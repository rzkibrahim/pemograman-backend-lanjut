package service

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"

	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
	"latihan-fiber/helper"
)

// CourseService menangani proses bisnis untuk domain courses, enrollments,
// dan transactional outbox notification events.
type CourseService struct {
	courses *repository.CourseRepository
	outbox  repository.OutboxRepository
	tx      *repository.TxManager
}

// NewCourseService membuat instance CourseService baru.
func NewCourseService(
	courses *repository.CourseRepository,
	outbox repository.OutboxRepository,
	tx *repository.TxManager,
) *CourseService {
	return &CourseService{courses: courses, outbox: outbox, tx: tx}
}

// List mengambil seluruh daftar kelas.
func (s *CourseService) List(c *fiber.Ctx) error {
	courses, err := s.courses.FindAll(c.Context())
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

	course, err := s.courses.FindByID(c.Context(), id)
	if err != nil {
		return translateCourseError(err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    course,
	})
}

// Enroll mendaftarkan mahasiswa ke kelas dengan proteksi konkurensi dan transactional outbox:
// 1. LockStudent (pessimistic lock) untuk mencegah request serentak tembus batas 24 SKS
// 2. SumCredits + WithinCreditLimit untuk validasi SKS
// 3. IncrementEnrolled (UPDATE bersyarat atomik) untuk kuota
// 4. Catat event enrollment.created ke outbox di dalam transaksi yang sama (Transactional Outbox)
// 5. InsertEnrollment (UNIQUE constraint) untuk mencegah daftar ganda
func (s *CourseService) Enroll(c *fiber.Ctx) error {
	courseID, err := strconv.Atoi(c.Params("id"))
	if err != nil || courseID <= 0 {
		return helper.BadRequest("ID course tidak valid")
	}

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	ctx := c.Context()
	course, err := s.courses.FindByID(ctx, courseID)
	if err != nil {
		return translateCourseError(err)
	}

	var enrolled model.Course
	err = s.tx.WithTx(ctx, func(tx pgx.Tx) error {
		repo := s.courses.WithTx(tx)

		// 1. Kunci baris student
		if err := repo.LockStudent(ctx, current.UserID); err != nil {
			return err
		}

		// 2. Cek akumulasi SKS
		currentCredits, err := repo.SumCredits(ctx, current.UserID)
		if err != nil {
			return err
		}
		if !WithinCreditLimit(currentCredits, course.Credits) {
			return errors.New("CREDIT_LIMIT_EXCEEDED")
		}

		// 3. Ambil satu kursi secara atomik.
		enrolled, err = repo.IncrementEnrolled(ctx, courseID)
		if err != nil {
			return err
		}

		// 4. Catat event-nya ke outbox, memakai transaction yang sama.
		//    Perbaikan Puzzle 2: Menggunakan s.outbox.WithTx(tx).Insert(...) BUKAN s.outbox.Insert(...)
		//    Dengan WithTx, jika langkah 5 (InsertEnrollment) gagal karena mahasiswa sudah terdaftar (409),
		//    baris outbox ini ikut di-rollback secara atomik bersama pembatalan kursi.
		if err := s.outbox.WithTx(tx).Insert(ctx, model.EventEnrollmentCreated, model.EnrollmentEvent{
			UserID: current.UserID, CourseID: enrolled.ID,
			CourseCode: enrolled.Code, CourseName: enrolled.Name,
		}); err != nil {
			return err
		}

		// 5. Catat pendaftarannya. Bila ternyata sudah terdaftar, error
		//    di sini membatalkan seluruh langkah sebelumnya.
		return repo.InsertEnrollment(ctx, current.UserID, courseID)
	})

	if err != nil {
		return translateCourseError(err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "berhasil mendaftar ke kelas",
		"data": fiber.Map{
			"student_id": current.UserID,
			"course_id":  courseID,
		},
	})
}

// Drop membatalkan pendaftaran mahasiswa dari suatu kelas secara atomik dan mencatat outbox event.
func (s *CourseService) Drop(c *fiber.Ctx) error {
	courseID, err := strconv.Atoi(c.Params("id"))
	if err != nil || courseID <= 0 {
		return helper.BadRequest("ID course tidak valid")
	}

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	ctx := c.Context()
	err = s.tx.WithTx(ctx, func(tx pgx.Tx) error {
		repo := s.courses.WithTx(tx)

		if err := repo.LockStudent(ctx, current.UserID); err != nil {
			return err
		}

		if err := repo.DeleteEnrollment(ctx, current.UserID, courseID); err != nil {
			return err
		}
		course, err := repo.DecrementEnrolled(ctx, courseID)
		if err != nil {
			return err
		}

		// Perbaikan Puzzle 3: EventType yang dicatat adalah model.EventEnrollmentDropped,
		// BUKAN model.EventEnrollmentCreated.
		return s.outbox.WithTx(tx).Insert(ctx, model.EventEnrollmentDropped, model.EnrollmentEvent{
			UserID: current.UserID, CourseID: course.ID,
			CourseCode: course.Code, CourseName: course.Name,
		})
	})

	if err != nil {
		return translateCourseError(err)
	}

	// Sesuai spesifikasi C.1 no 3: status 204 No Content
	return c.SendStatus(fiber.StatusNoContent)
}

// Swap menukar kelas mahasiswa secara atomik tanpa risiko deadlock.
func (s *CourseService) Swap(c *fiber.Ctx) error {
	var req model.SwapRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	if req.FromCourseID <= 0 || req.ToCourseID <= 0 {
		return helper.BadRequest("from_course_id dan to_course_id harus valid")
	}

	if req.FromCourseID == req.ToCourseID {
		return helper.Validation(map[string]string{
			"to_course_id": "kelas tujuan tidak boleh sama dengan kelas asal",
		})
	}

	ctx := c.Context()
	err := s.tx.WithTx(ctx, func(tx pgx.Tx) error {
		repo := s.courses.WithTx(tx)

		// Kunci baris student
		if err := repo.LockStudent(ctx, current.UserID); err != nil {
			return err
		}

		// Kunci kedua kelas secara terurut berdasarkan ID terkecil dulu (deadlock-free)
		courses, err := repo.LockCourses(ctx, req.FromCourseID, req.ToCourseID)
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
			currentCredits, err := repo.SumCredits(ctx, current.UserID)
			if err != nil {
				return err
			}
			if !WithinCreditLimit(currentCredits, sksDiff) {
				return errors.New("CREDIT_LIMIT_EXCEEDED")
			}
		}

		// 1. Hapus dari kelas lama
		if err := repo.DeleteEnrollment(ctx, current.UserID, req.FromCourseID); err != nil {
			return err
		}
		if _, err := repo.DecrementEnrolled(ctx, req.FromCourseID); err != nil {
			return err
		}

		// 2. Masukkan ke kelas baru
		if _, err := repo.IncrementEnrolled(ctx, req.ToCourseID); err != nil {
			return err
		}
		if err := repo.InsertEnrollment(ctx, current.UserID, req.ToCourseID); err != nil {
			return err
		}

		// Satu event "pindah", bukan dua event "keluar" dan "masuk".
		// Mahasiswa melakukan satu tindakan, maka ia menerima satu notifikasi.
		return s.outbox.WithTx(tx).Insert(ctx, model.EventEnrollmentSwapped, model.EnrollmentEvent{
			UserID: current.UserID, CourseID: toCourse.ID, CourseCode: toCourse.Code, CourseName: toCourse.Name,
			FromCourseCode: fromCourse.Code, FromCourseName: fromCourse.Name,
		})
	})

	if err != nil {
		return translateCourseError(err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "berhasil menukar kelas",
	})
}

// UpdateQuota mengubah kuota kelas (Optimistic Locking) dan mencatat outbox event (Tugas Mandiri D.2).
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

	ctx := c.Context()
	var updated model.Course
	// Tugas Mandiri D.2: Membungkus UpdateQuota dan pencatatan outbox ke dalam transaksi database
	// untuk menjamin bahwa pembaruan kuota dan pencatatan event notifikasi berjalan atomik.
	err = s.tx.WithTx(ctx, func(tx pgx.Tx) error {
		txRepo := s.courses.WithTx(tx)

		var err error
		updated, err = txRepo.UpdateQuota(ctx, id, req.Quota, req.Version)
		if err != nil {
			return err
		}

		// Ambil seluruh ID mahasiswa yang terdaftar di kelas ini
		studentIDs, err := txRepo.GetEnrolledStudents(ctx, id)
		if err != nil {
			return err
		}

		// Jika terdapat peserta kelas, catat event course.quota_changed ke outbox
		if len(studentIDs) > 0 {
			eventPayload := model.CourseQuotaChangedEvent{
				CourseID:   updated.ID,
				CourseCode: updated.Code,
				CourseName: updated.Name,
				NewQuota:   updated.Quota,
				UserIDs:    studentIDs,
			}
			if err := s.outbox.WithTx(tx).Insert(ctx, model.EventCourseQuotaChanged, eventPayload); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return translateCourseError(err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "kuota kelas berhasil diperbarui",
		"data":    updated,
	})
}

// EnrollNaive adalah versi naif (sengaja salah, check-then-act tanpa lock/update bersyarat)
// hanya disediakan untuk membuktikan race condition pada uji beban.
func (s *CourseService) EnrollNaive(c *fiber.Ctx) error {
	courseID, err := strconv.Atoi(c.Params("id"))
	if err != nil || courseID <= 0 {
		return helper.BadRequest("ID course tidak valid")
	}

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	ctx := c.Context()
	course, err := s.courses.FindByID(ctx, courseID)
	if err != nil {
		return translateCourseError(err)
	}

	if course.EnrolledCount >= course.Quota {
		return translateCourseError(repository.ErrCourseFull)
	}

	_, err = s.courses.FindAll(ctx)
	if err != nil {
		return translateCourseError(err)
	}

	if err := s.courses.InsertEnrollment(ctx, current.UserID, courseID); err != nil {
		return translateCourseError(err)
	}

	_, _ = s.courses.IncrementEnrolled(ctx, courseID)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "berhasil mendaftar (naif)",
	})
}
