package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"latihan-fiber/app/model"
)

// NotificationRepository mendefinisikan antarmuka akses data untuk tabel notifications.
type NotificationRepository interface {
	WithTx(tx DBTX) NotificationRepository

	// Claim mencatat bahwa notifikasi untuk sebuah event akan dikirim.
	// Mengembalikan false bila event itu sudah pernah tercatat (idempoten).
	Claim(ctx context.Context, n model.Notification) (bool, error)

	FindByUser(ctx context.Context, userID, limit int) ([]model.Notification, error)
	Count(ctx context.Context) (int, error)
}

type notificationPostgresRepository struct {
	db DBTX
}

// NewNotificationRepository membuat instance baru NotificationRepository.
func NewNotificationRepository(db DBTX) NotificationRepository {
	return &notificationPostgresRepository{db: db}
}

// WithTx mengembalikan instance NotificationRepository dalam transaksi DBTX.
func (r *notificationPostgresRepository) WithTx(tx DBTX) NotificationRepository {
	return &notificationPostgresRepository{db: tx}
}

// Claim memakai ON CONFLICT DO NOTHING untuk menjamin idempotency.
// Pola ini mencegah race condition jika ada pengiriman pesan duplikat (at-least-once delivery).
//
// Dengan ON CONFLICT DO NOTHING, jika notifikasi untuk outbox_id (atau outbox_id + user_id)
// sudah ada, kueri tidak akan menyisipkan baris baru dan RETURNING id menghasilkan pgx.ErrNoRows.
func (r *notificationPostgresRepository) Claim(ctx context.Context, n model.Notification) (bool, error) {
	var id int64
	err := r.db.QueryRow(ctx, `
		INSERT INTO notifications (outbox_id, user_id, channel, subject, body)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT DO NOTHING
		RETURNING id`,
		n.OutboxID, n.UserID, n.Channel, n.Subject, n.Body).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("mencatat notifikasi: %w", err)
	}
	return true, nil
}

// FindByUser mengambil daftar riwayat notifikasi milik mahasiswa tertentu,
// diurutkan berdasarkan outbox_id DESC (sesuai urutan terjadinya event transaksi).
func (r *notificationPostgresRepository) FindByUser(ctx context.Context, userID, limit int) ([]model.Notification, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, outbox_id, user_id, channel, subject, body, sent_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY outbox_id DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("mengambil notifikasi: %w", err)
	}
	defer rows.Close()

	result := []model.Notification{}
	for rows.Next() {
		var n model.Notification
		if err := rows.Scan(&n.ID, &n.OutboxID, &n.UserID, &n.Channel,
			&n.Subject, &n.Body, &n.SentAt); err != nil {
			return nil, fmt.Errorf("membaca row notifikasi: %w", err)
		}
		result = append(result, n)
	}
	return result, rows.Err()
}

// Count menghitung total seluruh notifikasi yang telah tercatat di sistem.
func (r *notificationPostgresRepository) Count(ctx context.Context) (int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications`).Scan(&total); err != nil {
		return 0, fmt.Errorf("menghitung notifikasi: %w", err)
	}
	return total, nil
}

