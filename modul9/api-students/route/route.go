package route

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"modul9/app/model"
	"modul9/app/service"
	"modul9/helper"
	"modul9/middleware"
)

func Register(
	app *fiber.App,
	pool *pgxpool.Pool,
	studentService *service.StudentService,
	userService *service.UserService,
	authService service.AuthService,
	courseService *service.CourseService,
	supervisorService *service.SupervisorService,
	perms *helper.PermissionSet,
	jwtSecret string,
) {
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("API Students - Praktikum Backend Lanjut (Pertemuan 9: Transactions & Concurrency)")
	})

	api := app.Group("/api/v1")

	api.Get("/health", healthCheck(pool))

	auth := api.Group("/auth")

	auth.Post("/register", middleware.RequireJSON, handleRegister(authService))
	auth.Post("/login", middleware.RequireJSON, middleware.LoginRateLimiter(), handleLogin(authService))
	auth.Post("/refresh", middleware.RequireJSON, handleRefreshToken(authService))
	auth.Post("/logout", middleware.RequireJSON, handleLogout(authService))

	auth.Get("/me", middleware.RequireAuth(jwtSecret), handleMe(authService))

	s := api.Group("/students", middleware.RequireAuth(jwtSecret))

	s.Get("/", middleware.RequirePermission(perms, "student:list"), studentService.List)
	s.Post("/", middleware.RequireJSON, middleware.RequirePermission(perms, "student:create"), studentService.Create)
	s.Get("/:id", studentService.Get)
	s.Put("/:id", middleware.RequireJSON, studentService.Replace)
	s.Patch("/:id", middleware.RequireJSON, studentService.Patch)
	s.Delete("/:id", middleware.RequirePermission(perms, "student:delete"), studentService.Delete)

	u := api.Group("/users", middleware.RequireAuth(jwtSecret))

	u.Get("/", middleware.RequirePermission(perms, "user:list"), userService.List)
	u.Get("/:id", userService.Get)
	u.Put("/:id", middleware.RequireJSON, userService.Update)
	u.Delete("/:id", middleware.RequirePermission(perms, "user:delete"), userService.Delete)
	u.Patch("/:id/role", middleware.RequireJSON, middleware.RequirePermission(perms, "role:assign"), userService.AssignRole)

	// Domain Courses (Modul 9 Transactions & Concurrency)
	cGroup := api.Group("/courses", middleware.RequireAuth(jwtSecret))

	cGroup.Get("/", middleware.RequirePermission(perms, "course:list"), courseService.List)
	cGroup.Get("/:id", middleware.RequirePermission(perms, "course:list"), courseService.Get)
	cGroup.Post("/:id/enroll", middleware.RequireJSON, middleware.RequirePermission(perms, "course:enroll"), courseService.Enroll)
	cGroup.Post("/:id/enroll-naive", middleware.RequireJSON, middleware.RequirePermission(perms, "course:enroll"), courseService.EnrollNaive)
	cGroup.Post("/:id/drop", middleware.RequireJSON, middleware.RequirePermission(perms, "course:drop"), courseService.Drop)
	cGroup.Post("/swap", middleware.RequireJSON, middleware.RequirePermission(perms, "course:swap"), courseService.Swap)
	cGroup.Patch("/:id", middleware.RequireJSON, middleware.RequirePermission(perms, "course:update"), courseService.UpdateQuota)

	// Domain Supervisors (Tugas Mandiri D.2 Pembimbing TA)
	sup := api.Group("/supervisors", middleware.RequireAuth(jwtSecret))

	sup.Get("/", supervisorService.List)
	sup.Get("/:id", supervisorService.Get)
	sup.Post("/:id/assign", middleware.RequireJSON, middleware.RequirePermission(perms, "supervision:manage"), supervisorService.Assign)
	sup.Patch("/:id", middleware.RequireJSON, middleware.RequirePermission(perms, "supervision:manage"), supervisorService.UpdateQuota)

	sups := api.Group("/supervisions", middleware.RequireAuth(jwtSecret))
	sups.Post("/switch", middleware.RequireJSON, middleware.RequirePermission(perms, "supervision:manage"), supervisorService.Switch)
}

func handleRegister(authService service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req model.RegisterRequest
		if err := c.BodyParser(&req); err != nil {
			return helper.BadRequest("body harus berupa JSON yang valid")
		}

		userResp, errs, err := authService.Register(c.UserContext(), req)
		if len(errs) > 0 {
			return helper.Validation(errs)
		}
		if err != nil {
			if errors.Is(err, service.ErrUserExists) {
				return helper.Conflict(err.Error())
			}
			return helper.Internal(err)
		}

		return helper.Success(c, fiber.StatusCreated, "user berhasil dibuat", userResp)
	}
}

func handleLogin(authService service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req model.LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return helper.BadRequest("body harus berupa JSON yang valid")
		}

		authResp, errs, err := authService.Login(c.UserContext(), req)
		if len(errs) > 0 {
			return helper.Validation(errs)
		}
		if err != nil {
			if errors.Is(err, service.ErrInvalidCredentials) {
				return helper.Unauthorized(err.Error())
			}
			return helper.Internal(err)
		}

		return helper.Success(c, fiber.StatusOK, "login berhasil", authResp)
	}
}

func handleRefreshToken(authService service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req model.RefreshTokenRequest
		if err := c.BodyParser(&req); err != nil {
			return helper.BadRequest("body harus berupa JSON yang valid")
		}

		authResp, err := authService.RefreshToken(c.UserContext(), req)
		if err != nil {
			if errors.Is(err, service.ErrTokenRevoked) || errors.Is(err, service.ErrRefreshTokenExpired) {
				return helper.Unauthorized(err.Error())
			}
			return helper.Unauthorized("refresh token tidak valid")
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
			return helper.Unauthorized("token autentikasi diperlukan")
		}

		userResp, err := authService.Me(c.UserContext(), currentUser.UserID)
		if err != nil {
			return helper.NotFound("user tidak ditemukan")
		}

		return helper.Success(c, fiber.StatusOK, "data pengguna berhasil diambil", userResp)
	}
}

func healthCheck(pool *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			return helper.ServiceUnavailable("database tidak dapat dihubungi")
		}
		return helper.Success(c, fiber.StatusOK, "server dan database berjalan", fiber.Map{
			"timestamp": time.Now(),
		})
	}
}
