package middleware

import (
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"modul6/helper"
)

// Register mendaftarkan seluruh middleware global ke aplikasi Fiber sesuai urutan:
// requestid -> recover -> helmet -> cors (restricted) -> RequestLogger.
func Register(app *fiber.App, logger *slog.Logger, allowedOrigins string) {
	app.Use(requestid.New())
	app.Use(recover.New())
	app.Use(helmet.New())

	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:3000,http://localhost:5173"
	}

	// Batasi origin CORS sesuai daftar di environment (menutup kerentanan CORS longgar)
	app.Use(cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, PATCH, DELETE, OPTIONS",
	}))

	app.Use(RequestLogger(logger))
}

// RequestLogger mencatat setiap HTTP request dalam format terstruktur memakai slog.
// Untuk memenuhi asas ketertelusuran (Langkah 8 & C.2 #5), RequestLogger juga mencatat
// user_id dan role jika pengguna telah terautentikasi (Locals currentUser).
func RequestLogger(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		latency := time.Since(start)

		reqID, _ := c.Locals("requestid").(string)
		status := c.Response().StatusCode()
		method := c.Method()
		path := c.Path()

		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}

		// Ambil identitas user dari context jika sudah terautentikasi
		userIDStr := "-"
		roleStr := "anonymous"
		if user, ok := helper.GetCurrentUser(c); ok && user != nil {
			userIDStr = strconv.Itoa(user.UserID)
			roleStr = user.Role
		}

		logger.Log(c.Context(), level, "http request",
			slog.String("request_id", reqID),
			slog.String("method", method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Duration("latency", latency),
			slog.String("ip", c.IP()),
			slog.String("user_id", userIDStr),
			slog.String("role", roleStr),
		)

		return err
	}
}

// RequireJSON menolak request POST, PUT, dan PATCH yang tidak ber-header application/json.
func RequireJSON(c *fiber.Ctx) error {
	method := c.Method()
	if method == fiber.MethodPost || method == fiber.MethodPut || method == fiber.MethodPatch {
		ct := c.Get("Content-Type")
		if !strings.HasPrefix(ct, fiber.MIMEApplicationJSON) {
			return helper.Fail(c, fiber.StatusUnsupportedMediaType, "Content-Type harus application/json")
		}
	}
	return c.Next()
}
