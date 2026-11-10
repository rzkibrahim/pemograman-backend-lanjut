//go:build integration

package integration_test

// Test di file ini melayani request ke aplikasi LENGKAP — middleware auth,
// permission, service, repository, transaction, outbox — dengan database
// PostgreSQL sungguhan, tetapi TANPA membuka port jaringan (app.Test).
//
// Ia membuktikan tiga skenario utama endpoint pendaftaran kelas:
//   1. mahasiswa yang boleh mendaftar diterima (201) dan event-nya
//      benar-benar tertulis di outbox,
//   2. mahasiswa yang sudah terdaftar ditolak (409 ALREADY_ENROLLED), dan
//      kuota tidak boleh bocor ketika transaksi dibatalkan,
//   3. dua pendaftaran SERENTAK mahasiswa yang sama tidak boleh dua-duanya
//      lolos pemeriksaan batas 24 SKS (LockStudent FOR UPDATE).

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
	"latihan-fiber/app/service"
	"latihan-fiber/config"
	"latihan-fiber/helper"
	"latihan-fiber/route"
)

// buildAPI menyusun aplikasi penuh di atas pool integration test.
func buildAPI(t *testing.T) (*helper.JWTManager, *fiber.App) {
	t.Helper()
	jwtManager := helper.NewJWTManager("secret-integrasi-test-minimal-32char", "test", time.Minute)
	perms := helper.NewPermissionSet(map[string][]string{"user": {
		"course:list", "course:enroll", "course:drop", "course:swap",
	}})
	courseService := service.NewCourseService(
		repository.NewCourseRepository(pool),
		repository.NewOutboxRepository(pool),
		repository.NewTxManager(pool),
	)
	app := config.NewApp(slog.New(slog.NewTextHandler(io.Discard, nil)), route.Dependencies{
		Pool:          pool,
		JWT:           jwtManager,
		Permissions:   perms,
		CourseService: courseService,
	})
	return jwtManager, app
}

func itoa(n int) string { return strconv.Itoa(n) }

func TestEnrollAPI_Success(t *testing.T) {
	ctx := context.Background()
	jwtManager, app := buildAPI(t)
	student := seedStudent(t)
	course := seedCourse(t, 3, 30)

	token, err := jwtManager.Generate(student.ID, student.Username, "user")
	if err != nil {
		t.Fatalf("membuat token: %v", err)
	}

	doEnroll := func() (*http.Response, error) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+itoa(course.ID)+"/enroll", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		return app.Test(req)
	}

	resp, err := doEnroll()
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var body map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&body)
		t.Fatalf("status = %d, seharusnya 201 (body: %v)", resp.StatusCode, body)
	}

	// Bukti paling kuat berada di database, bukan di response.
	var enrolled int
	if err := pool.QueryRow(ctx,
		`SELECT enrolled_count FROM courses WHERE id = $1`, course.ID).Scan(&enrolled); err != nil || enrolled != 1 {
		t.Errorf("enrolled_count = %d (err=%v), seharusnya 1", enrolled, err)
	}
	var outbox int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM outbox WHERE event_type = 'enrollment.created'
		 AND (payload->>'user_id')::INT = $1 AND (payload->>'course_id')::INT = $2`,
		student.ID, course.ID).Scan(&outbox); err != nil || outbox != 1 {
		t.Errorf("outbox event = %d (err=%v), seharusnya 1", outbox, err)
	}

	// Pendaftaran kedua mulai dari database yang sudah berubah: 409.
	resp2, err := doEnroll()
	if err != nil {
		t.Fatalf("app.Test kedua: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusConflict {
		t.Errorf("pendaftaran ganda: status = %d, seharusnya 409", resp2.StatusCode)
	}

	// Kuota tidak boleh bocor: apa pun yang IncrementEnrolled lakukan
	// harus ikut dibatalkan oleh rollback ketika InsertEnrollment gagal.
	var enrolledAfter int
	if err := pool.QueryRow(ctx,
		`SELECT enrolled_count FROM courses WHERE id = $1`, course.ID).Scan(&enrolledAfter); err != nil || enrolledAfter != 1 {
		t.Errorf("enrolled_count setelah 409 = %d (err=%v), seharusnya tetap 1", enrolledAfter, err)
	}
}

func TestEnrollAPI_SameStudentRace(t *testing.T) {
	// ALUR deterministik yang membuktikan LockStudent FOR UPDATE:
	// Transaksi A mengunci baris mahasiswa, melunasi pendaftarannya ke
	// targetA seluruhnya, namun TIDAK di-commit. Transaksi B mencoba
	// LockStudent. DENGAN FOR UPDATE, B menunggu di langkah kunci sampai
	// A commit, lalu membaca total SKS 24 dan ditolak (24 + 3 = 27 > 24).
	// TANPA FOR UPDATE, B lolos dari langkah kunci selagi A belum commit:
	// ia membaca SKS 21 (nilai lama), meloloskan dirinya, dan batas 24 SKS
	// tertembus (21 + 3 + 3 = 27). Dua target KELAS BERBEDA dipilih agar
	// pelindung satu-satunya adalah kunci, bukan UNIQUE student-course.
	ctx := context.Background()
	student := seedStudent(t)

	// Skema membatasi credits <= 6 per kelas, jadi total 21 SKS dirangkai
	// dari tujuh kelas @ 3 SKS yang langsung didaftarkan via SQL.
	for i := 0; i < 7; i++ {
		heavy := seedCourse(t, 3, 30)
		if _, err := pool.Exec(ctx,
			`INSERT INTO enrollments (student_id, course_id) VALUES ($1, $2)`,
			student.ID, heavy.ID); err != nil {
			t.Fatalf("seed KRS %s: %v", heavy.Code, err)
		}
	}
	targetA := seedCourse(t, 3, 30)
	targetB := seedCourse(t, 3, 30)

	courses := repository.NewCourseRepository(pool)
	outbox := repository.NewOutboxRepository(pool)

	txA, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("membuka transaksi A: %v", err)
	}
	defer func() { _ = txA.Rollback(ctx) }()

	repoA := courses.WithTx(txA)
	if err := repoA.LockStudent(ctx, student.ID); err != nil {
		t.Fatalf("A LockStudent: %v", err)
	}
	// A mendaftar targetA lengkap, tetapi BELUM di-commit.
	if _, err := repoA.IncrementEnrolled(ctx, targetA.ID); err != nil {
		t.Fatalf("A IncrementEnrolled: %v", err)
	}
	if err := outbox.WithTx(txA).Insert(ctx, "enrollment.created", model.EnrollmentEvent{
		UserID: student.ID, CourseID: targetA.ID,
		CourseCode: targetA.Code, CourseName: targetA.Name,
	}); err != nil {
		t.Fatalf("A outbox: %v", err)
	}
	if err := repoA.InsertEnrollment(ctx, student.ID, targetA.ID); err != nil {
		t.Fatalf("A InsertEnrollment: %v", err)
	}

	// B berjalan di goroutine: begitu langkah kuncinya kembali (apa pun
	// hasilnya), lockB dikirim; alur B lalu dilanjutkan sampai selesai.
	lockB := make(chan struct{}, 1)
	bResult := make(chan error, 1)
	go func() {
		txB, err := pool.Begin(ctx)
		if err != nil {
			bResult <- err
			return
		}
		defer func() { _ = txB.Rollback(ctx) }()
		repoB := courses.WithTx(txB)
		// Langkah yang, DENGAN kunci, menunggu A; TANPA kunci, lewat seketika.
		if err := repoB.LockStudent(ctx, student.ID); err == nil {
			lockB <- struct{}{}
		}
		taken, err := repoB.SumCredits(ctx, student.ID)
		if err != nil {
			bResult <- err
			return
		}
		if !service.WithinCreditLimit(taken, targetB.Credits) {
			bResult <- errors.New("CREDIT_LIMIT_EXCEEDED")
			return
		}
		if _, err := repoB.IncrementEnrolled(ctx, targetB.ID); err != nil {
			bResult <- err
			return
		}
		if err := outbox.WithTx(txB).Insert(ctx, "enrollment.created", model.EnrollmentEvent{
			UserID: student.ID, CourseID: targetB.ID,
			CourseCode: targetB.Code, CourseName: targetB.Name,
		}); err != nil {
			bResult <- err
			return
		}
		bResult <- repoB.InsertEnrollment(ctx, student.ID, targetB.ID)
	}()

	// Selama A belum commit, B TIDAK BOLEH lolos LockStudent. Polling
	// 40 x 50ms: dengan kunci, lockB tidak pernah terkirim di jendela ini.
	passedEarly := false
	for i := 0; i < 40 && !passedEarly; i++ {
		select {
		case <-lockB:
			passedEarly = true
		case <-time.After(50 * time.Millisecond):
		}
	}
	if passedEarly {
		t.Errorf("B lolos dari LockStudent padahal transaksi A belum commit — " +
			"perilaku TANPA FOR UPDATE: dua pendaftaran 3 SKS bisa lolos bersamaan")
	}

	// Baru sekarang A di-commit; setelah itu B lewat dan HARUS ditolak.
	if err := txA.Commit(ctx); err != nil {
		t.Fatalf("A commit: %v", err)
	}
	if err := <-bResult; err == nil {
		t.Errorf("B lolos sekalipun A sudah commit — total SKS melewati batas 24")
	}

	// Bukti level database: targetB TIDAK boleh terisi; total SKS 24.
	var enrolled int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM enrollments WHERE student_id = $1 AND course_id = $2`,
		student.ID, targetB.ID).Scan(&enrolled); err != nil || enrolled != 0 {
		t.Errorf("baris enrollment targetB = %d (err=%v), seharusnya 0 — batas SKS tertembus", enrolled, err)
	}
	var total int
	if err := pool.QueryRow(ctx, `SELECT COALESCE(SUM(c.credits),0)
	 FROM enrollments e JOIN courses c ON c.id = e.course_id WHERE e.student_id = $1`,
		student.ID).Scan(&total); err != nil || total != 24 {
		t.Errorf("total SKS = %d (err=%v), seharusnya 24", total, err)
	}
}

func TestEnrollAPI_Unauthorized(t *testing.T) {
	_, app := buildAPI(t)
	course := seedCourse(t, 2, 10)

	// Tanpa header Authorization: middleware RequireAuth harus menolak
	// SEBELUM service tersentuh — bukti bahwa pelindung bekerja pada
	// lapisan yang benar (lihat middleware/jwt_route.go).
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+itoa(course.ID)+"/enroll", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, seharusnya 401", resp.StatusCode)
	}

	// Kuota tidak berubah.
	var enrolled int
	if err := pool.QueryRow(context.Background(),
		`SELECT enrolled_count FROM courses WHERE id = $1`, course.ID).Scan(&enrolled); err != nil || enrolled != 0 {
		t.Errorf("enrolled_count = %d (err=%v), seharusnya 0", enrolled, err)
	}
}

// =========================================================================
// TUGAS MANDIRI D.2 — Integration Tests
// =========================================================================

// TestSwapEndpoint_TargetFull membuktikan: jika kelas tujuan sudah penuh,
// endpoint swap mengembalikan 409 COURSE_FULL, mahasiswa TETAP terdaftar di kelas asal,
// enrolled_count kedua kelas tidak berubah, dan tidak ada event enrollment.swapped di outbox.
func TestSwapEndpoint_TargetFull(t *testing.T) {
	jwtManager, app := buildAPI(t)
	ctx := context.Background()
	repo := repository.NewCourseRepository(pool)

	student := seedStudent(t)
	courseFrom := seedCourse(t, 3, 30)
	courseTo := seedCourse(t, 3, 1) // Kuota 1, akan diisi sampai penuh

	// Daftarkan mahasiswa uji ke kelas asal
	if err := repo.InsertEnrollment(ctx, student.ID, courseFrom.ID); err != nil {
		t.Fatalf("seed enrollment student: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE courses SET enrolled_count = 1 WHERE id = $1`, courseFrom.ID); err != nil {
		t.Fatalf("update enrolled_count courseFrom: %v", err)
	}

	// Penuhi kelas tujuan oleh mahasiswa lain
	otherStudent := seedStudent(t)
	if err := repo.InsertEnrollment(ctx, otherStudent.ID, courseTo.ID); err != nil {
		t.Fatalf("seed enrollment otherStudent: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE courses SET enrolled_count = 1 WHERE id = $1`, courseTo.ID); err != nil {
		t.Fatalf("update enrolled_count courseTo: %v", err)
	}

	token, err := jwtManager.Generate(student.ID, student.Username, "user")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	bodyJSON := fmt.Sprintf(`{"from_course_id":%d,"to_course_id":%d}`, courseFrom.ID, courseTo.ID)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/enrollments/swap", strings.NewReader(bodyJSON))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status code = %d, seharusnya 409", resp.StatusCode)
	}

	var resBody struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&resBody); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if resBody.Code != "COURSE_FULL" {
		t.Errorf("response code = %q, seharusnya COURSE_FULL", resBody.Code)
	}

	// 1. Mahasiswa TETAP terdaftar di kelas asal
	enrolledFrom, err := repo.IsEnrolled(ctx, student.ID, courseFrom.ID)
	if err != nil {
		t.Fatalf("check IsEnrolled courseFrom: %v", err)
	}
	if !enrolledFrom {
		t.Errorf("mahasiswa harus tetap terdaftar di kelas asal")
	}

	// 2. Mahasiswa TIDAK terdaftar di kelas tujuan
	enrolledTo, err := repo.IsEnrolled(ctx, student.ID, courseTo.ID)
	if err != nil {
		t.Fatalf("check IsEnrolled courseTo: %v", err)
	}
	if enrolledTo {
		t.Errorf("mahasiswa seharusnya TIDAK terdaftar di kelas tujuan yang penuh")
	}

	// 3. enrolled_count kedua kelas tidak berubah (tetap 1 dan 1)
	cFrom, err := repo.FindByID(ctx, courseFrom.ID)
	if err != nil {
		t.Fatalf("FindByID courseFrom: %v", err)
	}
	if cFrom.EnrolledCount != 1 {
		t.Errorf("enrolled_count courseFrom = %d, seharusnya tetap 1", cFrom.EnrolledCount)
	}

	cTo, err := repo.FindByID(ctx, courseTo.ID)
	if err != nil {
		t.Fatalf("FindByID courseTo: %v", err)
	}
	if cTo.EnrolledCount != 1 {
		t.Errorf("enrolled_count courseTo = %d, seharusnya tetap 1", cTo.EnrolledCount)
	}

	// 4. Tidak ada event enrollment.swapped di outbox untuk mahasiswa ini
	var outboxEvents int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM outbox WHERE event_type = $1 AND (payload->>'user_id')::INT = $2`,
		model.EventEnrollmentSwapped, student.ID).Scan(&outboxEvents)
	if err != nil {
		t.Fatalf("query outbox: %v", err)
	}
	if outboxEvents != 0 {
		t.Errorf("outbox events = %d, seharusnya 0", outboxEvents)
	}
}

// TestDropEndpoint_Success membuktikan: berhasil keluar kelas menghasilkan 204,
// baris enrollment hilang, enrolled_count berkurang satu, dan outbox berisi tepat satu event enrollment.dropped.
func TestDropEndpoint_Success(t *testing.T) {
	jwtManager, app := buildAPI(t)
	ctx := context.Background()
	repo := repository.NewCourseRepository(pool)

	student := seedStudent(t)
	course := seedCourse(t, 3, 30)

	// Daftarkan mahasiswa ke kelas
	if err := repo.InsertEnrollment(ctx, student.ID, course.ID); err != nil {
		t.Fatalf("seed enrollment: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE courses SET enrolled_count = 1 WHERE id = $1`, course.ID); err != nil {
		t.Fatalf("update enrolled_count: %v", err)
	}

	token, err := jwtManager.Generate(student.ID, student.Username, "user")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/courses/"+strconv.Itoa(course.ID)+"/enroll", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	// 1. Response status harus 204 No Content
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status code = %d, seharusnya 204", resp.StatusCode)
	}

	// 2. Baris enrollment hilang
	enrolled, err := repo.IsEnrolled(ctx, student.ID, course.ID)
	if err != nil {
		t.Fatalf("check IsEnrolled: %v", err)
	}
	if enrolled {
		t.Errorf("baris enrollment seharusnya sudah dihapus")
	}

	// 3. enrolled_count berkurang satu (dari 1 menjadi 0)
	c, err := repo.FindByID(ctx, course.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if c.EnrolledCount != 0 {
		t.Errorf("enrolled_count = %d, seharusnya 0", c.EnrolledCount)
	}

	// 4. Outbox berisi tepat satu event enrollment.dropped
	var dropEvents int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM outbox WHERE event_type = $1 AND (payload->>'user_id')::INT = $2 AND (payload->>'course_id')::INT = $3`,
		model.EventEnrollmentDropped, student.ID, course.ID).Scan(&dropEvents)
	if err != nil {
		t.Fatalf("query outbox: %v", err)
	}
	if dropEvents != 1 {
		t.Errorf("event enrollment.dropped = %d, seharusnya tepat 1", dropEvents)
	}
}

