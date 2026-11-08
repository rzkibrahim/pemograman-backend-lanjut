package config

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/app/model"
	"latihan-fiber/helper"
	"latihan-fiber/middleware"
	"latihan-fiber/route"
)

func newErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		requestID := helper.RequestID(c)

		var appErr *helper.AppError
		switch {
		case errors.As(err, &appErr):
			// Kegagalan yang sudah kita rencanakan.
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
				appErr = helper.Internal(err)
			}
		}

		// 5xx -> ERROR, 4xx -> WARN
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
			Fields:    appErr.Fields,
			RequestID: requestID,
		})
	}
}

// NewApp merakit instance fiber.App beserta middleware dan perutean route.Dependencies.
func NewApp(logger *slog.Logger, deps route.Dependencies) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "API Students - Praktikum Backend Lanjut (Modul 11: Background Jobs)",
		BodyLimit:    1 * 1024 * 1024,
		ErrorHandler: newErrorHandler(logger),
	})

	allowedOrigins := GetEnv("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173")
	middleware.Register(app, logger, allowedOrigins)

	route.Register(app, deps)

	app.Use(func(c *fiber.Ctx) error {
		return helper.NotFound("endpoint tidak ditemukan")
	})

	return app
}
