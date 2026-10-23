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
| [`modul6/`](./modul6) | **Modul 6 — Authorization & RBAC** | Otorisasi berbasis peran (RBAC) dengan skema relasional (`roles`, `permissions`, `role_permissions`), evaluasi *fail-closed* di memori (`PermissionSet`), pemisahan middleware (`RequirePermission`) vs *ownership check* di layer service (`CanAccessStudent`), proteksi *anti-spoofing* `owner_id`, jejak audit `user_id` & `role`, dan 22 pengujian matriks RBAC. | Selesai |
| [`modul7/`](./modul7) | **Modul 7 — Advanced API Design** | Validasi deklaratif berbasis struct tags (`go-playground/validator/v10`) & custom tag NIM/password, keyset/cursor pagination ($O(1)$ anti-page drift) dengan B-Tree composite index `(created_at DESC, id DESC)`, content negotiation (RFC 9110) format JSON & streaming CSV via header `Accept`, centralized error handling (`helper.AppError`), dan perbaikan 9 bug sengaja. | Selesai |
| [`modul10/`](./modul10) | **Modul 10 — API dengan MongoDB** | PostgreSQL + MongoDB polyglot persistence, ulasan kelas dengan update atomik dan aggregation `$facet`, portofolio mahasiswa embedded, validasi bertipe, serta health check dua database. | Selesai |
| [`latihan-fiber/UTS/`](./latihan-fiber/UTS) | **Proyek UTS — SIAKAD Mini API** | Sistem Informasi Akademik Mini berbasis Fiber v2 & PostgreSQL: 10 endpoint RESTful API (Auth JWT, RBAC Admin & Mahasiswa, manajemen Mahasiswa, katalog Mata Kuliah, KRS Enrollment dengan validasi batas SKS dinamis & row-locking kuota atomik `FOR UPDATE`, auto-seeder, rate limiter, serta suite 10 skenario pengujian otomatis). | Selesai |

> **Catatan Struktur Folder:** 
> - Proyek Ujian Tengah Semester (UTS) terletak di dalam direktori [`latihan-fiber/UTS/`](./latihan-fiber/UTS).
> - Folder `api-students/` tetap dipertahankan sebagai direktori cadangan/legacy dari pengerjaan awal.

---

## Struktur Direktori Repositori

```text
PBEL/
├── modul1/                 # Pertemuan 1: Dasar Go & Fiber
├── modul2/                 # Pertemuan 2: REST API & HTTP Deep Dive (Slice in-memory)
├── modul3/                 # Pertemuan 3: Database & Repository Pattern (PostgreSQL pgx)
├── modul4/                 # Pertemuan 4: Clean Architecture 4-Layer & Graceful Shutdown
├── modul5/                 # Pertemuan 5: Authentication & Security (JWT, bcrypt, Brute Force Limiter)
├── modul6/                 # Pertemuan 6: Authorization & RBAC (Roles & Permissions Database)
├── modul7/                 # Pertemuan 7: Advanced API Design (Cursor Pagination & Content Negotiation)
├── modul10/                # Pertemuan 10: MongoDB & Polyglot Persistence
├── latihan-fiber/
│   └── UTS/                # Proyek UTS: SIAKAD Mini RESTful API Engine
└── api-students/           # Direktori legacy / cadangan pengembangan awal
```

---

## Prasyarat Sistem & Tech Stack

Sebelum menjalankan modul-modul di repositori ini, pastikan lingkungan pengujian memenuhi spesifikasi berikut:

* **Go (Golang)**: Versi `1.22+` (disarankan `1.24+` / `1.26+`)
* **PostgreSQL Database Server**: Versi `14+` aktif pada port default `5432`
* **MongoDB**: Versi Community `9.0+` atau cluster Atlas (khusus Modul 10)
* **Node.js**: Versi `18+` (dibutuhkan untuk menjalankan suite verifikasi otomatis `test_runner.js` pada Modul 7)
* **PowerShell / Bash**: Untuk menjalankan skrip otomasi pengujian keamanan dan endpoint (`.ps1`)
* **Pustaka & Driver Utama**:
  * Web Framework: **Fiber v2** (`github.com/gofiber/fiber/v2`)
  * Database Driver: **pgx v5** (`github.com/jackc/pgx/v5`) dengan *connection pool* (`pgxpool`)
  * MongoDB Driver Modul 10: **MongoDB Go Driver v2** (`go.mongodb.org/mongo-driver/v2`)
  * Validasi Data: **go-playground/validator/v10** & custom validation tags
  * Autentikasi & Kriptografi: **golang-jwt/jwt/v5** & **golang.org/x/crypto/bcrypt** (cost 12)
  * Logging & Rotasi File: **gopkg.in/natefinch/lumberjack.v2**

---

## Matriks Konfigurasi Database & Port

Tabel berikut menyajikan ringkasan cepat konfigurasi port, basis data, dan metode verifikasi untuk setiap modul:

| Direktori | Modul / Topik | Port Default | Nama Database PostgreSQL | Skrip Verifikasi & Pengujian |
| :--- | :--- | :---: | :--- | :--- |
| [`modul1/`](./modul1) | Dasar Go & Fiber | `3000` | *Tanpa Database (In-Memory)* | Eksekusi `main.go`, `no2.go`, `no3.go`, `no4.go` |
| [`modul2/`](./modul2) | REST API & HTTP Deep Dive | `3000` | *Tanpa Database (Slice In-Memory)* | Unit testing: `go test -v ./...` (17 skenario) |
| [`modul3/`](./modul3) | Database & Repository Pattern | `3000` | `praktikum_backend` | Manual via cURL / Postman |
| [`modul4/`](./modul4) | Clean Architecture 4-Layer | `3000` | `praktikum_backend` | Unit testing: `go test -v ./...` |
| [`modul5/`](./modul5) | Authentication & Security | `3000` | `praktikum_backend` | Unit testing service & `.\test_security.ps1` |
| [`modul6/`](./modul6) | Authorization & RBAC | `3000` | `praktikum_backend` | Matriks RBAC: `powershell .\test_rbac.ps1` (22 skenario) |
| [`modul7/`](./modul7) | Advanced API Design | `3000` | `praktikum_backend` | Verifikasi cursor & format: `node test_runner.js` |
| [`modul10/`](./modul10) | API dengan MongoDB | `3000` | `praktikum_backend` + MongoDB `praktikum_backend` | Unit test: `go test ./...`; [panduan Modul 10](./modul10/README.md) |
| [`latihan-fiber/UTS/`](./latihan-fiber/UTS) | Proyek UTS (SIAKAD Mini) | `3000` | `siakad_mini` | Suite otomatis: `powershell .\test_all_endpoints.ps1` (10 endpoint) |

---

## Kredensial Akun Seeder & Pengujian

Untuk mempermudah pengujian API (login, autorisasi, dan pengambilan KRS) via Postman atau cURL, gunakan akun-akun bawaan seeder berikut:

### 1. Akun Modul 6 & Modul 7 (`DB: praktikum_backend`)
Dibuat otomatis dengan menjalankan `go run ./cmd/seed/main.go` dari folder modul masing-masing:

| Username | Email | Password | Role | Hak Akses Utama |
| :--- | :--- | :--- | :---: | :--- |
| `admin_budi` | `admin@univ.ac.id` | `Password123!` | `admin` | Akses penuh CRUD, manajemen role & permissions |
| `staff_sari` | `staff@univ.ac.id` | `Password123!` | `staff` | Read all students, create student, update student |
| `user_andi` | `andi@student.ac.id` | `Password123!` | `user` | Read-only mahasiswa data sendiri (*ownership check*) |
| `user_bambang` | `bambang@student.ac.id` | `Password123!` | `user` | Read-only mahasiswa data sendiri (*ownership check*) |

### 2. Akun Proyek UTS — SIAKAD Mini (`DB: siakad_mini`)
Dibuat otomatis dengan menjalankan `go run cmd/seed/main.go` dari dalam folder [`latihan-fiber/UTS`](./latihan-fiber/UTS):

| Role | Akun / Email | Password Default | Keterangan & Batasan |
| :--- | :--- | :--- | :--- |
| **Admin** | `admin@siakad.ac.id` | `admin12345` | Hak akses penuh: kelola mahasiswa, mata kuliah, dan KRS seluruh mahasiswa |
| **Mahasiswa 1** | `rina.putri@mhs.siakad.ac.id` | `187221000001` *(NIM)* | Profil mahasiswa, ambil/batalkan KRS, validasi batas SKS (IPK 3.85 = max 24 SKS) |
| **Mahasiswa 2** | `dimas.setiawan@mhs.siakad.ac.id` | `187221000002` *(NIM)* | Akun uji coba perebutan kuota mata kuliah & batas SKS (IPK 2.70 = max 20 SKS) |
| **Mahasiswa 3–20** | `[nama]@mhs.siakad.ac.id` | *(NIM Masing-masing)* | 18 akun mahasiswa tambahan untuk verifikasi pagination dan konkurensi kuota |

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

### Menjalankan & Menguji Modul 6:
```bash
cd modul6/api-students
# 1. Jalankan migrasi skema RBAC dan student permissions ke PostgreSQL:
psql -U postgres -d praktikum_backend -f migrations/003_rbac.sql
psql -U postgres -d praktikum_backend -f migrations/004_student_permissions.sql

# 2. Jalankan seeder akun pengujian (admin_budi, staff_sari, user_andi, user_bambang):
go run ./cmd/seed/main.go

# 3. Jalankan seluruh unit test (helper authz, rules authz, student authz, dsb.):
go test -v ./...

# 4. Jalankan server API terproteksi RBAC:
go run main.go

# 5. Jalankan pengujian otomatis matriks hak akses RBAC (22 skenario):
powershell -ExecutionPolicy Bypass -File .\test_rbac.ps1
```

### Menjalankan & Menguji Modul 7:
```bash
cd modul7/api-students
# 1. Jalankan migrasi database (termasuk B-Tree composite index untuk cursor pagination):
psql -U postgres -d praktikum_backend -f migrations/001_create_students.sql
psql -U postgres -d praktikum_backend -f migrations/002_auth.sql
psql -U postgres -d praktikum_backend -f migrations/003_rbac.sql
psql -U postgres -d praktikum_backend -f migrations/004_student_permissions.sql
psql -U postgres -d praktikum_backend -f migrations/005_cursor_index.sql
psql -U postgres -d praktikum_backend -f migrations/006_student_cursor_index.sql

# 2. Jalankan seeder akun pengujian:
go run ./cmd/seed/main.go

# 3. Jalankan server API Advanced Design:
go run main.go

# 4. Jalankan unit test:
go test -v ./...

# 5. Jalankan verifikasi otomatis (anti-drift, keyset pagination, content negotiation):
node test_runner.js
```

### Menjalankan Modul 10:
Panduan instalasi MongoDB, konfigurasi, endpoint, dan verifikasi tersedia di [`modul10/README.md`](./modul10/README.md).

```powershell
cd modul10/api-students
Copy-Item .env.example .env
# Isi DB_PASSWORD dan JWT_SECRET; pastikan PostgreSQL dan MongoDB aktif.
# Jalankan migration migrations/001_*.sql sampai migrations/010_*.sql.
go mod tidy
go run ./cmd/seed/main.go
go test ./...
go run main.go
```

### Menjalankan & Menguji Proyek UTS (SIAKAD Mini):
```bash
cd latihan-fiber/UTS

# 1. Konfigurasi kredensial database di .env (salin dari .env.example jika belum ada):
# Sesuaikan DB_USER, DB_PASSWORD, dan DB_NAME=siakad_mini

# 2. Jalankan seeder database (otomatis mengeksekusi migrasi skema, akun admin/mahasiswa, & katalog mata kuliah):
go run cmd/seed/main.go

# 3. Jalankan server API SIAKAD Mini (aktif di port 3000):
go run main.go

# 4. Jalankan pengujian otomatis 10 endpoint (PowerShell):
powershell -ExecutionPolicy Bypass -File .\test_all_endpoints.ps1
```

