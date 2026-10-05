package config

import (
	"io"
	"log/slog"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// NewLogger menginisialisasi structured logger (slog) dengan output ganda:
// terminal (stdout) dan berkas log berotasi otomatis (logs/app.log) melalui lumberjack.
func NewLogger() *slog.Logger {
	logFile := GetEnv("LOG_FILE", "logs/app.log")
	maxSize := GetEnvInt("LOG_MAX_SIZE", 10)       // batas ukuran per berkas (MB)
	maxBackups := GetEnvInt("LOG_MAX_BACKUPS", 5) // jumlah cadangan berkas lama
	maxAge := GetEnvInt("LOG_MAX_AGE", 30)         // masa retensi berkas (hari)

	fileWriter := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		Compress:   true,
	}

	// Menulis ke terminal dan file secara simultan
	multiWriter := io.MultiWriter(os.Stdout, fileWriter)

	handler := slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	return slog.New(handler)
}

