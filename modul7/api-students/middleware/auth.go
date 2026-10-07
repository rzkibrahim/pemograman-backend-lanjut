package middleware

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"

	"modul7/helper"
)

func RequireAuth(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.Set("WWW-Authenticate", "Bearer")
			return helper.Unauthorized("token autentikasi diperlukan")
		}

		tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if tokenString == "" {
			c.Set("WWW-Authenticate", "Bearer")
			return helper.Unauthorized("token autentikasi diperlukan")
		}

		claims, err := helper.ParseAccessToken(tokenString, jwtSecret)
		if err != nil {
			if errors.Is(err, helper.ErrTokenExpired) {
				c.Set("WWW-Authenticate", `Bearer error="invalid_token", error_description="access token telah kedaluwarsa"`)
				return helper.Unauthorized("access token telah kedaluwarsa")
			}
			c.Set("WWW-Authenticate", `Bearer error="invalid_token", error_description="access token tidak valid"`)
			return helper.Unauthorized("access token tidak valid")
		}

		helper.SetCurrentUser(c, claims)
		return c.Next()
	}
}

func LoginRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        60,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			c.Set("Retry-After", "60")
			return helper.TooManyRequests("terlalu banyak percobaan login, silakan coba lagi setelah 60 detik")
		},
	})
}