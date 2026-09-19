// Package route mendaftarkan seluruh endpoint REST API ke aplikasi Fiber.
package route

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"modul6/app/model"
	"modul6/app/service"
	"modul6/helper"
	"modul6/middleware"
)

// Register mendaftarkan seluruh rute endpoint API ke aplikasi Fiber.
// Memetakan secara visual dan tegas mana endpoint yang bebas,
// mana yang hanya butuh login (RequireAuth), dan mana yang butuh permission khusus (RequirePermission).
func Register(
	app *fiber.App,
	pool *pgxpool.Pool,
	studentService *service.StudentService,
	userService *service.UserService,
	authService service.AuthService,
	perms *helper.PermissionSet,
	jwtSecret string,
) {
	// Endpoint publik root
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("API Students - Praktikum Backend Lanjut (Pertemuan 6: Authorization & RBAC)")
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

	// Profil pengguna sendiri (/me) dilindungi token JWT
	auth.Get("/me", middleware.RequireAuth(jwtSecret), handleMe(authService))

	// ==========================================
	// 2. Endpoint Mahasiswa (/api/v1/students)
	// PENTING: RequireAuth selalu sebelum RequirePermission
	// ==========================================
	s := api.Group("/students", middleware.RequireAuth(jwtSecret))

	// GET /students -> Melihat daftar mahasiswa (butuh permission "student:list")
	s.Get("/", middleware.RequirePermission(perms, "student:list"), studentService.List)

	// POST /students -> Menambah mahasiswa baru (butuh permission "student:create", auto-fill owner_id)
	s.Post("/", middleware.RequireJSON, middleware.RequirePermission(perms, "student:create"), studentService.Create)

	// GET /students/:id -> Detail mahasiswa (Ownership check di service: pemilik atau punya student:read:any)
	s.Get("/:id", studentService.Get)

	// PUT /students/:id -> Ganti data mahasiswa (Ownership check di service: pemilik atau punya student:update:any)
	s.Put("/:id", middleware.RequireJSON, studentService.Replace)

	// PATCH /students/:id -> Update parsial mahasiswa (Ownership check di service: pemilik atau punya student:update:any)
	s.Patch("/:id", middleware.RequireJSON, studentService.Patch)

	// DELETE /students/:id -> Hapus mahasiswa (hanya role berhak "student:delete", yaitu admin)
	s.Delete("/:id", middleware.RequirePermission(perms, "student:delete"), studentService.Delete)

	// ==========================================
	// 3. Endpoint Manajemen Pengguna (/api/v1/users)
	// ==========================================
	u := api.Group("/users", middleware.RequireAuth(jwtSecret))

	// GET /users -> Daftar seluruh user (butuh permission "user:list", admin/staff)
	u.Get("/", middleware.RequirePermission(perms, "user:list"), userService.List)

	// GET /users/:id -> Detail user (Ownership check di service: pemilik akun atau punya user:read:any)
	u.Get("/:id", userService.Get)

	// PUT /users/:id -> Ubah profil user (Ownership check di service: pemilik akun atau punya user:update:any)
	u.Put("/:id", middleware.RequireJSON, userService.Update)

	// DELETE /users/:id -> Hapus akun user (butuh permission "user:delete", admin only)
	u.Delete("/:id", middleware.RequirePermission(perms, "user:delete"), userService.Delete)

	// PATCH /users/:id/role -> Ubah role user (butuh permission "role:assign", admin only)
	u.Patch("/:id/role", middleware.RequireJSON, middleware.RequirePermission(perms, "role:assign"), userService.AssignRole)
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
