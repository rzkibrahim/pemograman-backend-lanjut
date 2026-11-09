package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"latihan-fiber/app/model"
)

// OutboxRepository mendefinisikan antarmuka akses data untuk tabel outbox (Transactional Outbox Pattern).
type OutboxRepository interface {
	WithTx(tx DBTX) OutboxRepository

	// Insert WAJIB dipanggil lewat repository hasil WithTx, di dalam
	// transaction yang sama dengan perubahan data yang dicatatnya.
	Insert(ctx context.Context, eventType string, event any) error

	// FetchUnpublished mengunci baris yang diambilnya sampai transaction
	// selesai. Karena itu ia hanya bermakna bila dipanggil di dalam transaction.
	FetchUnpublished(ctx context.Context, limit int) ([]model.OutboxEvent, error)

	// MarkPublished menandai baris outbox yang sudah berhasil di-enqueue ke broker Redis.
	MarkPublished(ctx context.Context, ids []int64) error

	// Stats menghitung statistik antrean outbox untuk pemantauan sistem.
	Stats(ctx context.Context) (pending, published int, oldest time.Duration, err error)
}

type outboxPostgresRepository struct {
	db DBTX
}

// NewOutboxRepository membuat instance baru OutboxRepository berbasis PostgreSQL.
func NewOutboxRepository(db DBTX) OutboxRepository {
	return &outboxPostgresRepository{db: db}
}

// WithTx mengembalikan instance OutboxRepository yang berjalan di dalam transaksi database (DBTX).
func (r *outboxPostgresRepository) WithTx(tx DBTX) OutboxRepository {
	return &outboxPostgresRepository{db: tx}
}

// Insert menyisipkan baris event baru ke dalam tabel outbox dalam format JSONB.
func (r *outboxPostgresRepository) Insert(ctx context.Context, eventType string, event any) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("menyusun payload outbox: %w", err)
	}
	_, err = r.db.Exec(ctx,
		`INSERT INTO outbox (event_type, payload) VALUES ($1, $2)`,
		eventType, payload)
	if err != nil {
		return fmt.Errorf("mencatat outbox: %w", err)
	}
	return nil
}

// FetchUnpublished mengambil event yang belum di-publish dengan mekanisme konkurensi:
//   - FOR UPDATE   : baris yang diambil di-lock agar relay lain tidak mengambil baris yang sama.
//   - SKIP LOCKED  : baris yang sedang di-lock oleh proses relay lain dilewati tanpa menunggu (non-blocking).
//     Hal ini memungkinkan multiple worker/relay membagi beban outbox secara independen tanpa saling berebut.
func (r *outboxPostgresRepository) FetchUnpublished(ctx context.Context, limit int) ([]model.OutboxEvent, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, event_type, payload, created_at
		FROM outbox
		WHERE published_at IS NULL
		ORDER BY id
		LIMIT $1
		FOR UPDATE SKIP LOCKED`, limit)
	if err != nil {
		return nil, fmt.Errorf("mengambil outbox: %w", err)
	}
	defer rows.Close()

	events := []model.OutboxEvent{}
	for rows.Next() {
		var e model.OutboxEvent
		if err := rows.Scan(&e.ID, &e.EventType, &e.Payload, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("membaca row outbox: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// MarkPublished memperbarui kolom published_at menjadi waktu saat ini untuk baris outbox yang telah di-enqueue.
func (r *outboxPostgresRepository) MarkPublished(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.db.Exec(ctx,
		`UPDATE outbox SET published_at = NOW() WHERE id = ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("menandai outbox: %w", err)
	}
	return nil
}

// Stats menghitung jumlah pesan yang belum di-publish (pending), yang sudah di-publish,
// serta durasi usia pesan tertua yang masih tertahan di antrean outbox.
func (r *outboxPostgresRepository) Stats(ctx context.Context) (int, int, time.Duration, error) {
	var pending, published int
	var oldestSeconds float64
	err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE published_at IS NULL),
			COUNT(*) FILTER (WHERE published_at IS NOT NULL),
			COALESCE(EXTRACT(EPOCH FROM NOW() - MIN(created_at) FILTER (WHERE published_at IS NULL)), 0)
		FROM outbox`).Scan(&pending, &published, &oldestSeconds)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("menghitung outbox: %w", err)
	}
	return pending, published, time.Duration(oldestSeconds * float64(time.Second)), nil
}

