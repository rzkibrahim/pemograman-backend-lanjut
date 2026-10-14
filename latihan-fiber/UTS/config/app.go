// Package config - app.go
// Merakit aplikasi Fiber beserta ErrorHandler terpusat, middleware, dan route.
package config

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"siakad-mini/app/model"
	"siakad-mini/app/service"
	"siakad-mini/helper"
	"siakad-mini/middleware"
	"siakad-mini/route"
)

// newErrorHandler membuat ErrorHandler terpusat untuk Fiber.
// Semua error dari handler/service dialirkan ke sini untuk menghasilkan
// response JSON yang seragam dan logging yang konsisten.
func newErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		requestID := helper.RequestID(c)

		var appErr *helper.AppError
		switch {
		case errors.As(err, &appErr):
			// Error yang sudah kita bungkus dalam AppError
		case errors.Is(err, fiber.ErrRequestEntityTooLarge):
			appErr = &helper.AppError{
				Status:  fiber.StatusRequestEntityTooLarge,
				Code:    "PAYLOAD_TOO_LARGE",
				Message: "ukuran body melebihi batas yang diizinkan",
			}
		default:
			var fiberErr *fiber.Error
			if errors.As(err, &fiberErr) {
				appErr = &helper.AppError{
					Status:  fiberErr.Code,
					Code:    "HTTP_ERROR",
					Message: fiberErr.Message,
				}
			} else {
				// Error tak terduga -> 500 Internal Server Error
				appErr = helper.Internal(err)
			}
		}

		// 5xx -> ERROR level, 4xx -> WARN level
		if appErr.Status >= fiber.StatusInternalServerError {
			logger.Error("request_failed",
				slog.String("request_id", requestID),
				slog.String("path", c.Path()),
				slog.String("code", appErr.Code),
				slog.Int("status", appErr.Status),
				slog.String("error", appErr.Error()))
		} else {
			logger.Warn("request_rejected",
				slog.String("request_id", requestID),
				slog.String("path", c.Path()),
				slog.String("code", appErr.Code),
				slog.Int("status", appErr.Status))
		}

		return c.Status(appErr.Status).JSON(model.ErrorResponse{
			Success:   false,
			Code:      appErr.Code,
			Message:   appErr.Message,
			Errors:    appErr.Errors,
			RequestID: requestID,
		})
	}
}

// NewApp merakit seluruh komponen aplikasi SIAKAD Mini:
// Fiber instance, middleware, route, dan error handler.
func NewApp(
	logger *slog.Logger,
	pool *pgxpool.Pool,
	authService service.AuthService,
	studentService *service.StudentService,
	courseService *service.CourseService,
	enrollService *service.EnrollmentService,
	jwtSecret string,
) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "SIAKAD Mini - UTS Praktikum Backend Lanjut",
		BodyLimit:    1 * 1024 * 1024, // 1 MB
		ErrorHandler: newErrorHandler(logger),
	})

	allowedOrigins := GetEnv("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173")
	middleware.Register(app, logger, allowedOrigins)

	route.Register(app, pool, authService, studentService, courseService, enrollService, jwtSecret)

	// Catch-all: endpoint yang tidak ditemukan
	app.Use(func(c *fiber.Ctx) error {
		return helper.NotFound("endpoint tidak ditemukan")
	})

	return app
}
