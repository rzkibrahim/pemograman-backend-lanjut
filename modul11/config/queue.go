package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/hibiken/asynq"

	"latihan-fiber/app/model"
)

// RedisOpt membaca parameter konfigurasi koneksi Redis dari berkas .env.
// Objek asynq.RedisClientOpt yang sama digunakan bersama oleh client (relay),
// server (worker), dan inspector (endpoint monitoring admin).
func RedisOpt() asynq.RedisClientOpt {
	return asynq.RedisClientOpt{
		Addr:     GetEnv("REDIS_ADDR", "localhost:6379"),
		Password: GetEnv("REDIS_PASSWORD", ""),
		DB:       GetEnvInt("REDIS_DB", 0),
		// Timeout dibuat singkat (2 detik). Bila Redis mati/unreachable, kita ingin
		// segera mengetahuinya daripada menahan koneksi database yang ter-lock terlalu lama.
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	}
}

// NewWorkerServer menginisialisasi server Asynq pemroses background queue.
func NewWorkerServer(logger *slog.Logger) *asynq.Server {
	return asynq.NewServer(RedisOpt(), asynq.Config{
		// Jumlah task konkruen yang dikerjakan secara bersamaan.
		// Setiap task notifikasi dapat memegang satu koneksi PostgreSQL selama pengiriman,
		// sehingga batas ini harus disesuaikan lebih kecil dari DB_MAX_CONNS.
		Concurrency: GetEnvInt("WORKER_CONCURRENCY", 5),

		// Perbaikan Puzzle 8: Worker HANYA mengambil task dari antrean yang terdaftar di Queues!
		// Kode modul asli menulis Queues: map[string]int{model.TaskSendNotification: 1}, padahal
		// model.TaskSendNotification ("notification:send") adalah JENIS task (task type).
		// Nama queue sesungguhnya adalah model.QueueNotifications ("notifications").
		// Akibat kesalahan ini, worker tidak pernah mengambil task dari queue notifications.
		Queues: map[string]int{model.QueueNotifications: 1},

		// RetryDelayFunc menentukan jeda backoff sebelum melakukan percobaan ulang task.
		// n adalah jumlah retry yang SUDAH terjadi (mulai dari 0).
		// Jeda yang dihasilkan adalah: 2s, 4s, 6s, 8s, dan 10s (cocok untuk lingkungan pengujian praktikum).
		RetryDelayFunc: func(n int, err error, t *asynq.Task) time.Duration {
			return time.Duration((n+1)*2) * time.Second
		},

		// Saat proses worker menerima sinyal berhenti (Ctrl+C),
		// task yang sedang berjalan diberi tenggat waktu 10 detik untuk menyelesaikan eksekusinya (Graceful Shutdown).
		ShutdownTimeout: 10 * time.Second,

		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, t *asynq.Task, err error) {
			retry, _ := asynq.GetRetryCount(ctx)
			maxRetry, _ := asynq.GetMaxRetry(ctx)
			logger.Warn("task_failed",
				slog.String("type", t.Type()),
				slog.String("attempt", fmt.Sprintf("%d/%d", retry+1, maxRetry+1)),
				slog.String("error", err.Error()))
		}),

		Logger:   newAsynqLogger(logger),
		LogLevel: asynq.WarnLevel,
	})
}

// asynqLogger menjembatani logger internal Asynq ke slog bawaan aplikasi
// agar seluruh log terstandarisasi dalam format JSON terstruktur yang seragam.
type asynqLogger struct {
	l *slog.Logger
}

func newAsynqLogger(l *slog.Logger) asynqLogger {
	return asynqLogger{l: l.With(slog.String("component", "asynq"))}
}

func (a asynqLogger) Debug(args ...any) { a.l.Debug(fmt.Sprint(args...)) }
func (a asynqLogger) Info(args ...any)  { a.l.Info(fmt.Sprint(args...)) }
func (a asynqLogger) Warn(args ...any)  { a.l.Warn(fmt.Sprint(args...)) }
func (a asynqLogger) Error(args ...any) { a.l.Error(fmt.Sprint(args...)) }

// Fatal wajib menghentikan proses aplikasi sesuai kontrak antarmuka Logger Asynq.
func (a asynqLogger) Fatal(args ...any) {
	a.l.Error(fmt.Sprint(args...))
	os.Exit(1)
}

