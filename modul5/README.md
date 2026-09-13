# Modul 5 — Authentication & Security (API Students)

Implementasi lengkap **Modul Pertemuan 5: Authentication & Security** pada mata kuliah Praktikum Pemrograman Backend Lanjut (D4 Teknik Informatika, Fakultas Vokasi, Universitas Airlangga).

Proyek ini menambahkan sistem autentikasi berbasis **JSON Web Token (JWT)**, hashing kata sandi dengan **bcrypt (Cost Factor 12)**, proteksi endpoint dengan **middleware**, rotasi **refresh token** sekali pakai (one-time use), pencegahan serangan umum (brute force rate limiting, user enumeration, algorithm confusion, mass assignment), dan perlindungan menyeluruh terhadap resource data mahasiswa (`students`).

---

## 1. Arsitektur dan Fitur Keamanan

```
                                  KLIEN / PENGGUNA
                                         │
                   ┌─────────────────────┴─────────────────────┐
                   │                                           │
         Endpoint Publik                             Endpoint Terproteksi
    (/health, /auth/register,                 (/auth/me, /api/v1/students)
       /auth/login, /refresh)                                  │
                   │                                    RequireAuth Middleware
                   │                               (Cek JWT Signature, Exp, Alg)
                   │                                           │
                   ▼                                           ▼
          [ Rate Limiter ]                              [ Context User ]
     (Maks 5 request/menit per IP)                             │
                   │                                           │
                   ▼                                           ▼
       AuthService / Rules                        StudentService / Repository
  - Hash bcrypt (Cost 12)                         - CRUD Mahasiswa
  - Dummy Hash (Anti-Enumeration)                 - Database PostgreSQL
  - JWT Access Token (15m)
  - Refresh Token SHA-256 (7d)
                   │
                   ▼
         Database PostgreSQL
    (users, refresh_tokens)
```

### Kerentanan yang Ditutup Secara Ketat:
1. **Penyimpanan Password Plaintext**: Menggunakan `bcrypt` dengan cost factor 12 ($2a$12$).
2. **Algorithm Confusion (`alg: "none"`)**: Parsing JWT secara eksplisit memvalidasi bahwa algoritma penandatanganan adalah HMAC (`jwt.SigningMethodHMAC`).
3. **User Enumeration & Timing Attack**: Mengembalikan pesan identik `username atau password salah` dan mengeksekusi komputasi `VerifyDummyPassword()` saat username tidak ditemukan.
4. **Brute Force Login**: Middleware `LoginRateLimiter` membatasi login maksimal 5 kali/menit per alamat IP. Request ke-6 menghasilkan status `429 Too Many Requests` disertai header `Retry-After: 60`.
5. **Mass Assignment**: Struct `RegisterRequest` tidak memiliki field `role`. Role pengguna selalu ditetapkan sebagai `'user'` oleh server.
6. **Kebocoran Password**: Struct `User` memiliki tag `json:"-"` pada field password.
7. **Pencabutan Token (Revocation)**: Access token berumur pendek (15 menit) dan refresh token berumur panjang (7 hari) disimpan dalam bentuk hash SHA-256 dan dirotasi (sekali pakai).
8. **Memory Exhaustion (DoS)**: Konfigurasi Fiber `BodyLimit` dibatasi maksimal 1 MB.
9. **CORS Longgar**: Origin dibatasi melalui variabel environment `ALLOWED_ORIGINS`.

---

## 2. Struktur Berkas Modul 5

```
modul5/
├── app/
│   ├── model/
│   │   ├── auth.go                  <- DTO Register, Login, Refresh, AuthResponse
│   │   ├── student.go               <- Model entitas Student
│   │   └── user.go                  <- Model entitas User (dengan field Role)
│   ├── repository/
│   │   ├── student_repository.go    <- Repositori data mahasiswa
│   │   ├── token_repository.go      <- Repositori data refresh_tokens
│   │   └── user_repository.go       <- Repositori data users (FindByUsername)
│   └── service/
│       ├── auth_rules.go            <- Pure business rules: validasi register & password
│       ├── auth_rules_test.go       <- Unit test aturan validasi auth
│       ├── auth_service.go          <- Use case: Register, Login, Refresh, Logout, Me
│       ├── student_rules.go         <- Pure business rules mahasiswa
│       ├── student_rules_test.go    <- Unit test aturan mahasiswa
│       └── student_service.go       <- Use case mahasiswa
├── config/
│   ├── app.go                       <- Perakitan Fiber, BodyLimit 1MB, CORS
│   ├── env.go                       <- Pemuatan berkas .env
│   └── logger.go                    <- Structured logger dengan rotasi lumberjack
├── database/
│   └── postgres.go                  <- PostgreSQL connection pool
├── helper/
│   ├── context.go                   <- Get & Set user context dari Locals
│   ├── jwt.go                       <- Generator & Parser JWT dengan proteksi HMAC
│   ├── request.go                   <- Pembaca query string & DTO
│   ├── response.go                  <- Amplop baku WebResponse
│   └── security.go                  <- Bcrypt hashing, dummy verify, SHA256Hex, random token
├── middleware/
│   ├── auth.go                      <- RequireAuth & LoginRateLimiter
│   └── middleware.go                <- Global middleware (Helmet, Recover, CORS restricted, RequestLogger)
├── migrations/
│   ├── 001_create_students.sql      <- Skema tabel students
│   └── 002_auth.sql                 <- Skema tabel users, role, dan refresh_tokens
├── .env.example
├── .gitignore
├── go.mod
├── go.sum
├── LAPORAN.md                       <- Laporan komprehensif Tugas Mandiri Pertemuan 5
├── main.go                          <- Main entry point dengan validasi secret min 32 char
├── security_test_evidence.log       <- Log bukti eksekusi 14 pengujian keamanan
└── test_security.ps1                <- Skrip otomatis pengujian keamanan
```

---

## 3. Cara Menjalankan & Menguji

### 1. Persiapan Basis Data
```bash
psql -U postgres -d praktikum_backend -f migrations/001_create_students.sql
psql -U postgres -d praktikum_backend -f migrations/002_auth.sql
```

### 2. Menyiapkan Konfigurasi
Salin `.env.example` menjadi `.env` dan isi variabelnya:
```bash
cp .env.example .env
```
Pastikan `JWT_SECRET` diisi string acak minimal 32 karakter.

### 3. Menjalankan Unit Test
```bash
go test -v ./app/service/...
```

### 4. Menjalankan Server
```bash
go run .
```
Server berjalan di `http://localhost:3000`.

### 5. Menjalankan Pengujian Keamanan Otomatis
```powershell
.\test_security.ps1
```
Seluruh hasil pengujian akan dicatat ke berkas `security_test_evidence.log`.

---

## 4. Daftar Endpoint API

| Method | Endpoint | Akses | Keterangan |
|---|---|---|---|
| `GET` | `/` | Publik | Health / root check |
| `GET` | `/api/v1/health` | Publik | Status koneksi database |
| `POST` | `/api/v1/auth/register` | Publik | Registrasi akun baru (201 Created) |
| `POST` | `/api/v1/auth/login` | Publik (Rate Limited) | Login (200 OK, menghasilkan Access & Refresh Token) |
| `POST` | `/api/v1/auth/refresh` | Publik | Memperbarui token dengan rotasi refresh token |
| `POST` | `/api/v1/auth/logout` | Terproteksi / Publik | Mencabut refresh token |
| `GET` | `/api/v1/auth/me` | **Bearer Token** | Profil pengguna yang sedang login |
| `GET` | `/api/v1/students` | **Bearer Token** | Daftar mahasiswa (paginasi & filter) |
| `GET` | `/api/v1/students/:id` | **Bearer Token** | Detail satu mahasiswa |
| `POST` | `/api/v1/students` | **Bearer Token** | Menambah data mahasiswa |
| `PUT` | `/api/v1/students/:id` | **Bearer Token** | Mengganti seluruh data mahasiswa |
| `PATCH`| `/api/v1/students/:id` | **Bearer Token** | Mengubah sebagian data mahasiswa |
| `DELETE`| `/api/v1/students/:id`| **Bearer Token** | Menghapus data mahasiswa |
