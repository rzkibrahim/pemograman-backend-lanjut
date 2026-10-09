// Package route - route.go
// Mendaftarkan 10 endpoint RESTful API SIAKAD Mini sesuai spesifikasi teknis UTS.
// Menangani parsing input, validasi deklaratif, otorisasi per endpoint, dan pembentukan respons HTTP.
package route

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"siakad-mini/app/model"
	"siakad-mini/app/service"
	"siakad-mini/helper"
	"siakad-mini/middleware"
)

// Register mendaftarkan seluruh endpoint SIAKAD Mini ke instance Fiber router.
func Register(
	app *fiber.App,
	pool *pgxpool.Pool,
	authService service.AuthService,
	studentService *service.StudentService,
	courseService *service.CourseService,
	enrollService *service.EnrollmentService,
	jwtSecret string,
) {
	// Endpoint Health Check
	app.Get("/health", func(c *fiber.Ctx) error {
		ctx, cancel := helper.RequestContext(c)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			return helper.ServiceUnavailable("koneksi database terputus")
		}
		return helper.Success(c, fiber.StatusOK, "SIAKAD Mini API berjalan normal", fiber.Map{
			"status": "healthy",
		})
	})

	api := app.Group("/api/v1")

	// =========================================================================
	// ENDPOINT 1: POST /api/v1/auth/login (Publik)
	// Login, mengembalikan access token JWT
	// =========================================================================
	api.Post("/auth/login", func(c *fiber.Ctx) error {
		ip := c.IP()

		// 1. Cek rate limit: 429 jika gagal lebih dari 5 kali per menit
		if err := middleware.CheckLoginRateLimit(ip); err != nil {
			return err
		}

		var req model.LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return helper.BadRequest("format body JSON tidak valid")
		}

		// 2. Validasi deklaratif
		if errs := helper.ValidateStruct(req); errs != nil {
			return helper.Validation(errs)
		}

		ctx, cancel := helper.RequestContext(c)
		defer cancel()

		// 3. Eksekusi alur login
		resp, err := authService.Login(ctx, req)
		if err != nil {
			// Catat kegagalan untuk rate limiting
			middleware.RecordLoginFailure(ip)
			return err
		}

		// Reset riwayat kegagalan setelah berhasil login
		middleware.ResetLoginFailure(ip)

		return helper.Success(c, fiber.StatusOK, "Login berhasil", resp)
	})

	// Rute yang membutuhkan autentikasi token JWT
	authGroup := api.Group("", middleware.RequireAuth(jwtSecret))

	// =========================================================================
	// ENDPOINT 2: GET /api/v1/auth/me (Semua role)
	// Profil pengguna yang sedang login beserta data mahasiswa jika perannya mahasiswa
	// =========================================================================
	authGroup.Get("/auth/me", func(c *fiber.Ctx) error {
		user, err := helper.CurrentUser(c)
		if err != nil {
			return err
		}

		ctx, cancel := helper.RequestContext(c)
		defer cancel()

		resp, err := authService.Me(ctx, user.UserID)
		if err != nil {
			return err
		}

		return helper.Success(c, fiber.StatusOK, "Profil pengguna berhasil diambil", resp)
	})

	// =========================================================================
	// ENDPOINT 3: GET /api/v1/students (Admin)
	// Daftar mahasiswa dengan pagination, filtering, searching, dan sorting
	// =========================================================================
	authGroup.Get("/students", middleware.RequireRole(model.RoleAdmin), func(c *fiber.Ctx) error {
		page := helper.ParseQueryInt(c.Query("page"), 1, 1, 0)
		perPage := helper.ParseQueryInt(c.Query("per_page"), 10, 1, 50)
		angkatan := helper.ParseQueryInt(c.Query("angkatan"), 0, 0, 0)
		prodi := strings.TrimSpace(c.Query("prodi"))
		search := strings.TrimSpace(c.Query("search"))
		sort := strings.TrimSpace(c.Query("sort"))

		q := model.StudentListQuery{
			Page:     page,
			PerPage:  perPage,
			Prodi:    prodi,
			Angkatan: angkatan,
			Search:   search,
			Sort:     sort,
		}

		ctx, cancel := helper.RequestContext(c)
		defer cancel()

		items, meta, err := studentService.List(ctx, q)
		if err != nil {
			return err
		}

		return helper.SuccessList(c, "Data mahasiswa berhasil diambil", items, meta)
	})

	// =========================================================================
	// ENDPOINT 4: POST /api/v1/students (Admin)
	// Menambah mahasiswa sekaligus membuat akun user-nya dalam satu transaksi
	// =========================================================================
	authGroup.Post("/students", middleware.RequireRole(model.RoleAdmin), func(c *fiber.Ctx) error {
		var req model.CreateStudentRequest
		if err := c.BodyParser(&req); err != nil {
			return helper.BadRequest("format body JSON tidak valid")
		}

		if errs := helper.ValidateStruct(req); errs != nil {
			return helper.Validation(errs)
		}

		ctx, cancel := helper.RequestContext(c)
		defer cancel()

		createdStudent, err := studentService.Create(ctx, req)
		if err != nil {
			return err
		}

		location := fmt.Sprintf("/api/v1/students/%d", createdStudent.ID)
		return helper.Created(c, "Mahasiswa dan akun pengguna berhasil dibuat", createdStudent, location)
	})

	// =========================================================================
	// ENDPOINT 5: GET /api/v1/students/{id} (Admin, Mahasiswa data sendiri)
	// Detail mahasiswa beserta daftar mata kuliah, total SKS, dan batas SKS
	// =========================================================================
	authGroup.Get("/students/:id", func(c *fiber.Ctx) error {
		id, err := helper.ParamID(c, "id")
		if err != nil {
			return err
		}

		user, err := helper.CurrentUser(c)
		if err != nil {
			return err
		}

		ctx, cancel := helper.RequestContext(c)
		defer cancel()

		detail, err := studentService.GetDetail(ctx, id, user)
		if err != nil {
			return err
		}

		return helper.Success(c, fiber.StatusOK, "Detail mahasiswa berhasil diambil", detail)
	})

	// =========================================================================
	// ENDPOINT 6: PUT /api/v1/students/{id} (Admin)
	// Memperbarui data mahasiswa (NIM tidak boleh diubah)
	// =========================================================================
	authGroup.Put("/students/:id", middleware.RequireRole(model.RoleAdmin), func(c *fiber.Ctx) error {
		id, err := helper.ParamID(c, "id")
		if err != nil {
			return err
		}

		var req model.UpdateStudentRequest
		if err := c.BodyParser(&req); err != nil {
			return helper.BadRequest("format body JSON tidak valid")
		}

		if errs := helper.ValidateStruct(req); errs != nil {
			return helper.Validation(errs)
		}

		ctx, cancel := helper.RequestContext(c)
		defer cancel()

		updated, err := studentService.Update(ctx, id, req)
		if err != nil {
			return err
		}

		return helper.Success(c, fiber.StatusOK, "Data mahasiswa berhasil diperbarui", updated)
	})

	// =========================================================================
	// ENDPOINT 7: DELETE /api/v1/students/{id} (Admin)
	// Soft delete mahasiswa (mengisi deleted_at)
	// =========================================================================
	authGroup.Delete("/students/:id", middleware.RequireRole(model.RoleAdmin), func(c *fiber.Ctx) error {
		id, err := helper.ParamID(c, "id")
		if err != nil {
			return err
		}

		ctx, cancel := helper.RequestContext(c)
		defer cancel()

		if err := studentService.Delete(ctx, id); err != nil {
			return err
		}

		return helper.NoContent(c)
	})

	// =========================================================================
	// ENDPOINT 8: GET /api/v1/courses (Semua role)
	// Daftar mata kuliah beserta sisa kuota dinamis
	// =========================================================================
	authGroup.Get("/courses", func(c *fiber.Ctx) error {
		semester := helper.ParseQueryInt(c.Query("semester"), 0, 0, 8)
		search := strings.TrimSpace(c.Query("search"))
		available := false
		if c.Query("available") != "" {
			b, err := strconv.ParseBool(c.Query("available"))
			if err == nil {
				available = b
			}
		}

		q := model.CourseQuery{
			Semester:  semester,
			Search:    search,
			Available: available,
		}

		ctx, cancel := helper.RequestContext(c)
		defer cancel()

		courses, err := courseService.List(ctx, q)
		if err != nil {
			return err
		}

		return helper.Success(c, fiber.StatusOK, "Daftar mata kuliah berhasil diambil", courses)
	})

	// =========================================================================
	// ENDPOINT 9: POST /api/v1/enrollments (Mahasiswa)
	// Mengambil mata kuliah (tambah KRS) dengan validasi kuota, duplikasi, & SKS
	// =========================================================================
	authGroup.Post("/enrollments", middleware.RequireRole(model.RoleMahasiswa), func(c *fiber.Ctx) error {
		user, err := helper.CurrentUser(c)
		if err != nil {
			return err
		}

		var req model.CreateEnrollmentRequest
		if err := c.BodyParser(&req); err != nil {
			return helper.BadRequest("format body JSON tidak valid")
		}

		if errs := helper.ValidateStruct(req); errs != nil {
			return helper.Validation(errs)
		}

		ctx, cancel := helper.RequestContext(c)
		defer cancel()

		resp, err := enrollService.Create(ctx, user, req)
		if err != nil {
			return err
		}

		return helper.Success(c, fiber.StatusCreated, "Mata kuliah berhasil ditambahkan ke KRS", resp)
	})

	// =========================================================================
	// ENDPOINT 10: DELETE /api/v1/enrollments/{id} (Mahasiswa milik sendiri)
	// Membatalkan mata kuliah dari KRS (kuota kembali bertambah)
	// =========================================================================
	authGroup.Delete("/enrollments/:id", middleware.RequireRole(model.RoleMahasiswa), func(c *fiber.Ctx) error {
		id, err := helper.ParamID(c, "id")
		if err != nil {
			return err
		}

		user, err := helper.CurrentUser(c)
		if err != nil {
			return err
		}

		ctx, cancel := helper.RequestContext(c)
		defer cancel()

		if err := enrollService.Delete(ctx, user, id); err != nil {
			return err
		}

		return helper.NoContent(c)
	})
}
