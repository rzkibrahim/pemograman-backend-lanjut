package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TxManager mengelola siklus hidup transaction dengan pola defer Rollback
// dan retry otomatis pada error serialization_failure (40001) dan deadlock (40P01).
type TxManager struct {
	pool       *pgxpool.Pool
	maxRetries int
}

// NewTxManager membuat instance TxManager dengan pool dan batas retry.
func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool, maxRetries: 3}
}

// WithTx menjalankan fn di dalam transaction READ COMMITTED.
// Jika fn mengembalikan error dengan kode 40001 atau 40P01,
// transaction akan di-rollback dan fn dijalankan ulang hingga maxRetries.
func (tm *TxManager) WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	return tm.withTxOptions(ctx, pgx.TxOptions{}, fn)
}

// WithTxRepeatableRead menjalankan fn di dalam transaction REPEATABLE READ.
func (tm *TxManager) WithTxRepeatableRead(ctx context.Context, fn func(tx pgx.Tx) error) error {
	return tm.withTxOptions(ctx, pgx.TxOptions{
		IsoLevel: pgx.RepeatableRead,
	}, fn)
}

func (tm *TxManager) withTxOptions(ctx context.Context, opts pgx.TxOptions, fn func(tx pgx.Tx) error) error {
	var lastErr error

	for attempt := 1; attempt <= tm.maxRetries; attempt++ {
		tx, err := tm.pool.BeginTx(ctx, opts)
		if err != nil {
			return fmt.Errorf("memulai transaction: %w", err)
		}

		// Pola defer Rollback: jika fn berhasil dan tx sudah di-commit,
		// Rollback menjadi no-op. Jika fn gagal, Rollback memastikan
		// transaction tidak menggantung (idle in transaction).
		defer func() {
			_ = tx.Rollback(ctx)
		}()

		err = fn(tx)
		if err == nil {
			if commitErr := tx.Commit(ctx); commitErr != nil {
				if isRetryable(commitErr) && attempt < tm.maxRetries {
					slog.Warn("commit gagal, retrying",
						slog.Int("attempt", attempt),
						slog.String("error", commitErr.Error()))
					lastErr = commitErr
					continue
				}
				return fmt.Errorf("commit transaction: %w", commitErr)
			}
			return nil
		}

		// Rollback eksplisit pada error
		_ = tx.Rollback(ctx)

		if isRetryable(err) && attempt < tm.maxRetries {
			slog.Warn("transaction gagal (retryable), retrying",
				slog.Int("attempt", attempt),
				slog.String("error", err.Error()))
			lastErr = err
			continue
		}

		return err
	}

	return fmt.Errorf("transaction gagal setelah %d percobaan: %w", tm.maxRetries, lastErr)
}

// isRetryable memeriksa apakah error PostgreSQL memiliki kode 40001 (serialization_failure)
// atau 40P01 (deadlock_detected), yang dapat di-retry secara aman.
func isRetryable(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "40001" || pgErr.Code == "40P01"
	}
	return false
}
