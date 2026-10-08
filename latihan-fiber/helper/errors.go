// Package helper - errors.go
// Mendefinisikan AppError sebagai tipe error sentral aplikasi.
// Semua error dari service/repository dibungkus menjadi AppError
// agar ErrorHandler terpusat bisa menghasilkan response JSON seragam.
package helper

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// Konstanta kode error standar yang digunakan di seluruh API
const (
	CodeValidation       = "VALIDATION_ERROR"
	CodeBadRequest       = "BAD_REQUEST"
	CodeUnauthorized     = "UNAUTHORIZED"
	CodeForbidden        = "FORBIDDEN"
	CodeNotFound         = "NOT_FOUND"
	CodeConflict         = "CONFLICT"
	CodeUnsupportedMedia = "UNSUPPORTED_MEDIA_TYPE"
	CodeNotAcceptable    = "NOT_ACCEPTABLE"
	CodeTooManyRequests  = "TOO_MANY_REQUESTS"
	CodeInternal         = "INTERNAL_ERROR"
)

// AppError adalah tipe error sentral yang menyimpan:
// - Status: HTTP status code
// - Code: kode string error untuk konsumsi klien
// - Message: pesan error yang aman ditampilkan ke pengguna
// - Fields: detail error per-field (untuk validasi)
// - cause: error asli (unexported, tidak bocor ke klien)
type AppError struct {
	Status  int
	Code    string
	Message string
	Errors  map[string][]string
	cause   error
}

// Error mengimplementasikan interface error.
// Menyertakan cause jika ada untuk keperluan logging internal.
func (e *AppError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap mengembalikan error asli (cause) untuk mendukung errors.Is/As.
func (e *AppError) Unwrap() error { return e.cause }

// --- Konstruktor AppError untuk berbagai skenario ---

// BadRequest membuat AppError dengan status 400.
func BadRequest(message string) *AppError {
	return &AppError{Status: fiber.StatusBadRequest, Code: CodeBadRequest, Message: message}
}

// Unauthorized membuat AppError dengan status 401.
func Unauthorized(message string) *AppError {
	return &AppError{Status: fiber.StatusUnauthorized, Code: CodeUnauthorized, Message: message}
}

// Forbidden membuat AppError dengan status 403.
func Forbidden(message string) *AppError {
	return &AppError{Status: fiber.StatusForbidden, Code: CodeForbidden, Message: message}
}

// NotFound membuat AppError dengan status 404.
func NotFound(message string) *AppError {
	return &AppError{Status: fiber.StatusNotFound, Code: CodeNotFound, Message: message}
}

// Conflict membuat AppError dengan status 409.
func Conflict(message string) *AppError {
	return &AppError{Status: fiber.StatusConflict, Code: CodeConflict, Message: message}
}

// Validation membuat AppError dengan status 422 dan detail per-field berupa map string ke slice string.
func Validation(errors map[string][]string) *AppError {
	return &AppError{
		Status: fiber.StatusUnprocessableEntity, Code: CodeValidation,
		Message: "Validasi gagal", Errors: errors,
	}
}

// ValidationSingle membuat AppError 422 untuk satu field kesalahan.
func ValidationSingle(field, msg string) *AppError {
	return &AppError{
		Status: fiber.StatusUnprocessableEntity, Code: CodeValidation,
		Message: "Validasi gagal", Errors: map[string][]string{field: {msg}},
	}
}

// NotAcceptable membuat AppError dengan status 406.
func NotAcceptable(message string) *AppError {
	return &AppError{
		Status: fiber.StatusNotAcceptable, Code: CodeNotAcceptable, Message: message,
	}
}

// UnsupportedMediaType membuat AppError dengan status 415.
func UnsupportedMediaType(message string) *AppError {
	return &AppError{
		Status: fiber.StatusUnsupportedMediaType, Code: CodeUnsupportedMedia, Message: message,
	}
}

// TooManyRequests membuat AppError dengan status 429.
func TooManyRequests(message string) *AppError {
	return &AppError{
		Status: fiber.StatusTooManyRequests, Code: CodeTooManyRequests, Message: message,
	}
}

// ServiceUnavailable membuat AppError dengan status 503.
func ServiceUnavailable(message string) *AppError {
	return &AppError{
		Status: fiber.StatusServiceUnavailable, Code: "SERVICE_UNAVAILABLE", Message: message,
	}
}

// Internal membuat AppError dengan status 500.
// Pesan error asli disimpan di cause (tidak bocor ke klien).
func Internal(cause error) *AppError {
	return &AppError{
		Status: fiber.StatusInternalServerError, Code: CodeInternal,
		Message: "terjadi kesalahan pada server", cause: cause,
	}
}
