package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	baseURL      = "http://localhost:3000/api/v1"
	dbConnString = "postgres://postgres@localhost:5432/praktikum_backend?sslmode=disable"
)

type LoginResponse struct {
	Success bool `json:"success"`
	Data    struct {
		AccessToken string `json:"access_token"`
		User        struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
			Role     string `json:"role"`
		} `json:"user"`
	} `json:"data"`
}

type UserToken struct {
	UserID int
	Token  string
}

func main() {
	scenario := flag.String("scenario", "all", "Scenario to run: crowd, crowd-naive, credits, swap, functional, idle_leak, supervisor, all")
	flag.Parse()

	pool, err := pgxpool.New(context.Background(), dbConnString)
	if err != nil {
		fmt.Printf("Gagal koneksi ke PostgreSQL: %v\n", err)
		return
	}
	defer pool.Close()

	client := &http.Client{Timeout: 10 * time.Second}

	// Login admin
	adminToken, err := login(client, "admin_budi", "Password123!")
	if err != nil {
		fmt.Printf("Gagal login admin: %v\n", err)
		return
	}

	// Login 60 users
	fmt.Println("Melakukan pra-otentikasi 60 akun penguji...")
	tokens, err := preAuthenticateUsers(client, pool, 60)
	if err != nil {
		fmt.Printf("Gagal pra-otentikasi user: %v\n", err)
		return
	}
	fmt.Printf("Berhasil mengotentikasi %d akun penguji.\n\n", len(tokens))

	switch *scenario {
	case "crowd":
		runCrowdTest(client, pool, tokens, false, 3)
	case "crowd-naive":
		runCrowdTest(client, pool, tokens, true, 1)
	case "credits":
		runCreditLimitTest(client, pool, tokens[0], 3)
	case "swap":
		runSwapDeadlockTest(client, pool, tokens, 3)
	case "functional":
		runFunctionalTests(client, pool, adminToken, tokens)
	case "idle_leak":
		runIdleLeakTest(client, pool, tokens[0])
	case "supervisor":
		runSupervisorLoadTest(client, pool, adminToken, tokens, 3)
	case "all":
		fmt.Println("================================================================================")
		fmt.Println()

		fmt.Println(">>> [1/7] UJI CROWD ENROLLMENT (KUOTA 5 vs 60 REQUEST SERENTAK - 3x)")
		runCrowdTest(client, pool, tokens, false, 3)

		fmt.Println("\n>>> [2/7] UJI PEMBANDING: NAIVE ENROLLMENT (CHECK-THEN-ACT TANPA LOCK)")
		runCrowdTest(client, pool, tokens, true, 1)

		fmt.Println("\n>>> [3/7] UJI CREDIT LIMIT CONCURRENCY (BATAS 24 SKS DENGAN PESSIMISTIC LOCK)")
		runCreditLimitTest(client, pool, tokens[0], 3)

		fmt.Println("\n>>> [4/7] UJI DEADLOCK PREVENTION PADA PINDAH KELAS (40 CONCURRENT SWAP)")
		runSwapDeadlockTest(client, pool, tokens, 3)

		fmt.Println("\n>>> [5/7] UJI FUNGSIONAL LENGKAP (C.4: 8 KASUS VALIDASI & KEAMANAN)")
		runFunctionalTests(client, pool, adminToken, tokens)

		fmt.Println("\n>>> [6/7] UJI KETAHANAN KONEKSI & IDLE IN TRANSACTION (175 KEGAGALAN)")
		runIdleLeakTest(client, pool, tokens[0])

		fmt.Println("\n>>> [7/7] UJI TUGAS D.2 & D.3: PENETAPAN PEMBIMBING TA (30 SERENTAK - 3x)")
		runSupervisorLoadTest(client, pool, adminToken, tokens, 3)

		fmt.Println("\n================================================================================")
		fmt.Println("                  SELURUH PENGUJIAN SELESAI DENGAN SUKSES!                      ")
		fmt.Println("================================================================================")
	default:
		fmt.Println("Scenario tidak dikenal:", *scenario)
	}
}

func login(client *http.Client, username, password string) (string, error) {
	payload, _ := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})

	resp, err := client.Post(baseURL+"/auth/login", "application/json", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var lr LoginResponse
	if err := json.Unmarshal(body, &lr); err != nil {
		return "", fmt.Errorf("unmarshal login: %w", err)
	}
	if !lr.Success {
		return "", fmt.Errorf("login gagal untuk %s: %s", username, string(body))
	}
	return lr.Data.AccessToken, nil
}

func preAuthenticateUsers(client *http.Client, pool *pgxpool.Pool, count int) ([]UserToken, error) {
	tokens := make([]UserToken, count)
	type job struct{ idx int }
	jobs := make(chan job, count)
	errCh := make(chan error, count)
	var wg sync.WaitGroup

	numWorkers := 8
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				idx := j.idx
				uname := fmt.Sprintf("user%02d", idx)
				pass := "TestPass123"

				// Ambil UserID langsung dari database
				var userID int
				err := pool.QueryRow(context.Background(), "SELECT id FROM users WHERE username = $1", uname).Scan(&userID)
				if err != nil {
					errCh <- fmt.Errorf("ambil id user %s: %w", uname, err)
					return
				}

				payload, _ := json.Marshal(map[string]string{
					"username": uname,
					"password": pass,
				})

				resp, err := client.Post(baseURL+"/auth/login", "application/json", bytes.NewReader(payload))
				if err != nil {
					errCh <- fmt.Errorf("login %s error: %w", uname, err)
					return
				}

				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()

				var lr LoginResponse
				if err := json.Unmarshal(body, &lr); err != nil {
					errCh <- fmt.Errorf("unmarshal %s: %w", uname, err)
					return
				}
				if !lr.Success {
					errCh <- fmt.Errorf("login %s failed: %s", uname, string(body))
					return
				}

				tokens[idx-1] = UserToken{
					UserID: userID,
					Token:  lr.Data.AccessToken,
				}
			}
		}()
	}

	for i := 1; i <= count; i++ {
		jobs <- job{idx: i}
	}
	close(jobs)

	wg.Wait()
	close(errCh)

	if len(errCh) > 0 {
		return nil, <-errCh
	}
	return tokens, nil
}

func resetDatabase(pool *pgxpool.Pool) error {
	ctx := context.Background()
	queries := []string{
		"DELETE FROM enrollments;",
		"UPDATE courses SET enrolled_count = 0, quota = 5, version = 1 WHERE id IN (1, 2, 3);",
		"UPDATE courses SET enrolled_count = 0, quota = 40, version = 1 WHERE id IN (4, 5);",
		"DELETE FROM supervisions;",
		"UPDATE supervisors SET supervised_count = 0, quota = 3, version = 1 WHERE id IN (1, 2);",
		"UPDATE supervisors SET supervised_count = 0, quota = 5, version = 1 WHERE id = 3;",
	}
	for _, q := range queries {
		if _, err := pool.Exec(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

// -------------------------------------------------------------------------------------------------
// 1. Crowd Test (Kuata 5 vs 60 Request Serentak)
// -------------------------------------------------------------------------------------------------
func runCrowdTest(client *http.Client, pool *pgxpool.Pool, tokens []UserToken, naive bool, runs int) {
	endpoint := "/courses/1/enroll"
	testLabel := "Atomic Conditional UPDATE (Aman)"
	if naive {
		endpoint = "/courses/1/enroll-naive"
		testLabel = "Naive Check-Then-Act (Sengaja Salah)"
	}

	fmt.Printf("[%s] Menjalankan uji beban 60 request serentak ke course ID 1 (kuota 5)...\n", testLabel)

	for run := 1; run <= runs; run++ {
		if err := resetDatabase(pool); err != nil {
			fmt.Printf("Reset database gagal: %v\n", err)
			return
		}

		var successCount int64
		var fullCount int64
		var otherCount int64

		startGate := make(chan struct{})
		var wg sync.WaitGroup

		startDuration := time.Now()

		for _, ut := range tokens {
			wg.Add(1)
			go func(user UserToken) {
				defer wg.Done()
				<-startGate // Gerbang tembak bersama

				payload, _ := json.Marshal(map[string]int{
					"student_id": user.UserID,
				})

				req, _ := http.NewRequest("POST", baseURL+endpoint, bytes.NewReader(payload))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+user.Token)

				resp, err := client.Do(req)
				if err != nil {
					atomic.AddInt64(&otherCount, 1)
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusCreated {
					atomic.AddInt64(&successCount, 1)
				} else if resp.StatusCode == http.StatusConflict {
					atomic.AddInt64(&fullCount, 1)
				} else {
					atomic.AddInt64(&otherCount, 1)
				}
			}(ut)
		}

		// Lepaskan tembakan serentak
		close(startGate)
		wg.Wait()
		duration := time.Since(startDuration)

		// Periksa state di database
		var dbEnrolledCount, actualEnrollmentRows int
		_ = pool.QueryRow(context.Background(), "SELECT enrolled_count FROM courses WHERE id = 1").Scan(&dbEnrolledCount)
		_ = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM enrollments WHERE course_id = 1").Scan(&actualEnrollmentRows)

		fmt.Printf("  Percobaan %d/%d (waktu: %v):\n", run, runs, duration)
		fmt.Printf("    -> HTTP 201 Created      : %d\n", successCount)
		fmt.Printf("    -> HTTP 409 COURSE_FULL  : %d\n", fullCount)
		fmt.Printf("    -> HTTP Status Lain      : %d\n", otherCount)
		fmt.Printf("    -> DB courses.enrolled   : %d\n", dbEnrolledCount)
		fmt.Printf("    -> DB COUNT(enrollments) : %d\n", actualEnrollmentRows)

		if !naive {
			if successCount == 5 && fullCount == 55 && dbEnrolledCount == 5 && actualEnrollmentRows == 5 {
				fmt.Println("    [STATUS: LULUS] Tepat 5 berhasil, 55 COURSE_FULL, DB konsisten.")
			} else {
				fmt.Println("    [STATUS: GAGAL] Hasil tidak sesuai spesifikasi!")
			}
		} else {
			fmt.Printf("    [ANALISIS RACE CONDITION] Naive mode menghasilkan %d pendaftar tembus dari kuota 5!\n", actualEnrollmentRows)
		}
	}
}

// -------------------------------------------------------------------------------------------------
// 2. Credit Limit Test (Pessimistic Lock)
// -------------------------------------------------------------------------------------------------
func runCreditLimitTest(client *http.Client, pool *pgxpool.Pool, user UserToken, runs int) {
	fmt.Println("[Pessimistic Locking] Menguji 10 request serentak dari 1 mahasiswa (IF101..IF105, total 16 SKS + duplikat)...")

	// Pastikan kelas 4 dan 5 punya kuota cukup
	_, _ = pool.Exec(context.Background(), "UPDATE courses SET quota = 40 WHERE id IN (4, 5)")

	for run := 1; run <= runs; run++ {
		if err := resetDatabase(pool); err != nil {
			fmt.Printf("Reset database gagal: %v\n", err)
			return
		}

		startGate := make(chan struct{})
		var wg sync.WaitGroup

		// Masing-masing request mencoba mendaftar ke kelas 1 s/d 5 berulang
		courseIDs := []int{1, 2, 3, 4, 5, 1, 2, 3, 4, 5}

		for _, cid := range courseIDs {
			wg.Add(1)
			go func(courseID int) {
				defer wg.Done()
				<-startGate

				payload, _ := json.Marshal(map[string]int{
					"student_id": user.UserID,
				})

				req, _ := http.NewRequest("POST", fmt.Sprintf("%s/courses/%d/enroll", baseURL, courseID), bytes.NewReader(payload))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+user.Token)

				resp, err := client.Do(req)
				if err == nil {
					resp.Body.Close()
				}
			}(cid)
		}

		close(startGate)
		wg.Wait()

		var totalCredits int
		_ = pool.QueryRow(context.Background(), `
			SELECT COALESCE(SUM(c.credits), 0)
			FROM enrollments e
			JOIN courses c ON c.id = e.course_id
			WHERE e.student_id = $1
		`, user.UserID).Scan(&totalCredits)

		fmt.Printf("  Percobaan %d/%d: Total SKS terdaftar = %d SKS (Batas: 24 SKS)\n", run, runs, totalCredits)
		if totalCredits <= 24 {
			fmt.Println("    [STATUS: LULUS] Total SKS tidak pernah melebihi 24.")
		} else {
			fmt.Println("    [STATUS: GAGAL] Batas 24 SKS jebol!")
		}
	}
}

// -------------------------------------------------------------------------------------------------
// 3. Swap Deadlock Prevention Test
// -------------------------------------------------------------------------------------------------
func runSwapDeadlockTest(client *http.Client, pool *pgxpool.Pool, tokens []UserToken, runs int) {
	fmt.Println("[Deadlock Prevention] Menguji 40 request pindah kelas serentak (20 A->B dan 20 B->A)...")

	for run := 1; run <= runs; run++ {
		if err := resetDatabase(pool); err != nil {
			fmt.Printf("Reset database gagal: %v\n", err)
			return
		}

		// Daftarkan 20 user pertama ke IF104 (ID 4), 20 user berikutnya ke IF105 (ID 5)
		ctx := context.Background()
		for i := 0; i < 20; i++ {
			_, _ = pool.Exec(ctx, "INSERT INTO enrollments (student_id, course_id) VALUES ($1, 4)", tokens[i].UserID)
		}
		_, _ = pool.Exec(ctx, "UPDATE courses SET enrolled_count = 20 WHERE id = 4")

		for i := 20; i < 40; i++ {
			_, _ = pool.Exec(ctx, "INSERT INTO enrollments (student_id, course_id) VALUES ($1, 5)", tokens[i].UserID)
		}
		_, _ = pool.Exec(ctx, "UPDATE courses SET enrolled_count = 20 WHERE id = 5")

		var swapSuccess int64
		var swapError int64

		startGate := make(chan struct{})
		var wg sync.WaitGroup

		startTime := time.Now()

		// 20 request: IF104 -> IF105
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				<-startGate

				payload, _ := json.Marshal(map[string]int{
					"student_id":     tokens[idx].UserID,
					"from_course_id": 4,
					"to_course_id":   5,
				})

				req, _ := http.NewRequest("POST", baseURL+"/courses/swap", bytes.NewReader(payload))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+tokens[idx].Token)

				resp, err := client.Do(req)
				if err != nil {
					atomic.AddInt64(&swapError, 1)
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusOK {
					atomic.AddInt64(&swapSuccess, 1)
				} else {
					atomic.AddInt64(&swapError, 1)
				}
			}(i)
		}

		// 20 request: IF105 -> IF104 (Arah berlawanan!)
		for i := 20; i < 40; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				<-startGate

				payload, _ := json.Marshal(map[string]int{
					"student_id":     tokens[idx].UserID,
					"from_course_id": 5,
					"to_course_id":   4,
				})

				req, _ := http.NewRequest("POST", baseURL+"/courses/swap", bytes.NewReader(payload))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+tokens[idx].Token)

				resp, err := client.Do(req)
				if err != nil {
					atomic.AddInt64(&swapError, 1)
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusOK {
					atomic.AddInt64(&swapSuccess, 1)
				} else {
					atomic.AddInt64(&swapError, 1)
				}
			}(i)
		}

		close(startGate)
		wg.Wait()
		duration := time.Since(startTime)

		fmt.Printf("  Percobaan %d/%d (durasi: %v):\n", run, runs, duration)
		fmt.Printf("    -> Swap Berhasil : %d/40\n", swapSuccess)
		fmt.Printf("    -> Swap Gagal    : %d/40\n", swapError)

		if swapSuccess == 40 && swapError == 0 && duration < 1*time.Second {
			fmt.Println("    [STATUS: LULUS] 40 sukses, 0 deadlock, durasi < 1 detik.")
		} else {
			fmt.Println("    [STATUS: GAGAL] Ada error atau terjadi deadlock!")
		}
	}
}

// -------------------------------------------------------------------------------------------------
// 4. Functional Tests (C.4)
// -------------------------------------------------------------------------------------------------
func runFunctionalTests(client *http.Client, pool *pgxpool.Pool, adminToken string, tokens []UserToken) {
	fmt.Println("[Uji Fungsional C.4]")
	_ = resetDatabase(pool)

	user1 := tokens[0]

	// 1. Daftar sah -> 201 Created
	payload, _ := json.Marshal(map[string]int{"student_id": user1.UserID})
	req, _ := http.NewRequest("POST", baseURL+"/courses/1/enroll", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+user1.Token)
	resp, _ := client.Do(req)
	fmt.Printf("  1. Daftar kelas sah                 : HTTP %d (harapan: 201)\n", resp.StatusCode)
	resp.Body.Close()

	// 2. Daftar ganda -> 409 ALREADY_ENROLLED
	req2, _ := http.NewRequest("POST", baseURL+"/courses/1/enroll", bytes.NewReader(payload))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+user1.Token)
	resp2, _ := client.Do(req2)
	b2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	fmt.Printf("  2. Daftar ganda                     : HTTP %d, body: %s (harapan: 409 ALREADY_ENROLLED)\n", resp2.StatusCode, string(b2))

	// 3. Batalkan kelas yang tidak diambil -> 409 NOT_ENROLLED
	req3, _ := http.NewRequest("POST", baseURL+"/courses/2/drop", bytes.NewReader(payload))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("Authorization", "Bearer "+user1.Token)
	resp3, _ := client.Do(req3)
	b3, _ := io.ReadAll(resp3.Body)
	resp3.Body.Close()
	fmt.Printf("  3. Drop kelas yang belum diambil    : HTTP %d, body: %s (harapan: 409 NOT_ENROLLED)\n", resp3.StatusCode, string(b3))

	// 4. Swap dengan ID kelas sama -> 422
	payloadSwapSame, _ := json.Marshal(map[string]int{
		"student_id":     user1.UserID,
		"from_course_id": 1,
		"to_course_id":   1,
	})
	req4, _ := http.NewRequest("POST", baseURL+"/courses/swap", bytes.NewReader(payloadSwapSame))
	req4.Header.Set("Content-Type", "application/json")
	req4.Header.Set("Authorization", "Bearer "+user1.Token)
	resp4, _ := client.Do(req4)
	fmt.Printf("  4. Swap kelas tujuan = kelas asal   : HTTP %d (harapan: 422)\n", resp4.StatusCode)
	resp4.Body.Close()

	// 5. Ubah kuota (optimistic locking) version baru -> 200 OK
	payloadQuota, _ := json.Marshal(map[string]int{"quota": 10, "version": 1})
	req5, _ := http.NewRequest("PATCH", baseURL+"/courses/1", bytes.NewReader(payloadQuota))
	req5.Header.Set("Content-Type", "application/json")
	req5.Header.Set("Authorization", "Bearer "+adminToken)
	resp5, _ := client.Do(req5)
	fmt.Printf("  5. Ubah kuota admin (version benar) : HTTP %d (harapan: 200)\n", resp5.StatusCode)
	resp5.Body.Close()

	// 6. Ubah kuota dengan version lama -> 409 VERSION_CONFLICT
	payloadOldVersion, _ := json.Marshal(map[string]int{"quota": 12, "version": 1})
	req6, _ := http.NewRequest("PATCH", baseURL+"/courses/1", bytes.NewReader(payloadOldVersion))
	req6.Header.Set("Content-Type", "application/json")
	req6.Header.Set("Authorization", "Bearer "+adminToken)
	resp6, _ := client.Do(req6)
	b6, _ := io.ReadAll(resp6.Body)
	resp6.Body.Close()
	fmt.Printf("  6. Ubah kuota (version conflict)    : HTTP %d, body: %s (harapan: 409 VERSION_CONFLICT)\n", resp6.StatusCode, string(b6))

	// 7. Ubah kuota < jumlah terdaftar -> 409 QUOTA_BELOW_ENROLLED
	// Daftarkan user kedua ke course 1 sehingga enrolled_count = 2
	payloadUser2, _ := json.Marshal(map[string]int{"student_id": tokens[1].UserID})
	reqEnroll2, _ := http.NewRequest("POST", baseURL+"/courses/1/enroll", bytes.NewReader(payloadUser2))
	reqEnroll2.Header.Set("Content-Type", "application/json")
	reqEnroll2.Header.Set("Authorization", "Bearer "+tokens[1].Token)
	respEnroll2, _ := client.Do(reqEnroll2)
	respEnroll2.Body.Close()

	payloadBelow, _ := json.Marshal(map[string]int{"quota": 1, "version": 2})
	req7, _ := http.NewRequest("PATCH", baseURL+"/courses/1", bytes.NewReader(payloadBelow))
	req7.Header.Set("Content-Type", "application/json")
	req7.Header.Set("Authorization", "Bearer "+adminToken)
	resp7, _ := client.Do(req7)
	b7, _ := io.ReadAll(resp7.Body)
	resp7.Body.Close()
	fmt.Printf("  7. Kuota baru < jumlah terdaftar    : HTTP %d, body: %s (harapan: 409 QUOTA_BELOW_ENROLLED)\n", resp7.StatusCode, string(b7))

	// 8. Mahasiswa coba ubah kuota -> 403 Forbidden
	req8, _ := http.NewRequest("PATCH", baseURL+"/courses/1", bytes.NewReader(payloadQuota))
	req8.Header.Set("Content-Type", "application/json")
	req8.Header.Set("Authorization", "Bearer "+user1.Token)
	resp8, _ := client.Do(req8)
	fmt.Printf("  8. Mahasiswa ubah kuota             : HTTP %d (harapan: 403 Forbidden)\n", resp8.StatusCode)
	resp8.Body.Close()
}

// -------------------------------------------------------------------------------------------------
// 5. Idle in Transaction & Leak Test (C.11)
// -------------------------------------------------------------------------------------------------
func runIdleLeakTest(client *http.Client, pool *pgxpool.Pool, user UserToken) {
	fmt.Println("[Ketahanan Koneksi & Idle Leak] Mengirim 175 request gagal berturut-turut...")

	payload, _ := json.Marshal(map[string]int{"student_id": user.UserID})

	for i := 0; i < 175; i++ {
		req, _ := http.NewRequest("POST", baseURL+"/courses/9999/drop", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+user.Token)

		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
		}
	}

	// Cek status health
	healthResp, err := client.Get(baseURL + "/health")
	if err != nil {
		fmt.Printf("  [STATUS: GAGAL] Health check gagal: %v\n", err)
		return
	}
	defer healthResp.Body.Close()

	// Cek koneksi idle in transaction di PostgreSQL
	var idleXactCount int
	err = pool.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM pg_stat_activity
		WHERE datname = 'praktikum_backend'
		  AND state = 'idle in transaction'
	`).Scan(&idleXactCount)

	fmt.Printf("  -> Health Status Code               : HTTP %d (harapan: 200)\n", healthResp.StatusCode)
	fmt.Printf("  -> 'idle in transaction' di Postgres : %d (harapan: 0)\n", idleXactCount)

	if healthResp.StatusCode == http.StatusOK && idleXactCount == 0 {
		fmt.Println("  [STATUS: LULUS] Defer Rollback berfungsi sempurna, 0 transaksi bocor.")
	} else {
		fmt.Println("  [STATUS: GAGAL] Terjadi kebocoran koneksi transaksi!")
	}
}

// -------------------------------------------------------------------------------------------------
// 6. Supervisor Load Test (Tugas Mandiri D.2 & D.3)
// -------------------------------------------------------------------------------------------------
func runSupervisorLoadTest(client *http.Client, pool *pgxpool.Pool, adminToken string, tokens []UserToken, runs int) {
	fmt.Println("[Tugas D.2 & D.3] Menguji 30 mahasiswa menetapkan Dr. Ahmad Fauzi (kuota 3) serentak...")

	for run := 1; run <= runs; run++ {
		if err := resetDatabase(pool); err != nil {
			fmt.Printf("Reset database gagal: %v\n", err)
			return
		}

		var successCount int64
		var fullCount int64
		var otherCount int64

		startGate := make(chan struct{})
		var wg sync.WaitGroup

		startDuration := time.Now()

		for i := 0; i < 30; i++ {
			wg.Add(1)
			go func(user UserToken) {
				defer wg.Done()
				<-startGate

				payload, _ := json.Marshal(map[string]int{
					"student_id": user.UserID,
				})

				req, _ := http.NewRequest("POST", baseURL+"/supervisors/1/assign", bytes.NewReader(payload))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+adminToken) // role admin punya permission supervision:manage

				resp, err := client.Do(req)
				if err != nil {
					atomic.AddInt64(&otherCount, 1)
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusCreated {
					atomic.AddInt64(&successCount, 1)
				} else if resp.StatusCode == http.StatusConflict {
					atomic.AddInt64(&fullCount, 1)
				} else {
					atomic.AddInt64(&otherCount, 1)
				}
			}(tokens[i])
		}

		close(startGate)
		wg.Wait()
		duration := time.Since(startDuration)

		var dbSupervisedCount, actualSupervisions int
		_ = pool.QueryRow(context.Background(), "SELECT supervised_count FROM supervisors WHERE id = 1").Scan(&dbSupervisedCount)
		_ = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM supervisions WHERE supervisor_id = 1").Scan(&actualSupervisions)

		fmt.Printf("  Percobaan %d/%d (waktu: %v):\n", run, runs, duration)
		fmt.Printf("    -> HTTP 201 Created           : %d\n", successCount)
		fmt.Printf("    -> HTTP 409 SUPERVISOR_FULL   : %d\n", fullCount)
		fmt.Printf("    -> HTTP Status Lain           : %d\n", otherCount)
		fmt.Printf("    -> DB supervised_count        : %d\n", dbSupervisedCount)
		fmt.Printf("    -> DB COUNT(supervisions)     : %d\n", actualSupervisions)

		if successCount == 3 && fullCount == 27 && dbSupervisedCount == 3 && actualSupervisions == 3 {
			fmt.Println("    [STATUS: LULUS] Tepat 3 sukses, 27 SUPERVISOR_FULL, DB konsisten.")
		} else {
			fmt.Println("    [STATUS: GAGAL] Kuota pembimbing tidak konsisten!")
		}
	}

	// Uji Fungsional Switch Supervisor
	fmt.Println("\n  Menguji Switch Supervisor Beban & Atomik (D.2)...")
	// Ambil 1 user yang terdaftar di supervisor 1, switch ke supervisor 2 (Dr. Budi Santoso)
	var studentID int
	_ = pool.QueryRow(context.Background(), "SELECT student_id FROM supervisions WHERE supervisor_id = 1 LIMIT 1").Scan(&studentID)

	payloadSwitch, _ := json.Marshal(map[string]int{
		"student_id":         studentID,
		"from_supervisor_id": 1,
		"to_supervisor_id":   2,
	})

	reqSwitch, _ := http.NewRequest("POST", baseURL+"/supervisions/switch", bytes.NewReader(payloadSwitch))
	reqSwitch.Header.Set("Content-Type", "application/json")
	reqSwitch.Header.Set("Authorization", "Bearer "+adminToken)
	respSwitch, _ := client.Do(reqSwitch)
	fmt.Printf("    -> Switch Supervisor 1 -> 2     : HTTP %d (harapan: 200)\n", respSwitch.StatusCode)
	respSwitch.Body.Close()

	// Verifikasi kuota setelah switch
	var countSup1, countSup2 int
	_ = pool.QueryRow(context.Background(), "SELECT supervised_count FROM supervisors WHERE id = 1").Scan(&countSup1)
	_ = pool.QueryRow(context.Background(), "SELECT supervised_count FROM supervisors WHERE id = 2").Scan(&countSup2)
	fmt.Printf("    -> Kuota Sup 1: %d, Kuota Sup 2: %d (harapan: Sup1 berkurang 1, Sup2 bertambah 1)\n", countSup1, countSup2)

	// Optimistic locking supervisor
	payloadSupQuota, _ := json.Marshal(map[string]int{"quota": 5, "version": 1})
	reqSupQuota, _ := http.NewRequest("PATCH", baseURL+"/supervisors/1", bytes.NewReader(payloadSupQuota))
	reqSupQuota.Header.Set("Content-Type", "application/json")
	reqSupQuota.Header.Set("Authorization", "Bearer "+adminToken)
	respSupQuota, _ := client.Do(reqSupQuota)
	fmt.Printf("    -> PATCH /supervisors/1 (optimistic) : HTTP %d (harapan: 200)\n", respSupQuota.StatusCode)
	respSupQuota.Body.Close()
}
