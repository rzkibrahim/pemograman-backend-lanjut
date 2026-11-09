package model

import "time"

// Course merepresentasikan satu kelas mata kuliah.
type Course struct {
	ID            int       `json:"id"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	Credits       int       `json:"credits"`
	Quota         int       `json:"quota"`
	EnrolledCount int       `json:"enrolled_count"`
	Version       int       `json:"version"`
	CreatedAt     time.Time `json:"created_at"`
}

// Enrollment merepresentasikan baris pendaftaran mahasiswa ke kelas.
type Enrollment struct {
	ID        int       `json:"id"`
	StudentID int       `json:"student_id"`
	CourseID  int       `json:"course_id"`
	CreatedAt time.Time `json:"created_at"`
}

// MyCourses adalah isi field data pada GET /courses/mine.
//
// Sebelumnya response ini disusun dengan fiber.Map. Hasil JSON-nya sama
// persis, tetapi map tidak memiliki nama dan bentuk yang dapat dibaca swag,
// sehingga dokumentasi API tidak dapat menjelaskan isinya.
type MyCourses struct {
	Courses      []Course `json:"courses"`
	TotalCredits int      `json:"total_credits"`
	MaxCredits   int      `json:"max_credits"`
}

// EnrollRequest adalah payload untuk mendaftar ke kelas.
type EnrollRequest struct {
	StudentID int `json:"student_id" validate:"required,min=1"`
}

// SwapRequest adalah payload untuk menukar kelas.
type SwapRequest struct {
	StudentID    int `json:"student_id"     validate:"required,min=1"`
	FromCourseID int `json:"from_course_id" validate:"required,min=1"`
	ToCourseID   int `json:"to_course_id"   validate:"required,min=1"`
}

// UpdateQuotaRequest adalah payload untuk mengubah kuota kelas (optimistic locking).
type UpdateQuotaRequest struct {
	Quota   int `json:"quota"   validate:"required,min=1"`
	Version int `json:"version" validate:"required,min=1"`
}
