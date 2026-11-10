package service

import (
	"errors"
	"fmt"
	"testing"

	"latihan-fiber/app/repository"
	"latihan-fiber/helper"
)

// CATATAN PERBAIKAN (D.1 no.2): file dulu memakai `package service_test`
// padahal memanggil fungsi yang tidak di-export (translateReviewError).
// Compiler menolak; test.pemetaan error harus white-box (package service).
//
// Test ini memastikan setiap error dari repository berubah menjadi response
// yang tepat — termasuk error yang sudah dibungkus dengan fmt.Errorf.
func TestTranslateReviewError(t *testing.T) {
	tests := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{repository.ErrInvalidReviewCursor, 400, helper.CodeBadRequest},
		{repository.ErrReviewNotFound, 404, helper.CodeNotFound},
		{repository.ErrAlreadyReviewed, 409, "ALREADY_REVIEWED"},
		{repository.ErrNotReviewOwner, 403, helper.CodeForbidden},
		{repository.ErrOwnReviewHelpful, 409, "OWN_REVIEW"},
		{repository.ErrAlreadyHelpful, 409, "ALREADY_VOTED"},
		{repository.ErrMaxRepliesReached, 409, "REPLY_LIMIT"},
		{fmt.Errorf("lapisan luar: %w", repository.ErrAlreadyReviewed), 409, "ALREADY_REVIEWED"},
		{errors.New("koneksi MongoDB putus"), 500, helper.CodeInternal},
	}
	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			var appErr *helper.AppError
			err := translateReviewError(tt.err)
			if !errors.As(err, &appErr) {
				t.Fatalf("hasil bukan (*helper.AppError): %v", err)
			}
			if appErr.Status != tt.wantStatus || appErr.Code != tt.wantCode {
				t.Errorf("dapat %d %s, seharusnya %d %s", appErr.Status, appErr.Code,
					tt.wantStatus, tt.wantCode)
			}
		})
	}
}
