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

// SupervisorService menangani proses bisnis untuk penetapan pembimbing TA (Tugas D.2).
type SupervisorService struct {
	repo      *repository.SupervisorRepository
	txManager *repository.TxManager
}

// NewSupervisorService membuat instance SupervisorService baru.
func NewSupervisorService(repo *repository.SupervisorRepository, txManager *repository.TxManager) *SupervisorService {
	return &SupervisorService{
		repo:      repo,
		txManager: txManager,
	}
}

// List mengambil seluruh daftar pembimbing TA.
func (s *SupervisorService) List(c *fiber.Ctx) error {
	supervisors, err := s.repo.FindAll(c.Context())
	if err != nil {
		return translateSupervisorError(err)
	}
	return c.JSON(fiber.Map{
		"success": true,
		"message": "daftar pembimbing berhasil diambil",
		"data":    supervisors,
	})
}

// Get mengambil detail satu pembimbing berdasarkan ID.
func (s *SupervisorService) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return helper.BadRequest("ID pembimbing tidak valid")
	}

	supervisor, err := s.repo.FindByID(c.Context(), id)
	if err != nil {
		return translateSupervisorError(err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    supervisor,
	})
}

// Assign menetapkan pembimbing TA untuk mahasiswa (D.2):
// - Kunci baris student untuk serialisasi request
// - UPDATE bersyarat atomik untuk kuota supervisor (supervised_count < quota)
// - INSERT ke tabel supervisions (constraint UNIQUE memastikan 1 mahasiswa = 1 pembimbing)
func (s *SupervisorService) Assign(c *fiber.Ctx) error {
	supervisorID, err := strconv.Atoi(c.Params("id"))
	if err != nil || supervisorID <= 0 {
		return helper.BadRequest("ID pembimbing tidak valid")
	}

	var req model.AssignSupervisorRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if fields := helper.Validate(req); len(fields) > 0 {
		return helper.Validation(fields)
	}

	err = s.txManager.WithTx(c.Context(), func(tx pgx.Tx) error {
		txRepo := s.repo.WithTx(tx)

		// 1. Kunci baris student
		if err := txRepo.LockStudent(c.Context(), req.StudentID); err != nil {
			return err
		}

		// 2. Cek apakah mahasiswa sudah punya pembimbing
		currentSupervisor, err := txRepo.GetCurrentSupervisor(c.Context(), req.StudentID)
		if err != nil {
			return err
		}
		if currentSupervisor != 0 {
			return repository.ErrAlreadySupervised
		}

		// 3. Tambah kuota terpakai secara bersyarat atomik
		if err := txRepo.IncrementSupervised(c.Context(), supervisorID); err != nil {
			return err
		}

		// 4. Masukkan relasi supervisi
		if err := txRepo.InsertSupervision(c.Context(), req.StudentID, supervisorID); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return translateSupervisorError(err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "pembimbing TA berhasil ditetapkan",
		"data": fiber.Map{
			"student_id":    req.StudentID,
			"supervisor_id": supervisorID,
		},
	})
}

// Switch menukar pembimbing TA secara atomik dan bebas deadlock (D.2):
// - Mengurutkan supervisor ID terkecil dulu saat mengambil kunci baris
// - Mengurangi kuota pembimbing lama dan menambah kuota pembimbing baru
func (s *SupervisorService) Switch(c *fiber.Ctx) error {
	var req model.SwitchSupervisorRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if fields := helper.Validate(req); len(fields) > 0 {
		return helper.Validation(fields)
	}

	if req.FromSupervisorID == req.ToSupervisorID {
		return helper.Validation(map[string]string{
			"to_supervisor_id": "pembimbing tujuan tidak boleh sama dengan pembimbing asal",
		})
	}

	err := s.txManager.WithTx(c.Context(), func(tx pgx.Tx) error {
		txRepo := s.repo.WithTx(tx)

		// 1. Kunci baris student
		if err := txRepo.LockStudent(c.Context(), req.StudentID); err != nil {
			return err
		}

		// 2. Kunci kedua supervisor secara terurut untuk mencegah deadlock
		supervisors, err := txRepo.LockSupervisors(c.Context(), req.FromSupervisorID, req.ToSupervisorID)
		if err != nil {
			return err
		}
		if len(supervisors) < 2 {
			return errors.New("data tidak ditemukan")
		}

		// 3. Hapus relasi pembimbing lama
		if err := txRepo.DeleteSupervision(c.Context(), req.StudentID, req.FromSupervisorID); err != nil {
			return err
		}
		if err := txRepo.DecrementSupervised(c.Context(), req.FromSupervisorID); err != nil {
			return err
		}

		// 4. Tambah relasi ke pembimbing baru
		if err := txRepo.IncrementSupervised(c.Context(), req.ToSupervisorID); err != nil {
			return err
		}
		if err := txRepo.InsertSupervision(c.Context(), req.StudentID, req.ToSupervisorID); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return translateSupervisorError(err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "pembimbing TA berhasil dialihkan",
	})
}

// UpdateQuota memperbarui kuota pembimbing dengan optimistic locking (D.2).
func (s *SupervisorService) UpdateQuota(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return helper.BadRequest("ID pembimbing tidak valid")
	}

	var req model.UpdateSupervisorQuotaRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if fields := helper.Validate(req); len(fields) > 0 {
		return helper.Validation(fields)
	}

	supervisor, err := s.repo.UpdateQuota(c.Context(), id, req.Quota, req.Version)
	if err != nil {
		return translateSupervisorError(err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "kuota pembimbing berhasil diperbarui",
		"data":    supervisor,
	})
}
