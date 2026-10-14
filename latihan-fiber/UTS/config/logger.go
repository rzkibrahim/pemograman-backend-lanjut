package config

import (
	"io"
	"log/slog"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// NewLogger menginisialisasi structured logger (slog) dengan output ganda:
// terminal (stdout) dan berkas log berotasi otomatis melalui lumberjack.
// Logger menggunakan format JSON agar mudah di-parse oleh tools monitoring.
func NewLogger() *slog.Logger {
	logFile := GetEnv("LOG_FILE", "logs/app.log")
	maxSize := GetEnvInt("LOG_MAX_SIZE", 10)
	maxBackups := GetEnvInt("LOG_MAX_BACKUPS", 5)
	maxAge := GetEnvInt("LOG_MAX_AGE", 30)

	fileWriter := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		Compress:   true,
	}

	// MultiWriter menulis ke terminal dan file secara simultan
	multiWriter := io.MultiWriter(os.Stdout, fileWriter)

	handler := slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	return slog.New(handler)
}
