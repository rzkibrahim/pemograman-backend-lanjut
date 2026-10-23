# Modul 10 — API dengan MongoDB

Praktikum Pemrograman Backend Lanjut · D4 Teknik Informatika, Fakultas Vokasi, Universitas Airlangga

Modul ini melanjutkan API Modul 9 dengan PostgreSQL dan MongoDB. PostgreSQL tetap menjadi sumber data mahasiswa, kelas, KRS, kuota, dan SKS; MongoDB menyimpan ulasan kelas dan portofolio mahasiswa. Kode API berada di [`api-students/`](./api-students).

## Cakupan

- MongoDB Go Driver v2, koneksi `Ping`, dan health check PostgreSQL + MongoDB.
- Dokumen review dengan balasan embedded, tag, cursor ObjectID, unique compound index, multikey index, update atomik, dan summary `$facet`.
- Dokumen portofolio dengan unique index pada `student_id`, batas 20 proyek atomik, `$pull`, dan agregasi dua tahap `$unwind`.
- Validasi payload melalui struct bertipe; body tidak dipakai sebagai filter MongoDB.
- Pengecekan kelas/enrollment dan mahasiswa di PostgreSQL sebelum menulis dokumen MongoDB.

## Prasyarat

- Go `1.26+` (versi yang dideklarasikan di `api-students/go.mod`).
- PostgreSQL `14+` dan MongoDB Community `9.0+`, atau MongoDB Atlas.
- `psql`, `mongosh`, dan (opsional) MongoDB Compass.

### Instalasi MongoDB

Pilih instruksi resmi sesuai OS untuk memasang server, shell, dan Compass:

- [Windows](https://www.mongodb.com/docs/manual/tutorial/install-mongodb-on-windows/): unduh MongoDB Community MSI. Pasang [mongosh](https://www.mongodb.com/docs/mongodb-shell/install/) dan [Compass](https://www.mongodb.com/try/download/compass) sebagai aplikasi terpisah bila belum tersedia.
- [macOS](https://www.mongodb.com/docs/manual/tutorial/install-mongodb-on-os-x/): ikuti panduan MongoDB Community untuk Homebrew; pasang mongosh dan Compass bila diperlukan.
- [Ubuntu 22.04 / 24.04](https://www.mongodb.com/docs/manual/tutorial/install-mongodb-on-ubuntu/): ikuti panduan apt resmi untuk rilis Ubuntu yang dipakai. Jika menambahkan key secara manual, nama file key MongoDB 9 adalah `server-9.asc` (bukan `server-9.0.asc` atau `server-8.0.asc`).
- OS lain dapat menggunakan Atlas. Atur `MONGO_URI` ke connection string Atlas dan batasi akses jaringan sesuai konfigurasi cluster.

Verifikasi service lokal dan shell:

```powershell
mongosh "mongodb://127.0.0.1:27017" --eval "db.runCommand({ ping: 1 })"
```

## Menyiapkan aplikasi

Jalankan perintah dari `modul10/api-students`:

```powershell
Copy-Item .env.example .env
```

Isi `DB_PASSWORD` dan `JWT_SECRET` (minimal 32 karakter acak). Sesuaikan `MONGO_URI` dan `MONGO_DB` jika memakai Atlas. File `.env` sudah diabaikan Git; jangan unggah atau commit file tersebut.

Jalankan semua migration PostgreSQL secara berurutan. Contoh dengan `psql` lokal:

```powershell
1..10 | ForEach-Object {
  $migration = "migrations/{0:D3}_*.sql" -f $_
  Get-ChildItem $migration | ForEach-Object { psql -U postgres -d praktikum_backend -f $_.FullName }
}
```

Jika shell tidak mendukung glob tersebut, jalankan file `migrations/001_*.sql` sampai `migrations/010_*.sql` satu per satu. Migration `010_review_permissions.sql` menambahkan `review:moderate` untuk admin. Setelah migration, jalankan seeder dan test:

```powershell
go mod tidy
go run ./cmd/seed/main.go
go test ./...
```

Seeder membuat akun pengujian `admin_budi`, `staff_sari`, `user_andi`, dan `user_bambang` dengan password `Password123!`, serta akun load test. Jalankan server:

```powershell
go run main.go
```

Server default di `http://localhost:3000`. Startup menguji koneksi kedua database dan membuat index MongoDB; jika MongoDB tidak tersedia, aplikasi tidak akan mulai. `/api/v1/health` mengembalikan `503` bila salah satu database tidak sehat. Endpoint course yang hanya memakai PostgreSQL, misalnya `GET /api/v1/courses/`, tetap independen dari koneksi MongoDB setelah aplikasi berjalan.

## Endpoint

Semua endpoint API memakai prefix `/api/v1` dan memerlukan Bearer access token, kecuali root health sesuai routing aplikasi.

| Method | Path | Keterangan |
|---|---|---|
| `GET` | `/courses/:id/reviews?tag=...&before=<ObjectID>&limit=10` | Daftar ulasan dengan filter dan cursor |
| `GET` | `/courses/:id/reviews/summary` | Rata-rata, distribusi rating, dan lima tag teratas |
| `POST` | `/courses/:id/reviews` | Buat ulasan; user harus terdaftar pada kelas di PostgreSQL |
| `GET` | `/reviews/:id` | Detail ulasan |
| `PATCH` | `/reviews/:id` | Ubah rating, komentar, atau tag milik sendiri |
| `DELETE` | `/reviews/:id` | Hapus ulasan sendiri; admin dengan `review:moderate` dapat menghapus ulasan lain |
| `POST` | `/reviews/:id/replies` | Tambah balasan; maksimum 50 balasan per ulasan |
| `POST` | `/reviews/:id/helpful` | Tandai membantu satu kali, bukan pada ulasan sendiri |
| `GET` | `/students/:id/portfolio` | Ambil portofolio; mahasiswa harus ada di PostgreSQL |
| `PUT` | `/students/:id/portfolio` | Buat/perbarui headline dan bio |
| `POST` | `/students/:id/portfolio/projects` | Tambah proyek; batas 20 diberlakukan atomik |
| `DELETE` | `/students/:id/portfolio/projects/:projectId` | Hapus proyek embedded dengan `$pull` |
| `GET` | `/portfolios/stats/technologies` | Top 10 teknologi lintas portofolio |

Portofolio mengikuti ownership mahasiswa pada PostgreSQL: pemilik dapat mengelola miliknya, sedangkan role dengan permission `student:read:any` atau `student:update:any` dapat mengakses data lintas pemilik sesuai operasinya. Semua payload ditangani struct bertipe, bukan `map[string]any` sebagai filter.

Contoh request setelah memperoleh token:

```powershell
$headers = @{ Authorization = "Bearer $token"; "Content-Type" = "application/json" }
Invoke-RestMethod -Method Put -Uri http://localhost:3000/api/v1/students/1/portfolio `
  -Headers $headers -Body '{"headline":"Backend Developer","bio":"Membangun layanan Go dan sistem basis data."}'

Invoke-RestMethod -Method Post -Uri http://localhost:3000/api/v1/students/1/portfolio/projects `
  -Headers $headers -Body '{"title":"Sistem Akademik","technologies":["Go","PostgreSQL"],"link":"https://example.com/project","year":2026}'
```

## Latihan mongosh

Skrip latihan memakai database terpisah `praktikum_backend_latihan` dan menghapus collection latihan di database itu saja:

```powershell
mongosh .\api-students\latihan.js
```

Skrip mendemonstrasikan insert, ObjectID timestamp, `$push`, `$addToSet` + `$inc`, index unik gabungan, multikey index, dan `explain("executionStats")`.

## Pemeriksaan dan bukti tugas

```powershell
cd modul10/api-students
go test ./...
```

Gunakan endpoint di atas untuk mengambil bukti cURL/PowerShell. Untuk membuktikan batas proyek atomik, buat portofolio dengan 18 proyek lalu kirim 10 request `POST` serentak; tepat dua request penambahan harus berhasil. Periksa penggunaan index dengan `explain("executionStats")` di mongosh/Compass dan ambil screenshot satu dokumen lengkap di Compass.

Bukti screenshot, hasil request aktual, analisis C.4, dan PDF `Tugas10_NIM.pdf` harus dibuat dari lingkungan mahasiswa; README tidak mengklaim bukti runtime yang belum dijalankan. Sebelum pengumpulan, pastikan `.env` tidak masuk repo/PDF dan cantumkan link repository pada halaman pertama PDF.

## Struktur utama

- `app/model/`: model domain dan request bertipe, tanpa tipe driver MongoDB.
- `app/repository/`: adapter PostgreSQL dan MongoDB, termasuk BSON document internal.
- `app/service/`: aturan bisnis, validasi relasi antar database, dan ownership.
- `database/`: inisialisasi koneksi PostgreSQL dan MongoDB.
- `route/`: registrasi endpoint Fiber.
- `migrations/`: skema PostgreSQL dan permission RBAC.
