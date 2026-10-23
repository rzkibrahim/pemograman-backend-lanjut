package service

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"

	"modul9/app/model"
	"modul9/app/repository"
	"modul9/helper"
)

// CourseService menangani proses bisnis untuk domain courses dan enrollments.
type CourseService struct {
	repo      *repository.CourseRepository
	txManager *repository.TxManager
}

// NewCourseService membuat instance CourseService baru.
func NewCourseService(repo *repository.CourseRepository, txManager *repository.TxManager) *CourseService {
	return &CourseService{
		repo:      repo,
		txManager: txManager,
	}
}

// List mengambil seluruh daftar kelas.
func (s *CourseService) List(c *fiber.Ctx) error {
	courses, err := s.repo.FindAll(c.Context())
	if err != nil {
		return translateCourseError(err)
	}
	return c.JSON(fiber.Map{
		"success": true,
		"message": "daftar course berhasil diambil",
		"data":    courses,
	})
}

// Get mengambil detail satu kelas berdasarkan ID.
func (s *CourseService) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return helper.BadRequest("ID course tidak valid")
	}

	course, err := s.repo.FindByID(c.Context(), id)
	if err != nil {
		return translateCourseError(err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    course,
	})
}

// Enroll mendaftarkan mahasiswa ke kelas dengan proteksi konkurensi:
// 1. LockStudent (pessimistic lock) untuk mencegah request serentak tembus batas 24 SKS
// 2. SumCredits + WithinCreditLimit untuk validasi SKS
// 3. IncrementEnrolled (UPDATE bersyarat atomik) untuk kuota
// 4. InsertEnrollment (UNIQUE constraint) untuk mencegah daftar ganda
func (s *CourseService) Enroll(c *fiber.Ctx) error {
	courseID, err := strconv.Atoi(c.Params("id"))
	if err != nil || courseID <= 0 {
		return helper.BadRequest("ID course tidak valid")
	}

	var req model.EnrollRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if fields := helper.Validate(req); len(fields) > 0 {
		return helper.Validation(fields)
	}

	// Pastikan kelas tujuan ada
	course, err := s.repo.FindByID(c.Context(), courseID)
	if err != nil {
		return translateCourseError(err)
	}

	err = s.txManager.WithTx(c.Context(), func(tx pgx.Tx) error {
		txRepo := s.repo.WithTx(tx)

		// 1. Kunci baris student
		if err := txRepo.LockStudent(c.Context(), req.StudentID); err != nil {
			return err
		}

		// 2. Cek akumulasi SKS
		currentCredits, err := txRepo.SumCredits(c.Context(), req.StudentID)
		if err != nil {
			return err
		}
		if !WithinCreditLimit(currentCredits, course.Credits) {
			return errors.New("CREDIT_LIMIT_EXCEEDED")
		}

		// 3. UPDATE bersyarat atomik untuk kuota
		if err := txRepo.IncrementEnrolled(c.Context(), courseID); err != nil {
			return err
		}

		// 4. Catat baris enrollment
		if err := txRepo.InsertEnrollment(c.Context(), req.StudentID, courseID); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return translateCourseError(err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "berhasil mendaftar ke kelas",
		"data": fiber.Map{
			"student_id": req.StudentID,
			"course_id":  courseID,
		},
	})
}

// Drop membatalkan pendaftaran mahasiswa dari suatu kelas secara atomik.
func (s *CourseService) Drop(c *fiber.Ctx) error {
	courseID, err := strconv.Atoi(c.Params("id"))
	if err != nil || courseID <= 0 {
		return helper.BadRequest("ID course tidak valid")
	}

	var req model.EnrollRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if fields := helper.Validate(req); len(fields) > 0 {
		return helper.Validation(fields)
	}

	err = s.txManager.WithTx(c.Context(), func(tx pgx.Tx) error {
		txRepo := s.repo.WithTx(tx)

		// Kunci baris student
		if err := txRepo.LockStudent(c.Context(), req.StudentID); err != nil {
			return err
		}

		// Hapus baris enrollment
		if err := txRepo.DeleteEnrollment(c.Context(), req.StudentID, courseID); err != nil {
			return err
		}

		// Kurangi enrolled_count
		if err := txRepo.DecrementEnrolled(c.Context(), courseID); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return translateCourseError(err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "berhasil membatalkan pendaftaran kelas",
	})
}

// Swap menukar kelas mahasiswa secara atomik tanpa risiko deadlock.
// Menggunakan LockCourses yang mengurutkan ID kelas secara menaik.
func (s *CourseService) Swap(c *fiber.Ctx) error {
	var req model.SwapRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if fields := helper.Validate(req); len(fields) > 0 {
		return helper.Validation(fields)
	}

	if req.FromCourseID == req.ToCourseID {
		return helper.Validation(map[string]string{
			"to_course_id": "kelas tujuan tidak boleh sama dengan kelas asal",
		})
	}

	err := s.txManager.WithTx(c.Context(), func(tx pgx.Tx) error {
		txRepo := s.repo.WithTx(tx)

		// Kunci baris student
		if err := txRepo.LockStudent(c.Context(), req.StudentID); err != nil {
			return err
		}

		// Kunci kedua kelas secara terurut berdasarkan ID terkecil dulu (deadlock-free)
		courses, err := txRepo.LockCourses(c.Context(), req.FromCourseID, req.ToCourseID)
		if err != nil {
			return err
		}

		courseMap := make(map[int]model.Course, len(courses))
		for _, crs := range courses {
			courseMap[crs.ID] = crs
		}

		fromCourse, okFrom := courseMap[req.FromCourseID]
		toCourse, okTo := courseMap[req.ToCourseID]
		if !okFrom || !okTo {
			return errors.New("data tidak ditemukan")
		}

		// Cek batas SKS jika kelas baru memiliki SKS lebih besar
		sksDiff := toCourse.Credits - fromCourse.Credits
		if sksDiff > 0 {
			currentCredits, err := txRepo.SumCredits(c.Context(), req.StudentID)
			if err != nil {
				return err
			}
			if !WithinCreditLimit(currentCredits, sksDiff) {
				return errors.New("CREDIT_LIMIT_EXCEEDED")
			}
		}

		// 1. Hapus dari kelas lama
		if err := txRepo.DeleteEnrollment(c.Context(), req.StudentID, req.FromCourseID); err != nil {
			return err
		}
		if err := txRepo.DecrementEnrolled(c.Context(), req.FromCourseID); err != nil {
			return err
		}

		// 2. Masukkan ke kelas baru
		if err := txRepo.IncrementEnrolled(c.Context(), req.ToCourseID); err != nil {
			return err
		}
		if err := txRepo.InsertEnrollment(c.Context(), req.StudentID, req.ToCourseID); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return translateCourseError(err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "berhasil menukar kelas",
	})
}

// UpdateQuota mengubah kuota kelas menggunakan optimistic locking (version column).
func (s *CourseService) UpdateQuota(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return helper.BadRequest("ID course tidak valid")
	}

	var req model.UpdateQuotaRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if fields := helper.Validate(req); len(fields) > 0 {
		return helper.Validation(fields)
	}

	course, err := s.repo.UpdateQuota(c.Context(), id, req.Quota, req.Version)
	if err != nil {
		return translateCourseError(err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "kuota kelas berhasil diperbarui",
		"data":    course,
	})
}

// EnrollNaive adalah versi naif (sengaja salah, check-then-act tanpa lock/update bersyarat)
// hanya disediakan untuk membuktikan race condition pada uji beban.
func (s *CourseService) EnrollNaive(c *fiber.Ctx) error {
	courseID, err := strconv.Atoi(c.Params("id"))
	if err != nil || courseID <= 0 {
		return helper.BadRequest("ID course tidak valid")
	}

	var req model.EnrollRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if fields := helper.Validate(req); len(fields) > 0 {
		return helper.Validation(fields)
	}

	// Pola naif: BACA -> PERIKSA -> TULIS tanpa kunci dan tanpa UPDATE bersyarat
	course, err := s.repo.FindByID(c.Context(), courseID)
	if err != nil {
		return translateCourseError(err)
	}

	if course.EnrolledCount >= course.Quota {
		return translateCourseError(repository.ErrCourseFull)
	}

	// Simulasi sedikit waktu eksekusi agar race window terbuka
	// Perbarui count secara terpisah tanpa WHERE atomik
	_, err = s.repo.FindAll(c.Context())
	if err != nil {
		return translateCourseError(err)
	}

	// Update naif: enrolled_count = enrolled_count + 1 tanpa filter enrolled_count < quota
	if err := s.repo.InsertEnrollment(c.Context(), req.StudentID, courseID); err != nil {
		return translateCourseError(err)
	}

	// Naikkan count
	_ = s.repo.IncrementEnrolled(c.Context(), courseID)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "berhasil mendaftar (naif)",
	})
}
