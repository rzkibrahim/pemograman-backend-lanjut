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

	"modul9/app/repository"
	"modul9/app/service"
	"modul9/config"
	"modul9/database"
	"modul9/helper"
)

func main() {
	config.LoadEnv()

	appLogger := config.NewLogger()

	pool, err := database.NewPool(context.Background())
	if err != nil {
		appLogger.Error("koneksi database gagal", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	studentRepo := repository.NewStudentRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	tokenRepo := repository.NewTokenRepository(pool)
	roleRepo := repository.NewRoleRepository(pool)

	// Inisialisasi TxManager, CourseRepository, dan SupervisorRepository (Modul 9)
	txManager := repository.NewTxManager(pool)
	courseRepo := repository.NewCourseRepository(pool)
	supervisorRepo := repository.NewSupervisorRepository(pool)

	rolePermsMapping, err := roleRepo.GetRolePermissions(context.Background())
	if err != nil {
		appLogger.Error("gagal memuat skema hak akses RBAC dari database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	permissionSet := helper.NewPermissionSet(rolePermsMapping)
	appLogger.Info("skema RBAC berhasil dimuat ke memori (fail-closed)",
		slog.Int("roles_configured", len(rolePermsMapping)),
	)

	jwtSecret := config.GetEnv("JWT_SECRET", "")
	if len(jwtSecret) < 32 {
		appLogger.Error("konfigurasi keamanan gagal: JWT_SECRET wajib diisi di .env dengan panjang minimal 32 karakter acak")
		os.Exit(1)
	}

	jwtIssuer := config.GetEnv("JWT_ISSUER", "praktikum-backend")
	accessTTL := time.Duration(config.GetEnvInt("ACCESS_TOKEN_TTL_MINUTES", 15)) * time.Minute
	refreshTTL := time.Duration(config.GetEnvInt("REFRESH_TOKEN_TTL_DAYS", 7)) * 24 * time.Hour

	studentService := service.NewStudentService(studentRepo, permissionSet)
	userService := service.NewUserService(userRepo, permissionSet)
	authService := service.NewAuthService(userRepo, tokenRepo, jwtSecret, jwtIssuer, accessTTL, refreshTTL)
	courseService := service.NewCourseService(courseRepo, txManager)
	supervisorService := service.NewSupervisorService(supervisorRepo, txManager)

	app := config.NewApp(appLogger, pool, studentService, userService, authService, courseService, supervisorService, permissionSet, jwtSecret)

	port := config.GetEnv("APP_PORT", "3000")

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
