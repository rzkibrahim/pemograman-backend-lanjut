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
	courses repository.CourseRepository
	outbox  repository.OutboxRepository
	tx      *repository.TxManager
}

// NewCourseService membuat instance CourseService baru.
func NewCourseService(
	courses repository.CourseRepository,
	outbox repository.OutboxRepository,
	tx *repository.TxManager,
) *CourseService {
	return &CourseService{courses: courses, outbox: outbox, tx: tx}
}

// List godoc
// @Summary     Daftar kelas
// @Description Mengambil seluruh kelas yang tersedia.
// @Tags        courses
// @Security    BearerAuth
// @Produce     json
// @Success     200 {object} model.WebResponse "data: []model.Course"
// @Failure     500 {object} model.WebResponse
// @Router      /courses [get]
func (s *CourseService) List(c *fiber.Ctx) error {
	courses, err := s.courses.FindAll(c.Context())
	if err != nil {
		return translateCourseError(err)
	}
	return helper.Success(c, fiber.StatusOK, "daftar course berhasil diambil", courses)
}

// Get godoc
// @Summary     Detail kelas
// @Description Mengambil satu kelas berdasarkan ID.
// @Tags        courses
// @Security    BearerAuth
// @Produce     json
// @Param       id path int true "ID kelas"
// @Success     200 {object} model.WebResponse "data: model.Course"
// @Failure     400 {object} model.WebResponse
// @Failure     404 {object} model.WebResponse
// @Router      /courses/{id} [get]
func (s *CourseService) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return helper.BadRequest("ID course tidak valid")
	}

	course, err := s.courses.FindByID(c.Context(), id)
	if err != nil {
		return translateCourseError(err)
	}

	return helper.Success(c, fiber.StatusOK, "detail kelas berhasil diambil", course)
}

// Mine godoc
// @Summary     KRS mahasiswa (kelas milik pengguna)
// @Description Mengambil seluruh kelas yang diambil mahasiswa yang sedang login beserta total dan maksimum SKS.
// @Tags        courses
// @Security    BearerAuth
// @Produce     json
// @Success     200 {object} model.WebResponse "data: model.MyCourses"
// @Failure     401 {object} model.WebResponse
// @Failure     500 {object} model.WebResponse
// @Router      /courses/mine [get]
func (s *CourseService) Mine(c *fiber.Ctx) error {
	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	courses, err := s.courses.FindEnrolledByStudent(ctx, current.UserID)
	if err != nil {
		return translateCourseError(err)
	}
	total, err := s.courses.SumCredits(ctx, current.UserID)
	if err != nil {
		return translateCourseError(err)
	}

	return helper.Success(c, fiber.StatusOK, "KRS berhasil diambil", model.MyCourses{
		Courses:      courses,
		TotalCredits: total,
		MaxCredits:   MaxCreditsPerSemester,
	})
}

// Enroll mendaftarkan mahasiswa ke kelas dengan proteksi konkurensi dan transactional outbox:
// 1. LockStudent (pessimistic lock) untuk mencegah request serentak tembus batas 24 SKS
// 2. SumCredits + WithinCreditLimit untuk validasi SKS
// 3. IncrementEnrolled (UPDATE bersyarat atomik) untuk kuota
// 4. Catat event enrollment.created ke outbox di dalam transaksi yang sama (Transactional Outbox)
// 5. InsertEnrollment (UNIQUE constraint) untuk mencegah daftar ganda
//
// Enroll godoc
// @Summary     Daftar ke kelas
// @Description Mendaftarkan mahasiswa ke satu kelas. Dilindungi batas 24 SKS, kuota atomik, dan outbox pada transaksi yang sama.
// @Tags        courses
// @Security    BearerAuth
// @Param       id path int true "ID kelas"
// @Produce     json
// @Success     201 {object} model.WebResponse "data: model.Course; header Location"
// @Failure     400 {object} model.WebResponse
// @Failure     409 {object} model.WebResponse "ALREADY_ENROLLED / COURSE_FULL"
// @Failure     422 {object} model.WebResponse "CREDIT_LIMIT_EXCEEDED"
// @Router      /courses/{id}/enroll [post]
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

	return helper.Created(c, "berhasil mendaftar ke kelas", enrolled,
		"/api/v1/courses/"+strconv.Itoa(courseID))
}

// Drop membatalkan pendaftaran mahasiswa dari suatu kelas secara atomik dan mencatat outbox event.
//
// Drop godoc
// @Summary     Keluar dari kelas
// @Description Membatalkan pendaftaran mahasiswa dari kelas, secara atomik dan mencatat event outbox enrollment.dropped.
// @Tags        courses
// @Security    BearerAuth
// @Param       id path int true "ID kelas"
// @Success     204 "No Content"
// @Failure     400 {object} model.WebResponse
// @Failure     404 {object} model.WebResponse "NOT_ENROLLED"
// @Router      /courses/{id}/enroll [delete]
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
//
// Swap godoc
// @Summary     Tukar kelas
// @Description Mengganti keanggotaan satu kelas dengan kelas lain dalam satu transaksi (lock ID terurut, outbox satu event swap).
// @Tags        courses
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       request body model.SwapRequest true "Kelas asal dan tujuan"
// @Success     200 {object} model.WebResponse "data: model.Course (kelas yang didapat)"
// @Failure     400 {object} model.WebResponse
// @Failure     409 {object} model.WebResponse "COURSE_FULL / NOT_ENROLLED"
// @Failure     422 {object} model.WebResponse "CREDIT_LIMIT_EXCEEDED"
// @Router      /courses/swap [post]
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
	var swapped model.Course
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
		swapped = courseMap[req.ToCourseID]

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
		enrolledTo, err := repo.IncrementEnrolled(ctx, req.ToCourseID)
		if err != nil {
			return err
		}
		if err := repo.InsertEnrollment(ctx, current.UserID, req.ToCourseID); err != nil {
			return err
		}
		swapped = enrolledTo

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
		"data":    swapped,
	})
}

// UpdateQuota godoc
// @Summary      Mengubah kuota kelas (admin)
// @Description  Optimistic locking: kirim version yang terakhir dilihat. Membutuhkan permission course:update.
// @Tags         courses
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id path int true "ID kelas"
// @Param        body body model.UpdateQuotaRequest true "Kuota baru dan version"
// @Success      200 {object} model.WebResponse{data=model.Course}
// @Failure      400 {object} model.ErrorResponse
// @Failure      401 {object} model.ErrorResponse
// @Failure      403 {object} model.ErrorResponse
// @Failure      409 {object} model.ErrorResponse "VERSION_CONFLICT / QUOTA_BELOW_ENROLLED"
// @Router       /courses/{id} [patch]
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

