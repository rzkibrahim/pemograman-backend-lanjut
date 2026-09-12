# Modul 4 - Clean Architecture (API Students)

REST API untuk mengelola data mahasiswa (students) dengan basis data PostgreSQL, direstrukturisasi penuh ke dalam 4 lapisan **Clean Architecture** (Robert C. Martin).

---

## Arsitektur 4 Lapisan

Proyek ini dipisahkan secara ketat sesuai prinsip *Dependency Rule*, di mana dependensi hanya mengarah ke dalam (ke arah Domain/Entities):

1. **Layer 1 - Entities (`app/model/`):**
   - Struct data murni (`Student`, DTO Request/Response, `Meta`, `ListQuery`).
   - Tidak mengimpor package apa pun dari framework atau layer luar.

2. **Layer 2 - Repositories (`app/repository/`):**
   - Kontrak interface `StudentRepository` dan implementasi PostgreSQL `studentPostgresRepository`.
   - Menggunakan query berparameter aman dari SQL injection, connection pool (`pgxpool`), dan sentinel error (`ErrNotFound`, `ErrDuplicate`).
   - Tidak ada ketergantungan pada `fiber.Ctx`.

3. **Layer 3 - Use Cases / Service (`app/service/`):**
   - `student_service.go`: Mengorkestrasikan alur bisnis antara HTTP adapter dan database repository.
   - `student_rules.go`: Fungsi validasi dan kalkulasi bisnis murni (`ValidateCreate`, `ValidateReplace`, `ApplyPatch`, `CountTotalPages`) tanpa dependensi HTTP maupun SQL, sehingga 100% dapat diuji secara unit test.
   - `student_rules_test.go`: Suite unit test yang berjalan cepat tanpa memerlukan basis data aktif.

4. **Layer 4 - Delivery / Frameworks & Drivers:**
   - `helper/request.go`: Membaca dan memvalidasi input HTTP Fiber.
   - `helper/response.go`: Membungkus respons JSON seragam (`WebResponse`).
   - `middleware/`: Middleware request ID, logging, CORS, dan validasi Content-Type (`requireJSON`).
   - `route/route.go`: Mendaftarkan endpoint ke handler Fiber.
   - `config/app.go`: Perakitan aplikasi Fiber dan dependency injection.
   - `config/logger.go`: Structured logging rotasi berkas harian (`gopkg.in/natefinch/lumberjack.v2`).
   - `main.go`: Entry point bersih yang hanya merakit dependensi dan menangani graceful shutdown (SIGINT / SIGTERM).

---

## Struktur Direktori

```
modul4/
├── app/
│   ├── model/
│   │   └── student.go
│   ├── repository/
│   │   └── student_repository.go
│   └── service/
│       ├── student_rules.go
│       ├── student_rules_test.go
│       └── student_service.go
├── config/
│   ├── app.go
│   ├── env.go
│   └── logger.go
├── database/
│   └── postgres.go
├── helper/
│   ├── request.go
│   └── response.go
├── middleware/
│   ├── logger.go
│   └── require_json.go
├── migrations/
│   └── 001_create_students.sql
├── route/
│   └── route.go
├── .env.example
├── .gitignore
├── go.mod
├── go.sum
├── main.go
└── README.md
```

---

## Cara Menjalankan

1. **Pastikan database PostgreSQL sudah disiapkan:**
   ```bash
   psql -U postgres -c "CREATE DATABASE praktikum_backend;"
   psql -U postgres -d praktikum_backend -f migrations/001_create_students.sql
   ```

2. **Siapkan berkas konfigurasi `.env`:**
   ```bash
   cp .env.example .env
   ```
   Sesuaikan `DB_USER`, `DB_PASSWORD`, dan `DB_NAME`.

3. **Jalankan Unit Test:**
   ```bash
   go test -v ./...
   ```
   Seluruh pengujian business rules di `app/service/student_rules_test.go` akan dijalankan.

4. **Jalankan Server:**
   ```bash
   go run .
   ```
   Server akan aktif di `http://localhost:3000` dengan dukungan graceful shutdown.

---

## Endpoint API

| Method   | Path                   | Keterangan                                   |
|----------|------------------------|----------------------------------------------|
| `GET`    | `/`                    | Health / Root check                          |
| `GET`    | `/api/v1/health`       | Status koneksi database & server             |
| `GET`    | `/api/v1/students`     | Daftar mahasiswa (paginasi, urut, filter)    |
| `GET`    | `/api/v1/students/:id` | Detail mahasiswa berdasarkan ID              |
| `POST`   | `/api/v1/students`     | Tambah mahasiswa baru (201 Created)          |
| `PUT`    | `/api/v1/students/:id` | Ganti seluruh data mahasiswa                 |
| `PATCH`  | `/api/v1/students/:id` | Perbarui sebagian field mahasiswa            |
| `DELETE` | `/api/v1/students/:id` | Hapus data mahasiswa (204 No Content)        |
