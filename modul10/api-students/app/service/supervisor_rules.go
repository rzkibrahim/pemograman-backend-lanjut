package service

import (
	"github.com/gofiber/fiber/v2"

	"modul10/helper"
)

// translateSupervisorError memetakan sentinel error domain supervisor ke AppError HTTP.
func translateSupervisorError(err error) *helper.AppError {
	if err == nil {
		return nil
	}

	msg := err.Error()

	switch msg {
	case "SUPERVISOR_FULL":
		return &helper.AppError{
			Status:  fiber.StatusConflict,
			Code:    "SUPERVISOR_FULL",
			Message: "kuota pembimbing sudah penuh",
		}
	case "ALREADY_SUPERVISED":
		return &helper.AppError{
			Status:  fiber.StatusConflict,
			Code:    "ALREADY_SUPERVISED",
			Message: "mahasiswa sudah memiliki pembimbing",
		}
	case "NOT_SUPERVISED":
		return &helper.AppError{
			Status:  fiber.StatusConflict,
			Code:    "NOT_SUPERVISED",
			Message: "mahasiswa tidak dibimbing oleh pembimbing ini",
		}
	case "VERSION_CONFLICT":
		return &helper.AppError{
			Status:  fiber.StatusConflict,
			Code:    "VERSION_CONFLICT",
			Message: "data supervisor telah diperbarui oleh pengguna lain (version conflict)",
		}
	case "QUOTA_BELOW_SUPERVISED":
		return &helper.AppError{
			Status:  fiber.StatusConflict,
			Code:    "QUOTA_BELOW_SUPERVISED",
			Message: "kuota tidak boleh lebih kecil dari jumlah mahasiswa yang sedang dibimbing",
		}
	case "data tidak ditemukan":
		return helper.NotFound("data pembimbing tidak ditemukan")
	default:
		return helper.Internal(err)
	}
}
