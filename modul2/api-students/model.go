package main

import "time"

// Student adalah entitas utama yang merepresentasikan data mahasiswa.
// Field NIM ditambahkan sebagai penanda unik selain ID auto-increment.
// Password di-tag json:"-" agar tidak pernah muncul di respons.
type Student struct {
	ID        int       `json:"id"`
	NIM       string    `json:"nim"`
	Name      string    `json:"name"`
	Grade     float64   `json:"grade"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateStudentRequest digunakan untuk POST ΓÇö semua field wajib diisi.
// Struct terpisah dari Student agar pengguna tidak bisa mengatur ID atau CreatedAt sendiri.
type CreateStudentRequest struct {
	NIM      string  `json:"nim"`
	Name     string  `json:"name"`
	Grade    float64 `json:"grade"`
	IsActive bool    `json:"is_active"`
}

// ReplaceStudentRequest digunakan untuk PUT ΓÇö mengganti seluruh isi.
// Semua field bertipe biasa (bukan pointer) dan semuanya wajib dikirim.
// Field yang tidak dikirim dianggap dikosongkan / bernilai zero value.
type ReplaceStudentRequest struct {
	NIM      string  `json:"nim"`
	Name     string  `json:"name"`
	Grade    float64 `json:"grade"`
	IsActive bool    `json:"is_active"`
}

// PatchStudentRequest digunakan untuk PATCH ΓÇö mengubah sebagian.
// Semua field bertipe pointer agar server bisa membedakan antara:
//   - "tidak dikirim" (nil) ΓåÆ field dibiarkan apa adanya
//   - "dikirim bernilai kosong/false" (pointer ke zero value) ΓåÆ field diubah
//
// Contoh: *bool nil berarti tidak dikirim, *bool ke false berarti sengaja dimatikan.
type PatchStudentRequest struct {
	NIM      *string  `json:"nim,omitempty"`
	Name     *string  `json:"name,omitempty"`
	Grade    *float64 `json:"grade,omitempty"`
	IsActive *bool    `json:"is_active,omitempty"`
}

// WebResponse adalah amplop baku untuk seluruh respons API.
// Bentuk yang konsisten memudahkan pengguna: cukup periksa field "success"
// untuk tahu berhasil atau gagal, tanpa perlu membaca status HTTP secara manual.
type WebResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Meta    *Meta  `json:"meta,omitempty"`
	Errors  any    `json:"errors,omitempty"`
}

// Meta menyertai respons daftar agar pengguna dapat menggambar navigasi halaman.
type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// ListQuery menampung parameter query string yang sudah divalidasi.
type ListQuery struct {
	Page     int
	Limit    int
	Search   string
	Sort     string
	Order    string
	IsActive *bool
	GradeMin *float64
	GradeMax *float64
}
