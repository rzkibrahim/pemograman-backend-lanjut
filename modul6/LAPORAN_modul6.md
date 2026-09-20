# LAPORAN TUGAS MANDIRI PERTEMUAN 6
## Authorization & Role Based Access Control (RBAC) pada REST API Go Fiber

| Informasi Mahasiswa | Keterangan |
|---|---|
| **Mata Kuliah** | Praktikum Pemrograman Backend Lanjut |
| **Program Studi** | D4 Teknik Informatika, Fakultas Vokasi, Universitas Airlangga |
| **Topik** | Modul 6 — Authorization & Role Based Access Control (RBAC) |
| **Repository GitHub** | `rzkibrahim/pemograman-backend-lanjut` |

---

## 1. Skema Basis Data dan Migrasi

### DDL Skema RBAC (`migrations/003_rbac.sql`):
```sql
-- 1. Tabel roles
CREATE TABLE IF NOT EXISTS roles (
    id          VARCHAR(20)  PRIMARY KEY,
    name        VARCHAR(50)  NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- 2. Tabel permissions
CREATE TABLE IF NOT EXISTS permissions (
    id          VARCHAR(50)  PRIMARY KEY,
    description TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- 3. Tabel role_permissions (Composite Primary Key)
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id       VARCHAR(20) NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id VARCHAR(50) NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX IF NOT EXISTS role_permissions_role_id_idx ON role_permissions(role_id);
CREATE INDEX IF NOT EXISTS role_permissions_permission_id_idx ON role_permissions(permission_id);

-- 4. Seed data Roles
INSERT INTO roles (id, name, description)
VALUES 
    ('admin', 'Administrator', 'Akses penuh ke seluruh modul administratif dan sistem'),
    ('staff', 'Staf Akademik', 'Akses operasional data dan pembacaan umum'),
    ('user',  'Pengguna Reguler', 'Akses mandiri ke data pribadi dan kepemilikan data')
ON CONFLICT (id) DO NOTHING;

-- 5. Seed data Permissions domain User
INSERT INTO permissions (id, description)
VALUES
    ('user:list',        'Melihat daftar seluruh user'),
    ('user:read:any',    'Melihat data profil user mana pun'),
    ('user:update:any',  'Mengubah profil data user mana pun'),
    ('user:delete',      'Menghapus akun user dari sistem'),
    ('role:assign',      'Mengubah role akun milik user lain')
ON CONFLICT (id) DO NOTHING;

-- 6. Hubungkan Permission ke Role
INSERT INTO role_permissions (role_id, permission_id)
VALUES
    ('admin', 'user:list'),
    ('admin', 'user:read:any'),
    ('admin', 'user:update:any'),
    ('admin', 'user:delete'),
    ('admin', 'role:assign'),
    ('staff', 'user:list'),
    ('staff', 'user:read:any')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- 7. Normalisasi data lama sebelum menambahkan Foreign Key
UPDATE users 
SET role = 'user' 
WHERE role IS NULL OR role NOT IN ('admin', 'staff', 'user');

-- 8. Tambahkan FOREIGN KEY constraint pada users.role
ALTER TABLE users 
ADD CONSTRAINT fk_users_role 
FOREIGN KEY (role) REFERENCES roles(id) ON UPDATE CASCADE;
```

### DDL Skema Permissions Mahasiswa & Kepemilikan (`migrations/004_student_permissions.sql`):
```sql
-- 1. Tambahkan permission domain student
INSERT INTO permissions (id, description)
VALUES
    ('student:list',        'Melihat daftar seluruh mahasiswa'),
    ('student:read:any',    'Melihat data mahasiswa mana pun'),
    ('student:create',      'Menambahkan data mahasiswa baru'),
    ('student:update:any',  'Mengubah data mahasiswa milik siapa pun'),
    ('student:delete',      'Menghapus data mahasiswa')
ON CONFLICT (id) DO NOTHING;

-- 2. Hubungkan permission student ke role
INSERT INTO role_permissions (role_id, permission_id)
VALUES
    ('admin', 'student:list'),
    ('admin', 'student:read:any'),
    ('admin', 'student:create'),
    ('admin', 'student:update:any'),
    ('admin', 'student:delete'),
    ('staff', 'student:list'),
    ('staff', 'student:read:any'),
    ('staff', 'student:create')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- 3. Tambahkan kolom owner_id pada tabel students
ALTER TABLE students ADD COLUMN IF NOT EXISTS owner_id INTEGER;

-- 4. Penanganan data lama: kaitkan ke user pertama
UPDATE students 
SET owner_id = (SELECT id FROM users ORDER BY id ASC LIMIT 1) 
WHERE owner_id IS NULL;

-- 5. Tambahkan constraint FOREIGN KEY ke tabel users
ALTER TABLE students 
ADD CONSTRAINT fk_students_owner 
FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS students_owner_id_idx ON students (owner_id);
```

### Alasan Penanganan Data Lama Sebelum `ALTER TABLE ADD CONSTRAINT`:
Pada basis data yang telah berjalan sejak Modul 3 dan Modul 5, tabel `users` dan tabel `students` telah terisi oleh baris data riil.
1. Pada tabel `users`: jika terdapat data lama yang bernilai `NULL` atau berisi string di luar `'admin'`, `'staff'`, dan `'user'`, perintah penambahan batasan foreign key `REFERENCES roles(id)` akan ditolak seketika oleh PostgreSQL karena terjadi inkonsistensi data (*referential integrity violation*). Oleh karena itu, baris `UPDATE users SET role = 'user'` wajib dijalankan terlebih dahulu.
2. Pada tabel `students`: saat kolom baru `owner_id` ditambahkan, seluruh record lama akan bernilai `NULL`. Sebelum batasan foreign key aktif, baris `UPDATE students SET owner_id = (SELECT id FROM users ...)` dijalankan untuk mengaitkan baris data mahasiswa lama ke akun pengguna yang valid di sistem.

### Keunggulan Desain Composite Primary Key pada `role_permissions`:
Tabel relasi `role_permissions` menggunakan composite primary key gabungan `PRIMARY KEY (role_id, permission_id)`. Desain ini memiliki keunggulan:
1. **Pencegahan Data Duplikat**: Menjamin secara mutlak di level engine database bahwa kombinasi satu peran dengan satu izin tidak dapat diduplikasi (misalnya role `admin` tidak bisa memiliki entri ganda untuk `student:list`).
2. **Efisiensi Indeks B-Tree**: Basis data secara otomatis membangun indeks gabungan yang mempercepat query pencarian seluruh izin untuk role tertentu.

### Bukti Verifikasi Relasi & Foreign Key di Database:
Query verifikasi pemetaan permission ke role:
```sql
SELECT role_id, permission_id FROM role_permissions ORDER BY role_id, permission_id;
```
Hasil:
```
 role_id |   permission_id    
---------+--------------------
 admin   | role:assign
 admin   | student:create
 admin   | student:delete
 admin   | student:list
 admin   | student:read:any
 admin   | student:update:any
 admin   | user:delete
 admin   | user:list
 admin   | user:read:any
 admin   | user:update:any
 staff   | student:create
 staff   | student:list
 staff   | student:read:any
 staff   | user:list
 staff   | user:read:any
(15 rows)
```
Tabel `students` membuktikan kolom `owner_id` terpasang dengan batasan:
```
Foreign-key constraints:
    "fk_students_owner" FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
```

---

## 2. Implementasi Otorisasi & Evaluasi Fail-Closed

### 2.1 In-Memory PermissionSet & Fail-Closed (`helper/authz.go`)
Pemeriksaan otorisasi dilakukan menggunakan struktur data dalam memori (*in-memory*) `PermissionSet` yang dimuat sekali saat aplikasi menyala. Struktur ini memanfaatkan `map[role]map[permission]bool` yang memberikan kompleksitas waktu konstan $O(1)$ tanpa membebani database dengan query berulang setiap HTTP request.

Potongan kode `helper/authz.go`:
```go
func (ps *PermissionSet) Can(role, permission string) bool {
	if ps == nil {
		return false
	}
	normalizedRole := strings.ToLower(strings.TrimSpace(role))
	normalizedPerm := strings.ToLower(strings.TrimSpace(permission))

	if normalizedRole == "" || normalizedPerm == "" {
		return false
	}

	ps.mu.RLock()
	defer ps.mu.RUnlock()

	rolePerms, exists := ps.permissions[normalizedRole]
	if !exists || rolePerms == nil {
		return false
	}

	return rolePerms[normalizedPerm]
}
```

### Prinsip Fail-Closed:
Fail-closed adalah filosofi desain keamanan di mana suatu sistem **secara default menolak akses** bila terjadi keraguan, galat, atau kondisi tak terduga:
- Jika `PermissionSet` bernilai `nil` $\to$ tolak (`false`).
- Jika role bernilai kosong atau permission kosong $\to$ tolak (`false`).
- Jika role tidak dikenal di basis data $\to$ tolak (`false`).
- Jika permission tidak terdaftar untuk role tersebut $\to$ tolak (`false`).

Kebalikan dari fail-closed adalah *fail-open* (mengizinkan akses bila tidak menemukan alasan eksplisit untuk menolak). Bila pengembang membuat kesalahan ketik (*typo*) pada sistem fail-open, endpoint rahasia dapat terbuka ke publik tanpa terdeteksi. Pada sistem fail-closed, kesalahan ketik satu huruf langsung menutup akses dan segera terlapor saat pengujian.

### 2.2 Middleware `RequirePermission` (`middleware/authz.go`)
```go
func RequirePermission(perms *helper.PermissionSet, requiredPerm string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := helper.GetCurrentUser(c)
		if !ok || user == nil {
			return helper.Fail(c, fiber.StatusUnauthorized, "token autentikasi diperlukan")
		}

		if perms == nil || !perms.Can(user.Role, requiredPerm) {
			return helper.Fail(c, fiber.StatusForbidden, "akses ditolak: Anda tidak memiliki hak akses ("+requiredPerm+")")
		}

		return c.Next()
	}
}
```

### Mengapa `RequireAuth` Selalu Dipasang Sebelum `RequirePermission`:
Otorisasi (*authorization*) adalah proses menjawab pertanyaan *"Anda boleh melakukan apa?"*. Pertanyaan ini hanya memiliki makna setelah identitas pengguna diketahui secara pasti (*authentication* — *"Siapa Anda?"*). Urutan ini tidak dapat dibalik. Bila `RequirePermission` dieksekusi sebelum `RequireAuth`, sistem belum memiliki konteks pengguna (`currentUser`), sehingga tidak ada role yang bisa diperiksa. Pemasangan `RequireAuth` di depan menjamin:
- Ketiadaan token menghasilkan status **401 Unauthorized** ("Saya tidak kenal Anda, silakan login").
- Token valid namun kekurangan hak menghasilkan status **403 Forbidden** ("Saya kenal Anda, tetapi Anda tidak diizinkan mengakses resource ini").

### Bukti Unit Test Evaluasi Fail-Closed:
```
=== RUN   TestPermissionSet_FailClosed
=== RUN   TestPermissionSet_FailClosed/Admin_memiliki_user:list_->_True
=== RUN   TestPermissionSet_FailClosed/Admin_case-insensitive_->_True
=== RUN   TestPermissionSet_FailClosed/Staff_tidak_memiliki_student:create_->_False
=== RUN   TestPermissionSet_FailClosed/Role_user_kosong_->_False
=== RUN   TestPermissionSet_FailClosed/Fail-closed:_Role_tidak_dikenal_->_False
=== RUN   TestPermissionSet_FailClosed/Fail-closed:_Permission_tidak_terdaftar_->_False
=== RUN   TestPermissionSet_FailClosed/Fail-closed:_Role_string_kosong_->_False
=== RUN   TestPermissionSet_FailClosed/Fail-closed:_Permission_string_kosong_->_False
=== RUN   TestPermissionSet_FailClosed/Fail-closed:_PermissionSet_adalah_nil_->_False
--- PASS: TestPermissionSet_FailClosed (0.00s)
PASS
ok  	modul6/helper	0.702s
```

---

## 3. Proteksi Endpoint Mahasiswa (`students`) & Kepemilikan Data

Endpoint `/api/v1/students` telah diproteksi menyeluruh dengan pembagian dua lapis:
1. **Lapis Middleware**: Dipasang pada endpoint yang keputusannya tidak memerlukan query isi baris data:
   - `GET /api/v1/students` $\to$ mewajibkan permission `student:list` (hanya admin dan staff).
   - `POST /api/v1/students` $\to$ mewajibkan permission `student:create` (admin dan staff).
   - `DELETE /api/v1/students/:id` $\to$ mewajibkan permission `student:delete` (hanya admin).
2. **Lapis Service (Ownership Check)**: Dipasang pada endpoint yang keputusannya memerlukan pengetahuan atas pemilik data (`owner_id`):
   - `GET /api/v1/students/:id`
   - `PUT /api/v1/students/:id`
   - `PATCH /api/v1/students/:id`

### 3.1 Aturan Murni Kepemilikan Mahasiswa (`app/service/student_authz_rules.go`):
```go
package service

import (
	"modul6/app/model"
	"modul6/helper"
)

// CanAccessStudent adalah pure function tanpa dependensi ke Fiber atau Repository
func CanAccessStudent(current model.AuthUser, ownerID int, perms *helper.PermissionSet, anyPermission string) bool {
	if current.ID <= 0 || ownerID <= 0 {
		return false
	}

	// 1. Ownership Check: Pemilik data selalu berhak atas datanya sendiri
	if current.ID == ownerID {
		return true
	}

	// 2. Bukan pemilik: Wajib memiliki permission :any (misal: student:read:any atau student:update:any)
	if perms == nil || anyPermission == "" {
		return false
	}

	return perms.Can(current.Role, anyPermission)
}
```

### 3.2 Pencegahan Spoofing Kepemilikan (`owner_id`):
Pada endpoint `POST /api/v1/students`:
- Nilai `owner_id` **diisi secara sepihak dan otomatis oleh server** dari identitas pemanggil pada access token JWT (`current.ID`).
- DTO `CreateStudentRequest` sengaja tidak menyertakan tag atau field `owner_id`. Bila klien mengirimkan JSON `"owner_id": 9999`, nilai tersebut diabaikan total. Hal ini menutup celah *ownership spoofing* (penyerang mengaku bahwa data dibuat oleh pengguna lain).

### 3.3 Pencatatan Jejak Audit Terstruktur:
Setiap permintaan HTTP yang masuk dicatat oleh `RequestLogger` pada `middleware/middleware.go` dengan menangkap `user_id` dan `role` pemanggil ke dalam file `logs/app.log`:
```json
{"time":"2026-09-18T22:51:54+07:00","level":"INFO","msg":"http request","method":"PUT","path":"/api/v1/students/6","status":200,"user_id":"9","role":"user"}
{"time":"2026-09-18T22:51:54+07:00","level":"WARN","msg":"http request","method":"PUT","path":"/api/v1/students/6","status":403,"user_id":"10","role":"user"}
{"time":"2026-09-18T22:51:54+07:00","level":"WARN","msg":"http request","method":"PUT","path":"/api/v1/students/6","status":403,"user_id":"8","role":"staff"}
{"time":"2026-09-18T22:51:54+07:00","level":"INFO","msg":"http request","method":"PUT","path":"/api/v1/students/6","status":200,"user_id":"7","role":"admin"}
```

### Bukti Unit Test Pure Rules Mahasiswa:
```
=== RUN   TestCanAccessStudent
=== RUN   TestCanAccessStudent/Pemilik_data_sendiri_(role_user)_boleh_akses_tanpa_permission_:any
=== RUN   TestCanAccessStudent/Pemilik_data_sendiri_(role_user)_boleh_update_tanpa_permission_student:update:any
=== RUN   TestCanAccessStudent/Bukan_pemilik_(role_user)_DITOLAK_saat_membaca_data_orang_lain
=== RUN   TestCanAccessStudent/Bukan_pemilik_(role_user)_DITOLAK_saat_mengedit_data_orang_lain
=== RUN   TestCanAccessStudent/Staff_boleh_membaca_data_mahasiswa_milik_orang_lain_(memiliki_student:read:any)
=== RUN   TestCanAccessStudent/Staff_DITOLAK_saat_mengedit_data_mahasiswa_milik_orang_lain_(tidak_memiliki_student:update:any)
=== RUN   TestCanAccessStudent/Admin_boleh_membaca_data_mahasiswa_mana_pun_(memiliki_student:read:any)
=== RUN   TestCanAccessStudent/Admin_boleh_mengubah_data_mahasiswa_mana_pun_(memiliki_student:update:any)
=== RUN   TestCanAccessStudent/Fail-closed:_PermissionSet_bernilai_nil
=== RUN   TestCanAccessStudent/Fail-closed:_current.ID_tidak_valid_(<=_0)
=== RUN   TestCanAccessStudent/Fail-closed:_ownerID_tidak_valid_(<=_0)
=== RUN   TestCanAccessStudent/Fail-closed:_permission_string_kosong
=== RUN   TestCanAccessStudent/Fail-closed:_role_tidak_dikenal
--- PASS: TestCanAccessStudent (0.00s)
PASS
ok  	modul6/app/service	0.753s
```

---

## 4. Bukti Hasil Pengujian Matriks Hak Akses & Pengujian Negatif

Seluruh skenario pengujian dieksekusi secara otomatis melalui skrip `test_rbac.ps1` terhadap server lokal (`http://localhost:3000`) dan dicatat pada berkas `security_test_evidence.log`:

| No | Skenario Pengujian | Role Pemanggil | Kepemilikan | Permission Syarat | Status Expected | Status Actual | Hasil |
|:---:|---|:---:|:---:|:---:|:---:|:---:|:---:|
| 1 | `GET /students` | admin | - | `student:list` | `200 OK` | `200 OK` | **PASS** |
| 2 | `GET /students` | staff | - | `student:list` | `200 OK` | `200 OK` | **PASS** |
| 3 | `GET /students` | user | - | `student:list` | `403 Forbidden` | `403 Forbidden` | **PASS** |
| 4 | `GET /students` (Tanpa Token) | anonymous | - | RequireAuth | `401 Unauthorized` | `401 Unauthorized` | **PASS** |
| 5 | `POST /students` | admin | - | `student:create` | `201 Created` | `201 Created` | **PASS** |
| 6 | `POST /students` | staff | - | `student:create` | `201 Created` | `201 Created` | **PASS** |
| 7 | `POST /students` | user | - | `student:create` | `403 Forbidden` | `403 Forbidden` | **PASS** |
| 8 | **Negatif 1: Spoofing `owner_id`** | staff (ID 8) | - | `student:create` | `201 Created` | `201 Created` | **PASS** |
| 9 | `GET /students/:id` | user (Andi) | **Pemilik** | - (Ownership) | `200 OK` | `200 OK` | **PASS** |
| 10 | `GET /students/:id` | user (Bambang) | Bukan Pemilik | `student:read:any` | `403 Forbidden` | `403 Forbidden` | **PASS** |
| 11 | `GET /students/:id` | staff | Bukan Pemilik | `student:read:any` | `200 OK` | `200 OK` | **PASS** |
| 12 | `GET /students/:id` | admin | Bukan Pemilik | `student:read:any` | `200 OK` | `200 OK` | **PASS** |
| 13 | `PUT /students/:id` | user (Andi) | **Pemilik** | - (Ownership) | `200 OK` | `200 OK` | **PASS** |
| 14 | **Negatif 2: Cross-User Edit** | user (Bambang) | Bukan Pemilik | `student:update:any` | `403 Forbidden` | `403 Forbidden` | **PASS** |
| 15 | `PUT /students/:id` | staff | Bukan Pemilik | `student:update:any` | `403 Forbidden` | `403 Forbidden` | **PASS** |
| 16 | `PUT /students/:id` | admin | Bukan Pemilik | `student:update:any` | `200 OK` | `200 OK` | **PASS** |
| 17 | `DELETE /students/:id` | user | - | `student:delete` | `403 Forbidden` | `403 Forbidden` | **PASS** |
| 18 | `DELETE /students/:id` | staff | - | `student:delete` | `403 Forbidden` | `403 Forbidden` | **PASS** |
| 19 | `DELETE /students/:id` | admin | - | `student:delete` | `204 No Content` | `204 No Content` | **PASS** |
| 20 | `GET /users` | admin | - | `user:list` | `200 OK` | `200 OK` | **PASS** |
| 21 | `GET /users` | staff | - | `user:list` | `200 OK` | `200 OK` | **PASS** |
| 22 | `GET /users` | user | - | `user:list` | `403 Forbidden` | `403 Forbidden` | **PASS** |
| 23 | `GET /users/:id` | user (Andi) | **Pemilik** | - (Ownership) | `200 OK` | `200 OK` | **PASS** |
| 24 | `GET /users/:id` | user (Andi) | Bukan Pemilik | `user:read:any` | `403 Forbidden` | `403 Forbidden` | **PASS** |
| 25 | `PATCH /users/:id/role` | staff | - | `role:assign` | `403 Forbidden` | `403 Forbidden` | **PASS** |
| 26 | `PATCH /users/:id/role` | admin | - | `role:assign` | `200 OK` | `200 OK` | **PASS** |

---

### Detail Pengujian Negatif (Checklist C.3):

#### 1. Pengujian Negatif 1: Spoofing `owner_id` pada `POST /students`
- **Tindakan:** Pengguna `staff_sari` (User ID = 8) mengirimkan permintaan `POST /api/v1/students` dengan menyelundupkan `"owner_id": 9999` di dalam request body.
- **Hasil Pengujian:** Permintaan menghasilkan status `201 Created`. Pada respons data dan di tabel database, nilai `owner_id` tercatat bernilai **`8`** (sesuai ID pengguna yang terotentikasi di token), bukan `9999`.
- **Kesimpulan:** Sistem backend kebal terhadap serangan manipulasi kepemilikan.

#### 2. Pengujian Negatif 2: Cross-User Mutation (`PUT /students/:id`) oleh Role User
- **Tindakan:** Pengguna `user_bambang` (User ID = 10, role `user`) mencoba mengubah data mahasiswa ID `6` yang dimiliki oleh `user_andi` (User ID = 9).
- **Hasil Pengujian:** Permintaan langsung digagalkan dengan status **`403 Forbidden`** dan pesan JSON:
  ```json
  {
    "success": false,
    "message": "akses ditolak: Anda bukan pemilik data ini dan tidak memiliki izin student:update:any"
  }
  ```
- **Kesimpulan:** Otorisasi kepemilikan di layer service berhasil mencegah modifikasi lintas pengguna.

---

## 5. Analisis Jawaban Tugas Mandiri (Bagian C.4)

### A. Mengapa Ownership Check Tidak Bisa Dipindah ke Middleware?
Ownership check (pemeriksaan kepemilikan data) tidak dapat dipindahkan ke layer middleware karena middleware hanya memiliki akses terhadap konteks request HTTP mentah (seperti header token dan URL parameter `:id`), tetapi **tidak mengetahui isi baris data konkret** yang tersimpan di dalam database (yaitu nilai kolom `owner_id`). Rujuk pada implementasi `student_service.go` baris 75–86:
```go
student, err := s.repo.FindByID(ctx, id)
...
if !CanAccessStudent(current, student.OwnerID, s.perms, "student:read:any") { ... }
```
Bila pemeriksaan kepemilikan dipaksakan berada di middleware, maka middleware harus mengimpor repository dan melakukan query database untuk membaca record mahasiswa tersebut, kemudian lapisan service di belakangnya akan meng-query ulang baris yang sama untuk memproses bisnisnya. Hal ini memicu duplikasi query (*redundant round-trip*) yang menurunkan performa serta melanggar prinsip *Clean Architecture* (Modul 4), di mana middleware seharusnya murni menangani protokol HTTP dan terisolasi dari lapisan data.

### B. Risiko Konkret dari Dua Tempat Pemeriksaan Akses (Route & Service), dan Cara Menguranginya
Risiko konkret dari adanya dua tempat pemeriksaan otorisasi yang terpisah (middleware pada `route.go` dan ownership check pada `student_service.go`) adalah **inkonsistensi kebijakan akibat kelalaian pengembang (*developer oversight*)**. Ketika endpoint baru ditambahkan, pengembang sangat mungkin lupa menyematkan middleware `RequirePermission` pada definisi route, atau lupa memanggil fungsi `CanAccessStudent` pada use case baru di service layer, yang berpotensi membuka celah *Broken Access Control*. Cara konkret menguranginya meliputi:
1. Menyediakan *automated integration test* seperti `test_rbac.ps1` yang wajib dieksekusi pada pipeline CI/CD untuk memverifikasi matriks seluruh role sebelum kode di-*merge*.
2. Mewajibkan penulisan unit test pada setiap fungsi otorisasi murni (`student_authz_rules_test.go`) dengan target cakupan 100%.
3. Menerapkan konvensi arsitektur yang seragam dan *code review checklist* yang mewajibkan setiap method service yang menerima parameter `:id` untuk selalu menempatkan pemeriksaan kepemilikan pada baris pertama setelah data diambil.

### C. Apakah RBAC Masih Memadai untuk Kasus "Dosen Wali Hanya Boleh Lihat Mahasiswa Bimbingannya"? Bila Tidak, Apa yang Perlu Ditambahkan?
Model RBAC murni **tidak lagi memadai** untuk menangani kasus ini. RBAC hanya mampu memetakan hak akses secara statis berdasarkan peran global (contoh: peran "dosen" boleh membaca entitas "mahasiswa"), namun tidak dapat mengevaluasi relasi kontekstual dinamis antar-entitas ("dosen A hanya boleh melihat mahasiswa yang dibimbing oleh dosen A, bukan bimbingan dosen B"). Untuk mendukung kasus ini, arsitektur harus diperluas menggunakan model **ABAC (*Attribute-Based Access Control*)** atau **ReBAC (*Relationship-Based Access Control*)**, dengan menambahkan:
1. Atribut relasi eksplisit pada tabel basis data: kolom `dosen_wali_id INT REFERENCES dosens(id)` pada tabel `students`.
2. Evaluasi aturan relasional di lapisan service:
   ```go
   if current.Role == "dosen" && current.DosenID != student.DosenWaliID {
       return helper.Fail(c, fiber.StatusForbidden, "akses ditolak: bukan mahasiswa bimbingan Anda")
   }
   ```

---

## 6. Rencana Commit Bertahap

Untuk memenuhi ketentuan pengerjaan tugas mandiri yang bertahap dan teratur (rentang 18–20 September 2026), commit git dijalankan per bagian fungsional:

### Tahap 1: Jumat, 18 September 2026 — Skema RBAC & Fondasi Authz
```bash
git add migrations/003_rbac.sql
git commit -m "chore(db): add RBAC schema migration (roles, permissions, role_permissions)"

git add helper/authz.go helper/authz_test.go
git commit -m "feat(helper): add PermissionSet with fail-closed Can()"

git add app/repository/role_repository.go
git commit -m "feat(repository): add role repository to load role-permission mapping"

git add main.go
git commit -m "feat(main): load PermissionSet once at startup"
```

### Tahap 2: Sabtu, 19 September 2026 — Middleware, Route, dan Authz Students
```bash
git add middleware/authz.go
git commit -m "feat(middleware): add RequirePermission middleware"

git add route/route.go config/app.go
git commit -m "feat(route): wire RequirePermission onto endpoints without data-dependent access"

git add migrations/004_student_permissions.sql
git commit -m "chore(db): add student_permissions migration and owner_id column"

git add app/service/student_authz_rules.go app/service/student_authz_rules_test.go app/service/authz_rules.go app/service/authz_rules_test.go
git commit -m "feat(service): add CanAccessStudent pure ownership-check function"

git add app/service/student_service.go app/repository/student_repository.go
git commit -m "feat(service): apply ownership check on GET/PUT/PATCH /students/:id"
```

### Tahap 3: Minggu, 20 September 2026 — Ownership, Logging, Testing, Laporan
```bash
git add app/model/student.go app/model/user.go app/service/user_service.go app/repository/user_repository.go
git commit -m "feat(service): auto-fill owner_id from caller identity on POST /students"

git add middleware/middleware.go
git commit -m "feat(middleware): log user_id and role on every students request"

git add test_rbac.ps1 cmd/seed/main.go
git commit -m "test: security matrix testing (admin/staff/user x owner/non-owner) via powershell"

git add security_test_evidence.log
git commit -m "test: negative testing evidence (owner_id spoofing, cross-user PUT with user role)"

git add LAPORAN_modul6.md README.md
git commit -m "docs: add Tugas6 report (schema, code snippets, access matrix, C.4 analysis)"
```

---

## 7. Pernyataan Penggunaan Alat Bantu AI

Penyusunan tugas praktikum ini dibantu oleh alat bantu AI melalui platform Google DeepMind Antigravity IDE yang menyediakan dua model berbeda. Seluruh hasil keluaran AI diperiksa, diuji, dan dipertanggungjawabkan secara mandiri oleh mahasiswa.

### SUMBER BANTUAN

| Model AI | Bagian yang Dibantu |
|---|---|
| **Claude (Anthropic)** | - Membantu perancangan skema relasional RBAC `migrations/003_rbac.sql` (`roles`, `permissions`, `role_permissions` dengan composite primary key) dan penanganan update data lama sebelum pemasangan constraint foreign key. |
| | - Membantu perancangan struktur in-memory `PermissionSet` dan method `Can()` fail-closed pada `helper/authz.go`. |
| | - Membantu implementasi `RoleRepository` di `app/repository/role_repository.go` dan pemuatan skema RBAC saat *startup* di `main.go`. |
| | - Membantu implementasi middleware `RequirePermission` di `middleware/authz.go` dan penyusunan rute terproteksi di `route/route.go` (menjamin `RequireAuth` selalu mendahului `RequirePermission`). |
| | - Membantu implementasi pure function `CanAccessStudent` di `app/service/student_authz_rules.go` dan `CanAccessUser` di `app/service/authz_rules.go` tanpa dependensi eksternal. |
| | - Membantu integrasi *ownership check* dan pencegahan *ownership spoofing* pada `app/service/student_service.go`. |
| | - Membantu pemutakhiran `RequestLogger` pada `middleware/middleware.go` untuk pencatatan `user_id` dan `role` (jejak audit). |
| **Gemini** | - Membantu pembuatan unit test fail-closed di `helper/authz_test.go` dan pure ownership rules di `app/service/student_authz_rules_test.go` serta `app/service/authz_rules_test.go`. |
| | - Membantu pembuatan seeder Go `cmd/seed/main.go` untuk penyiapan akun uji dengan hash bcrypt yang valid. |
| | - Membantu pembuatan skrip pengujian keamanan otomatis `test_rbac.ps1` untuk 26 kasus uji matriks RBAC dan skenario negatif (spoofing `owner_id`, *cross-user mutation*). |
| | - Membantu penyusunan dan penyesuaian format laporan teknis `LAPORAN_modul6.md` agar konsisten dengan struktur baku laporan Modul 5. |
| | - Membantu analisis mendalam untuk Bagian C.4 (pemisahan middleware vs service, mitigasi risiko dua lokasi pengecekan, serta keterbatasan RBAC untuk kasus dosen wali). |
