// Package middleware - auth.go
// Mengelola autentikasi berbasis token JWT (Bearer Token) dan otorisasi berbasis Role (RBAC).
package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"siakad-mini/helper"
)

// RequireAuth memvalidasi keberadaan dan validitas JWT access token pada header Authorization.
func RequireAuth(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return helper.Unauthorized("token autentikasi diperlukan (Authorization: Bearer <token>)")
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return helper.Unauthorized("format authorization header tidak valid, gunakan format 'Bearer <token>'")
		}

		tokenString := strings.TrimSpace(parts[1])
		payload, err := helper.ParseAccessToken(tokenString, jwtSecret)
		if err != nil {
			return helper.Unauthorized("token tidak valid atau telah kedaluwarsa")
		}

		// Simpan payload identitas pengguna di konteks request Fiber
		c.Locals("user", payload)
		return c.Next()
	}
}

// RequireRole memverifikasi apakah role pengguna yang sedang login memenuhi daftar role yang diizinkan.
func RequireRole(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := helper.CurrentUser(c)
		if err != nil {
			return err
		}

		for _, role := range allowedRoles {
			if strings.EqualFold(user.Role, role) {
				return c.Next()
			}
		}

		return helper.Forbidden("akses ditolak: peran Anda (" + user.Role + ") tidak memiliki hak akses ke endpoint ini")
	}
}
