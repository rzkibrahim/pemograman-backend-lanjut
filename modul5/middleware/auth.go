package middleware

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"

	"modul5/app/model"
	"modul5/helper"
)

// RequireAuth melindungi endpoint yang mewajibkan autentikasi JWT.
// Membaca header Authorization: Bearer <token>, memvalidasi signature & masa berlaku,
// dan menyematkan identitas user ke Locals context.
func RequireAuth(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.Set("WWW-Authenticate", "Bearer")
			return c.Status(fiber.StatusUnauthorized).JSON(model.WebResponse{
				Success: false,
				Message: "token autentikasi diperlukan",
			})
		}

		tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if tokenString == "" {
			c.Set("WWW-Authenticate", "Bearer")
			return c.Status(fiber.StatusUnauthorized).JSON(model.WebResponse{
				Success: false,
				Message: "token autentikasi diperlukan",
			})
		}

		claims, err := helper.ParseAccessToken(tokenString, jwtSecret)
		if err != nil {
			if errors.Is(err, helper.ErrTokenExpired) {
				c.Set("WWW-Authenticate", `Bearer error="invalid_token", error_description="access token telah kedaluwarsa"`)
				return c.Status(fiber.StatusUnauthorized).JSON(model.WebResponse{
					Success: false,
					Message: "access token telah kedaluwarsa",
				})
			}

			c.Set("WWW-Authenticate", `Bearer error="invalid_token", error_description="access token tidak valid"`)
			return c.Status(fiber.StatusUnauthorized).JSON(model.WebResponse{
				Success: false,
				Message: "access token tidak valid",
			})
		}

		// Simpan payload user ke context
		helper.SetCurrentUser(c, claims)
		return c.Next()
	}
}

// LoginRateLimiter membatasi percobaan login untuk mencegah serangan Brute Force.
// Maksimal 5 permintaan per menit per alamat IP. Permintaan ke-6 menghasilkan status 429 + Retry-After.
func LoginRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			c.Set("Retry-After", "60")
			return c.Status(fiber.StatusTooManyRequests).JSON(model.WebResponse{
				Success: false,
				Message: "terlalu banyak percobaan login, silakan coba lagi setelah 60 detik",
			})
		},
	})
}
