// Package model - course.go
// Mendefinisikan entitas mata kuliah, query filter, dan response DTO
// yang menyertakan kuota terisi dan sisa kuota.
package model

import "time"

// Course merepresentasikan entitas mata kuliah pada tabel `courses`.
type Course struct {
	ID        int       `json:"id"`
	KodeMK    string    `json:"kode_mk"`
	NamaMK    string    `json:"nama_mk"`
	SKS       int       `json:"sks"`
	Semester  int       `json:"semester"`
	Kuota     int       `json:"kuota"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CourseItemResponse adalah format DTO untuk endpoint GET /api/v1/courses.
// Menyertakan kolom terisi dan sisa_kuota yang dihitung secara dinamis dari tabel enrollments.
type CourseItemResponse struct {
	ID        int    `json:"id"`
	KodeMK    string `json:"kode_mk"`
	NamaMK    string `json:"nama_mk"`
	SKS       int    `json:"sks"`
	Semester  int    `json:"semester"`
	Kuota     int    `json:"kuota"`
	Terisi    int    `json:"terisi"`
	SisaKuota int    `json:"sisa_kuota"`
}

// CourseQuery menampung filter pencarian mata kuliah:
// - Semester: filter semester tertentu (opsional)
// - Search: pencarian pada kode_mk atau nama_mk (case-insensitive)
// - Available: jika true, hanya menampilkan mata kuliah yang kuotanya belum penuh (sisa_kuota > 0)
type CourseQuery struct {
	Semester  int
	Search    string
	Available bool
}
