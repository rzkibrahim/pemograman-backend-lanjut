package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

// metodeBerbody mendaftarkan metode HTTP yang seharusnya membawa body.
// Hanya metode-metode ini yang diperiksa Content-Type-nya.
var metodeBerbody = map[string]bool{
	fiber.MethodPost:  true,
	fiber.MethodPut:   true,
	fiber.MethodPatch: true,
}

// requireJSON adalah middleware yang menolak request berisi body yang
// Content-Type-nya bukan application/json.
// Status yang tepat untuk kasus ini adalah 415 Unsupported Media Type, bukan 400.
func requireJSON(c *fiber.Ctx) error {
	if metodeBerbody[c.Method()] {
		ct := c.Get("Content-Type")
		if !strings.HasPrefix(ct, fiber.MIMEApplicationJSON) {
			return fail(c, fiber.StatusUnsupportedMediaType,
				"Content-Type harus application/json")
		}
	}
	return c.Next()
}

func main() {
	app := fiber.New(fiber.Config{
		AppName: "API Students - Praktikum Backend Lanjut Pertemuan 2",
		// ErrorHandler kustom agar semua error dikembalikan dalam amplop WebResponse
		// yang konsisten, bukan format bawaan Fiber.
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			status := fiber.StatusInternalServerError
			pesan := "terjadi kesalahan pada server"
			if e, ok := err.(*fiber.Error); ok {
				status = e.Code
				pesan = e.Message
			}
			return fail(c, status, pesan)
		},
	})

	// === Middleware Global ===
	// Urutan pemasangan menentukan urutan eksekusi:
	// 1. requestid ΓÇö memberi setiap request ID unik untuk pelacakan di log
	// 2. logger ΓÇö mencatat setiap request beserta ID-nya
	// 3. cors ΓÇö mengizinkan akses lintas origin (untuk frontend)
	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${locals:requestid} ${method} ${path} ${status} ${latency}\n",
	}))
	app.Use(cors.New())

	// Endpoint root ΓÇö tetap dipertahankan untuk perbandingan
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("API Students - Praktikum Backend Lanjut Pertemuan 2")
	})

	// Grup API dengan versioning di awal jalur
	api := app.Group("/api/v1")

	// Health check ΓÇö berguna untuk monitoring dan memastikan server berjalan
	api.Get("/health", func(c *fiber.Ctx) error {
		return ok(c, "server berjalan", fiber.Map{"timestamp": time.Now()})
	})

	// Grup students ΓÇö requireJSON dipasang per grup, bukan global,
	// agar endpoint lain yang mungkin menerima format berbeda tidak tertolak.
	s := api.Group("/students", requireJSON)
	s.Get("/", listStudents)
	s.Get("/:id", getStudent)
	s.Post("/", createStudent)
	s.Put("/:id", replaceStudent)
	s.Patch("/:id", patchStudent)
	s.Delete("/:id", deleteStudent)

	// Catch-all ΓÇö endpoint yang tidak dikenal mendapat 404
	app.Use(func(c *fiber.Ctx) error {
		return fail(c, fiber.StatusNotFound, "endpoint tidak ditemukan")
	})

	fmt.Println("Server berjalan di http://localhost:3000")
	log.Fatal(app.Listen(":3000"))
}
