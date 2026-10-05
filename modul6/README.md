# Modul 6 — Authorization & Role-Based Access Control (RBAC)

Implementasi lengkap **Modul Pertemuan 6: Authorization & Role-Based Access Control (RBAC)** pada mata kuliah Praktikum Pemrograman Backend Lanjut (D4 Teknik Informatika, Fakultas Vokasi, Universitas Airlangga).

Proyek ini menutup celah keamanan dari Modul 5 (*Broken Access Control*) dengan menerapkan otorisasi berlapis:
1. **RBAC Relasional**: Skema tabel `roles`, `permissions`, dan `role_permissions` dengan composite primary key dan foreign key.
2. **In-Memory Fail-Closed PermissionSet**: Pemuatan skema otorisasi ke memori saat startup dengan evaluasi $O(1)$ yang selalu menolak akses (*fail-closed*) secara default jika role atau permission tidak terdaftar.
3. **Pemisahan Tanggung Jawab Otorisasi**:
   - **Middleware Layer (`RequirePermission`)**: untuk keputusan akses independen data (`GET /students`, `POST /students`, `DELETE /students/:id`, `GET /users`).
   - **Service Layer (Ownership Check)**: untuk keputusan akses yang bergantung pada data baris (`GET /students/:id`, `PUT /students/:id`, `PATCH /students/:id`, `/users/:id`).
4. **Anti-Spoofing `owner_id`**: Penetapan atribut kepemilikan secara otomatis dari token pemanggil saat `POST /students`, menolak manipulasi dari body JSON.
5. **Jejak Audit Terstruktur**: Pencatatan `user_id` dan `role` pada setiap permintaan HTTP di structured logger.

---

## 1. Struktur Berkas & Penjelasan Fungsi Per-Folder & Per-File

Kode sumber utama Modul 6 berada di dalam direktori `modul6/api-students/`:

```
modul6/
├── LAPORAN.md                         <- Laporan lengkap tugas mandiri (analisis C.4 & bukti uji)
├── LAPORAN_modul6.md                  <- Salinan arsip laporan resmi Modul 6
├── README.md                          <- Dokumentasi teknis Modul 6 ini
└── api-students/                      <- Direktori proyek utama Go Fiber
    ├── app/
    │   ├── model/                     <- [FOLDER MODEL / ENTITY DTO]
    │   │   ├── auth.go                <- DTO Auth, RegisterRequest, LoginRequest, AuthUser, JWTPayload
    │   │   ├── student.go             <- Model Student (kolom OwnerID), DTO Create/Replace/PatchStudentRequest
    │   │   └── user.go                <- Model User, UpdateUserRequest, AssignRoleRequest
    │   ├── repository/                <- [FOLDER REPOSITORY / DATABASE ACCESS]
    │   │   ├── role_repository.go     <- Query pemetaan role-permission dari database PostgreSQL
    │   │   ├── student_repository.go  <- Repositori mahasiswa (dukungan penyimpanan & filter owner_id)
    │   │   ├── token_repository.go    <- Repositori refresh token dengan SHA-256 hash dan mekanisme revoke
    │   │   └── user_repository.go     <- Repositori pengguna (FindAll, Update, UpdateRole, Delete)
    │   └── service/                   <- [FOLDER SERVICE / USE CASE & BUSINESS RULES]
    │       ├── auth_rules.go          <- Pure rules: validasi registrasi manual & kekuatan kata sandi
    │       ├── auth_rules_test.go     <- Unit test validasi auth (zero external dependencies)
    │       ├── auth_service.go        <- Use case autentikasi (Register, Login, Refresh, Me)
    │       ├── authz_rules.go         <- Pure function CanAccessUser (aturan akses profil pengguna)
    │       ├── authz_rules_test.go    <- Unit test otorisasi user
    │       ├── student_authz_rules.go <- Pure function CanAccessStudent (aturan kepemilikan data mahasiswa)
    │       ├── student_authz_rules_test.go <- Unit test otorisasi & ownership mahasiswa
    │       ├── student_rules.go       <- Pure rules validasi data mahasiswa secara imperatif
    │       ├── student_rules_test.go  <- Unit test validasi mahasiswa
    │       ├── student_service.go     <- Use case mahasiswa dengan ownership check & anti-spoofing
    │       └── user_service.go        <- Use case manajemen user dengan otorisasi berbasis peran
    ├── cmd/
    │   └── seed/                      <- [FOLDER SEEDER]
    │       └── main.go                <- Seeder akun uji: admin_budi, staff_sari, user_andi, user_bambang
    ├── config/                        <- [FOLDER CONFIGURATION]
    │   ├── app.go                     <- Perakitan Fiber, pendaftaran middleware, dan injeksi dependensi
    │   ├── env.go                     <- Pembaca variabel environment .env
    │   └── logger.go                  <- Structured logger (slog) dengan rotasi file lumberjack
    ├── database/                      <- [FOLDER DATABASE CONNECTION]
    │   └── postgres.go                <- PostgreSQL connection pool menggunakan pgx/v5
    ├── helper/                        <- [FOLDER HELPER UTILITIES]
    │   ├── authz.go                   <- In-memory PermissionSet dengan method Can() fail-closed
    │   ├── authz_test.go              <- Unit test pembuktian fail-closed pada PermissionSet
    │   ├── context.go                 <- Ekstraksi context HTTP (GetCurrentUser & GetCurrentAuthUser)
    │   ├── jwt.go                     <- Pembuat & parser access token JWT HS256
    │   ├── request.go                 <- Pembaca query string limit/offset & parameter ID
    │   ├── response.go                <- Amplop JSON standar WebResponse
    │   └── security.go                <- Bcrypt hashing (cost 12), dummy verify timing-safe, SHA-256
    ├── middleware/                    <- [FOLDER MIDDLEWARE]
    │   ├── auth.go                    <- RequireAuth (JWT) & LoginRateLimiter (anti-brute force)
    │   ├── authz.go                   <- RequirePermission middleware (pengecekan hak akses route)
    │   └── middleware.go              <- Global middleware + RequestLogger (audit user_id & role)
    ├── migrations/                    <- [FOLDER DATABASE MIGRATIONS]
    │   ├── 001_create_students.sql    <- Migrasi tabel students dasar
    │   ├── 002_auth.sql               <- Migrasi tabel users & refresh_tokens
    │   ├── 003_rbac.sql               <- Skema RBAC (roles, permissions, role_permissions)
    │   └── 004_student_permissions.sql<- Permission student & penambahan kolom owner_id pada students
    ├── route/
    │   └── route.go                   <- Pendaftaran rute HTTP & pemetaan middleware otorisasi
    ├── .env.example                   <- Template environment lokal
    ├── .gitignore                     <- Pengecualian berkas git
    ├── go.mod                         <- Deklarasi modul Go
    ├── go.sum                         <- Checksum dependensi Go
    ├── main.go                        <- Entry point: load RBAC dari DB ke memori saat startup & jalankan server
    ├── security_test_evidence.log     <- Log bukti eksekusi 22 pengujian keamanan & matriks RBAC
    └── test_rbac.ps1                  <- Skrip pengujian otomatis 22 skenario matriks RBAC
```

---

## 2. Cara Menjalankan Aplikasi

### Persyaratan:
- Go versi 1.22 atau lebih baru
- PostgreSQL 15+ (berjalan di port 5432)
- Basis data `praktikum_backend`

### Langkah Menjalankan:
1. Pindah ke direktori `modul6/api-students`:
   ```bash
   cd c:\laragon\www\PBEL\modul6\api-students
   ```

2. Salin dan sesuaikan konfigurasi `.env`:
   ```bash
   copy .env.example .env
   ```

3. Jalankan migrasi basis data PostgreSQL:
   ```bash
   psql -U postgres -d praktikum_backend -f migrations/001_create_students.sql
   psql -U postgres -d praktikum_backend -f migrations/002_auth.sql
   psql -U postgres -d praktikum_backend -f migrations/003_rbac.sql
   psql -U postgres -d praktikum_backend -f migrations/004_student_permissions.sql
   ```

4. Jalankan seeder akun uji:
   ```bash
   go run ./cmd/seed/main.go
   ```

5. Jalankan server:
   ```bash
   go run main.go
   ```

6. Jalankan pengujian unit:
   ```bash
   go test -v ./...
   ```

7. Jalankan pengujian otomatis matriks RBAC (22 skenario):
   ```powershell
   powershell -ExecutionPolicy Bypass -File .\test_rbac.ps1
   ```
