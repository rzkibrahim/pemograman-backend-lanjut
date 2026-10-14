// Package model - student.go
// Mendefinisikan entitas mahasiswa, DTO create/update, query pagination,
// serta DTO detail mahasiswa beserta perhitungan batas SKS.
package model

import "time"

// Student merepresentasikan entitas mahasiswa pada tabel `students`.
type Student struct {
	ID          int        `json:"id"`
	UserID      int        `json:"user_id"`
	NIM         string     `json:"nim"`
	Nama        string     `json:"nama"`
	Prodi       string     `json:"prodi"`
	Angkatan    int        `json:"angkatan"`
	IPKTerakhir float64    `json:"ipk_terakhir"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// StudentListItem adalah format DTO mahasiswa untuk endpoint GET /api/v1/students.
// Sesuai dengan spesifikasi contoh: id, nim, nama, prodi, angkatan, ipk_terakhir.
type StudentListItem struct {
	ID          int     `json:"id"`
	NIM         string  `json:"nim"`
	Nama        string  `json:"nama"`
	Prodi       string  `json:"prodi"`
	Angkatan    int     `json:"angkatan"`
	IPKTerakhir float64 `json:"ipk_terakhir"`
}

// CreateStudentRequest adalah DTO request body untuk endpoint POST /api/v1/students (Admin).
type CreateStudentRequest struct {
	NIM         string   `json:"nim"          validate:"required,nim_12digit"`
	Nama        string   `json:"nama"         validate:"required,min=2,max=100"`
	Email       string   `json:"email"        validate:"required,email"`
	Prodi       string   `json:"prodi"        validate:"required,min=2,max=100"`
	Angkatan    int      `json:"angkatan"     validate:"required,angkatan_valid"`
	IPKTerakhir *float64 `json:"ipk_terakhir" validate:"omitempty,min=0,max=4"`
}

// UpdateStudentRequest adalah DTO request body untuk endpoint PUT /api/v1/students/{id} (Admin).
// Catatan: Sesuai spesifikasi, field NIM tidak boleh diubah.
type UpdateStudentRequest struct {
	Nama        string   `json:"nama"         validate:"required,min=2,max=100"`
	Prodi       string   `json:"prodi"        validate:"required,min=2,max=100"`
	Angkatan    int      `json:"angkatan"     validate:"required,angkatan_valid"`
	IPKTerakhir *float64 `json:"ipk_terakhir" validate:"omitempty,min=0,max=4"`
}

// StudentListQuery menampung parameter query string untuk filter, search, sort, dan pagination.
type StudentListQuery struct {
	Page     int
	PerPage  int
	Prodi    string
	Angkatan int
	Search   string
	Sort     string // "nama", "-nama", "ipk_terakhir", "-ipk_terakhir"
}

// EnrolledCourseItem merepresentasikan mata kuliah yang diambil oleh mahasiswa pada KRS.
type EnrolledCourseItem struct {
	EnrollmentID  int    `json:"enrollment_id"`
	CourseID      int    `json:"course_id"`
	KodeMK        string `json:"kode_mk"`
	NamaMK        string `json:"nama_mk"`
	SKS           int    `json:"sks"`
	Semester      int    `json:"semester"`
	TahunAkademik string `json:"tahun_akademik"`
}

// StudentDetailResponse adalah DTO untuk endpoint GET /api/v1/students/{id}.
// Memuat data profil mahasiswa, daftar mata kuliah, total SKS yang diambil,
// dan batas SKS maksimal sesuai business rule IPK.
type StudentDetailResponse struct {
	ID          int                  `json:"id"`
	NIM         string               `json:"nim"`
	Nama        string               `json:"nama"`
	Prodi       string               `json:"prodi"`
	Angkatan    int                  `json:"angkatan"`
	IPKTerakhir float64              `json:"ipk_terakhir"`
	TotalSKS    int                  `json:"total_sks"`
	BatasSKS    int                  `json:"batas_sks"`
	Courses     []EnrolledCourseItem `json:"courses"`
}

// HitungBatasSKS mengimplementasikan Business Rule 1 UTS:
// IPK >= 3.00 -> 24 SKS
// IPK 2.50 - 2.99 -> 21 SKS
// IPK < 2.50 -> 18 SKS
func HitungBatasSKS(ipk float64) int {
	if ipk >= 3.00 {
		return 24
	}
	if ipk >= 2.50 {
		return 21
	}
	return 18
}
