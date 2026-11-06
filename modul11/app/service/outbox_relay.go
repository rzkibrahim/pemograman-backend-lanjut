package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"

	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
)

// TaskEnqueuer adalah kemampuan melakukan enqueue task ke broker Asynq.
// *asynq.Client memenuhi interface ini.
type TaskEnqueuer interface {
	EnqueueContext(ctx context.Context, task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

// OutboxRelay memindahkan event dari tabel outbox ke queue Redis secara berkala.
//
// Relay adalah satu-satunya komponen yang membaca outbox. API tidak pernah
// berbicara dengan Redis; ia cukup menulis ke PostgreSQL. Karena itu
// mahasiswa tetap dapat mengisi KRS walaupun Redis sedang mati — notifikasinya
// hanya tertunda, tidak hilang.
type OutboxRelay struct {
	outbox   repository.OutboxRepository
	tx       *repository.TxManager
	queue    TaskEnqueuer
	logger   *slog.Logger
	interval time.Duration
	batch    int
}

// NewOutboxRelay membuat instance OutboxRelay baru.
func NewOutboxRelay(
	outbox repository.OutboxRepository,
	tx *repository.TxManager,
	queue TaskEnqueuer,
	logger *slog.Logger,
	interval time.Duration,
	batch int,
) *OutboxRelay {
	return &OutboxRelay{
		outbox:   outbox,
		tx:       tx,
		queue:    queue,
		logger:   logger,
		interval: interval,
		batch:    batch,
	}
}

// Run menjalankan polling loop relay outbox sampai context dibatalkan (graceful shutdown).
func (r *OutboxRelay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		n, err := r.RelayOnce(ctx)
		if err != nil {
			r.logger.Warn("relay_delayed",
				slog.Int("published", n), slog.String("error", err.Error()))
		} else if n > 0 {
			r.logger.Info("relay_done", slog.Int("published", n))
		}

		// Satu batch penuh berarti kemungkinan besar masih ada sisa baris di outbox:
		// langsung lanjutkan ke batch berikutnya tanpa menunggu ticker.
		if err == nil && n == r.batch && ctx.Err() == nil {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// RelayOnce mem-publish paling banyak satu batch baris outbox ke queue Redis.
// Mengembalikan jumlah event yang berhasil di-publish dan error jika terjadi kendala.
func (r *OutboxRelay) RelayOnce(ctx context.Context) (int, error) {
	var published int
	var enqueueErr error

	err := r.tx.WithTxOnce(ctx, func(tx repository.DBTX) error {
		repo := r.outbox.WithTx(tx)
		events, err := repo.FetchUnpublished(ctx, r.batch)
		if err != nil {
			return err
		}

		ids := make([]int64, 0, len(events))
		for _, e := range events {
			// Mencoba publish ke broker Redis
			if err := r.publish(ctx, e); err != nil {
				enqueueErr = err
				// Berhenti pada kegagalan pertama (misal Redis mati).
				// Event yang gagal TIDAK dimasukkan ke ids sehingga tidak ditandai published_at.
				break
			}
			// Perbaikan Puzzle 4: e.ID HANYA di-append ke ids SETELAH r.publish berhasil!
			// Pada kode modul asli, ids di-append SEBELUM publish sehingga baris yang gagal
			// di-enqueue ke Redis justru ditandai sebagai sudah ter-publish dan hilang selamanya.
			ids = append(ids, e.ID)
		}

		// Hanya event yang SUDAH benar-benar berhasil di-enqueue yang ditandai published_at = NOW()
		if len(ids) > 0 {
			if err := repo.MarkPublished(ctx, ids); err != nil {
				return err
			}
		}
		published = len(ids)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return published, enqueueErr
}

// publish menyusun struct NotificationJob, melakukan serialisasi JSON, dan mengirim task ke Asynq.
func (r *OutboxRelay) publish(ctx context.Context, e model.OutboxEvent) error {
	job := model.NotificationJob{
		OutboxID:  e.ID,
		EventType: e.EventType,
		Payload:   e.Payload,
	}

	// Perbaikan Puzzle 5: Asynq NewTask membutuhkan payload berupa []byte, bukan struct model.NotificationJob.
	// Kita harus memanggil json.Marshal(job) terlebih dahulu.
	payloadBytes, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("serialize outbox %d payload: %w", e.ID, err)
	}

	task := asynq.NewTask(model.TaskSendNotification, payloadBytes,
		// TaskID yang diturunkan dari id outbox membuat enqueue ulang
		// event yang sama ditolak Redis (ErrTaskIDConflict). Ini menutup celah ketika relay
		// mati SETELAH melakukan enqueue tetapi SEBELUM sempat commit MarkPublished di PostgreSQL.
		asynq.TaskID(fmt.Sprintf("outbox-%d", e.ID)),
		asynq.Queue(model.QueueNotifications),
		asynq.MaxRetry(5),
		asynq.Timeout(30*time.Second),
		// Task yang selesai disimpan selama masa Retention (24 jam)
		// agar riwayat TaskID tetap tercatat di Redis untuk pencegahan duplikasi.
		asynq.Retention(24*time.Hour),
	)

	_, err = r.queue.EnqueueContext(ctx, task)
	if errors.Is(err, asynq.ErrTaskIDConflict) {
		// Sudah pernah di-enqueue pada putaran sebelumnya. Bagi relay, itu dianggap berhasil.
		return nil
	}
	if err != nil {
		return fmt.Errorf("enqueue outbox %d: %w", e.ID, err)
	}
	return nil
}

