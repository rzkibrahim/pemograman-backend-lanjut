# Praktikum Pemrograman Backend Lanjut (PBEL)

Repositori ini berisi implementasi materi dan tugas praktikum Pemrograman Backend Lanjut yang telah dirapikan ke dalam folder terpisah untuk masing-masing modul. Dengan struktur ini, evolusi arsitektur dan progres pembelajaran dari setiap pertemuan dapat dipelajari dan dijalankan secara mandiri.

---

## Daftar Modul & Progres Pembelajaran

| Direktori | Pertemuan / Topik | Deskripsi Singkat & Fitur Utama | Status |
| :--- | :--- | :--- | :---: |
| [`modul1/`](./modul1) | **Modul 1 — Dasar Go & Fiber** | Sintaks dasar, variabel & map (`no2.go`), pointer (`no3.go`), struct `Student` & method (`no4.go`), server Fiber Hello World (`main.go`). | Selesai |
| [`modul2/`](./modul2) | **Modul 2 — REST API & HTTP Deep Dive** | REST API CRUD `students` berbasis *in-memory slice*, validasi query string (paginasi, search, sort whitelist, filter), status HTTP presisi (200, 201, 204, 400, 404, 409, 415, 422), pembuktian perbedaan PUT vs PATCH, dan suite 17 unit test. | Selesai |
| [`modul3/`](./modul3) | **Modul 3 — Database & Repository Pattern** | Migrasi penyimpanan ke PostgreSQL dengan driver `pgx/v5` (`pgxpool`), connection pool, migrasi DDL berparameter aman dari SQL Injection, abstraksi antarmuka `StudentRepository`, dan penerjemahan kode error PostgreSQL ke status HTTP. | Selesai |
| [`modul4/`](./modul4) | **Modul 4 — Clean Architecture** | Restrukturisasi menyeluruh ke dalam 4 lapisan Clean Architecture: Entities (`app/model`), Repositories (`app/repository`), Use Cases (`app/service`), dan Delivery/Frameworks (`route`, `middleware`, `config`, `helper`). Dilengkapi *pure business rules*, unit testing, rotasi log file (`lumberjack`), dan *graceful shutdown*. | Selesai |
| [`modul5/`](./modul5) | **Modul 5 — Authentication & Security** | Autentikasi JWT (HS256), hashing password bcrypt (cost 12), rotasi refresh token SHA-256 (sekali pakai), rate limiter brute force, pencegahan user enumeration & algorithm confusion, dan proteksi endpoint `/students`. | Selesai |

> **Catatan:** Folder `api-students/` tetap dipertahankan sebagai direktori cadangan/legacy dari pengerjaan awal.

---

## Cara Menjalankan Setiap Modul

Masing-masing modul memiliki file `go.mod` dan dependensi sendiri:

### Menjalankan Modul 1:
```bash
cd modul1
go run main.go
# atau menjalankan demo latihan mandiri:
go run no2.go
go run no3.go
go run no4.go
```

### Menjalankan & Menguji Modul 2:
```bash
cd modul2
go test -v ./...
go run .
```

### Menjalankan Modul 3:
```bash
cd modul3
# Pastikan migrasi database migrations/001_create_students.sql sudah dijalankan di PostgreSQL
go run .
```

### Menjalankan & Menguji Modul 4:
```bash
cd modul4
# Jalankan unit test business rules (tanpa perlu koneksi database):
go test -v ./...
# Jalankan server dengan PostgreSQL aktif:
go run .
```

### Menjalankan & Menguji Modul 5:
```bash
cd modul5
# Jalankan unit test aturan autentikasi & validasi:
go test -v ./app/service/...
# Jalankan server API terproteksi:
go run .
# Jalankan pengujian keamanan otomatis (curl suite):
.\test_security.ps1
```
