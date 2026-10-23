# Modul 9 — Transactions & Concurrency

Praktikum Pemrograman Backend Lanjut · D4 Teknik Informatika, Fakultas Vokasi, Universitas Airlangga

Modul ini melanjutkan REST API Go/Fiber dan PostgreSQL dengan fokus menjaga integritas KRS saat request berjalan bersamaan. Source proyek mandiri berada di [`api-students/`](../api-students).

## Tujuan

- Memahami race condition, lost update, dan batas transaksi PostgreSQL pada isolation level READ COMMITTED.
- Memakai constraint database sebagai perlindungan integritas data.
- Menerapkan conditional update, pessimistic locking, dan optimistic locking.
- Mengelola transaction melalui `DBTX` dan `TxManager` agar repository dapat dipakai dengan pool maupun `pgx.Tx`.
- Menjaga konsistensi kuota, enrollment, batas SKS, dan penugasan supervisor.
- Menguji perilaku API dengan request paralel dan membandingkannya dengan implementasi naif.

## Ringkasan Konsep

### Race condition dan ACID

Pola baca–periksa–tulis yang dijalankan tanpa koordinasi memungkinkan beberapa request membaca nilai kuota yang sama dan sama-sama menganggap kursi masih tersedia. Transaksi biasa pada READ COMMITTED tidak otomatis mengunci hasil `SELECT` untuk mencegah pola ini. Constraint dan perubahan bersyarat harus berada pada lapisan yang tepat.

| Sifat | Penerapan |
|---|---|
| Atomicity | Perubahan multi-langkah KRS seperti swap di-commit atau di-rollback sebagai satu kesatuan |
| Consistency | `CHECK` kuota dan `UNIQUE` enrollment menjaga invariant database |
| Isolation | Locking/conditional update mengoordinasikan request bersamaan |
| Durability | Commit PostgreSQL dicatat melalui mekanisme WAL |

### Tiga strategi konkurensi

1. **Conditional update atomik:** `UPDATE ... WHERE enrolled_count < quota`; nol baris berubah berarti kuota penuh.
2. **Pessimistic locking:** `SELECT ... FOR UPDATE` pada mahasiswa saat validasi agregat SKS atau perubahan relasi supervisor.
3. **Optimistic locking:** kolom `version` pada perubahan kuota; versi usang menghasilkan konflik, bukan blind overwrite.

Untuk menghindari deadlock saat mengunci beberapa kelas atau supervisor, lock diambil konsisten menurut ID menaik. External side effects seperti pengiriman email tidak ditempatkan dalam closure transaksi yang dapat diulang.

## Langkah Praktikum

1. Jalankan migration PostgreSQL berurutan, termasuk skema courses/enrollments di `api-students/migrations/007_courses.sql`.
2. Siapkan akun pengujian dan reset data KRS menggunakan migration seed serta `api-students/migrations/reset.sql`.
3. Pelajari abstraksi `DBTX`, repository, dan transaksi pada `api-students/app/repository/`.
4. Jalankan `api-students/tools/loadtest/main.go` untuk menguji mode crowd, credits, swap, dan supervisor.
5. Bandingkan hasil endpoint transaksional dengan endpoint naif yang disediakan khusus sebagai demonstrasi race condition.
6. Periksa respons HTTP, counter `enrolled_count`, jumlah baris enrollment, batas 24 SKS, serta konsistensi kuota supervisor.

Jalankan perintah Go dari `modul9/api-students`:

```powershell
go test ./...
go vet ./...
go run ./cmd/seed/main.go
go run main.go
```

Pastikan PostgreSQL aktif dan konfigurasi lokal dibuat dari `.env.example`. Jangan commit `.env`.

## Endpoint Domain

| Area | Endpoint utama | Aturan |
|---|---|---|
| Courses/KRS | `GET /api/v1/courses`, `POST /api/v1/courses/:id/enroll`, `POST /api/v1/courses/:id/drop` | Kuota dan batas SKS dijaga secara atomik dalam transaksi |
| Swap | `POST /api/v1/courses/swap` | Mengunci resource dengan urutan konsisten dan rollback jika tujuan gagal |
| Quota | `PATCH /api/v1/courses/:id` | Optimistic locking melalui `version` |
| Supervisors | `POST /api/v1/supervisors/:id/assign`, `PATCH /api/v1/supervisors/:id` | RBAC, kuota atomik, dan optimistic locking |
| Supervisions | `POST /api/v1/supervisions/switch` | Perpindahan supervisor transaksional dan lock berurutan |

## Spesifikasi Penerimaan

- Uji beban tidak boleh melampaui kuota; jumlah baris enrollment dan counter harus konsisten.
- Request mahasiswa yang sama tidak boleh melampaui batas kredit semester.
- Swap gagal harus mengembalikan enrollment ke keadaan sebelum transaksi.
- Version lama menghasilkan konflik, dan kuota tidak boleh lebih kecil daripada peserta aktif.
- Kegagalan berulang tidak boleh meninggalkan koneksi `idle in transaction`.
- Uji pembanding boleh menunjukkan race condition hanya pada endpoint naif; jangan menghapus constraint atau memakai hasil pembanding sebagai bukti implementasi aman.

## Tugas Mandiri

Implementasikan domain supervisor dengan constraint kuota, satu supervisor aktif per mahasiswa, assignment/switch atomik, optimistic locking, dan pengujian paralel. Catat hasil eksekusi aktual untuk tiap percobaan; jangan menulis angka hasil sebelum uji benar-benar dilakukan.

Analisis yang perlu disiapkan: perbedaan conditional update dan lock mahasiswa; dampak pessimistic lock pada form yang lama terbuka dibanding optimistic lock; dan alasan side effect eksternal harus berada di luar closure transaction yang dapat di-retry.
