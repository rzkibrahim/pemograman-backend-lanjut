package main

import "fmt"

// RunStructDemo menjalankan demo struct Student dari BAGIAN 3
func RunStructDemo() {
	RunPointerDemo()

	// ========================================
	// BAGIAN 3: Struct Student & Method (dari folder tugas-mandiri/4-struct-student)
	// ========================================

	fmt.Println("\n=== Struct Student ===")
	student := Student{ID: 1, Name: "Rizki", Grade: 0, IsActive: false}

	fmt.Println("Data awal:")
	fmt.Println(" ", student.GetInfo())

	student.Activate()
	student.UpdateGrade(87.5)
	fmt.Println("\nSetelah Activate() dan UpdateGrade(87.5):")
	fmt.Println(" ", student.GetInfo())

	student.UpdateGrade(91.0)
	fmt.Println("\nSetelah UpdateGrade(91.0):")
	fmt.Println(" ", student.GetInfo())

	student.Deactivate()
	fmt.Println("\nSetelah Deactivate():")
	fmt.Println(" ", student.GetInfo())
}

// ============================
// BAGIAN 3: Struct Student & Method (dari folder tugas-mandiri/4-struct-student)
// ============================

type Student struct {
	ID       int
	Name     string
	Grade    float64
	IsActive bool
}

// GetInfo hanya MEMBACA data, tidak mengubahnya, sehingga value receiver
// =sudah cukup. Menggunakannya juga membuat niat method ini jelas: "aman
// dipanggil tanpa mengkhawatirkan efek samping pada data aslinya".
func (s Student) GetInfo() string {
	status := "nonaktif"
	if s.IsActive {
		status = "aktif"
	}
	return fmt.Sprintf("[%d] %s - Nilai: %.2f - Status: %s", s.ID, s.Name, s.Grade, status)
}

// UpdateGrade, Activate, dan Deactivate semuanya MENGUBAH field struct,
// jadi wajib pakai pointer receiver — kalau pakai value receiver,
// perubahan hanya akan terjadi pada salinan lokal dan hilang begitu
// method selesai dijalankan.
func (s *Student) UpdateGrade(grade float64) {
	s.Grade = grade
}

func (s *Student) Activate() {
	s.IsActive = true
}

func (s *Student) Deactivate() {
	s.IsActive = false
}