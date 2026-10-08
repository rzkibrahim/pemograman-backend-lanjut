// Package model - enrollment.go
// Mendefinisikan entitas pengambilan mata kuliah (KRS) dan request DTO pendaftaran KRS.
package model

import "time"

// Enrollment merepresentasikan pengambilan mata kuliah oleh mahasiswa pada semester tertentu.
// Memiliki relasi ke Student dan Course dengan unique constraint (student_id, course_id, tahun_akademik).
type Enrollment struct {
	ID            int       `json:"id"`
	StudentID     int       `json:"student_id"`
	CourseID      int       `json:"course_id"`
	TahunAkademik string    `json:"tahun_akademik"`
	CreatedAt     time.Time `json:"created_at"`
}

// CreateEnrollmentRequest adalah DTO request body untuk endpoint POST /api/v1/enrollments (Mahasiswa).
type CreateEnrollmentRequest struct {
	CourseID      int    `json:"course_id"      validate:"required,min=1"`
	TahunAkademik string `json:"tahun_akademik" validate:"required,academic_year"`
}

// EnrollmentResponse adalah DTO respons sukses ketika mahasiswa berhasil mengambil mata kuliah.
type EnrollmentResponse struct {
	ID            int       `json:"id"`
	StudentID     int       `json:"student_id"`
	CourseID      int       `json:"course_id"`
	KodeMK        string    `json:"kode_mk,omitempty"`
	NamaMK        string    `json:"nama_mk,omitempty"`
	SKS           int       `json:"sks,omitempty"`
	TahunAkademik string    `json:"tahun_akademik"`
	CreatedAt     time.Time `json:"created_at"`
}
