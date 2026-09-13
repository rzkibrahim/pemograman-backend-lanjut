package config

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"modul5/app/service"
	"modul5/helper"
	"modul5/middleware"
	"modul5/route"
)

// NewApp merakit seluruh komponen aplikasi web Fiber:
// inisialisasi Fiber dengan error handler terpusat, registrasi middleware global,
// pendaftaran endpoint route (auth & protected students), dan penanganan catch-all 404.
func NewApp(
	logger *slog.Logger,
	pool *pgxpool.Pool,
	studentService *service.StudentService,
	authService service.AuthService,
	jwtSecret string,
) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName: "API Students - Praktikum Backend Lanjut (Pertemuan 5)",
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

	// 1. Pasang seluruh middleware global (dengan pembatasan CORS)
	allowedOrigins := GetEnv("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173")
	middleware.Register(app, logger, allowedOrigins)

	// 2. Daftarkan seluruh rute API
	route.Register(app, pool, studentService, authService, jwtSecret)

	// 3. Tangani endpoint yang tidak ditemukan
	app.Use(func(c *fiber.Ctx) error {
		return helper.Fail(c, fiber.StatusNotFound, "endpoint tidak ditemukan")
	})

	return app
}
