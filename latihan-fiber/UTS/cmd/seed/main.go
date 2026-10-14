// Package main - cmd/seed/main.go
// Seeder database untuk SIAKAD Mini:
// Mengisi minimal 1 admin, 20 mahasiswa, 10 mata kuliah, dan data KRS awal.
// Seluruh password disimpan dalam bentuk hash bcrypt yang aman.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"siakad-mini/config"
	"siakad-mini/database"
	"siakad-mini/helper"
)

type studentSeed struct {
	NIM      string
	Nama     string
	Email    string
	Prodi    string
	Angkatan int
	IPK      float64
}

type courseSeed struct {
	KodeMK   string
	NamaMK   string
	SKS      int
	Semester int
	Kuota    int
}

func main() {
	log.Println("Memulai proses seeding database SIAKAD Mini...")

	config.LoadEnv()
	logger := config.NewLogger()
	pool, err := database.NewPostgresPool(logger)
	if err != nil {
		log.Fatalf("Gagal terhubung ke basis data: %v", err)
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Eksekusi migrasi skema terlebih dahulu untuk memastikan tabel tersedia
	if err := runMigrations(ctx, pool); err != nil {
		log.Fatalf("Gagal menjalankan migrasi: %v", err)
	}

	// 2. Bersihkan data lama agar seeder bersifat idempotent (bersih saat dijalankan ulang)
	log.Println("Membersihkan data lama...")
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE enrollments, students, courses, users RESTART IDENTITY CASCADE;")

	// 3. Seed 1 Akun Admin
	adminEmail := "admin@siakad.ac.id"
	adminPass := "admin12345"
	adminHash, err := helper.HashPassword(adminPass)
	if err != nil {
		log.Fatalf("Gagal hash password admin: %v", err)
	}

	var adminID int
	err = pool.QueryRow(ctx, `
		INSERT INTO users (email, password, role, created_at, updated_at)
		VALUES ($1, $2, 'admin', NOW(), NOW())
		RETURNING id;
	`, adminEmail, adminHash).Scan(&adminID)
	if err != nil {
		log.Fatalf("Gagal seed admin: %v", err)
	}
	log.Printf("✓ Admin berhasil dibuat: %s (Password: %s)", adminEmail, adminPass)

	// 4. Seed 20 Mahasiswa (dan akun pengguna masing-masing)
	studentsData := []studentSeed{
		{"187221000001", "Rina Putri", "rina.putri@mhs.siakad.ac.id", "Sistem Informasi", 2022, 3.85},
		{"187221000002", "Budi Santoso", "budi.santoso@mhs.siakad.ac.id", "Sistem Informasi", 2022, 3.45},
		{"187221000003", "Dewi Lestari", "dewi.lestari@mhs.siakad.ac.id", "Teknik Informatika", 2023, 3.10},
		{"187221000004", "Ahmad Fauzi", "ahmad.fauzi@mhs.siakad.ac.id", "Teknik Informatika", 2021, 2.80},
		{"187221000005", "Siti Nurhaliza", "siti.nurhaliza@mhs.siakad.ac.id", "Teknologi Sains Data", 2023, 2.35},
		{"187221000006", "Rizki Pratama", "rizki.pratama@mhs.siakad.ac.id", "Sistem Informasi", 2022, 3.90},
		{"187221000007", "Maya Anggraini", "maya.anggraini@mhs.siakad.ac.id", "Teknik Informatika", 2022, 2.95},
		{"187221000008", "Fajar Hidayat", "fajar.hidayat@mhs.siakad.ac.id", "Teknologi Sains Data", 2021, 2.40},
		{"187221000009", "Anisa Rahmawati", "anisa.rahma@mhs.siakad.ac.id", "Sistem Informasi", 2023, 3.65},
		{"187221000010", "Dimas Setiawan", "dimas.setiawan@mhs.siakad.ac.id", "Teknik Informatika", 2022, 3.25},
		{"187221000011", "Eka Wahyuni", "eka.wahyuni@mhs.siakad.ac.id", "Sistem Informasi", 2023, 2.75},
		{"187221000012", "Gilang Ramadhan", "gilang.ramadhan@mhs.siakad.ac.id", "Teknik Informatika", 2021, 3.50},
		{"187221000013", "Hani Safitri", "hani.safitri@mhs.siakad.ac.id", "Teknologi Sains Data", 2022, 3.70},
		{"187221000014", "Indra Kusuma", "indra.kusuma@mhs.siakad.ac.id", "Sistem Informasi", 2023, 2.15},
		{"187221000015", "Joko Widodo", "joko.widodo@mhs.siakad.ac.id", "Teknik Informatika", 2022, 3.05},
		{"187221000016", "Kartika Sari", "kartika.sari@mhs.siakad.ac.id", "Teknologi Sains Data", 2021, 2.60},
		{"187221000017", "Lukman Hakim", "lukman.hakim@mhs.siakad.ac.id", "Sistem Informasi", 2022, 3.30},
		{"187221000018", "Mega Utami", "mega.utami@mhs.siakad.ac.id", "Teknik Informatika", 2023, 3.60},
		{"187221000019", "Nanda Pratama", "nanda.pratama@mhs.siakad.ac.id", "Teknologi Sains Data", 2022, 2.85},
		{"187221000020", "Oscar Permana", "oscar.permana@mhs.siakad.ac.id", "Sistem Informasi", 2021, 3.40},
	}

	studentIDs := make([]int, 0, len(studentsData))
	for _, s := range studentsData {
		// Password default mahasiswa adalah NIM
		passHash, err := helper.HashPassword(s.NIM)
		if err != nil {
			log.Fatalf("Gagal hash password mahasiswa %s: %v", s.NIM, err)
		}

		var uID int
		err = pool.QueryRow(ctx, `
			INSERT INTO users (email, password, role, created_at, updated_at)
			VALUES ($1, $2, 'mahasiswa', NOW(), NOW())
			RETURNING id;
		`, s.Email, passHash).Scan(&uID)
		if err != nil {
			log.Fatalf("Gagal buat user mahasiswa %s: %v", s.Email, err)
		}

		var sID int
		err = pool.QueryRow(ctx, `
			INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
			RETURNING id;
		`, uID, s.NIM, s.Nama, s.Prodi, s.Angkatan, s.IPK).Scan(&sID)
		if err != nil {
			log.Fatalf("Gagal buat record student %s: %v", s.NIM, err)
		}

		studentIDs = append(studentIDs, sID)
	}
	log.Printf("✓ 20 Mahasiswa berhasil dibuat (Password: NIM masing-masing)")

	// 5. Seed 10 Mata Kuliah
	coursesData := []courseSeed{
		{"SI101", "Dasar Pemrograman", 3, 1, 30},
		{"SI102", "Struktur Data & Algoritma", 3, 2, 25},
		{"SI201", "Basis Data Relasional", 3, 3, 25},
		{"SI202", "Pemrograman Berorientasi Objek", 3, 3, 20},
		{"SI301", "Pemrograman Backend Lanjut", 3, 5, 15},
		{"SI302", "Arsitektur Perangkat Lunak", 3, 5, 20},
		{"SI303", "Keamanan Sistem Informasi", 3, 5, 25},
		{"SI401", "Cloud Computing & DevOps", 3, 7, 20},
		{"SI402", "Analisis & Desain Sistem", 4, 4, 25},
		{"SI403", "Proyek Capstone", 4, 7, 10},
	}

	courseIDs := make([]int, 0, len(coursesData))
	for _, c := range coursesData {
		var cID int
		err := pool.QueryRow(ctx, `
			INSERT INTO courses (kode_mk, nama_mk, sks, semester, kuota, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
			RETURNING id;
		`, c.KodeMK, c.NamaMK, c.SKS, c.Semester, c.Kuota).Scan(&cID)
		if err != nil {
			log.Fatalf("Gagal buat course %s: %v", c.KodeMK, err)
		}
		courseIDs = append(courseIDs, cID)
	}
	log.Printf("✓ 10 Mata Kuliah berhasil dibuat")

	// 6. Seed beberapa enrollment awal untuk demonstrasi kuota terisi & detail KRS
	thAkademik := "2026/2027-Ganjil"
	// Mahasiswa 1 mengambil mata kuliah 1, 2, 5 (Total 9 SKS)
	_, _ = pool.Exec(ctx, `
		INSERT INTO enrollments (student_id, course_id, tahun_akademik, created_at)
		VALUES 
			($1, $2, $4, NOW()),
			($1, $3, $4, NOW());
	`, studentIDs[0], courseIDs[0], courseIDs[4], thAkademik)

	// Mahasiswa 2 mengambil mata kuliah 5 (Backend Lanjut)
	_, _ = pool.Exec(ctx, `
		INSERT INTO enrollments (student_id, course_id, tahun_akademik, created_at)
		VALUES ($1, $2, $3, NOW());
	`, studentIDs[1], courseIDs[4], thAkademik)

	log.Println("✓ Sample enrollments KRS berhasil disisipkan")
	fmt.Println("\n========================================================")
	fmt.Println("Proses Seeding SIAKAD Mini Selesai dengan Sukses!")
	fmt.Println("Admin: admin@siakad.ac.id | Password: admin12345")
	fmt.Println("Mahasiswa 1: rina.putri@mhs.siakad.ac.id | Password: 187221000001")
	fmt.Println("========================================================")
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migrationSQL := `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL UNIQUE,
			password VARCHAR(255) NOT NULL,
			role VARCHAR(20) NOT NULL CHECK (role IN ('admin', 'mahasiswa')),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS students (
			id SERIAL PRIMARY KEY,
			user_id INT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
			nim VARCHAR(12) NOT NULL UNIQUE,
			nama VARCHAR(255) NOT NULL,
			prodi VARCHAR(100) NOT NULL,
			angkatan INT NOT NULL,
			ipk_terakhir NUMERIC(3, 2) NOT NULL DEFAULT 0.00 CHECK (ipk_terakhir >= 0.00 AND ipk_terakhir <= 4.00),
			deleted_at TIMESTAMPTZ DEFAULT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_students_deleted_at ON students(deleted_at);
		CREATE INDEX IF NOT EXISTS idx_students_prodi ON students(prodi) WHERE deleted_at IS NULL;
		CREATE INDEX IF NOT EXISTS idx_students_angkatan ON students(angkatan) WHERE deleted_at IS NULL;
		CREATE INDEX IF NOT EXISTS idx_students_nim ON students(nim);

		CREATE TABLE IF NOT EXISTS courses (
			id SERIAL PRIMARY KEY,
			kode_mk VARCHAR(20) NOT NULL UNIQUE,
			nama_mk VARCHAR(255) NOT NULL,
			sks INT NOT NULL CHECK (sks > 0),
			semester INT NOT NULL CHECK (semester >= 1 AND semester <= 8),
			kuota INT NOT NULL CHECK (kuota >= 0),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_courses_semester ON courses(semester);
		CREATE INDEX IF NOT EXISTS idx_courses_kode_mk ON courses(kode_mk);

		CREATE TABLE IF NOT EXISTS enrollments (
			id SERIAL PRIMARY KEY,
			student_id INT NOT NULL REFERENCES students(id) ON DELETE CASCADE,
			course_id INT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
			tahun_akademik VARCHAR(30) NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT uq_student_course_year UNIQUE (student_id, course_id, tahun_akademik)
		);

		CREATE INDEX IF NOT EXISTS idx_enrollments_student_year ON enrollments(student_id, tahun_akademik);
		CREATE INDEX IF NOT EXISTS idx_enrollments_course_id ON enrollments(course_id);
	`
	_, err := pool.Exec(ctx, migrationSQL)
	return err
}
