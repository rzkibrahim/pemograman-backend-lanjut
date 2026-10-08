// Package middleware - middleware.go
// Mendaftarkan middleware global: CORS, Request Logger terstruktur (slog),
// Panic Recovery, dan Security Headers.
package middleware

import (
	"errors"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"siakad-mini/helper"
)

// Register mendaftarkan seluruh middleware global ke router Fiber.
func Register(app *fiber.App, logger *slog.Logger, allowedOrigins string) {
	// 1. Request ID generator
	app.Use(requestid.New(requestid.Config{
		Header: "X-Request-ID",
	}))

	// 2. CORS (Cross-Origin Resource Sharing)
	app.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Request-ID",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
		AllowCredentials: true,
	}))

	// 3. Panic Recovery (mencegah server crash akibat unhandled runtime panic)
	app.Use(recover.New(recover.Config{
		EnableStackTrace: false, // Tidak membocorkan internal stack trace
	}))

	// 4. Security Headers
	app.Use(func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "1; mode=block")
		return c.Next()
	})

	// 5. Structured Request Logger
	app.Use(func(c *fiber.Ctx) error {
		start := time.Now()
		reqID := helper.RequestID(c)

		// Lanjutkan eksekusi ke handler berikutnya
		err := c.Next()

		latency := time.Since(start)
		statusCode := c.Response().StatusCode()

		// Tangani jika ada error yang belum terformat status kodenya
		if err != nil {
			var appErr *helper.AppError
			if errors.As(err, &appErr) {
				statusCode = appErr.Status
			}
		}

		logger.Info("http_request",
			slog.String("request_id", reqID),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", statusCode),
			slog.Duration("latency", latency),
			slog.String("ip", c.IP()),
		)

		return err
	})
}
