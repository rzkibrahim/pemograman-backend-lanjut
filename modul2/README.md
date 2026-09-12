# Modul 2 — API Students (REST API & HTTP Deep Dive)

REST API untuk mengelola data mahasiswa, dibangun dengan [Go Fiber v2](https://docs.gofiber.io).

Data disimpan **di memori** (in-memory). Setiap kali server dimatikan, seluruh data hilang.

## Cara Menjalankan

```bash
cd modul2
go mod tidy
go run .
```

Server berjalan di `http://localhost:3000`.

> **Catatan:** Gunakan `go run .` (bukan `go run main.go`) karena proyek ini terdiri dari beberapa berkas Go.

---

## Struktur Berkas

| Berkas       | Tanggung Jawab                                                  |
| ------------ | --------------------------------------------------------------- |
| `main.go`    | Konfigurasi aplikasi, middleware, dan pendaftaran route          |
| `model.go`   | Struct entitas, request, dan respons                            |
| `helper.go`  | Amplop respons, parsing query string, dan daftar putih          |
| `handler.go` | Fungsi penangan untuk setiap endpoint                           |

---

## Bentuk Respons (Amplop Baku)

Seluruh endpoint menggunakan bentuk respons yang sama:

**Berhasil:**
```json
{
  "success": true,
  "message": "...",
  "data": { ... }
}
```

**Berhasil (daftar):**
```json
{
  "success": true,
  "message": "...",
  "data": [ ... ],
  "meta": {
    "page": 1,
    "limit": 10,
    "total": 50,
    "total_pages": 5
  }
}
```

**Gagal biasa:**
```json
{
  "success": false,
  "message": "student tidak ditemukan"
}
```

**Gagal validasi (422):**
```json
{
  "success": false,
  "message": "validasi gagal",
  "errors": {
    "nim": "wajib diisi",
    "name": "wajib diisi"
  }
}
```

---

## Kontrak API

### 1. Health Check

| Item            | Detail                          |
| --------------- | ------------------------------- |
| **Metode**      | `GET`                           |
| **Endpoint**    | `/api/v1/health`                |
| **Parameter**   | ΓÇö                               |
| **Body**        | ΓÇö                               |
| **Status**      | `200 OK`                        |

**Contoh Respons:**
```json
{
  "success": true,
  "message": "server berjalan",
  "data": {
    "timestamp": "2026-08-21T14:30:00Z"
  }
}
```

---

### 2. Daftar Student (dengan Paginasi, Pencarian, Filter, Pengurutan)

| Item            | Detail                                      |
| --------------- | ------------------------------------------- |
| **Metode**      | `GET`                                       |
| **Endpoint**    | `/api/v1/students`                          |
| **Status**      | `200 OK`                                    |

**Query Parameters:**

| Parameter    | Tipe    | Default | Keterangan                                                   |
| ------------ | ------- | ------- | ------------------------------------------------------------ |
| `page`       | int     | `1`     | Halaman keberapa                                             |
| `limit`      | int     | `10`    | Jumlah data per halaman (maks 100)                           |
| `search`     | string  | ΓÇö       | Kata kunci pencarian pada nama dan NIM (case-insensitive)    |
| `sort`       | string  | `id`    | Field pengurutan: `id`, `nim`, `name`, `grade`, `created_at` |
| `order`      | string  | `asc`   | Arah pengurutan: `asc` atau `desc`                           |
| `is_active`  | bool    | ΓÇö       | Filter berdasarkan status aktif: `true` atau `false`         |
| `grade_min`  | float64 | ΓÇö       | Filter nilai minimal                                        |
| `grade_max`  | float64 | ΓÇö       | Filter nilai maksimal                                       |

**Contoh Request:**
```
GET /api/v1/students?page=1&limit=5&search=rizki&sort=name&order=asc&is_active=true
```

**Contoh Respons:**
```json
{
  "success": true,
  "message": "daftar student berhasil diambil",
  "data": [
    {
      "id": 1,
      "nim": "1234567890",
      "name": "Rizki Ibrahim",
      "grade": 87.5,
      "is_active": true,
      "created_at": "2026-08-21T14:30:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "limit": 5,
    "total": 1,
    "total_pages": 1
  }
}
```

---

### 3. Ambil Satu Student

| Item            | Detail                              |
| --------------- | ----------------------------------- |
| **Metode**      | `GET`                               |
| **Endpoint**    | `/api/v1/students/:id`              |
| **Parameter**   | `id` (path) ΓÇö ID student            |
| **Body**        | ΓÇö                                   |
| **Status**      | `200 OK`, `400 Bad Request`, `404 Not Found` |

**Contoh Request:**
```
GET /api/v1/students/1
```

**Contoh Respons (200):**
```json
{
  "success": true,
  "message": "student ditemukan",
  "data": {
    "id": 1,
    "nim": "1234567890",
    "name": "Rizki Ibrahim",
    "grade": 87.5,
    "is_active": true,
    "created_at": "2026-08-21T14:30:00Z"
  }
}
```

**Contoh Respons (404):**
```json
{
  "success": false,
  "message": "student tidak ditemukan"
}
```

---

### 4. Tambah Student Baru

| Item            | Detail                                                     |
| --------------- | ---------------------------------------------------------- |
| **Metode**      | `POST`                                                     |
| **Endpoint**    | `/api/v1/students`                                         |
| **Header**      | `Content-Type: application/json` **(wajib)**               |
| **Status**      | `201 Created`, `400`, `409 Conflict`, `415`, `422`         |

**Contoh Body Request:**
```json
{
  "nim": "1234567890",
  "name": "Rizki Ibrahim",
  "grade": 87.5,
  "is_active": true
}
```

**Contoh Respons (201):**

Header: `Location: /api/v1/students/1`

```json
{
  "success": true,
  "message": "student berhasil dibuat",
  "data": {
    "id": 1,
    "nim": "1234567890",
    "name": "Rizki Ibrahim",
    "grade": 87.5,
    "is_active": true,
    "created_at": "2026-08-21T14:30:00Z"
  }
}
```

**Contoh Respons (409 ΓÇö NIM duplikat):**
```json
{
  "success": false,
  "message": "NIM '1234567890' sudah terdaftar"
}
```

**Contoh Respons (422 ΓÇö Validasi gagal):**
```json
{
  "success": false,
  "message": "validasi gagal",
  "errors": {
    "nim": "wajib diisi",
    "name": "wajib diisi",
    "grade": "harus antara 0 dan 100"
  }
}
```

---

### 5. Ganti Seluruh Data Student (PUT)

| Item            | Detail                                                     |
| --------------- | ---------------------------------------------------------- |
| **Metode**      | `PUT`                                                      |
| **Endpoint**    | `/api/v1/students/:id`                                     |
| **Header**      | `Content-Type: application/json` **(wajib)**               |
| **Parameter**   | `id` (path) ΓÇö ID student                                   |
| **Status**      | `200 OK`, `400`, `404`, `409`, `415`, `422`                |

> **PUT mengganti SELURUH isi.** Semua field wajib dikirim. Field yang tidak dikirim dianggap dikosongkan (zero value Go).

**Contoh Body Request (semua field wajib):**
```json
{
  "nim": "1234567890",
  "name": "Rizki Ibrahim Baru",
  "grade": 91.0,
  "is_active": false
}
```

**Contoh Respons (200):**
```json
{
  "success": true,
  "message": "student berhasil diganti seluruhnya",
  "data": {
    "id": 1,
    "nim": "1234567890",
    "name": "Rizki Ibrahim Baru",
    "grade": 91.0,
    "is_active": false,
    "created_at": "2026-08-21T14:30:00Z"
  }
}
```

**Contoh Respons (422 ΓÇö field tidak lengkap):**
```json
{
  "success": false,
  "message": "validasi gagal",
  "errors": {
    "nim": "wajib diisi pada PUT",
    "name": "wajib diisi pada PUT"
  }
}
```

---

### 6. Ubah Sebagian Data Student (PATCH)

| Item            | Detail                                                     |
| --------------- | ---------------------------------------------------------- |
| **Metode**      | `PATCH`                                                    |
| **Endpoint**    | `/api/v1/students/:id`                                     |
| **Header**      | `Content-Type: application/json` **(wajib)**               |
| **Parameter**   | `id` (path) ΓÇö ID student                                   |
| **Status**      | `200 OK`, `400`, `404`, `409`, `415`, `422`                |

> **PATCH hanya mengubah field yang dikirim.** Field yang tidak dikirim tetap apa adanya. Minimal satu field harus dikirim.

**Contoh Body Request (hanya ubah is_active):**
```json
{
  "is_active": true
}
```

**Contoh Respons (200):**
```json
{
  "success": true,
  "message": "student berhasil diperbarui sebagian",
  "data": {
    "id": 1,
    "nim": "1234567890",
    "name": "Rizki Ibrahim Baru",
    "grade": 91.0,
    "is_active": true,
    "created_at": "2026-08-21T14:30:00Z"
  }
}
```

---

### 7. Hapus Student

| Item            | Detail                                         |
| --------------- | ---------------------------------------------- |
| **Metode**      | `DELETE`                                       |
| **Endpoint**    | `/api/v1/students/:id`                         |
| **Parameter**   | `id` (path) ΓÇö ID student                       |
| **Body**        | ΓÇö                                              |
| **Status**      | `204 No Content`, `400`, `404`                 |

**Contoh Request:**
```
DELETE /api/v1/students/1
```

**Respons:** Status `204 No Content` ΓÇö tanpa body.

---

## Ringkasan Status HTTP yang Digunakan

| Status | Nama                   | Situasi                                                       |
| ------ | ---------------------- | ------------------------------------------------------------- |
| `200`  | OK                     | Pengambilan atau perubahan berhasil                            |
| `201`  | Created                | Penambahan berhasil, disertai header `Location`                |
| `204`  | No Content             | Penghapusan berhasil                                           |
| `400`  | Bad Request            | Body bukan JSON valid, atau `id` bukan angka                   |
| `404`  | Not Found              | Data tidak ditemukan / endpoint tidak ada                       |
| `409`  | Conflict               | NIM sudah terdaftar (duplikat)                                 |
| `415`  | Unsupported Media Type | `Content-Type` bukan `application/json`                        |
| `422`  | Unprocessable Entity   | Validasi isi gagal, dengan rincian per field                   |

---

## Perbedaan PUT dan PATCH

| Aspek                   | PUT                               | PATCH                                  |
| ----------------------- | --------------------------------- | -------------------------------------- |
| Arti                    | Ganti seluruhnya                  | Ubah sebagian                          |
| Field wajib dikirim     | Semua                             | Hanya yang ingin diubah                |
| Tipe field di struct Go | Tipe biasa (`string`, `bool`)     | Pointer (`*string`, `*bool`)           |
| Idempotent              | Ya                                | Tidak selalu                           |
| Bila field tidak dikirim| Dikosongkan (zero value)          | Dibiarkan apa adanya                   |

---

## Contoh Pengujian dengan curl

```bash
# 1. Buat student pertama
curl -i -X POST http://localhost:3000/api/v1/students \
  -H "Content-Type: application/json" \
  -d '{"nim":"240001","name":"Rizki Ibrahim","grade":87.5,"is_active":true}'

# 2. Buat student kedua
curl -i -X POST http://localhost:3000/api/v1/students \
  -H "Content-Type: application/json" \
  -d '{"nim":"240002","name":"Sari Dewi","grade":92.0,"is_active":true}'

# 3. Buat student ketiga
curl -i -X POST http://localhost:3000/api/v1/students \
  -H "Content-Type: application/json" \
  -d '{"nim":"240003","name":"Budi Santoso","grade":75.0,"is_active":false}'

# 4. Daftar semua (paginasi)
curl "http://localhost:3000/api/v1/students?page=1&limit=2"

# 5. Cari berdasarkan nama
curl "http://localhost:3000/api/v1/students?search=rizki"

# 6. Filter aktif + urutkan berdasarkan grade descending
curl "http://localhost:3000/api/v1/students?is_active=true&sort=grade&order=desc"

# 7. Filter berdasarkan rentang grade
curl "http://localhost:3000/api/v1/students?grade_min=80&grade_max=95"

# 8. PUT ΓÇö ganti seluruh isi (semua field wajib)
curl -i -X PUT http://localhost:3000/api/v1/students/1 \
  -H "Content-Type: application/json" \
  -d '{"nim":"240001","name":"Rizki Ibrahim Baru","grade":91.0,"is_active":false}'

# 9. PUT tanpa field lengkap (akan ditolak 422)
curl -i -X PUT http://localhost:3000/api/v1/students/1 \
  -H "Content-Type: application/json" \
  -d '{"name":"Rizki"}'

# 10. PATCH ΓÇö ubah sebagian saja
curl -i -X PATCH http://localhost:3000/api/v1/students/1 \
  -H "Content-Type: application/json" \
  -d '{"is_active":true}'

# 11. POST dengan NIM duplikat (409 Conflict)
curl -i -X POST http://localhost:3000/api/v1/students \
  -H "Content-Type: application/json" \
  -d '{"nim":"240001","name":"Duplikat","grade":80.0,"is_active":true}'

# 12. POST tanpa Content-Type (415)
curl -i -X POST http://localhost:3000/api/v1/students \
  -d '{"nim":"240099","name":"Test"}'

# 13. GET student yang tidak ada (404)
curl -i http://localhost:3000/api/v1/students/999

# 14. POST dengan validasi gagal (422)
curl -i -X POST http://localhost:3000/api/v1/students \
  -H "Content-Type: application/json" \
  -d '{"nim":"","name":"","grade":-5}'

# 15. DELETE
curl -i -X DELETE http://localhost:3000/api/v1/students/3

# 16. ID bukan angka (400)
curl -i http://localhost:3000/api/v1/students/abc
```
