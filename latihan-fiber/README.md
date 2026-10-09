# SIAKAD Mini — RESTful API Back End (UTS)

RESTful API back end untuk **SIAKAD Mini**, sistem informasi akademik sederhana yang mengelola data mahasiswa, mata kuliah, dan Kartu Rencana Studi (KRS) menggunakan **Go (Golang)**, **Fiber v2**, dan **PostgreSQL**.

## 📌 Fitur Utama & 10 Endpoint

1. `POST /api/v1/auth/login` — Login pengguna (Admin & Mahasiswa), return JWT Access Token, dilengkapi rate limiting kegagalan login (HTTP 429).
2. `GET /api/v1/auth/me` — Menampilkan profil akun pengguna yang sedang login (menyertakan data mahasiswa jika role mahasiswa).
3. `GET /api/v1/students` — [Admin] Daftar mahasiswa dengan pagination (`page`, `per_page`), filter (`prodi`, `angkatan`), pencarian (`search`), dan sorting (`nama`, `-ipk_terakhir`).
4. `POST /api/v1/students` — [Admin] Mendaftarkan mahasiswa baru sekaligus membuat akun login dalam 1 transaksi atomik (password default = NIM).
5. `GET /api/v1/students/:id` — [Admin & Mahasiswa Data Sendiri] Detail mahasiswa lengkap beserta daftar mata kuliah KRS, total SKS diambil, dan batas SKS.
6. `PUT /api/v1/students/:id` — [Admin] Memperbarui nama, prodi, angkatan, dan IPK mahasiswa (NIM tidak boleh diubah).
7. `DELETE /api/v1/students/:id` — [Admin] Soft-delete mahasiswa (mengisi `deleted_at`, akun tidak dapat login lagi).
8. `GET /api/v1/courses` — [Semua Role] Katalog mata kuliah beserta perhitungan dinamis kuota `terisi` dan `sisa_kuota`.
9. `POST /api/v1/enrollments` — [Mahasiswa] Mengambil mata kuliah dengan validasi batas SKS (berdasarkan IPK), duplikasi semester (HTTP 409), dan row-locking kuota (`FOR UPDATE`).
10. `DELETE /api/v1/enrollments/:id` — [Mahasiswa Milik Sendiri] Membatalkan mata kuliah dari KRS (kuota kembali bertambah).

## 🚀 Cara Menjalankan

### 1. Konfigurasi Lingkungan (`.env`)
Salin file `.env.example` menjadi `.env` dan sesuaikan kredensial PostgreSQL Anda:
```env
APP_PORT=3000
DB_HOST=127.0.0.1
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=
DB_NAME=siakad_mini
DB_SSLMODE=disable
DB_MAX_CONNS=10
JWT_SECRET=c1f4e92a83b071de49265f279c138d45e0a6b7c89123456789abcdef01234567
JWT_ISSUER=siakad-mini-uts
ACCESS_TOKEN_TTL_MINUTES=60
```

### 2. Jalankan Migrasi & Seeding Data
Seeder akan membuat skema database, 1 akun Admin, 20 akun Mahasiswa, 10 Mata Kuliah, dan KRS awal:
```bash
go run cmd/seed/main.go
```
Akun bawaan seeder:
- **Admin**: `admin@siakad.ac.id` / `admin12345`
- **Mahasiswa 1**: `rina.putri@mhs.siakad.ac.id` / `187221000001` (Password adalah NIM)

### 3. Jalankan Server API
```bash
go run main.go
```
Server akan aktif di `http://localhost:3000`.

### 4. Eksekusi Pengujian Otomatis
Jalankan skrip PowerShell untuk menguji ke-10 endpoint dan seluruh aturan bisnis secara otomatis:
```powershell
powershell -ExecutionPolicy Bypass -File .\test_all_endpoints.ps1
```

## 📄 Laporan Akademik
Laporan resmi dalam format Markdown tersedia pada file:
- `LAPORAN_UTS_PBE.md` (dapat langsung dikonversi ke format `.docx` melalui Pandoc atau aplikasi Markdown viewer/editor).
