package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
	"latihan-fiber/helper"
)

// NotificationWorker mengeksekusi task asynchronous "notification:send" dari queue broker Asynq.
type NotificationWorker struct {
	users         repository.UserRepository
	notifications repository.NotificationRepository
	tx            *repository.TxManager
	mailer        helper.Mailer
	logger        *slog.Logger
}

// NewNotificationWorker membuat instance baru NotificationWorker.
func NewNotificationWorker(
	users repository.UserRepository,
	notifications repository.NotificationRepository,
	tx *repository.TxManager,
	mailer helper.Mailer,
	logger *slog.Logger,
) *NotificationWorker {
	return &NotificationWorker{
		users:         users,
		notifications: notifications,
		tx:            tx,
		mailer:        mailer,
		logger:        logger,
	}
}

// HandleSend memproses task pengiriman notifikasi.
//
// Perbaikan Puzzle 6: Signature handler Asynq WAJIB menerima pointer *asynq.Task
// (func(ctx context.Context, t *asynq.Task) error), bukan struct value (asynq.Task).
// Menggunakan value menyebabkan kompilasi gagal saat didaftarkan ke mux.HandleFunc.
func (w *NotificationWorker) HandleSend(ctx context.Context, t *asynq.Task) error {
	var job model.NotificationJob
	if err := json.Unmarshal(t.Payload(), &job); err != nil {
		return fmt.Errorf("%w: payload task rusak: %v", asynq.SkipRetry, err)
	}

	// Penanganan event perubahan kuota kelas (Tugas Mandiri D.2)
	if job.EventType == model.EventCourseQuotaChanged {
		return w.handleQuotaChanged(ctx, job)
	}

	var event model.EnrollmentEvent
	if err := json.Unmarshal(job.Payload, &event); err != nil {
		return fmt.Errorf("%w: payload event %d rusak: %v", asynq.SkipRetry, job.OutboxID, err)
	}

	user, err := w.users.FindByID(ctx, event.UserID)
	if errors.Is(err, repository.ErrNotFound) {
		// Akunnya sudah dihapus; permanent failure -> langsung skip retry
		return fmt.Errorf("%w: user %d tidak ditemukan", asynq.SkipRetry, event.UserID)
	}
	if err != nil {
		return err // Gangguan database -> temporary failure, layak di-retry
	}

	msg, err := BuildMessage(job.EventType, event, user)
	if err != nil {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}

	alreadySent := false
	err = w.tx.WithTxOnce(ctx, func(tx repository.DBTX) error {
		// Perbaikan Puzzle 9: Menggunakan w.notifications.WithTx(tx).Claim(...) BUKAN w.notifications.Claim(...)!
		// Pada kode modul asli, Claim dipanggil tanpa WithTx(tx) sehingga commit secara mandiri di luar transaksi.
		// Akibatnya, jika pengiriman email (mailer.Send) gagal, claim tidak ikut ter-rollback.
		// Ketika task di-retry kemudian, claim melihat sudah ada baris notifikasi dan email tidak pernah dikirim ulang!
		claimed, err := w.notifications.WithTx(tx).Claim(ctx, model.Notification{
			OutboxID: job.OutboxID,
			UserID:   user.ID,
			Channel:  "email",
			Subject:  msg.Subject,
			Body:     msg.Body,
		})
		if err != nil {
			return err
		}
		if !claimed {
			alreadySent = true
			return nil
		}
		return w.mailer.Send(ctx, msg)
	})

	// Perbaikan Puzzle 7: asynq.SkipRetry WAJIB dibungkus dengan verb %w, BUKAN %v!
	// Kode modul asli menulis fmt.Errorf("%v: %w", asynq.SkipRetry, err) yang mana %v membuat
	// errors.Is(err, asynq.SkipRetry) bernilai false. Akibatnya Asynq menganggapnya temporary failure
	// dan tetap melakukan retry sampai 5 kali alih-alih langsung diarsipkan ke archived.
	if errors.Is(err, helper.ErrPermanentDelivery) {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}
	if err != nil {
		return err
	}

	retry, _ := asynq.GetRetryCount(ctx)
	w.logger.Info("notification_processed",
		slog.Int64("outbox_id", job.OutboxID),
		slog.String("event", job.EventType),
		slog.Int("retry", retry),
		slog.Bool("already_sent", alreadySent))
	return nil
}

// handleQuotaChanged menangani pengiriman notifikasi broadcast perubahan kuota untuk Tugas Mandiri D.2.
// Setiap mahasiswa diproses secara independen dalam sub-transaksi WithTxOnce masing-masing.
// Jika email ke-7 gagal, mahasiswa 1-6 telah ter-commit dan retry Asynq tidak akan mengirim email ganda kepada mereka.
func (w *NotificationWorker) handleQuotaChanged(ctx context.Context, job model.NotificationJob) error {
	var event model.CourseQuotaChangedEvent
	if err := json.Unmarshal(job.Payload, &event); err != nil {
		return fmt.Errorf("%w: payload kuota rusak: %v", asynq.SkipRetry, err)
	}

	var firstErr error
	var sentCount int

	for _, userID := range event.UserIDs {
		user, err := w.users.FindByID(ctx, userID)
		if err != nil {
			continue
		}

		msg := BuildQuotaChangedMessage(event, user)
		alreadySent := false

		txErr := w.tx.WithTxOnce(ctx, func(tx repository.DBTX) error {
			claimed, err := w.notifications.WithTx(tx).Claim(ctx, model.Notification{
				OutboxID: job.OutboxID,
				UserID:   user.ID,
				Channel:  "email",
				Subject:  msg.Subject,
				Body:     msg.Body,
			})
			if err != nil {
				return err
			}
			if !claimed {
				alreadySent = true
				return nil
			}
			return w.mailer.Send(ctx, msg)
		})

		if txErr != nil {
			if errors.Is(txErr, helper.ErrPermanentDelivery) {
				w.logger.Warn("permanent_failure_quota_email", slog.Int("user_id", userID), slog.String("error", txErr.Error()))
				continue
			}
			// Catat error pertama untuk memicu retry Asynq bagi penerima yang belum sukses
			if firstErr == nil {
				firstErr = txErr
			}
			break
		}

		if !alreadySent {
			sentCount++
		}
	}

	if firstErr != nil {
		return firstErr
	}

	retry, _ := asynq.GetRetryCount(ctx)
	w.logger.Info("notification_processed",
		slog.Int64("outbox_id", job.OutboxID),
		slog.String("event", job.EventType),
		slog.Int("retry", retry),
		slog.Int("recipients_sent", sentCount),
		slog.Bool("already_sent", sentCount == 0))
	return nil
}
