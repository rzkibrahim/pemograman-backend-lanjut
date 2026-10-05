# Modul 7 — Advanced API Design (Declarative Validation, Keyset Pagination, Content Negotiation & Centralized Error Handling)

Implementasi lengkap **Modul Pertemuan 7: Advanced API Design** pada mata kuliah Praktikum Pemrograman Backend Lanjut (D4 Teknik Informatika, Fakultas Vokasi, Universitas Airlangga).

Proyek ini merevolusi arsitektur API dari Modul 6 menjadi REST API tingkat lanjut yang memenuhi standar industri modern (RFC 9110, Keyset/Cursor Pagination, Centralized Error Handling, dan Validasi Deklaratif berbasis Struct Tag).

---

## Daftar Isi

1. [Perbedaan Mendasar Modul 6 vs Modul 7](#1-perbedaan-mendasar-modul-6-vs-modul-7)
2. [4 Pilar Arsitektur Utama Modul 7](#2-4-pilar-arsitektur-utama-modul-7)
3. [Struktur Berkas & Penjelasan Fungsi Per-Folder & Per-File](#3-struktur-berkas--penjelasan-fungsi-per-folder--per-file)
4. [Tabel Berita Acara Perbaikan (9 Bug Sengaja Bagian B)](#4-tabel-berita-acara-perbaikan-9-bug-sengaja-bagian-b)
5. [Analisis Kaskade 85+ Error Refactoring](#5-analisis-kaskade-85-error-refactoring)
6. [Panduan Menjalankan & Menguji Aplikasi](#6-panduan-menjalankan--menguji-aplikasi)
7. [Panduan & Rencana Git Commit Bertahap](#7-panduan--rencana-git-commit-bertahap)
8. [Bank Tanya Jawab Responsi / Ujian Dosen](#8-bank-tanya-jawab-responsi--ujian-dosen)

---

## 1. Perbedaan Mendasar Modul 6 vs Modul 7

| Aspek Arsitektur | Modul 6 (RBAC & Broken Access Control) | Modul 7 (Advanced API Design) |
|---|---|---|
| **Fokus Utama** | Otorisasi & Role-Based Access Control (Relasional + In-Memory fail-closed) | Kehandalan API, Kinerja Skala Besar, Validasi Deklaratif, & Standar Protokol Web |
| **Metode Validasi** | **Imperatif**: Fungsi validasi manual (`if-else`) bertumpuk di layer service | **Deklaratif**: Berbasis struct tags (`go-playground/validator/v10`) + custom tag (`nim_format`, `strongpassword`, `omitnil`) |
| **Metode Paginasi** | **Offset-based** (`LIMIT n OFFSET m`): Biaya komputasi $O(N)$ di database dan rentan *Page Drift* | **Keyset / Cursor-based**: Menggunakan klausa tuple `WHERE (created_at, id) < (cursor_created_at, cursor_id)` berkecepatan $O(1)$ dan anti-*drift* |
| **Representasi Data (MIME)** | Hanya mengembalikan format tunggal **JSON** | **Content Negotiation (RFC 9110)**: Melayani **JSON** & **CSV** via header `Accept`, menolak format asing dengan HTTP 406 *Not Acceptable* |
| **Manajemen Error** | Tersebar imperatif via pemanggilan langsung `helper.Fail(c, ...)` di tiap handler | **Centralized Error Handling**: Mengembalikan `helper.AppError` murni ke satu `fiber.ErrorHandler` terpusat |
| **Logging Error** | Logging tidak konsisten, status HTTP salah catat (status 200 pada request gagal) | **Audit Terstruktur Presisi**: 4xx dicatat sebagai `WARN` (`request_rejected`), 5xx dicatat sebagai `ERROR` (`request_failed`), status asli tercatat akurat |
| **Indeks Database** | Indeks primer standar & foreign key | Penambahan **B-Tree Composite Index** `(created_at DESC, id DESC)` untuk performa query cursor |

---

## 2. 4 Pilar Arsitektur Utama Modul 7

### A. Declarative Validation & Pointer Types
- Menghapus fungsi validasi imperatif yang berulang (`ValidateCreate`, `ValidateReplace`, dll.) dan menggantinya dengan struct tag:
  - `validate:"required,min=3,max=100"`
  - `validate:"required,nim_format"` (Custom validator NIM universitas)
  - `validate:"required,strongpassword"` (Aturan kombinasi huruf besar, kecil, angka, dan pencegahan kata sandi umum)
  - `validate:"omitempty"` dan `validate:"omitnil"` pada model parsial (PATCH).
- **Pentingnya Pointer pada PATCH**: Pada model `PatchUserRequest` dan `PatchStudentRequest`, field opsional wajib bertipe pointer (misal `*string`). Jika menggunakan string biasa, nilai default adalah string kosong `""` sehingga validator tetap memeriksa string kosong tersebut dan memicu error palsu (*false positive*). Dengan tipe pointer, field yang tidak dikirim klien bernilai `nil` dan diabaikan secara aman oleh tag `omitnil`.

### B. Keyset / Cursor-Based Pagination
- Mengatasi kelemahan fatal **Offset Pagination**:
  1. **Degradasi Kinerja Database $O(N)$**: Pada offset besar (misal halaman 10.000 dengan offset 100.000), database PostgreSQL harus membaca dan membuang 100.000 baris sebelum mengambil limit baris berikutnya.
  2. **Page Drift / Duplikasi Baris**: Jika baris baru disisipkan saat pengguna membaca halaman berikutnya, seluruh baris bergeser sehingga pengguna melihat data yang sama dua kali.
- Keyset pagination menggunakan penanda unik dari baris terakhir (`created_at` dan `id`) yang di-encode ke format **Base64URL** (`nanodetik|id`).
- Kueri SQL mengeksekusi klausa tuple berkecepatan tinggi:
  ```sql
  WHERE (created_at, id) < ($1, $2)
  ORDER BY created_at DESC, id DESC
  LIMIT $3;
  ```
- Didukung oleh composite index B-Tree pada database sehingga query dieksekusi secara instan (*Index Scan* berbiaya konstan).

### C. Content Negotiation (RFC 9110)
- REST API membaca header HTTP `Accept` dari klien menggunakan `helper.Negotiate`:
  - `Accept: application/json` atau `*/*` (default) $\rightarrow$ Mengembalikan payload JSON standar `WebResponse`.
  - `Accept: text/csv` $\rightarrow$ Menghasilkan streaming file CSV terformat (`users.csv` atau `students.csv`) dengan header `Content-Disposition: attachment`.
  - Header yang tidak didukung (misal `Accept: application/xml`) $\rightarrow$ Ditolak langsung dengan status **HTTP 406 Not Acceptable**.

### D. Centralized Error Handling (`AppError`)
- Tidak ada controller atau service yang langsung menulis response gagal. Seluruh kegagalan dibungkus ke dalam objek `helper.AppError`:
  - `helper.BadRequest(msg)` (HTTP 400)
  - `helper.Unauthorized(msg)` (HTTP 401)
  - `helper.Forbidden(msg)` (HTTP 403)
  - `helper.NotFound(msg)` (HTTP 404)
  - `helper.Conflict(msg)` (HTTP 409)
  - `helper.UnsupportedMediaType(msg)` (HTTP 415)
  - `helper.Validation(fields)` (HTTP 422 - semantic error)
  - `helper.NotAcceptable(msg)` (HTTP 406)
  - `helper.Internal(err)` (HTTP 500 - generic safe message)
- Central error handler di `config/app.go` mencegat semua error, mencatat structured log sesuai tingkat keparahan (4xx = WARN, 5xx = ERROR), dan mengembalikan format JSON seragam tanpa membocorkan detail teknis internal sistem ke publik.

---

## 3. Struktur Berkas & Penjelasan Fungsi Per-Folder & Per-File

Berikut adalah dokumentasi komprehensif mengenai peranan masing-masing folder dan berkas pada codebase `modul7/api-students`:

```
modul7/
├── LAPORAN_modul7.md                  <- Laporan resmi tugas mandiri (9 bug, 85 error, analisis arsitektur)
├── README.md                          <- Dokumentasi teknis utama Modul 7 ini
└── api-students/                      <- Direktori proyek utama Go Fiber
    ├── app/
    │   ├── model/                     <- [FOLDER MODEL / ENTITY DTO]
    │   │   ├── auth.go                <- Model DTO Auth, RegisterRequest, LoginRequest, AuthUser, JWTPayload
    │   │   ├── student.go             <- Model Student, DTO Create/Replace/Patch (pointer + tags), CursorMeta
    │   │   └── user.go                <- Model User, PatchUserRequest (*string + omitnil), Cursor DTO
    │   ├── repository/                <- [FOLDER REPOSITORY / DATABASE ACCESS]
    │   │   ├── role_repository.go     <- Query RBAC: pemetaan relasional role dan permission
    │   │   ├── student_repository.go  <- Query CRUD mahasiswa, termasuk FindAfterCursor (keyset pagination)
    │   │   ├── token_repository.go    <- Penyimpanan dan pencabutan (revocation) refresh token SHA-256
    │   │   └── user_repository.go     <- Query CRUD pengguna, termasuk FindAfterCursor (keyset pagination)
    │   └── service/                   <- [FOLDER SERVICE / USE CASE & BUSINESS RULES]
    │       ├── auth_service.go        <- Use case autentikasi: Register, Login, RefreshToken, Logout, Me
    │       ├── authz_rules.go         <- Aturan murni otorisasi User (CanAccessUser)
    │       ├── student_authz_rules.go <- Aturan murni otorisasi Mahasiswa (CanAccessStudent & ownership check)
    │       ├── student_rules.go       <- Helper manipulasi domain data mahasiswa (ApplyStudentPatch)
    │       ├── student_service.go     <- Controller/Use case mahasiswa: List (keyset + CSV), CRUD, Anti-spoofing
    │       ├── user_rules.go          <- Helper manipulasi domain data pengguna (ApplyUserPatch)
    │       └── user_service.go        <- Controller/Use case pengguna: List (keyset + CSV), CRUD, AssignRole
    ├── cmd/
    │   └── seed/                      <- [FOLDER CLI / SEEDER]
    │       └── main.go                <- Seeder database akun demo (admin_budi, staff_sari, user_andi, bambang)
    ├── config/                        <- [FOLDER CONFIGURATION & INITIALIZATION]
    │   ├── app.go                     <- Perakitan Fiber, pendaftaran route, dan Centralized ErrorHandler
    │   ├── env.go                     <- Parser & validasi variabel lingkungan (.env)
    │   └── logger.go                  <- Konfigurasi structured logger (slog) dengan rotasi file (lumberjack)
    ├── database/                      <- [FOLDER DATABASE CONNECTION]
    │   └── postgres.go                <- Inisialisasi PostgreSQL connection pool menggunakan pgx/v5
    ├── helper/                        <- [FOLDER HELPER / CROSS-CUTTING UTILITIES]
    │   ├── authz.go                   <- In-memory PermissionSet evaluasi O(1) fail-closed
    │   ├── authz_test.go              <- Unit test pembuktian fail-closed pada PermissionSet
    │   ├── context.go                 <- Ekstraksi context HTTP (CurrentUser, RequestID, RequestContext)
    │   ├── cursor.go                  <- Encoder & Decoder Base64URL untuk Keyset Pagination
    │   ├── errors.go                  <- Struct AppError terpusat & seluruh fungsi konstruktor status HTTP
    │   ├── jwt.go                     <- Pembuat, validator, dan ekstraktor klaim JWT HS256
    │   ├── negotiate.go               <- Parser header Accept (Content Negotiation) & generator streaming CSV
    │   ├── request.go                 <- Parser query string limit, ID parameter URL, dan cursor query
    │   ├── response.go                <- Amplop standar response sukses JSON (WebResponse)
    │   ├── security.go                <- Password hashing (bcrypt cost 12), verifikasi timing-safe, SHA-256
    │   └── validator.go               <- Integrasi go-playground/validator/v10 & aturan custom nim_format
    ├── middleware/                    <- [FOLDER MIDDLEWARE]
    │   ├── auth.go                    <- RequireAuth (JWT verification) & LoginRateLimiter (anti-brute force)
    │   ├── authz.go                   <- RequirePermission (pengecekan izin endpoint via PermissionSet)
    │   └── middleware.go              <- RequestLogger (audit status, duration, user_id) & RequireJSON
    ├── migrations/                    <- [FOLDER DATABASE MIGRATIONS]
    │   ├── 001_create_students.sql    <- Tabel dasar students
    │   ├── 002_auth.sql               <- Tabel users dan refresh_tokens
    │   ├── 003_rbac.sql               <- Tabel roles, permissions, dan role_permissions
    │   ├── 004_student_permissions.sql<- Izin operasional student & penambahan kolom owner_id
    │   ├── 005_cursor_index.sql       <- Composite index B-Tree users(created_at DESC, id DESC)
    │   └── 006_student_cursor_index.sql<- Composite index B-Tree students(created_at DESC, id DESC)
    ├── .env.example                   <- Template konfigurasi environment (PORT, DB, JWT_SECRET, dll.)
    ├── .gitignore                     <- Pengecualian berkas git (.env, binary, logs/)
    ├── go.mod                         <- Deklarasi modul Go dan daftar dependensi pustaka
    ├── go.sum                         <- Checksum integritas pustaka dependensi Go
    ├── main.go                        <- Titik masuk aplikasi: load konfigurasi, koneksi DB, cache RBAC, & start server
    ├── test_modul7.ps1                <- Skrip pengujian otomatis modul 7 via PowerShell
    ├── test_rbac.ps1                  <- Skrip pengujian regresi otorisasi RBAC (22 skenario)
    └── test_runner.js                 <- Skrip Node.js otomatis untuk verifikasi anti-drift, cursor, & omitnil
```

---

## 4. Tabel Berita Acara Perbaikan (9 Bug Sengaja Bagian B)

Modul praktikum sengaja menyertakan 9 kesalahan desain (*intentional bugs*) pada Bagian B untuk menguji kemampuan debugging mahasiswa:

| No | Berkas dan Baris | Kategori Bug | Gejala Masalah | Akar Masalah | Solusi Konkret |
|---|---|---|---|---|---|
| **1** | `app/model/user.go` (`PatchUserRequest`) | Salah Perilaku | Request PATCH sah `{"email":"baru@univ.ac.id"}` gagal divalidasi dengan pesan `Username: minimal 3 karakter` | Field `Username` bertipe string biasa (`string`). Nilai zero value-nya adalah `""` sehingga tag `omitnil` tidak aktif dan string kosong diuji terhadap `min=3`. | Ubah tipe data menjadi pointer: `Username *string` |
| **2** | `config/app.go` (`newErrorHandler`) | Salah Perilaku | Error 404 dicatat sebagai log `ERROR`, sedangkan crash DB 500 dicatat sebagai log `WARN` | Logika perbandingan terbalik: menggunakan `< fiber.StatusInternalServerError` untuk log ERROR. | Ubah operator perbandingan menjadi `>= fiber.StatusInternalServerError` |
| **3** | `middleware/middleware.go` (`RequestLogger`) | Salah Perilaku | Access log di terminal/file selalu menampilkan `status: 200` meskipun respons sebenarnya bernilai 400, 404, atau 500 | Blok atribut log membaca langsung `c.Response().StatusCode()` sebelum dieksekusi handler error. | Ganti dengan membaca variabel lokal `status` yang menangkap kode dari error |
| **4** | `app/service/user_service.go` (`translateError`) | Salah Perilaku | Ketika database down, API mengembalikan response HTTP 200 OK kosong ke klien | Klausa default pada switch-case `translateError` mengembalikan `nil` alih-alih `helper.Internal(err)`. | Tambahkan penanganan `default: return helper.Internal(err)` |
| **5** | `app/repository/user_repository.go` (`FindAfterCursor`) | Salah Perilaku | Paginasi halaman 1 menampilkan user terlama (ID 1, 2, ...), bukan user terbaru | Kueri SQL memakai pengurutan `ORDER BY created_at ASC, id ASC`. | Ubah klausa SQL menjadi `ORDER BY created_at DESC, id DESC` |
| **6** | `helper/validator.go` (`strongpassword`) | Salah Perilaku | Password kuat ditolak, sedangkan password lemah dibiarkan lolos | Logika fungsi validator terbalik: mengembalikan `passwordStrength(...) != ""` (padahal string kosong menandakan valid). | Ubah perbandingan menjadi `passwordStrength(...) == ""` |
| **7** | `helper/negotiate.go` (`Negotiate`) | Salah Perilaku | Request dengan header standar `Accept: */*` ditolak dengan HTTP 406 *Not Acceptable* | Fiber mengembalikan string kosong saat mengevaluasi wildcard `*/*` terhadap opsi format yang ditawarkan. | Tambahkan penanganan eksplisit: `if accept == "" \|\| accept == "*/*" { return offered[0], nil }` |
| **8** | `helper/errors.go` (`Validation`) | Salah Perilaku | Kesalahan input formulir/JSON mengembalikan status HTTP 400 Bad Request, bukan 422 | Konstruktor `Validation` menggunakan status 400 padahal standar REST mensyaratkan 422 untuk semantic error. | Ganti status code menjadi `fiber.StatusUnprocessableEntity` (422) |
| **9** | `config/app.go` & `helper/errors.go` | Ditolak Compiler | Kompilasi gagal: `appErr.cause undefined (type *helper.AppError has no field or method cause)` | Field `cause error` diawali huruf kecil (*unexported*) sehingga tidak dapat diakses dari package luar. | Tambahkan method pengakses `Cause() error` atau panggil `appErr.Error()` |

---

## 5. Analisis Kaskade 85+ Error Refactoring

Saat berpindah dari arsitektur Modul 6 ke Modul 7, penghapusan fungsi `helper.Fail` dan transisi ke validasi deklaratif memicu kaskade kompilasi hingga 85 error. Berikut ringkasan 6 kategori penyelesaiannya:

1. **Kategori A (19 Error) — Eliminasi `helper.Fail` & `helper.FailValidation`**:
   - Seluruh pemanggilan lama diganti dengan melempar konstruktor `helper.AppError` (seperti `helper.BadRequest`, `helper.Unauthorized`, `helper.NotFound`, `helper.Validation`).
2. **Kategori B (14 Error) — Eliminasi Validasi Manual**:
   - Menghapus fungsi imperatif `ValidateCreate`, `ValidateReplace`, `ValidateLogin` dan mendaftarkan tag validasi ke struct model.
3. **Kategori C (12 Error) — Tipe Balik & Signature Service**:
   - Mengubah tanda tangan method service agar mengembalikan `error` murni alih-alih langsung merender context Fiber.
4. **Kategori D (16 Error) — Keyset Pagination & Content Negotiation**:
   - Mengubah query repository `FindAll` menjadi `FindAfterCursor` dengan klausa tuple dan DTO `CursorMeta`.
5. **Kategori E (15 Error) — Cascading Unused Imports**:
   - Pembersihan paket impor Go (`strings`, `strconv`, dll.) yang tidak lagi dipakai setelah logika imperatif dihapus.
6. **Kategori F (9 Error) — Perbaikan 9 Bug Sengaja Bagian B**.

---

## 6. Panduan Menjalankan & Menguji Aplikasi

### Persyaratan Sistem
- **Go**: Versi 1.22 atau lebih baru
- **PostgreSQL**: Versi 15 atau lebih baru (berjalan pada port 5432)
- **Database**: `praktikum_backend`

### Langkah-Langkah Menjalankan

1. **Buka direktori proyek `modul7/api-students`**:
   ```bash
   cd c:\laragon\www\PBEL\modul7\api-students
   ```

2. **Salin berkas konfigurasi `.env`**:
   ```bash
   copy .env.example .env
   ```
   *Pastikan nilai `DB_NAME`, `DB_USER`, `DB_PASSWORD`, dan `JWT_SECRET` sesuai dengan konfigurasi PostgreSQL lokal Anda.*

3. **Jalankan seluruh migrasi database (termasuk composite index keyset)**:
   ```bash
   psql -U postgres -d praktikum_backend -f migrations/001_create_students.sql
   psql -U postgres -d praktikum_backend -f migrations/002_auth.sql
   psql -U postgres -d praktikum_backend -f migrations/003_rbac.sql
   psql -U postgres -d praktikum_backend -f migrations/004_student_permissions.sql
   psql -U postgres -d praktikum_backend -f migrations/005_cursor_index.sql
   psql -U postgres -d praktikum_backend -f migrations/006_student_cursor_index.sql
   ```

4. **Jalankan seeder akun pengguna**:
   ```bash
   go run ./cmd/seed/main.go
   ```

5. **Jalankan server Fiber**:
   ```bash
   go run main.go
   ```
   *Server akan aktif pada port `http://localhost:3000`.*

---

### Panduan Pengujian Menggunakan Postman / cURL

#### 1. Login untuk Mendapatkan Token JWT
- **Method**: `POST`
- **URL**: `http://localhost:3000/api/v1/auth/login`
- **Headers**: `Content-Type: application/json`
- **Body**:
  ```json
  {
    "username": "admin_budi",
    "password": "Password123!"
  }
  ```
- Salin token dari respons: `data.access_token`.

#### 2. Uji Keyset Pagination (Page 1)
- **Method**: `GET`
- **URL**: `http://localhost:3000/api/v1/students?limit=2`
- **Headers**:
  - `Authorization: Bearer <TOKEN>`
  - `Accept: application/json`
- Respons akan mengembalikan 2 data mahasiswa beserta `meta.next_cursor`.

#### 3. Uji Keyset Pagination (Page 2 menggunakan Cursor)
- **Method**: `GET`
- **URL**: `http://localhost:3000/api/v1/students?limit=2&cursor=<NEXT_CURSOR_DARI_PAGE_1>`
- **Headers**:
  - `Authorization: Bearer <TOKEN>`
  - `Accept: application/json`
- Menampilkan data berikutnya tanpa terjadi duplikasi baris (*anti-drift*).

#### 4. Uji Content Negotiation (Ekspor Format CSV)
- **Method**: `GET`
- **URL**: `http://localhost:3000/api/v1/students?limit=10`
- **Headers**:
  - `Authorization: Bearer <TOKEN>`
  - `Accept: text/csv`
- Server akan merespons dengan berkas teks CSV terformat (`attachment; filename="students.csv"`).

#### 5. Uji Format Ditolak (HTTP 406 Not Acceptable)
- **Method**: `GET`
- **URL**: `http://localhost:3000/api/v1/students`
- **Headers**:
  - `Authorization: Bearer <TOKEN>`
  - `Accept: application/xml`
- Server mengembalikan status **HTTP 406** dengan kode error `NOT_ACCEPTABLE`.

#### 6. Uji Validasi Deklaratif & Custom NIM (HTTP 422 Unprocessable Entity)
- **Method**: `POST`
- **URL**: `http://localhost:3000/api/v1/students`
- **Headers**:
  - `Authorization: Bearer <TOKEN>`
  - `Content-Type: application/json`
- **Body (NIM Salah / Field Kosong)**:
  ```json
  {
    "name": "Budi Santoso",
    "nim": "12345",
    "major": "Teknik Kedokteran"
  }
  ```
- Server mengembalikan status **HTTP 422** beserta rincian field yang melanggar aturan deklaratif.

#### 7. Menjalankan Suite Pengujian Otomatis
```bash
# Pengujian Unit Go:
go test -v ./...

# Pengujian Otomatis Node.js (Anti-Drift, Keyset, omitnil PATCH):
node test_runner.js
```

---

## 7. Panduan & Rencana Git Commit Bertahap

Jika dosen Anda menanyakan rekam jejak commit atau Anda ingin membuat commit bertahap yang bersih (*conventional commits*), berikut alur logis yang telah dirancang:

### Commit 1: Inisialisasi Dependensi & Model Error
- **Pesan**: `chore: inisialisasi dependensi validator v10 dan kontrak AppError`
- **Berkas**: `go.mod`, `go.sum`, `helper/errors.go`
- **Penjelasan**: Menambahkan library `go-playground/validator/v10` dan membuat struktur `AppError` untuk standardisasi format JSON response error di seluruh aplikasi.

### Commit 2: Centralized Error Handler & Access Logger
- **Pesan**: `fix(config): konfigurasi central ErrorHandler dan perbaikan status access log`
- **Berkas**: `config/app.go`, `middleware/middleware.go`
- **Penjelasan**: Memperbaiki 2 bug sengaja pada penanganan status error log (4xx sebagai WARN, 5xx sebagai ERROR) dan memastikan access logger membaca status kode yang sebenarnya.

### Commit 3: Validasi Deklaratif & Custom Tag NIM
- **Pesan**: `feat(validator): implementasi validasi deklaratif struct tags dan custom nim_format`
- **Berkas**: `helper/validator.go`, `app/model/student.go`, `app/model/user.go`, `app/model/auth.go`
- **Penjelasan**: Mengganti validasi imperatif dengan struct tags, menambahkan validasi custom `nim_format` dan `strongpassword`, serta mengubah field `PatchUserRequest` dan `PatchStudentRequest` menjadi pointer dengan tag `omitnil`.

### Commit 4: Indeks Komposit Database untuk Keyset Pagination
- **Pesan**: `feat(db): migrasi composite index created_at dan id untuk keyset pagination`
- **Berkas**: `migrations/005_cursor_index.sql`, `migrations/006_student_cursor_index.sql`
- **Penjelasan**: Menambahkan indeks B-Tree majemuk `(created_at DESC, id DESC)` pada tabel `users` dan `students` agar eksekusi kueri kursor mencapai kecepatan $O(1)$.

### Commit 5: Keyset Pagination & Content Negotiation (JSON/CSV)
- **Pesan**: `feat(api): implementasi cursor pagination dan content negotiation RFC 9110`
- **Berkas**: `helper/cursor.go`, `helper/negotiate.go`, `app/repository/`, `app/service/`
- **Penjelasan**: Menyediakan helper encoding/decoding cursor Base64URL, mekanisme pemilihan format response (JSON vs CSV), penanganan wildcard `*/*`, dan status HTTP 406.

### Commit 6: Otomatisasi Pengujian & Dokumentasi Lengkap
- **Pesan**: `test(e2e): penambahan test suite otomatis dan dokumentasi teknis modul 7`
- **Berkas**: `test_runner.js`, `test_modul7.ps1`, `LAPORAN_modul7.md`, `README.md`
- **Penjelasan**: Menyediakan skrip verifikasi otomatis anti-page drift dan mendokumentasikan penyelesaian 85 error serta jawaban atas pertanyaan arsitektur API.

---

## 8. Bank Tanya Jawab Responsi / Ujian Dosen

Berikut adalah persiapan jawaban argumentatif jika dosen pengampu mengajukan pertanyaan saat asistensi atau ujian:

### Q1: Mengapa pada struct PATCH kita wajib memakai tipe data pointer (`*string`), bukan string biasa (`string`)?
> **Jawaban:**  
> Pada bahasa Go, tipe data string biasa memiliki nilai default (*zero value*) berupa string kosong `""`. Ketika klien mengirimkan request PATCH parsial yang hanya mengubah email (`{"email": "baru@mail.com"}`), field `Username` pada struct akan bernilai `""`. Pustaka validator tidak bisa membedakan apakah pengguna sengaja mengosongkan username atau tidak mengirimkan username sama sekali. Akibatnya, aturan `min=3` akan terpicu dan memicu error palsu. Dengan menggunakan tipe pointer (`*string`), field yang tidak dikirimkan klien akan bernilai `nil`. Bersama tag `omitnil`, validator akan melewati field tersebut secara aman.

### Q2: Mengapa Keyset Pagination jauh lebih unggul daripada Offset Pagination pada tabel dengan jutaan data?
> **Jawaban:**  
> Pada Offset Pagination (`OFFSET 100000 LIMIT 10`), PostgreSQL harus memindai dan membuang 100.000 baris sebelum mengambil 10 baris yang diinginkan, sehingga kompleksitasnya bernilai $O(N)$ dan semakin lambat seiring bertambahnya halaman. Selain itu, jika terjadi penambahan data baru di awal tabel saat pengguna berpindah halaman, baris data akan bergeser (*page drift*) dan pengguna melihat data ganda.  
> Sebaliknya, Keyset Pagination menggunakan kondisi langsung `WHERE (created_at, id) < (cursor_time, cursor_id)` yang memanfaatkan indeks B-Tree majemuk. Database langsung melompat (*Index Scan*) ke titik data yang dicari secara $O(1)$ tanpa perlu memindai baris-baris sebelumnya, serta bebas dari pergeseran data.

### Q3: Apakah aman menyimpan `user_id` atau `role` di dalam payload cursor Base64URL?
> **Jawaban:**  
> **Sangat TIDAK AMAN.** Kursor Base64URL hanyalah serialisasi teks biasa tanpa enkripsi maupun tanda tangan kriptografi (*unsigned*). Siapa pun dapat mendekode, mengubah isi nilai, dan meng-encode-nya kembali. Jika identitas pengguna atau hak akses disimpan di dalam kursor dan dipercaya oleh server, penyerang dapat memanipulasi kursor untuk membaca data milik pengguna lain (kerentanan *Insecure Direct Object References* / IDOR). Konteks identitas dan otorisasi wajib selalu diambil dari token JWT yang terverifikasi di header `Authorization`.

### Q4: Apakah semua aturan validasi bisnis bisa dipindahkan ke struct tag validator?
> **Jawaban:**  
> **Tidak.** Tag validator hanya dirancang untuk validasi lokal di memori (*in-memory syntax/format validation*), seperti panjang karakter, format email, regex NIM, dan keberadaan field. Aturan yang melibatkan **pemeriksaan state basis data** (misalnya *Keunikan NIM di database* atau *Pemeriksaan kuota kelas aktif*) serta aturan bisnis lintas entitas yang kompleks tetap wajib diimplementasikan di **Service Layer**.
