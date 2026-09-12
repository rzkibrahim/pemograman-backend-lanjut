// Package middleware berisi seluruh middleware global yang dipasang ke aplikasi Fiber.
//
// Urutan pemasangan middleware penting karena dieksekusi secara berantai (chain):
//  1. requestid â€” menyisipkan UUID unik ke setiap request untuk tracing.
//  2. recover   â€” menangkap panic agar server tidak crash.
//  3. helmet    â€” menambahkan header keamanan HTTP (XSS, clickjacking, dsb).
//  4. cors      â€” mengizinkan request lintas origin (Cross-Origin Resource Sharing).
//  5. RequestLogger â€” mencatat log terstruktur setiap HTTP request via slog.
//
// Middleware RequireJSON dipakai secara selektif pada grup route yang membutuhkan
// validasi Content-Type (POST, PUT, PATCH).
package middleware

import (
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"modul4/helper"
)

// Register mendaftarkan seluruh middleware global ke aplikasi Fiber sesuai urutan:
// requestid -> recover -> helmet -> cors -> RequestLogger.
func Register(app *fiber.App, logger *slog.Logger) {
	app.Use(requestid.New())
	app.Use(recover.New())
	app.Use(helmet.New())
	app.Use(cors.New())
	app.Use(RequestLogger(logger))
}

// RequestLogger mencatat setiap HTTP request dalam format terstruktur memakai slog.
func RequestLogger(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		latency := time.Since(start)

		reqID, _ := c.Locals("requestid").(string)
		status := c.Response().StatusCode()

		attrs := []slog.Attr{
			slog.String("request_id", reqID),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", status),
			slog.Duration("latency", latency),
			slog.String("ip", c.IP()),
		}

		if err != nil {
			attrs = append(attrs, slog.String("error", err.Error()))
		}

		lvl := slog.LevelInfo
		if status >= 500 {
			lvl = slog.LevelError
		} else if status >= 400 {
			lvl = slog.LevelWarn
		}

		logger.LogAttrs(c.UserContext(), lvl, "HTTP request", attrs...)
		return err
	}
}

// metodeBerbody mendaftarkan metode HTTP yang seharusnya membawa body.
var metodeBerbody = map[string]bool{
	fiber.MethodPost:  true,
	fiber.MethodPut:   true,
	fiber.MethodPatch: true,
}

// RequireJSON adalah middleware yang menolak request berisi body yang
// Content-Type-nya bukan application/json.
func RequireJSON(c *fiber.Ctx) error {
	if metodeBerbody[c.Method()] {
		ct := c.Get("Content-Type")
		if !strings.HasPrefix(ct, fiber.MIMEApplicationJSON) {
			return helper.Fail(c, fiber.StatusUnsupportedMediaType, "Content-Type harus application/json")
		}
	}
	return c.Next()
}

