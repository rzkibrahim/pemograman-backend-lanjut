// Package database mengelola koneksi dan pool basis data PostgreSQL
// menggunakan pgxpool untuk SIAKAD Mini.
package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"siakad-mini/config"
)

// NewPool membuat connection pool ke PostgreSQL.
// Pool ini mendukung koneksi konkurensi tinggi dengan konfigurasi
// max connections, idle time, dan lifetime yang optimal.
func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		config.GetEnv("DB_USER", "postgres"),
		config.GetEnv("DB_PASSWORD", ""),
		config.GetEnv("DB_HOST", "localhost"),
		config.GetEnv("DB_PORT", "5432"),
		config.GetEnv("DB_NAME", "siakad_mini"),
		config.GetEnv("DB_SSLMODE", "disable"),
	)

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("konfigurasi database tidak valid: %w", err)
	}

	cfg.MaxConns = int32(config.GetEnvInt("DB_MAX_CONNS", 10))
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat pool: %w", err)
	}

	// Ping memastikan kredensial benar dan server dapat dihubungi
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("gagal terhubung ke database: %w", err)
	}

	return pool, nil
}

// NewPostgresPool membuat pool koneksi PostgreSQL dan mencatat log keberhasilan.
func NewPostgresPool(logger *slog.Logger) (*pgxpool.Pool, error) {
	ctx := context.Background()
	pool, err := NewPool(ctx)
	if err != nil {
		return nil, err
	}
	if logger != nil {
		logger.Info("berhasil terhubung ke basis data PostgreSQL",
			slog.String("database", config.GetEnv("DB_NAME", "siakad_mini")),
			slog.String("host", config.GetEnv("DB_HOST", "localhost")),
		)
	}
	return pool, nil
}
