// Package route mendaftarkan seluruh endpoint REST API ke aplikasi Fiber.
package route

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"modul5/app/model"
	"modul5/app/service"
	"modul5/helper"
	"modul5/middleware"
)

// Register mendaftarkan seluruh endpoint rute API ke aplikasi Fiber.
func Register(
	app *fiber.App,
	pool *pgxpool.Pool,
	studentService *service.StudentService,
	authService service.AuthService,
	jwtSecret string,
) {
	// Endpoint publik root
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("API Students - Praktikum Backend Lanjut (Pertemuan 5: Authentication & Security)")
	})

	// Grup API dengan versioning
	api := app.Group("/api/v1")

	// Endpoint publik: health check
	api.Get("/health", healthCheck(pool))

	// ==========================================
	// 1. Endpoint Autentikasi (/api/v1/auth)
	// ==========================================
	auth := api.Group("/auth")

	// Register & Login (dilindungi requireJSON & rate limiter khusus login)
	auth.Post("/register", middleware.RequireJSON, handleRegister(authService))
	auth.Post("/login", middleware.RequireJSON, middleware.LoginRateLimiter(), handleLogin(authService))
	auth.Post("/refresh", middleware.RequireJSON, handleRefreshToken(authService))
	auth.Post("/logout", middleware.RequireJSON, handleLogout(authService))

	// Profil pengguna (/me) dilindungi token JWT
	auth.Get("/me", middleware.RequireAuth(jwtSecret), handleMe(authService))

	// ==========================================
	// 2. Endpoint Mahasiswa (/api/v1/students)
	// Terproteksi penuh: mewajibkan JWT valid via RequireAuth
	// ==========================================
	s := api.Group("/students", middleware.RequireAuth(jwtSecret), middleware.RequireJSON)
	s.Get("/", studentService.List)
	s.Get("/:id", studentService.Get)
	s.Post("/", studentService.Create)
	s.Put("/:id", studentService.Replace)
	s.Patch("/:id", studentService.Patch)
	s.Delete("/:id", studentService.Delete)
}

func handleRegister(authService service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req model.RegisterRequest
		if err := c.BodyParser(&req); err != nil {
			return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
		}

		userResp, errs, err := authService.Register(c.UserContext(), req)
		if len(errs) > 0 {
			return helper.FailValidation(c, errs)
		}
		if err != nil {
			if errors.Is(err, service.ErrUserExists) {
				return helper.Fail(c, fiber.StatusConflict, err.Error())
			}
			return helper.Fail(c, fiber.StatusInternalServerError, "gagal mendaftarkan pengguna")
		}

		return helper.Success(c, fiber.StatusCreated, "user berhasil dibuat", userResp)
	}
}

func handleLogin(authService service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req model.LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
		}

		authResp, errs, err := authService.Login(c.UserContext(), req)
		if len(errs) > 0 {
			return helper.FailValidation(c, errs)
		}
		if err != nil {
			if errors.Is(err, service.ErrInvalidCredentials) {
				return helper.Fail(c, fiber.StatusUnauthorized, err.Error())
			}
			return helper.Fail(c, fiber.StatusInternalServerError, "gagal memproses login")
		}

		return helper.Success(c, fiber.StatusOK, "login berhasil", authResp)
	}
}

func handleRefreshToken(authService service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req model.RefreshTokenRequest
		if err := c.BodyParser(&req); err != nil {
			return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
		}

		authResp, err := authService.RefreshToken(c.UserContext(), req)
		if err != nil {
			if errors.Is(err, service.ErrTokenRevoked) || errors.Is(err, service.ErrRefreshTokenExpired) {
				return helper.Fail(c, fiber.StatusUnauthorized, err.Error())
			}
			return helper.Fail(c, fiber.StatusUnauthorized, "refresh token tidak valid")
		}

		return helper.Success(c, fiber.StatusOK, "token berhasil diperbarui", authResp)
	}
}

func handleLogout(authService service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req model.RefreshTokenRequest
		_ = c.BodyParser(&req)
		_ = authService.Logout(c.UserContext(), req.RefreshToken)
		return helper.Success(c, fiber.StatusOK, "logout berhasil", nil)
	}
}

func handleMe(authService service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		currentUser, ok := helper.GetCurrentUser(c)
		if !ok || currentUser == nil {
			return helper.Fail(c, fiber.StatusUnauthorized, "token autentikasi diperlukan")
		}

		userResp, err := authService.Me(c.UserContext(), currentUser.UserID)
		if err != nil {
			return helper.Fail(c, fiber.StatusNotFound, "user tidak ditemukan")
		}

		return helper.Success(c, fiber.StatusOK, "data pengguna berhasil diambil", userResp)
	}
}

// healthCheck memeriksa ketersediaan basis data dan mengembalikan status kesehatan sistem.
func healthCheck(pool *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			return helper.Fail(c, fiber.StatusServiceUnavailable, "database tidak dapat dihubungi")
		}
		return helper.Success(c, fiber.StatusOK, "server dan database berjalan", fiber.Map{
			"timestamp": time.Now(),
		})
	}
}
