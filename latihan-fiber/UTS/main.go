// Package main - main.go
// Titik masuk utama (entry point) aplikasi SIAKAD Mini API.
// Memuat konfigurasi environment, menginisialisasi database pool, merakit service,
// dan menjalankan server web Fiber dengan dukungan graceful shutdown.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"siakad-mini/app/repository"
	"siakad-mini/app/service"
	"siakad-mini/config"
	"siakad-mini/database"
)

func main() {
	// 1. Muat environment dari file .env
	config.LoadEnv()

	// 2. Inisialisasi structured logger (slog) ke terminal dan file berotasi
	logger := config.NewLogger()
	logger.Info("memulai server SIAKAD Mini...")

	// 3. Inisialisasi pool koneksi database PostgreSQL (pgxpool)
	pool, err := database.NewPostgresPool(logger)
	if err != nil {
		logger.Error("gagal menghubungkan ke database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	// 4. Inisialisasi layer Repository
	userRepo := repository.NewUserRepository(pool)
	studentRepo := repository.NewStudentRepository(pool)
	courseRepo := repository.NewCourseRepository(pool)
	enrollmentRepo := repository.NewEnrollmentRepository(pool)

	// 5. Inisialisasi konfigurasi autentikasi JWT
	jwtSecret := config.GetEnv("JWT_SECRET", "super-secret-key-at-least-32-bytes-long")
	jwtIssuer := config.GetEnv("JWT_ISSUER", "siakad-mini-uts")
	tokenTTL := config.GetEnvInt("ACCESS_TOKEN_TTL_MINUTES", 60)

	// 6. Inisialisasi layer Service
	authService := service.NewAuthService(userRepo, studentRepo, jwtSecret, jwtIssuer, tokenTTL)
	studentService := service.NewStudentService(pool, userRepo, studentRepo, enrollmentRepo)
	courseService := service.NewCourseService(courseRepo)
	enrollService := service.NewEnrollmentService(pool, studentRepo, courseRepo, enrollmentRepo)

	// 7. Rakit aplikasi Fiber dengan middleware, routing, dan centralized error handler
	app := config.NewApp(
		logger,
		pool,
		authService,
		studentService,
		courseService,
		enrollService,
		jwtSecret,
	)

	// 8. Jalankan HTTP server pada goroutine terpisah
	port := config.GetEnv("APP_PORT", "3000")
	addr := fmt.Sprintf(":%s", port)

	go func() {
		logger.Info("server HTTP aktif", slog.String("alamat", "http://localhost"+addr))
		if err := app.Listen(addr); err != nil {
			logger.Info("server berhenti menerima koneksi baru", slog.String("alasan", err.Error()))
		}
	}()

	// 9. Penanganan sinyal OS untuk Graceful Shutdown (SIGINT, SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("menerima sinyal terminasi, mematikan server secara graceful...")

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := app.ShutdownWithContext(ctxShutdown); err != nil {
		logger.Error("terjadi kesalahan saat mematikan server", slog.String("error", err.Error()))
	}

	logger.Info("server SIAKAD Mini berhasil dihentikan dengan aman.")
}
