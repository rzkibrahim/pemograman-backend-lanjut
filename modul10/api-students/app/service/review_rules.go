package service

import (
	"errors"

	"modul10/app/repository"
	"modul10/helper"
)

func translateReviewError(err error) *helper.AppError {
	switch {
	case errors.Is(err, repository.ErrInvalidReviewCursor):
		return helper.BadRequest("cursor before bukan ObjectID yang valid")
	case errors.Is(err, repository.ErrReviewNotFound):
		return helper.NotFound("ulasan tidak ditemukan")
	case errors.Is(err, repository.ErrNotReviewOwner):
		return helper.Forbidden("hanya pemilik ulasan yang dapat mengubah atau menghapusnya")
	case errors.Is(err, repository.ErrAlreadyReviewed):
		return helper.Conflict("Anda sudah mengulas kelas ini")
	case errors.Is(err, repository.ErrMaxRepliesReached):
		return helper.Conflict("ulasan sudah mencapai batas 50 balasan")
	case errors.Is(err, repository.ErrAlreadyHelpful):
		return helper.Conflict("ulasan sudah ditandai membantu")
	case errors.Is(err, repository.ErrOwnReviewHelpful):
		return helper.Conflict("ulasan sendiri tidak dapat ditandai membantu")
	default:
		return helper.Internal(err)
	}
}
