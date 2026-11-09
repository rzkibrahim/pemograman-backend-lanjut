package service

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/app/repository"
	"latihan-fiber/helper"
)

// NormalizeTags membersihkan daftar tag ulasan sebelum disimpan: setiap
// tag di-trim dan di-lowercase, string kosong dibuang, duplikat dihapus
// dengan mempertahankan urutan kemunculan pertama. Hasil tidak pernah nil
// agar response JSON berisi [] dan bukan null.
func NormalizeTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	result := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		result = append(result, t)
	}
	return result
}

func translateReviewError(err error) *helper.AppError {
	switch {
	case errors.Is(err, repository.ErrInvalidReviewCursor):
		return helper.BadRequest("cursor before bukan ObjectID yang valid")
	case errors.Is(err, repository.ErrReviewNotFound):
		return helper.NotFound("ulasan tidak ditemukan")
	case errors.Is(err, repository.ErrNotReviewOwner):
		return helper.Forbidden("hanya pemilik ulasan yang dapat mengubah atau menghapusnya")
	case errors.Is(err, repository.ErrAlreadyReviewed):
		return &helper.AppError{
			Status:  fiber.StatusConflict,
			Code:    "ALREADY_REVIEWED",
			Message: "Anda sudah mengulas kelas ini",
		}
	case errors.Is(err, repository.ErrMaxRepliesReached):
		return &helper.AppError{
			Status:  fiber.StatusConflict,
			Code:    "REPLY_LIMIT",
			Message: "ulasan sudah mencapai batas 50 balasan",
		}
	case errors.Is(err, repository.ErrAlreadyHelpful):
		return &helper.AppError{
			Status:  fiber.StatusConflict,
			Code:    "ALREADY_VOTED",
			Message: "ulasan sudah ditandai membantu",
		}
	case errors.Is(err, repository.ErrOwnReviewHelpful):
		return &helper.AppError{
			Status:  fiber.StatusConflict,
			Code:    "OWN_REVIEW",
			Message: "ulasan sendiri tidak dapat ditandai membantu",
		}
	default:
		return helper.Internal(err)
	}
}
