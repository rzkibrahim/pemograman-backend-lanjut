# LAPORAN TUGAS MANDIRI PERTEMUAN 5
## Authentication & Security pada REST API Go Fiber

| Informasi Mahasiswa | Keterangan |
|---|---|
| **Mata Kuliah** | Praktikum Pemrograman Backend Lanjut |
| **Program Studi** | D4 Teknik Informatika, Fakultas Vokasi, Universitas Airlangga |
| **Topik** | Modul 5 — Authentication & Security |
| **Repository GitHub** | `rzkibrahim/pemograman-backend-lanjut` |

---

## 1. Skema Basis Data dan Migrasi

### DDL Skema (`migrations/002_auth.sql`):
```sql
-- 1. Tabel users
CREATE TABLE IF NOT EXISTS users (
    id         SERIAL       PRIMARY KEY,
    username   VARCHAR(50)  NOT NULL,
    email      VARCHAR(255) NOT NULL,
    password   VARCHAR(255) NOT NULL,
    role       VARCHAR(20)  NOT NULL DEFAULT 'user',
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS users_username_lower_key ON users (LOWER(username));
CREATE INDEX IF NOT EXISTS users_email_lower_idx ON users (LOWER(email));
ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'user';

-- 2. Kosongkan tabel users lama
TRUNCATE TABLE users CASCADE;

-- 3. Tabel refresh_tokens
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         SERIAL       PRIMARY KEY,
    user_id    INT          NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64)  NOT NULL,
    expires_at TIMESTAMPTZ  NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS refresh_tokens_token_hash_idx ON refresh_tokens (token_hash);
CREATE INDEX IF NOT EXISTS refresh_tokens_user_id_idx ON refresh_tokens (user_id);
```

### Alasan Pengosongan Tabel (`TRUNCATE TABLE users CASCADE`):
Hashing adalah fungsi matematika satu arah (*one-way cryptographic hash*). Pada pertemuan-pertemuan sebelumnya, password pengguna disimpan dalam bentuk teks mentah (*plaintext*). Plaintext tersebut tidak dapat didekripsi atau diubah secara otomatis menjadi hash yang sah tanpa pengguna memasukkan kembali password aslinya. Oleh karena itu, langkah yang paling aman dan bersih dalam praktikum ini adalah mengosongkan tabel `users` dan mewajibkan pendaftaran ulang akun dengan password yang langsung di-hash menggunakan `bcrypt`. Di lingkungan produksi sungguhan, kebijakan yang diambil adalah memicu prosedur *forced password reset* via email.

### Alasan Pemilihan Bcrypt Cost Factor 12:
Bcrypt sengaja dirancang lambat untuk menggagalkan serangan *brute force* dan penggunaan tabel *rainbow*. Cost factor $12$ berarti algoritma melakukan hashing sebanyak $2^{12} = 4.096$ putaran. Di mesin pengujian modern, waktu eksekusi berada pada kisaran 250–350 ms per hash. Durasi ini hampir tidak terasa oleh pengguna saat login sesekali, tetapi menjadikannya sangat mahal secara komputasi bagi penyerang yang mencoba menebak jutaan kombinasi kata sandi per detik.

### Bukti Format Hash di Database:
Query verifikasi:
```sql
SELECT username, substring(password from 1 for 7) as prefix, role FROM users WHERE username = 'sari_rahma_test';
```
Hasil:
```
username         | prefix  | role
sari_rahma_test  | $2a$12$ | user
```
Prefix `$2a$12$` membuktikan bahwa algoritma yang digunakan adalah bcrypt varian `2a` dengan cost factor `12`.

---

## 2. Implementasi Endpoint Autentikasi

Terdapat 5 endpoint autentikasi utama:
1. `POST /api/v1/auth/register`: Menerima `username`, `email`, `password`. Memvalidasi input, melakukan hashing dengan bcrypt cost 12, menetapkan `role = "user"`, dan mengembalikan status `201 Created` tanpa membocorkan field password.
2. `POST /api/v1/auth/login`: Dilindungi `LoginRateLimiter`. Memverifikasi kecocokan kredensial (dengan mekanisme `VerifyDummyPassword` untuk mencegah *user enumeration*), menerbitkan Access Token JWT (15 menit) dan Refresh Token acak (7 hari) yang disimpan dalam bentuk hash SHA-256 di basis data.
3. `POST /api/v1/auth/refresh`: Memverifikasi hash refresh token, memastikan token belum kedaluwarsa atau dicabut, lalu mengeksekusi **rotasi token** (token lama dicabut dan diganti dengan pasangan token baru).
4. `POST /api/v1/auth/logout`: Mencabut refresh token di basis data sehingga tidak dapat digunakan kembali.
5. `GET /api/v1/auth/me`: Mengembalikan data profil pengguna saat ini berdasarkan ID yang tersimpan di dalam klaim Access Token JWT.

### Bukti Unit Test Aturan Validasi Password & Auth:
```
=== RUN   TestValidatePasswordStrength
--- PASS: TestValidatePasswordStrength (0.00s)
=== RUN   TestValidateRegister
--- PASS: TestValidateRegister (0.00s)
=== RUN   TestValidateLogin
--- PASS: TestValidateLogin (0.00s)
PASS
ok  	modul5/app/service	0.552s
```

---

## 3. Proteksi Endpoint Mahasiswa (`students`)

Endpoint `/api/v1/students` diproteksi secara penuh menggunakan middleware `RequireAuth`. Setiap request wajib menyertakan header:
```http
Authorization: Bearer <access_token>
```
Endpoint `/health` dan `/` tetap dibiarkan publik agar sistem *monitoring* dan *load balancer* dapat memeriksa kesiapan server tanpa kredensial.

### Pembedaan Pesan Error Expired vs Invalid:
- Jika token kedaluwarsa: status `401 Unauthorized` dengan pesan `"access token telah kedaluwarsa"` dan header `WWW-Authenticate: Bearer error="invalid_token", error_description="access token telah kedaluwarsa"`.
- Jika token tidak valid / signature salah: status `401 Unauthorized` dengan pesan `"access token tidak valid"` dan header `WWW-Authenticate: Bearer error="invalid_token", error_description="access token tidak valid"`.

**Alasan Keamanan:** Pembedaan ini sangat penting bagi aplikasi klien (frontend/mobile) agar mengetahui tindakan lanjutan yang tepat: jika token kedaluwarsa (*expired*), klien dapat memanggil endpoint `/api/v1/auth/refresh` secara transparan tanpa mengganggu pengguna. Namun jika token tidak valid (*invalid* / rusak), klien harus menghapus token lokal dan mengarahkan pengguna ke halaman login. Pada saat yang sama, detail kesalahan algoritma internal tidak dibocorkan ke penyerang.

---

## 4. Bukti Hasil Pengujian Keamanan

Seluruh skenario pengujian dieksekusi secara otomatis dan tercatat pada berkas `security_test_evidence.log`:

| No | Skenario Uji | Status HTTP | Bukti / Catatan Penting |
|:---:|---|:---:|---|
| 1 | Register Normal | `201 Created` | Response JSON tidak memuat field password |
| 2 | Register Password Lemah | `422 Unprocessable Entity` | Pesan: `password terlalu umum` (blacklist terdeteksi) |
| 3 | Verifikasi Hash DB | — | Hash diawali `$2a$12$`, role = `user` |
| 4 | Cegah Mass Assignment | `201 Created` | Request mengirim `role: "admin"`, namun role di DB tetap `'user'` |
| 5 | Akses `/students` Tanpa Token | `401 Unauthorized` | Header `WWW-Authenticate: Bearer` disertakan |
| 6 | Pencegahan User Enumeration | `401 Unauthorized` | Login user tidak ada dan password salah mengembalikan status dan pesan yang identik (`username atau password salah`) dengan latensi seimbang |
| 7 | Login Valid | `200 OK` | Mengembalikan `access_token` JWT dan `refresh_token` acak |
| 8 | Akses `/students` dengan Token | `200 OK` | Endpoint terbuka dan mengembalikan data mahasiswa |
| 9 | Token Dimodifikasi 1 Karakter | `401 Unauthorized` | Signature rusak terdeteksi, pesan `access token tidak valid` |
| 10 | Algorithm Confusion (`alg: "none"`) | `401 Unauthorized` | Ditolak oleh keyfunc karena bukan algoritma HMAC |
| 11 | Rotasi Refresh Token | `200 OK` lalu `401 Unauthorized` | Token pertama berhasil; pemakaian kedua terhadap token lama ditolak (`refresh token telah dicabut`) |
| 12 | Endpoint Profil `/auth/me` | `200 OK` | Mengembalikan data user pemilik token tanpa field password |
| 13 | Brute Force Rate Limiter | `429 Too Many Requests` | Permintaan login ke-6 berturut-turut ditolak disertai header `Retry-After: 60` |

---

## 5. Analisis Jawaban Tugas Mandiri

### A. Perbandingan Penyimpanan Token: `localStorage` vs `Cookie httpOnly + Secure + SameSite`
Penyimpanan token di `localStorage` sangat rentan terhadap serangan **Cross-Site Scripting (XSS)** karena nilai token dapat diakses dan dibaca secara langsung oleh JavaScript pihak ketiga atau skrip jahat yang berhasil disuntikkan ke dalam halaman. Sebaliknya, penyimpanan token menggunakan `Cookie` dengan atribut `httpOnly` secara mutlak mencegah token dibaca oleh skrip JavaScript apa pun di peramban, atribut `Secure` memastikan cookie hanya dikirim melalui saluran terenkripsi HTTPS, dan atribut `SameSite=Lax` atau `SameSite=Strict` secara efektif mencegah serangan **Cross-Site Request Forgery (CSRF)**. Oleh karena itu, untuk aplikasi web berbasis browser, `Cookie httpOnly+Secure+SameSite` menawarkan tingkat keamanan yang jauh lebih tinggi dibandingkan `localStorage`.

### B. Dampak Pengubahan Masa Berlaku (TTL) Access Token
Mengubah TTL *Access Token* menciptakan kompromi langsung antara keamanan dan beban sistem (*performance overhead*). Jika TTL dipersingkat (misalnya menjadi 5 menit), tingkat keamanan meningkat pesat karena celah waktu bagi penyerang yang berhasil mencegat token menjadi sangat sempit, namun konsekuensinya klien harus lebih sering menghubungi endpoint `/auth/refresh` sehingga meningkatkan frekuensi request dan beban basis data. Sebaliknya, jika TTL diperpanjang (misalnya menjadi beberapa jam atau hari), lalu lintas jaringan dan beban verifikasi berkurang, namun jika token tersebut bocor, penyerang dapat menggunakannya leluasa selama masa aktif tanpa ada cara mudah dari sisi server untuk mencabutnya sebelum waktu kedaluwarsa tercapai.

### C. Satu Kerentanan Tersisa dan Cara Menutupnya
Kerentanan paling fundamental yang masih tersisa pada sistem saat ini adalah ketiadaan enkripsi saluran komunikasi di tingkat jaringan (**belum menggunakan HTTPS/TLS**): pada jaringan lokal atau Wi-Fi publik, siapapun dapat menjalankan perangkat lunak *packet sniffer* (seperti Wireshark) untuk membaca header `Authorization: Bearer <token>` secara teks mentah (*plaintext*). Cara menutup kerentanan ini adalah dengan memasang sertifikat SSL/TLS (misalnya dari Let's Encrypt) di level web server/reverse proxy (Nginx, Traefik, atau Caddy), mengaktifkan HTTP Strict Transport Security (HSTS), serta mengarahkan (*redirect*) seluruh lalu lintas port 80 (HTTP) secara otomatis ke port 443 (HTTPS).

---

## 6. Pernyataan Penggunaan Alat Bantu AI

Penyusunan tugas praktikum ini dibantu oleh alat bantu AI (Google DeepMind Antigravity) dalam proses perencanaan urutan kompilasi arsitektur Clean Architecture, perancangan skrip pengujian keamanan otomatis menggunakan `curl.exe`, dan perapian format laporan teknis. Seluruh implementasi kode sumber di Go Fiber, konfigurasi migrasi PostgreSQL, eksekusi unit test `go test`, dan verifikasi keluaran status HTTP diuji dan dipertanggungjawabkan secara mandiri oleh mahasiswa.
