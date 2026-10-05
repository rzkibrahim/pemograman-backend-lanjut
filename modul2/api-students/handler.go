package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// === Penyimpanan in-memory ===
// Data disimpan di memori ΓÇö setiap kali server dimatikan, seluruh data hilang.
// Masalah ini baru diselesaikan pada pertemuan 3 dengan PostgreSQL.
var students []Student
var nextID = 1

// findStudentIndex mencari posisi student di slice berdasarkan ID.
// Mengembalikan -1 bila tidak ditemukan.
func findStudentIndex(id int) int {
	for i := range students {
		if students[i].ID == id {
			return i
		}
	}
	return -1
}

// cocokPencarian memeriksa apakah kata kunci muncul di name atau nim.
// Pencarian bersifat case-insensitive.
func cocokPencarian(s Student, kata string) bool {
	kata = strings.ToLower(kata)
	return strings.Contains(strings.ToLower(s.Name), kata) ||
		strings.Contains(strings.ToLower(s.NIM), kata)
}

// paramID mengambil dan memvalidasi parameter :id dari URL path.
// Mengembalikan false bila id bukan angka positif.
func paramID(c *fiber.Ctx) (int, bool) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return 0, false
	}
	return id, true
}

// isNIMDuplicate memeriksa apakah NIM sudah dipakai oleh student lain.
// Parameter excludeID digunakan agar saat update, NIM milik student itu sendiri
// tidak dianggap duplikat.
func isNIMDuplicate(nim string, excludeID int) bool {
	for _, s := range students {
		if strings.EqualFold(s.NIM, nim) && s.ID != excludeID {
			return true
		}
	}
	return false
}

// ============================================================
// HANDLER: LIST ΓÇö GET /api/v1/students
// Mendukung paginasi, pencarian, penyaringan, dan pengurutan.
// ============================================================

func listStudents(c *fiber.Ctx) error {
	q := parseListQuery(c)

	// 1) Saring ΓÇö terapkan semua filter yang dikirim pengguna
	hasil := []Student{}
	for _, s := range students {
		// Filter is_active
		if q.IsActive != nil && s.IsActive != *q.IsActive {
			continue
		}
		// Filter grade_min
		if q.GradeMin != nil && s.Grade < *q.GradeMin {
			continue
		}
		// Filter grade_max
		if q.GradeMax != nil && s.Grade > *q.GradeMax {
			continue
		}
		// Pencarian pada name dan nim
		if q.Search != "" && !cocokPencarian(s, q.Search) {
			continue
		}
		hasil = append(hasil, s)
	}

	// 2) Urutkan berdasarkan field dan arah yang diminta
	sort.SliceStable(hasil, func(i, j int) bool {
		var lebihKecil bool
		switch q.Sort {
		case "nim":
			lebihKecil = hasil[i].NIM < hasil[j].NIM
		case "name":
			lebihKecil = hasil[i].Name < hasil[j].Name
		case "grade":
			lebihKecil = hasil[i].Grade < hasil[j].Grade
		case "created_at":
			lebihKecil = hasil[i].CreatedAt.Before(hasil[j].CreatedAt)
		default: // "id"
			lebihKecil = hasil[i].ID < hasil[j].ID
		}
		if q.Order == "desc" {
			return !lebihKecil
		}
		return lebihKecil
	})

	// 3) Potong sesuai halaman (paginasi)
	total := len(hasil)
	totalPages := 0
	if total > 0 {
		totalPages = (total + q.Limit - 1) / q.Limit
	}
	mulai := (q.Page - 1) * q.Limit
	if mulai > total {
		mulai = total
	}
	akhir := mulai + q.Limit
	if akhir > total {
		akhir = total
	}

	return okList(c, "daftar student berhasil diambil", hasil[mulai:akhir], &Meta{
		Page: q.Page, Limit: q.Limit, Total: total, TotalPages: totalPages,
	})
}

// ============================================================
// HANDLER: GET ONE ΓÇö GET /api/v1/students/:id
// ============================================================

func getStudent(c *fiber.Ctx) error {
	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	i := findStudentIndex(id)
	if i == -1 {
		return fail(c, fiber.StatusNotFound, "student tidak ditemukan")
	}

	return ok(c, "student ditemukan", students[i])
}

// ============================================================
// HANDLER: CREATE ΓÇö POST /api/v1/students
// Mengembalikan 201 Created dengan header Location.
// ============================================================

func createStudent(c *fiber.Ctx) error {
	var req CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	// Validasi per field ΓÇö pesan kesalahan spesifik agar pengguna tahu
	// kolom mana yang bermasalah
	errs := map[string]string{}
	req.NIM = strings.TrimSpace(req.NIM)
	req.Name = strings.TrimSpace(req.Name)

	if req.NIM == "" {
		errs["nim"] = "wajib diisi"
	}
	if req.Name == "" {
		errs["name"] = "wajib diisi"
	}
	if req.Grade < 0 || req.Grade > 100 {
		errs["grade"] = "harus antara 0 dan 100"
	}

	if len(errs) > 0 {
		return failValidation(c, errs)
	}

	// Cek duplikasi NIM ΓÇö 409 Conflict
	if isNIMDuplicate(req.NIM, 0) {
		return failConflict(c, fmt.Sprintf("NIM '%s' sudah terdaftar", req.NIM))
	}

	baru := Student{
		ID:        nextID,
		NIM:       req.NIM,
		Name:      req.Name,
		Grade:     req.Grade,
		IsActive:  req.IsActive,
		CreatedAt: time.Now(),
	}
	students = append(students, baru)
	nextID++

	return created(c, "student berhasil dibuat", baru,
		"/api/v1/students/"+strconv.Itoa(baru.ID))
}

// ============================================================
// HANDLER: REPLACE ΓÇö PUT /api/v1/students/:id
// PUT mengganti SELURUH isi sumber daya. Semua field wajib dikirim.
// Field yang tidak dikirim dianggap dikosongkan (zero value).
// ============================================================

func replaceStudent(c *fiber.Ctx) error {
	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	i := findStudentIndex(id)
	if i == -1 {
		return fail(c, fiber.StatusNotFound, "student tidak ditemukan")
	}

	var req ReplaceStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	// Validasi ΓÇö pada PUT semua field wajib dikirim dan valid
	errs := map[string]string{}
	req.NIM = strings.TrimSpace(req.NIM)
	req.Name = strings.TrimSpace(req.Name)

	if req.NIM == "" {
		errs["nim"] = "wajib diisi pada PUT"
	}
	if req.Name == "" {
		errs["name"] = "wajib diisi pada PUT"
	}
	if req.Grade < 0 || req.Grade > 100 {
		errs["grade"] = "harus antara 0 dan 100"
	}
	if len(errs) > 0 {
		return failValidation(c, errs)
	}

	// Cek duplikasi NIM ΓÇö kecuali NIM milik student ini sendiri
	if isNIMDuplicate(req.NIM, id) {
		return failConflict(c, fmt.Sprintf("NIM '%s' sudah dipakai student lain", req.NIM))
	}

	// Ganti seluruh field (kecuali ID dan CreatedAt yang bersifat immutable)
	students[i].NIM = req.NIM
	students[i].Name = req.Name
	students[i].Grade = req.Grade
	students[i].IsActive = req.IsActive

	return ok(c, "student berhasil diganti seluruhnya", students[i])
}

// ============================================================
// HANDLER: PATCH ΓÇö PATCH /api/v1/students/:id
// PATCH hanya mengubah field yang benar-benar dikirim.
// Field bertipe pointer: nil = tidak dikirim, non-nil = dikirim.
// ============================================================

func patchStudent(c *fiber.Ctx) error {
	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	i := findStudentIndex(id)
	if i == -1 {
		return fail(c, fiber.StatusNotFound, "student tidak ditemukan")
	}

	var req PatchStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	// Minimal satu field harus dikirim ΓÇö PATCH tanpa perubahan tidak masuk akal
	if req.NIM == nil && req.Name == nil && req.Grade == nil && req.IsActive == nil {
		return fail(c, fiber.StatusBadRequest, "tidak ada field yang diubah")
	}

	// Validasi dan terapkan hanya field yang dikirim
	if req.NIM != nil {
		nim := strings.TrimSpace(*req.NIM)
		if nim == "" {
			return failValidation(c, map[string]string{"nim": "tidak boleh kosong"})
		}
		if isNIMDuplicate(nim, id) {
			return failConflict(c, fmt.Sprintf("NIM '%s' sudah dipakai student lain", nim))
		}
		students[i].NIM = nim
	}
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			return failValidation(c, map[string]string{"name": "tidak boleh kosong"})
		}
		students[i].Name = *req.Name
	}
	if req.Grade != nil {
		if *req.Grade < 0 || *req.Grade > 100 {
			return failValidation(c, map[string]string{"grade": "harus antara 0 dan 100"})
		}
		students[i].Grade = *req.Grade
	}
	if req.IsActive != nil {
		students[i].IsActive = *req.IsActive
	}

	return ok(c, "student berhasil diperbarui sebagian", students[i])
}

// ============================================================
// HANDLER: DELETE ΓÇö DELETE /api/v1/students/:id
// Mengembalikan 204 No Content ΓÇö berhasil, tanpa body.
// ============================================================

func deleteStudent(c *fiber.Ctx) error {
	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	i := findStudentIndex(id)
	if i == -1 {
		return fail(c, fiber.StatusNotFound, "student tidak ditemukan")
	}

	// Hapus elemen dari slice dengan menggabungkan bagian sebelum dan sesudahnya
	students = append(students[:i], students[i+1:]...)

	return noContent(c) // 204: berhasil, dan memang tidak ada yang perlu dikirim
}
