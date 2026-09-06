// Package model berisi definisi entitas inti (domain) dan DTO (Data Transfer Object)
// yang dipakai oleh seluruh lapisan aplikasi. Package ini TIDAK BOLEH mengimpor
// package lain di dalam proyek (zero internal dependencies) agar tetap menjadi
// fondasi paling bawah pada diagram dependensi Clean Architecture.
package model

import "time"

// Student adalah entitas utama yang merepresentasikan data mahasiswa.
// Struct ini digunakan oleh repository (untuk membaca/menulis dari database)
// maupun oleh service (untuk menjalankan business rules).
type Student struct {
	ID        int       `json:"id"`         // Primary key, di-generate oleh database (SERIAL)
	NIM       string    `json:"nim"`        // Nomor Induk Mahasiswa, harus unik (case-insensitive)
	Name      string    `json:"name"`       // Nama lengkap mahasiswa
	Grade     float64   `json:"grade"`      // Nilai, rentang 0–100
	IsActive  bool      `json:"is_active"`  // Status keaktifan mahasiswa
	CreatedAt time.Time `json:"created_at"` // Waktu pembuatan record, diisi otomatis oleh database
}

// CreateStudentRequest adalah DTO untuk endpoint POST /api/v1/students.
// Seluruh field wajib diisi; validasi dilakukan oleh ValidateCreate di student_rules.go.
type CreateStudentRequest struct {
	NIM      string  `json:"nim"`       // NIM wajib diisi dan tidak boleh kosong
	Name     string  `json:"name"`      // Nama wajib diisi dan tidak boleh kosong
	Grade    float64 `json:"grade"`     // Nilai harus dalam rentang 0–100
	IsActive bool    `json:"is_active"` // Status keaktifan (default: false jika tidak dikirim)
}

// ReplaceStudentRequest adalah DTO untuk endpoint PUT /api/v1/students/:id.
// Semantik PUT mengharuskan client mengirim seluruh field (full replacement).
// Validasi dilakukan oleh ValidateReplace di student_rules.go.
type ReplaceStudentRequest struct {
	NIM      string  `json:"nim"`       // NIM wajib diisi pada PUT
	Name     string  `json:"name"`      // Nama wajib diisi pada PUT
	Grade    float64 `json:"grade"`     // Nilai harus dalam rentang 0–100
	IsActive bool    `json:"is_active"` // Status keaktifan
}

// PatchStudentRequest digunakan untuk PATCH -- mengubah sebagian.
// Semua field bertipe pointer agar bisa membedakan "tidak dikirim" (nil)
// dengan "dikirim bernilai kosong/false" (pointer ke zero value).
type PatchStudentRequest struct {
	NIM      *string  `json:"nim,omitempty"`
	Name     *string  `json:"name,omitempty"`
	Grade    *float64 `json:"grade,omitempty"`
	IsActive *bool    `json:"is_active,omitempty"`
}

// WebResponse adalah amplop (envelope) baku untuk seluruh respons API.
// Setiap respons JSON yang dikirim ke client mengikuti format ini agar
// konsisten dan mudah di-parse oleh frontend.
type WebResponse struct {
	Success bool   `json:"success"`           // true jika permintaan berhasil, false jika gagal
	Message string `json:"message"`           // pesan deskriptif untuk pengguna
	Data    any    `json:"data,omitempty"`    // payload data (hanya ada jika sukses)
	Meta    *Meta  `json:"meta,omitempty"`    // metadata paginasi (hanya pada respons daftar)
	Errors  any    `json:"errors,omitempty"` // detail error per field (hanya pada validasi gagal)
}

// Meta menyertai respons daftar agar pengguna dapat menggambar navigasi halaman.
type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// ListQuery menampung parameter query string yang sudah divalidasi dan
// dinormalisasi oleh helper.ParseListQuery. Struct ini menjadi kontrak antara
// lapisan helper (pembaca HTTP) dan lapisan repository (pembuat SQL query).
type ListQuery struct {
	Page     int      // nomor halaman saat ini (minimal 1)
	Limit    int      // jumlah data per halaman (1–100)
	Search   string   // kata kunci pencarian nama/NIM (opsional)
	Sort     string   // kolom pengurutan: id, nim, name, grade, created_at
	Order    string   // arah pengurutan: "asc" atau "desc"
	IsActive *bool    // filter status keaktifan (nil = semua)
	GradeMin *float64 // batas bawah nilai (nil = tanpa batas)
	GradeMax *float64 // batas atas nilai (nil = tanpa batas)
}

// Offset menghitung berapa baris yang dilewati untuk halaman ini.
func (q ListQuery) Offset() int {
	return (q.Page - 1) * q.Limit
}