package route

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-students/app/service"
	"api-students/helper"
	"api-students/middleware"
)

// Register mendaftarkan seluruh endpoint rute API ke aplikasi Fiber.
func Register(app *fiber.App, pool *pgxpool.Pool, studentService *service.StudentService) {
	// Grup API dengan versioning
	api := app.Group("/api/v1")

	// Endpoint health check (memeriksa kesiapan basis data)
	api.Get("/health", healthCheck(pool))

	// Endpoint students (dengan middleware pemeriksaan tipe konten JSON)
	s := api.Group("/students", middleware.RequireJSON)
	s.Get("/", studentService.List)
	s.Get("/:id", studentService.Get)
	s.Post("/", studentService.Create)
	s.Put("/:id", studentService.Replace)
	s.Patch("/:id", studentService.Patch)
	s.Delete("/:id", studentService.Delete)
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
