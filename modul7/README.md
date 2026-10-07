# Modul 6 — Authorization & Role-Based Access Control (RBAC)

Implementasi lengkap **Modul Pertemuan 6: Authorization & Role-Based Access Control (RBAC)** pada mata kuliah Praktikum Pemrograman Backend Lanjut (D4 Teknik Informatika, Fakultas Vokasi, Universitas Airlangga).

Proyek ini menutup celah keamanan dari Modul 5 (*Broken Access Control*) dengan menerapkan otorisasi berlapis:
1. **RBAC Relasional**: Skema tabel `roles`, `permissions`, dan `role_permissions` dengan composite primary key dan foreign key.
2. **In-Memory Fail-Closed PermissionSet**: Pemuatan skema otorisasi ke memori saat startup dengan evaluasi $O(1)$ yang selalu menolak akses (*fail-closed*) secara default.
3. **Pemisahan Tanggung Jawab Otorisasi**:
   - **Middleware Layer (`RequirePermission`)**: untuk keputusan akses independen data (`GET /students`, `POST /students`, `DELETE /students/:id`, `GET /users`).
   - **Service Layer (Ownership Check)**: untuk keputusan akses yang bergantung pada data baris (`GET /students/:id`, `PUT /students/:id`, `PATCH /students/:id`, `/users/:id`).
4. **Anti-Spoofing `owner_id`**: Penetapan atribut kepemilikan secara otomatis dari token pemanggil saat `POST /students`, menolak manipulasi dari body JSON.
5. **Jejak Audit Terstruktur**: Pencatatan `user_id` dan `role` pada setiap permintaan HTTP di structured logger.

---

## 1. Struktur Berkas Modul 6

```
modul6/
├── app/
│   ├── model/
│   │   ├── auth.go                  <- DTO Auth, AuthUser, JWTPayload
│   │   ├── student.go               <- Model Student (tambah OwnerID), DTO Create/Replace/Patch
│   │   └── user.go                  <- Model User & DTO UpdateUser/AssignRole
│   ├── repository/
│   │   ├── role_repository.go       <- Query pemetaan role-permission dari database
│   │   ├── student_repository.go    <- Repositori mahasiswa (dukungan owner_id)
│   │   ├── token_repository.go      <- Repositori token refresh
│   │   └── user_repository.go       <- Repositori pengguna (FindAll, Update, UpdateRole, Delete)
│   └── service/
│       ├── auth_rules.go            <- Pure rules: validasi registrasi & kekuatan kata sandi
│       ├── auth_rules_test.go       <- Unit test validasi auth
│       ├── auth_service.go          <- Use case autentikasi (Register, Login, Refresh, Me)
│       ├── authz_rules.go           <- Pure function CanAccessUser (zero dependencies)
│       ├── authz_rules_test.go      <- Unit test otorisasi user
│       ├── student_authz_rules.go   <- Pure function CanAccessStudent (zero dependencies)
│       ├── student_authz_rules_test.go <- Unit test otorisasi & ownership mahasiswa
│       ├── student_rules.go         <- Pure rules validasi mahasiswa
│       ├── student_rules_test.go    <- Unit test validasi mahasiswa
│       ├── student_service.go       <- Use case mahasiswa dengan ownership check & anti-spoofing
│       └── user_service.go          <- Use case manajemen user dengan otorisasi
├── cmd/
│   └── seed/
│       └── main.go                  <- Seeder akun admin_budi, staff_sari, user_andi, user_bambang
├── config/
│   ├── app.go                       <- Perakitan Fiber & injeksi dependensi
│   ├── env.go                       <- Pembaca variabel environment .env
│   └── logger.go                    <- Structured logger dengan rotasi lumberjack
├── database/
│   └── postgres.go                  <- PostgreSQL connection pool pgx
├── helper/
│   ├── authz.go                     <- In-memory PermissionSet dengan Can() fail-closed
│   ├── authz_test.go                <- Unit test PermissionSet fail-closed
│   ├── context.go                   <- GetCurrentUser & GetCurrentAuthUser
│   ├── jwt.go                       <- Pembuat & parser access token JWT HS256
│   ├── request.go                   <- Pembaca query string & pagination
│   ├── response.go                  <- Amplop JSON standar WebResponse
│   └── security.go                  <- Bcrypt hashing, dummy verify, SHA-256
├── middleware/
│   ├── auth.go                      <- RequireAuth & LoginRateLimiter
│   ├── authz.go                     <- RequirePermission middleware
│   └── middleware.go                <- Global middleware + RequestLogger (user_id & role audit)
├── migrations/
│   ├── 001_create_students.sql      <- Migrasi tabel students dasar
│   ├── 002_auth.sql                 <- Migrasi users & refresh_tokens
│   ├── 003_rbac.sql                 <- Skema RBAC (roles, permissions, role_permissions)
│   └── 004_student_permissions.sql  <- Permission student & penambahan kolom owner_id
├── .env.example
├── .gitignore
├── go.mod
├── go.sum
├── LAPORAN.md                       <- Laporan lengkap tugas mandiri (analisis C.4 & bukti uji)
├── main.go                          <- Entry point: load RBAC saat startup & assemble app
├── security_test_evidence.log       <- Log bukti eksekusi 22 pengujian keamanan & matriks RBAC
└── test_rbac.ps1                    <- Skrip pengujian otomatis matriks RBAC
```

---

## 2. Cara Menjalankan Aplikasi

### Persyaratan:
- Go versi 1.22 atau lebih baru
- PostgreSQL 15+ (sudah berjalan di port 5432)
- Basis data `praktikum_backend`

### Langkah Menjalankan:
1. Pindah ke direktori `modul6`:
   ```bash
   cd c:\laragon\www\PBEL\modul6
   ```

2. Salin dan sesuaikan konfigurasi `.env`:
   ```bash
   cp .env.example .env
   ```

3. Jalankan migrasi basis data:
   ```bash
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

7. Jalankan pengujian otomatis matriks RBAC:
   ```powershell
   powershell -ExecutionPolicy Bypass -File .\test_rbac.ps1
   ```
