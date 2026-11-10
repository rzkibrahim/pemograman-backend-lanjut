//go:build integration

package integration_test

// Test di file ini memanggil repository langsung (tanpa HTTP) untuk
// memastikan kueri SQL benar-benar berperilaku sesuai janjinya di atas
// PostgreSQL sungguhan — bukan pada fake di memory.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
)

func newCourseRepo() repository.CourseRepository {
	return repository.NewCourseRepository(pool)
}

// TestIncrementEnrolled_StopsAtQuota membuktikan UPDATE bersyarat atomik:
// setiap kelas berkapasitas 2 hanya boleh mengeluarkan 2 kursi, request
// ketiga harus ditolak dengan ErrCourseFull — meski diberi jeda panjang.
func TestIncrementEnrolled_StopsAtQuota(t *testing.T) {
	ctx := context.Background()
	repo := newCourseRepo()
	course := seedCourse(t, 2, 2)

	for i := 1; i <= 2; i++ {
		c, err := repo.IncrementEnrolled(ctx, course.ID)
		if err != nil {
			t.Fatalf("kursi ke-%d harus tersedia, dapat error: %v", i, err)
		}
		if c.EnrolledCount != i {
			t.Errorf("kursi ke-%d: enrolled_count = %d, seharusnya %d", i, c.EnrolledCount, i)
		}
	}

	// Kursi ketiga: kuota penuh, query tidak meng-update baris apa pun.
	// Beri jeda agar test ini TIDAK bisa lolos hanya karena cepat.
	time.Sleep(50 * time.Millisecond)
	if _, err := repo.IncrementEnrolled(ctx, course.ID); !errors.Is(err, repository.ErrCourseFull) {
		t.Errorf("kursi ketiga: error = %v, seharusnya repository.ErrCourseFull", err)
	}
}

// TestInsertEnrollment_Duplicate membuktikan UNIQUE constraint pada
// enrollments: baris kedua dengan pasangan {student, course} yang sama
// dipugar menjadi ErrAlreadyEnrolled (terjemahan isUniqueViolation).
func TestInsertEnrollment_Duplicate(t *testing.T) {
	ctx := context.Background()
	repo := newCourseRepo()
	student := seedStudent(t)
	course := seedCourse(t, 3, 30)

	if err := repo.InsertEnrollment(ctx, student.ID, course.ID); err != nil {
		t.Fatalf("pendaftaran pertama tidak seharusnya gagal: %v", err)
	}
	err := repo.InsertEnrollment(ctx, student.ID, course.ID)
	if !errors.Is(err, repository.ErrAlreadyEnrolled) {
		t.Errorf("pendaftaran kedua: error = %v, seharusnya repository.ErrAlreadyEnrolled", err)
	}
}

// TestWithTx_Rollback membuktikan commit dan rollback: sebuah enrollment
// yang dibuat DI DALAM transaksi lalu dibatalkan (rollback) tidak boleh
// pernah terlihat oleh pembaca di luar transaksi.
func TestWithTx_Rollback(t *testing.T) {
	ctx := context.Background()
	repo := newCourseRepo()
	student := seedStudent(t)
	course := seedCourse(t, 2, 10)

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("membuka transaksi: %v", err)
	}
	txRepo := repo.WithTx(tx)
	if err := txRepo.InsertEnrollment(ctx, student.ID, course.ID); err != nil {
		t.Fatalf("menyisipkan di dalam transaksi: %v", err)
	}
	// Terlihat oleh pembaca yang SATU transaksi (konsistensi baca-sendiri).
	if enrolled, err := repo.IsEnrolled(ctx, student.ID, course.ID); err != nil {
		t.Fatalf("memeriksa di luar transaksi: %v", err)
	} else if enrolled {
		t.Fatalf("enrollment belum di-commit, tidak boleh terlihat")
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if enrolled, err := repo.IsEnrolled(ctx, student.ID, course.ID); err != nil {
		t.Fatalf("memeriksa setelah rollback: %v", err)
	} else if enrolled {
		t.Errorf("enrollment masih ada setelah rollback")
	}
}

// TestClaim_OnlyOnce membuktikan idempotency Claim via ON CONFLICT DO
// NOTHING: pemanggilan kedua untuk outbox_id + user_id yang sama tidak
// membuat baris baru, tetapi mengembalikan false.
func TestClaim_OnlyOnce(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewNotificationRepository(pool)

	student := seedStudent(t)
	// Claim butuh outbox_id yang valid (foreign key).
	var outboxID int64
	err := pool.QueryRow(ctx,
		`INSERT INTO outbox (event_type, payload) VALUES ('enrollment.created', $1) RETURNING id`,
		[]byte(fmt.Sprintf(`{"user_id":%d}`, student.ID)),
	).Scan(&outboxID)
	if err != nil {
		t.Fatalf("seed outbox: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM notifications WHERE outbox_id = $1`, outboxID)
		_, _ = pool.Exec(ctx, `DELETE FROM outbox WHERE id = $1`, outboxID)
	})

	n := model.Notification{
		OutboxID: outboxID, UserID: student.ID, Channel: "email",
		Subject: "uji", Body: "uji claim",
	}
	first, err := repo.Claim(ctx, n)
	if err != nil {
		t.Fatalf("claim pertama: %v", err)
	}
	if !first {
		t.Fatal("claim pertama seharusnya berhasil (true)")
	}
	time.Sleep(50 * time.Millisecond)
	second, err := repo.Claim(ctx, n)
	if err != nil {
		t.Fatalf("claim kedua: %v", err)
	}
	if second {
		t.Error("claim kedua seharusnya false: idempotency dilanggar")
	}
}

func init() { _ = pgx.ErrNoRows } // jaga import pgx tetap terpakai
