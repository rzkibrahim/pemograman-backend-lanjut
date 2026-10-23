package config

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"modul9/app/model"
	"modul9/app/service"
	"modul9/helper"
	"modul9/middleware"
	"modul9/route"
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

func NewApp(
	logger *slog.Logger,
	pool *pgxpool.Pool,
	studentService *service.StudentService,
	userService *service.UserService,
	authService service.AuthService,
	courseService *service.CourseService,
	supervisorService *service.SupervisorService,
	perms *helper.PermissionSet,
	jwtSecret string,
) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "API Students - Praktikum Backend Lanjut (Pertemuan 9: Transactions & Concurrency)",
		BodyLimit:    1 * 1024 * 1024,
		ErrorHandler: newErrorHandler(logger),
	})

	allowedOrigins := GetEnv("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173")
	middleware.Register(app, logger, allowedOrigins)

	route.Register(app, pool, studentService, userService, authService, courseService, supervisorService, perms, jwtSecret)

	app.Use(func(c *fiber.Ctx) error {
		return helper.NotFound("endpoint tidak ditemukan")
	})

	return app
}
