package config

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-students/app/service"
	"api-students/helper"
	"api-students/middleware"
	"api-students/route"
)

// NewApp merakit seluruh komponen aplikasi web Fiber:
// inisialisasi Fiber dengan error handler terpusat, registrasi middleware global,
// pendaftaran endpoint route, dan penanganan catch-all 404 (not found).
func NewApp(
	logger *slog.Logger,
	pool *pgxpool.Pool,
	studentService *service.StudentService,
) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName: "API Students - Praktikum Backend Lanjut",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			status := fiber.StatusInternalServerError
			pesan := "terjadi kesalahan pada server"
			if e, ok := err.(*fiber.Error); ok {
				status = e.Code
				pesan = e.Message
			}

			logger.Error("unhandled error",
				slog.String("path", c.Path()),
				slog.String("method", c.Method()),
				slog.Int("status", status),
				slog.String("error", err.Error()),
			)

			return helper.Fail(c, status, pesan)
		},
	})

	// 1. Pasang seluruh middleware global
	middleware.Register(app, logger)

	// 2. Daftarkan seluruh rute API
	route.Register(app, pool, studentService)

	// 3. Tangani endpoint yang tidak ditemukan
	app.Use(func(c *fiber.Ctx) error {
		return helper.Fail(c, fiber.StatusNotFound, "endpoint tidak ditemukan")
	})

	return app
}
