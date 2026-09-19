package service

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"modul6/app/model"
	"modul6/app/repository"
	"modul6/helper"
)

// StudentService memegang dua tanggung jawab pada arsitektur aplikasi ini:
// menerima *fiber.Ctx (layer controller/delivery) dan menjalankan business rules
// serta evaluasi otorisasi kepemilikan (ownership check pada use case).
type StudentService struct {
	repo  repository.StudentRepository
	perms *helper.PermissionSet
}

// NewStudentService menerima interface StudentRepository dan in-memory PermissionSet.
func NewStudentService(repo repository.StudentRepository, perms *helper.PermissionSet) *StudentService {
	return &StudentService{
		repo:  repo,
		perms: perms,
	}
}

// translateError memetakan error milik repository menjadi status HTTP yang sesuai.
func translateError(c *fiber.Ctx, err error, generalMessage string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.Fail(c, fiber.StatusNotFound, "student tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.Fail(c, fiber.StatusConflict, "NIM sudah dipakai")
	default:
		return helper.Fail(c, fiber.StatusInternalServerError, generalMessage)
	}
}

// ============================================================
// HANDLER: LIST - GET /api/v1/students
// Endpoint dilindungi oleh RequirePermission("student:list") di route.go
// ============================================================

func (s *StudentService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	q := helper.ParseListQuery(c)

	students, total, err := s.repo.FindAll(ctx, q)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal mengambil data student")
	}

	return helper.SuccessList(c, "daftar student berhasil diambil", students, &model.Meta{
		Page:       q.Page,
		Limit:      q.Limit,
		Total:      total,
		TotalPages: CountTotalPages(total, q.Limit),
	})
}

// ============================================================
// HANDLER: GET ONE - GET /api/v1/students/:id
// Ownership Check: Pemilik data selalu boleh melihat datanya sendiri.
// Pengguna lain WAJIB memiliki permission "student:read:any".
// ============================================================

func (s *StudentService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	// 1. Ambil data dari database untuk mengetahui siapa pemiliknya (owner_id)
	student, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(c, err, "gagal mengambil data student")
	}

	// 2. Ambil identitas pemanggil dari context JWT
	current, ok := helper.GetCurrentAuthUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "token autentikasi diperlukan")
	}

	// 3. Evaluasi kepemilikan melalui Pure Rule CanAccessStudent
	if !CanAccessStudent(current, student.OwnerID, s.perms, "student:read:any") {
		return helper.Fail(c, fiber.StatusForbidden, "akses ditolak: Anda bukan pemilik data ini dan tidak memiliki izin student:read:any")
	}

	return helper.Success(c, fiber.StatusOK, "student ditemukan", student)
}

// ============================================================
// HANDLER: CREATE - POST /api/v1/students
// Endpoint dilindungi oleh RequirePermission("student:create") di route.go.
// Anti-Spoofing: owner_id selalu diisi otomatis dari token pemanggil.
// ============================================================

func (s *StudentService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	// 1. Ambil identitas pemanggil
	current, ok := helper.GetCurrentAuthUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "token autentikasi diperlukan")
	}

	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	// 2. Business rules validasi input
	if errs := ValidateCreate(req); len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	// 3. Simpan data mahasiswa
	// PENTING: OwnerID diisi secara sepihak dari token (current.ID) untuk mencegah spoofing kepemilikan
	baru, err := s.repo.Create(ctx, model.Student{
		NIM:      strings.TrimSpace(req.NIM),
		Name:     strings.TrimSpace(req.Name),
		Grade:    req.Grade,
		IsActive: req.IsActive,
		OwnerID:  current.ID,
	})
	if err != nil {
		return translateError(c, err, "gagal menyimpan student")
	}

	return helper.Created(c, "student berhasil dibuat", baru,
		"/api/v1/students/"+strconv.Itoa(baru.ID))
}

// ============================================================
// HANDLER: REPLACE - PUT /api/v1/students/:id
// Ownership Check: Pemilik data boleh mengubah seluruh datanya.
// Pengguna lain WAJIB memiliki permission "student:update:any".
// ============================================================

func (s *StudentService) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	// 1. Ambil identitas pemanggil
	current, ok := helper.GetCurrentAuthUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "token autentikasi diperlukan")
	}

	// 2. Periksa keberadaan data dan kepemilikannya
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(c, err, "gagal mengambil data student")
	}

	// 3. Evaluasi kepemilikan: hanya pemilik atau pemegang student:update:any yang diizinkan
	if !CanAccessStudent(current, existing.OwnerID, s.perms, "student:update:any") {
		return helper.Fail(c, fiber.StatusForbidden, "akses ditolak: Anda bukan pemilik data ini dan tidak memiliki izin student:update:any")
	}

	var req model.ReplaceStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	if errs := ValidateReplace(req); len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	hasil, err := s.repo.Update(ctx, model.Student{
		ID:       id,
		NIM:      strings.TrimSpace(req.NIM),
		Name:     strings.TrimSpace(req.Name),
		Grade:    req.Grade,
		IsActive: req.IsActive,
		OwnerID:  existing.OwnerID, // Menjaga integritas agar owner_id tidak bergeser
	})
	if err != nil {
		return translateError(c, err, "gagal memperbarui student")
	}

	return helper.Success(c, fiber.StatusOK, "student berhasil diganti seluruhnya", hasil)
}

// ============================================================
// HANDLER: PATCH - PATCH /api/v1/students/:id
// Ownership Check: Pemilik data boleh mengubah sebagian datanya.
// Pengguna lain WAJIB memiliki permission "student:update:any".
// ============================================================

func (s *StudentService) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	// 1. Ambil identitas pemanggil
	current, ok := helper.GetCurrentAuthUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "token autentikasi diperlukan")
	}

	// 2. Ambil data existing dari database
	currentStudent, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(c, err, "gagal mengambil data student")
	}

	// 3. Evaluasi kepemilikan
	if !CanAccessStudent(current, currentStudent.OwnerID, s.perms, "student:update:any") {
		return helper.Fail(c, fiber.StatusForbidden, "akses ditolak: Anda bukan pemilik data ini dan tidak memiliki izin student:update:any")
	}

	var req model.PatchStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}
	if IsEmptyPatch(req) {
		return helper.Fail(c, fiber.StatusBadRequest, "tidak ada field yang diubah")
	}

	updated, errs := ApplyPatch(currentStudent, req)
	if len(errs) > 0 {
		return helper.FailValidation(c, errs)
	}

	// Pastikan owner_id tetap terjaga
	updated.OwnerID = currentStudent.OwnerID

	result, err := s.repo.Update(ctx, updated)
	if err != nil {
		return translateError(c, err, "gagal memperbarui student")
	}
	return helper.Success(c, fiber.StatusOK, "student berhasil diperbarui sebagian", result)
}

// ============================================================
// HANDLER: DELETE - DELETE /api/v1/students/:id
// Endpoint dilindungi oleh RequirePermission("student:delete") di route.go
// ============================================================

func (s *StudentService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return translateError(c, err, "gagal menghapus student")
	}
	return helper.NoContent(c)
}
