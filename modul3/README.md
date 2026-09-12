# Modul 3 — Database & Repository Pattern (API Students)

REST API untuk mengelola data mahasiswa (students) dengan penyimpanan di PostgreSQL, menggunakan pola repository pattern.

## Skema Tabel

```sql
CREATE TABLE IF NOT EXISTS students (
    id         SERIAL        PRIMARY KEY,
    nim        VARCHAR(20)   NOT NULL,
    name       VARCHAR(100)  NOT NULL,
    grade      NUMERIC(5,2)  NOT NULL DEFAULT 0,
    is_active  BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- Keunikan NIM (case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS students_nim_lower_key
    ON students (LOWER(nim));

-- Indeks pencarian nama
CREATE INDEX IF NOT EXISTS students_name_lower_idx
    ON students (LOWER(name));

-- Indeks penyaringan grade
CREATE INDEX IF NOT EXISTS students_grade_idx
    ON students (grade);
```

## Cara Menyiapkan Basis Data dari Nol

1. **Pastikan PostgreSQL sudah berjalan** (misal lewat Laragon atau instalasi mandiri).

2. **Buat basis data:**
   ```bash
   psql -U postgres -c "CREATE DATABASE praktikum_backend;"
   ```

3. **Jalankan migrasi:**
   ```bash
   psql -U postgres -d praktikum_backend -f migrations/001_create_students.sql
   ```

4. **Verifikasi tabel berhasil dibuat:**
   ```bash
   psql -U postgres -d praktikum_backend -c "\d students"
   ```

## Variabel Environment

Salin `.env.example` menjadi `.env`, lalu isi nilainya:

```bash
cp .env.example .env
```

| Variabel       | Keterangan                          | Contoh Nilai         |
|----------------|-------------------------------------|----------------------|
| `APP_PORT`     | Port yang dipakai server            | `3000`               |
| `DB_HOST`      | Host basis data                     | `localhost`           |
| `DB_PORT`      | Port basis data                     | `5432`               |
| `DB_USER`      | Username basis data                 | `postgres`            |
| `DB_PASSWORD`  | Password basis data                 | *(isi sendiri)*       |
| `DB_NAME`      | Nama basis data                     | `praktikum_backend`   |
| `DB_SSLMODE`   | Mode SSL koneksi                    | `disable`             |
| `DB_MAX_CONNS` | Jumlah koneksi maksimal di pool     | `10`                  |

> **Penting:** Berkas `.env` berisi kredensial dan **tidak boleh di-commit** ke Git. Gunakan `.env.example` sebagai panduan.

## Cara Menjalankan

```bash
go mod tidy
go run .
```

Server akan berjalan di `http://localhost:3000` (atau sesuai `APP_PORT`).

## Endpoint

| Method   | Path                    | Keterangan                          |
|----------|-------------------------|-------------------------------------|
| `GET`    | `/api/v1/health`        | Health check (termasuk cek DB)      |
| `GET`    | `/api/v1/students`      | Daftar student (paginasi, filter)   |
| `GET`    | `/api/v1/students/:id`  | Detail satu student                 |
| `POST`   | `/api/v1/students`      | Tambah student baru                 |
| `PUT`    | `/api/v1/students/:id`  | Ganti seluruh data student          |
| `PATCH`  | `/api/v1/students/:id`  | Ubah sebagian data student          |
| `DELETE` | `/api/v1/students/:id`  | Hapus student                       |

### Query Parameters (GET daftar)

| Parameter   | Tipe     | Default | Keterangan                              |
|-------------|----------|---------|-----------------------------------------|
| `page`      | int      | 1       | Halaman ke-                             |
| `limit`     | int      | 10      | Jumlah per halaman (maks 100)           |
| `search`    | string   | -       | Cari di nama dan NIM (ILIKE)            |
| `sort`      | string   | `id`    | Kolom pengurutan (id/nim/name/grade/created_at) |
| `order`     | string   | `asc`   | Arah pengurutan (asc/desc)              |
| `is_active` | bool     | -       | Filter berdasarkan status aktif         |
| `grade_min` | float    | -       | Filter nilai minimum                   |
| `grade_max` | float    | -       | Filter nilai maksimum                   |

## Struktur Proyek

```
api-students/
+-- app/
ª   +-- model/
ª   ª   +-- student.go         # Struct entitas, request, dan respons
ª   +-- repository/
ª       +-- student_repository.go  # Kontrak dan implementasi akses data
+-- config/
ª   +-- env.go                 # Memuat variabel environment
+-- database/
ª   +-- postgres.go            # Koneksi dan connection pool
+-- migrations/
ª   +-- 001_create_students.sql  # Skema tabel
+-- .env                       # Konfigurasi lokal (JANGAN di-commit)
+-- .env.example               # Contoh isian untuk rekan Anda
+-- .gitignore
+-- main.go                    # Entry point dan routing
+-- handler.go                 # Handler HTTP (method pada StudentHandler)
+-- helper.go                  # Fungsi pembantu respons dan parsing
+-- go.mod
+-- go.sum
```
