// Package main adalah titik masuk (entry point) aplikasi API Students.
//
// Pada Modul 6 (Authorization & RBAC), main.go memuat pemetaan role-permission
// dari basis data ke dalam memori (PermissionSet) sekali saat startup,
// lalu menyuntikkannya ke middleware dan service layer untuk evaluasi hak akses fail-closed.
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

	"modul6/app/repository"
	"modul6/app/service"
	"modul6/config"
	"modul6/database"
	"modul6/helper"
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
	roleRepo := repository.NewRoleRepository(pool)

	// 5. Muat Skema RBAC ke Memori (Langkah 4 Panduan)
	// Query tabel role_permissions sekali saja saat startup untuk performa maksimal O(1)
	rolePermsMapping, err := roleRepo.GetRolePermissions(context.Background())
	if err != nil {
		appLogger.Error("gagal memuat skema hak akses RBAC dari database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	permissionSet := helper.NewPermissionSet(rolePermsMapping)
	appLogger.Info("skema RBAC berhasil dimuat ke memori (fail-closed)",
		slog.Int("roles_configured", len(rolePermsMapping)),
	)

	// 6. Konfigurasi Autentikasi JWT (Validasi ketat: wajib min 32 karakter acak)
	jwtSecret := config.GetEnv("JWT_SECRET", "")
	if len(jwtSecret) < 32 {
		appLogger.Error("konfigurasi keamanan gagal: JWT_SECRET wajib diisi di .env dengan panjang minimal 32 karakter acak")
		os.Exit(1)
	}

	jwtIssuer := config.GetEnv("JWT_ISSUER", "praktikum-backend")
	accessTTL := time.Duration(config.GetEnvInt("ACCESS_TOKEN_TTL_MINUTES", 15)) * time.Minute
	refreshTTL := time.Duration(config.GetEnvInt("REFRESH_TOKEN_TTL_DAYS", 7)) * 24 * time.Hour

	// 7. Lapisan Service (dengan injeksi PermissionSet)
	studentService := service.NewStudentService(studentRepo, permissionSet)
	userService := service.NewUserService(userRepo, permissionSet)
	authService := service.NewAuthService(userRepo, tokenRepo, jwtSecret, jwtIssuer, accessTTL, refreshTTL)

	// 8. Perakitan Aplikasi Fiber (middleware, route, handler error terpusat)
	app := config.NewApp(appLogger, pool, studentService, userService, authService, permissionSet, jwtSecret)

	port := config.GetEnv("APP_PORT", "3000")

	// 9. Menjalankan server pada goroutine terpisah
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

	// 10. Graceful Shutdown
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
