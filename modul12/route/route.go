package route

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"latihan-fiber/app/model"
	"latihan-fiber/app/service"
	"latihan-fiber/helper"
	"latihan-fiber/middleware"
)

// Dependencies membungkus seluruh dependensi perutean sistem.
type Dependencies struct {
	// EnableSwagger menyalakan Swagger UI di /swagger. Di production
	// sebaiknya dimatikan: dokumentasi lengkap adalah peta gratis bagi
	// penyerang.
	EnableSwagger       bool
	Pool                *pgxpool.Pool
	Mongo               *mongo.Client
	JWT                 *helper.JWTManager
	Permissions         *helper.PermissionSet
	UserService         *service.UserService
	AuthService         service.AuthService
	CourseService       *service.CourseService
	ReviewService       *service.ReviewService
	NotificationService *service.NotificationService
	StudentService      *service.StudentService
	SupervisorService   *service.SupervisorService
	PortfolioService    *service.PortfolioService
}

// Register mendaftarkan seluruh endpoint API dan middleware pelindungnya.
func Register(app *fiber.App, deps Dependencies) {
	if deps.EnableSwagger {
		// Swagger UI: http://localhost:3000/swagger/index.html
		app.Get("/swagger/*", swagger.HandlerDefault)
	}

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("API Students - Praktikum Backend Lanjut (Modul 11: Background Jobs & Asynq)")
	})

	api := app.Group("/api/v1")

	api.Get("/health", healthCheck(deps.Pool, deps.Mongo))

	// --- Auth Endpoints ---
	auth := api.Group("/auth")
	auth.Post("/register", middleware.RequireJSON, handleRegister(deps.AuthService))
	auth.Post("/login", middleware.RequireJSON, middleware.LoginRateLimiter(), handleLogin(deps.AuthService))
	auth.Post("/refresh", middleware.RequireJSON, handleRefreshToken(deps.AuthService))
	auth.Post("/logout", middleware.RequireJSON, handleLogout(deps.AuthService))
	auth.Get("/me", middleware.RequireAuth(deps.JWT), handleMe(deps.AuthService))

	perms := deps.Permissions

	// --- Students Endpoints (jika service tersedia) ---
	if deps.StudentService != nil {
		s := api.Group("/students", middleware.RequireAuth(deps.JWT))
		s.Get("/", middleware.RequirePermission(perms, "student:list"), deps.StudentService.List)
		s.Post("/", middleware.RequireJSON, middleware.RequirePermission(perms, "student:create"), deps.StudentService.Create)
		s.Get("/:id", deps.StudentService.Get)
		s.Put("/:id", middleware.RequireJSON, deps.StudentService.Replace)
		s.Patch("/:id", middleware.RequireJSON, deps.StudentService.Patch)
		s.Delete("/:id", middleware.RequirePermission(perms, "student:delete"), deps.StudentService.Delete)

		if deps.PortfolioService != nil {
			s.Get("/:id/portfolio", deps.PortfolioService.Get)
			s.Put("/:id/portfolio", middleware.RequireJSON, deps.PortfolioService.Upsert)
			s.Post("/:id/portfolio/projects", middleware.RequireJSON, deps.PortfolioService.AddProject)
			s.Delete("/:id/portfolio/projects/:projectId", deps.PortfolioService.DeleteProject)
		}
	}

	if deps.PortfolioService != nil {
		portfolios := api.Group("/portfolios", middleware.RequireAuth(deps.JWT))
		portfolios.Get("/stats/technologies", deps.PortfolioService.TechnologyStats)
	}

	// --- Users Endpoints ---
	if deps.UserService != nil {
		u := api.Group("/users", middleware.RequireAuth(deps.JWT))
		u.Get("/", middleware.RequirePermission(perms, "user:list"), deps.UserService.List)
		u.Get("/:id", deps.UserService.Get)
		u.Put("/:id", middleware.RequireJSON, deps.UserService.Update)
		u.Delete("/:id", middleware.RequirePermission(perms, "user:delete"), deps.UserService.Delete)
		u.Patch("/:id/role", middleware.RequireJSON, middleware.RequirePermission(perms, "role:assign"), deps.UserService.AssignRole)
	}

	// --- Courses Endpoints (Modul 9 & 11) ---
	if deps.CourseService != nil {
		cGroup := api.Group("/courses", middleware.RequireAuth(deps.JWT))
		cGroup.Get("/", middleware.RequirePermission(perms, "course:list"), deps.CourseService.List)
		cGroup.Get("/mine", middleware.RequirePermission(perms, "course:list"), deps.CourseService.Mine)
		cGroup.Get("/:id", middleware.RequirePermission(perms, "course:list"), deps.CourseService.Get)
		cGroup.Post("/:id/enroll", middleware.RequirePermission(perms, "course:enroll"), deps.CourseService.Enroll)
		cGroup.Delete("/:id/enroll", middleware.RequirePermission(perms, "course:drop"), deps.CourseService.Drop)
		cGroup.Post("/:id/enroll-naive", middleware.RequirePermission(perms, "course:enroll"), deps.CourseService.EnrollNaive)
		cGroup.Post("/:id/drop", middleware.RequirePermission(perms, "course:drop"), deps.CourseService.Drop)
		cGroup.Post("/swap", middleware.RequireJSON, middleware.RequirePermission(perms, "course:swap"), deps.CourseService.Swap)
		cGroup.Patch("/:id", middleware.RequireJSON, middleware.RequirePermission(perms, "course:update"), deps.CourseService.UpdateQuota)

		if deps.ReviewService != nil {
			cGroup.Get("/:id/reviews/summary", deps.ReviewService.Summary)
			cGroup.Get("/:id/reviews", deps.ReviewService.ListByCourse)
			cGroup.Post("/:id/reviews", middleware.RequireJSON, deps.ReviewService.Create)
		}

		// Alias rute /enrollments/swap sesuai spesifikasi uji C.1
		enrollments := api.Group("/enrollments", middleware.RequireAuth(deps.JWT))
		enrollments.Post("/swap", middleware.RequireJSON, middleware.RequirePermission(perms, "course:swap"), deps.CourseService.Swap)
	}

	// --- Reviews Endpoints ---
	if deps.ReviewService != nil {
		reviews := api.Group("/reviews", middleware.RequireAuth(deps.JWT))
		reviews.Get("/:id", deps.ReviewService.Get)
		reviews.Patch("/:id", middleware.RequireJSON, deps.ReviewService.Update)
		reviews.Delete("/:id", deps.ReviewService.Delete)
		reviews.Post("/:id/replies", middleware.RequireJSON, deps.ReviewService.AddReply)
		reviews.Post("/:id/helpful", deps.ReviewService.MarkHelpful)
	}

	// --- Supervisors Endpoints ---
	if deps.SupervisorService != nil {
		sup := api.Group("/supervisors", middleware.RequireAuth(deps.JWT))
		sup.Get("/", deps.SupervisorService.List)
		sup.Get("/:id", deps.SupervisorService.Get)
		sup.Post("/:id/assign", middleware.RequireJSON, middleware.RequirePermission(perms, "supervision:manage"), deps.SupervisorService.Assign)
		sup.Patch("/:id", middleware.RequireJSON, middleware.RequirePermission(perms, "supervision:manage"), deps.SupervisorService.UpdateQuota)

		sups := api.Group("/supervisions", middleware.RequireAuth(deps.JWT))
		sups.Post("/switch", middleware.RequireJSON, middleware.RequirePermission(perms, "supervision:manage"), deps.SupervisorService.Switch)
	}

	// --- Notifikasi (diisi oleh worker, dibaca lewat API) ---
	if deps.NotificationService != nil {
		notifications := api.Group("/notifications",
			middleware.RequireAuth(deps.JWT))
		notifications.Get("/mine", deps.NotificationService.Mine)

		admin := api.Group("/admin",
			middleware.RequireAuth(deps.JWT))
		admin.Get("/queue", middleware.RequirePermission(perms, "queue:monitor"), deps.NotificationService.Monitor)
	}
}

// handleRegister godoc
// @Summary      Mendaftarkan akun baru
// @Description  Role akun baru selalu "user". Password minimal 8 karakter dan memuat huruf serta angka.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body model.RegisterRequest true "Data akun baru"
// @Success      201 {object} model.WebResponse{data=model.UserResponse}
// @Failure      400 {object} model.ErrorResponse "Body bukan JSON yang valid"
// @Failure      409 {object} model.ErrorResponse "Username atau email sudah dipakai"
// @Failure      422 {object} model.ErrorResponse "Validasi gagal; detail per field ada di fields"
// @Router       /auth/register [post]
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

// handleLogin godoc
// @Summary      Login dan mendapatkan pasangan token
// @Description  Dibatasi 5 kali gagal per menit per IP.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body model.LoginRequest true "Username dan password"
// @Success      200 {object} model.WebResponse{data=model.AuthResponse}
// @Failure      400 {object} model.ErrorResponse
// @Failure      401 {object} model.ErrorResponse "Username atau password salah"
// @Failure      422 {object} model.ErrorResponse
// @Router       /auth/login [post]
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

// handleRefreshToken godoc
// @Summary      Menukar refresh token dengan pasangan token baru
// @Description  Refresh token lama langsung dicabut (rotation).
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body model.RefreshTokenRequest true "Refresh token"
// @Success      200 {object} model.WebResponse{data=model.AuthResponse}
// @Failure      400 {object} model.ErrorResponse
// @Failure      401 {object} model.ErrorResponse "Refresh token tidak valid, kedaluwarsa, atau sudah dicabut"
// @Router       /auth/refresh [post]
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

// handleLogout godoc
// @Summary      Logout pengguna
// @Description  Mencabut refresh token yang aktif
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body model.RefreshTokenRequest true "Refresh token yang akan dicabut"
// @Success      200 {object} model.WebResponse
// @Router       /auth/logout [post]
func handleLogout(authService service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req model.RefreshTokenRequest
		_ = c.BodyParser(&req)
		_ = authService.Logout(c.UserContext(), req.RefreshToken)
		return helper.Success(c, fiber.StatusOK, "logout berhasil", nil)
	}
}

// handleMe godoc
// @Summary      Profil pengguna saat ini
// @Description  Mengambil data profil pengguna yang sedang login dari token JWT
// @Tags         auth
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} model.WebResponse{data=model.UserResponse}
// @Failure      401 {object} model.ErrorResponse
// @Failure      404 {object} model.ErrorResponse
// @Router       /auth/me [get]
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

func healthCheck(pool *pgxpool.Pool, mongoClient *mongo.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			return helper.ServiceUnavailable("database tidak dapat dihubungi")
		}
		if mongoClient != nil {
			if err := mongoClient.Ping(ctx, nil); err != nil {
				return helper.ServiceUnavailable("database tidak dapat dihubungi")
			}
		}
		return helper.Success(c, fiber.StatusOK, "server, PostgreSQL, dan MongoDB berjalan", fiber.Map{
			"timestamp": time.Now(),
		})
	}
}
