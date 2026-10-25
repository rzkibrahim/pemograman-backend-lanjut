package service

import (
	"github.com/gofiber/fiber/v2"

	"modul9/helper"
)

const maxCredits = 24

// WithinCreditLimit memeriksa apakah menambahkan sejumlah SKS masih dalam batas 24.
func WithinCreditLimit(currentCredits, additionalCredits int) bool {
	return currentCredits+additionalCredits <= maxCredits
}

// translateCourseError memetakan sentinel error repository ke AppError HTTP.
func translateCourseError(err error) *helper.AppError {
	if err == nil {
		return nil
	}

	msg := err.Error()

	switch msg {
	case "COURSE_FULL":
		return &helper.AppError{
			Status:  fiber.StatusConflict,
			Code:    "COURSE_FULL",
			Message: "kelas sudah penuh",
		}
	case "ALREADY_ENROLLED":
		return &helper.AppError{
			Status:  fiber.StatusConflict,
			Code:    "ALREADY_ENROLLED",
			Message: "mahasiswa sudah terdaftar di kelas ini",
		}
	case "NOT_ENROLLED":
		return &helper.AppError{
			Status:  fiber.StatusConflict,
			Code:    "NOT_ENROLLED",
			Message: "mahasiswa tidak terdaftar di kelas ini",
		}
	case "VERSION_CONFLICT":
		return &helper.AppError{
			Status:  fiber.StatusConflict,
			Code:    "VERSION_CONFLICT",
			Message: "data sudah diubah oleh pengguna lain (version conflict)",
		}
	case "QUOTA_BELOW_ENROLLED":
		return &helper.AppError{
			Status:  fiber.StatusConflict,
			Code:    "QUOTA_BELOW_ENROLLED",
			Message: "kuota tidak boleh lebih kecil dari jumlah peserta terdaftar",
		}
	case "CREDIT_LIMIT_EXCEEDED":
		return &helper.AppError{
			Status:  fiber.StatusConflict,
			Code:    "CREDIT_LIMIT_EXCEEDED",
			Message: "total SKS melebihi batas maksimal 24 SKS",
		}
	case "data tidak ditemukan":
		return helper.NotFound("data tidak ditemukan")
	default:
		return helper.Internal(err)
	}
}
