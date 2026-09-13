// Package main adalah titik masuk (entry point) aplikasi API Students.
//
// Setelah refactoring Clean Architecture, main.go hanya bertugas sebagai
// orchestrator perakitan dependensi (dependency assembly) dengan urutan:
//
//	config.LoadEnv → config.NewLogger → database.NewPool →
//	repository.NewStudentRepository / NewUserRepository / NewTokenRepository →
//	service.NewStudentService / NewAuthService →
//	config.NewApp → app.Listen + graceful shutdown
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"modul5/app/repository"
	"modul5/app/service"
	"modul5/config"
	"modul5/database"
)

func main() {
	// 1. Konfigurasi environment
	config.LoadEnv()

	// 2. Structured Logger dengan rotasi file otomatis (lumberjack)
	appLogger := config.NewLogger()

	// 3. Basis data: connection pool PostgreSQL
	pool, err := database.NewPool(context.Background())
	if err != nil {
		appLogger.Error("koneksi database gagal", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	// 4. Lapisan Repository
	studentRepo := repository.NewStudentRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	tokenRepo := repository.NewTokenRepository(pool)

	// 5. Konfigurasi Autentikasi JWT
	jwtSecret := config.GetEnv("JWT_SECRET", "super-secret-key-that-is-at-least-32-bytes-long!")
	jwtIssuer := config.GetEnv("JWT_ISSUER", "praktikum-backend")
	accessTTL := time.Duration(config.GetEnvInt("ACCESS_TOKEN_TTL_MINUTES", 15)) * time.Minute
	refreshTTL := time.Duration(config.GetEnvInt("REFRESH_TOKEN_TTL_DAYS", 7)) * 24 * time.Hour

	// 6. Lapisan Service
	studentService := service.NewStudentService(studentRepo)
	authService := service.NewAuthService(userRepo, tokenRepo, jwtSecret, jwtIssuer, accessTTL, refreshTTL)

	// 7. Perakitan Aplikasi Fiber (middleware, route, handler error terpusat)
	app := config.NewApp(appLogger, pool, studentService, authService, jwtSecret)

	port := config.GetEnv("APP_PORT", "3000")

	// 8. Menjalankan server pada goroutine terpisah
	go func() {
		appLogger.Info("server dimulai",
			slog.String("port", port),
			slog.String("url", fmt.Sprintf("http://localhost:%s", port)),
		)

		if err := app.Listen(":" + port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			appLogger.Error("server berhenti tidak normal", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// 9. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	appLogger.Info("menerima sinyal penghentian, mematikan server dengan anggun...",
		slog.String("signal", sig.String()),
	)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		appLogger.Error("gagal mematikan server dengan anggun", slog.String("error", err.Error()))
		os.Exit(1)
	}

	appLogger.Info("server berhasil dimatikan dengan bersih")
}
