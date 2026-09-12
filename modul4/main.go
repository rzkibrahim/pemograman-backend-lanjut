// Package main adalah titik masuk (entry point) aplikasi API Students.
//
// Setelah refactoring Clean Architecture, main.go hanya bertugas sebagai
// orchestrator perakitan dependensi (dependency assembly) dengan urutan:
//
//	config.LoadEnv â†’ config.NewLogger â†’ database.NewPool â†’
//	repository.NewStudentRepository â†’ service.NewStudentService â†’
//	config.NewApp â†’ app.Listen + graceful shutdown
//
// Tidak ada business logic, routing, atau middleware di file ini.
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

	"modul4/app/repository"
	"modul4/app/service"
	"modul4/config"
	"modul4/database"
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

	// 5. Lapisan Service
	studentService := service.NewStudentService(studentRepo)

	// 6. Perakitan Aplikasi Fiber (middleware, route, handler error terpusat)
	app := config.NewApp(appLogger, pool, studentService)

	port := config.GetEnv("APP_PORT", "3000")

	// 7. Menjalankan server pada goroutine terpisah
	go func() {
		appLogger.Info("server dimulai",
			slog.String("port", port),
			slog.String("url", fmt.Sprintf("http://localhost:%s", port)),
		)

		if err := app.Listen(":" + port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			appLogger.Error("server berhenti dengan kesalahan", slog.String("error", err.Error()))
		}
	}()

	// 8. Menunggu sinyal sistem untuk graceful shutdown (SIGINT / Ctrl+C, SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	appLogger.Info("sinyal penghentian diterima, memulai graceful shutdown...")

	// Memberi tenggat waktu 5 detik untuk menyelesaikan request yang sedang berjalan
	if err := app.ShutdownWithTimeout(5 * time.Second); err != nil {
		appLogger.Error("gagal mematikan server secara graceful", slog.String("error", err.Error()))
	} else {
		appLogger.Info("server berhasil dimatikan dengan aman")
	}
}
