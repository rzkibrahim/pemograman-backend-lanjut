package service

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"

	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
	"latihan-fiber/helper"
)

// QueueInspector adalah kemampuan membaca keadaan metrik queue Asynq.
// *asynq.Inspector memenuhi antarmuka ini.
type QueueInspector interface {
	GetQueueInfo(queue string) (*asynq.QueueInfo, error)
}

// NotificationService melayani endpoint-endpoint HTTP seputar notifikasi dan antrean sistem.
type NotificationService struct {
	notifications repository.NotificationRepository
	outbox        repository.OutboxRepository
	inspector     QueueInspector
}

// NewNotificationService membuat instance baru NotificationService.
func NewNotificationService(
	notifications repository.NotificationRepository,
	outbox repository.OutboxRepository,
	inspector QueueInspector,
) *NotificationService {
	return &NotificationService{
		notifications: notifications,
		outbox:        outbox,
		inspector:     inspector,
	}
}

// Mine godoc
// @Summary      Riwayat notifikasi user
// @Description  Mengambil riwayat notifikasi milik user yang sedang login
// @Tags         notifications
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} model.WebResponse{data=[]model.Notification}
// @Failure      401 {object} model.ErrorResponse
// @Router       /notifications/mine [get]
func (s *NotificationService) Mine(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	list, err := s.notifications.FindByUser(ctx, current.UserID, 20)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "notifikasi berhasil diambil", list)
}

// Monitor godoc
// @Summary      Status antrean queue dan outbox (admin)
// @Description  Memantau keadaan antrean broker Asynq dan outbox yang belum terkirim
// @Tags         admin
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} model.WebResponse{data=model.QueueStats}
// @Failure      401 {object} model.ErrorResponse
// @Failure      403 {object} model.ErrorResponse
// @Router       /admin/queue [get]
func (s *NotificationService) Monitor(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	pending, published, oldest, err := s.outbox.Stats(ctx)
	if err != nil {
		return helper.Internal(err)
	}
	sent, err := s.notifications.Count(ctx)
	if err != nil {
		return helper.Internal(err)
	}

	stats := model.QueueStats{
		OutboxPending:     pending,
		OutboxPublished:   published,
		OldestPendingAge:  oldest.Round(time.Millisecond).String(),
		NotificationsSent: sent,
	}

	// Kegagalan koneksi Redis tidak boleh menggagalkan pembacaan metrik outbox di PostgreSQL.
	// Saat Redis padam, administrator justru paling membutuhkan data outbox_pending untuk mengetahui beban yang tertahan.
	info, err := s.inspector.GetQueueInfo(model.QueueNotifications)
	if err != nil {
		stats.QueueError = "queue tidak dapat dibaca: Redis tidak dapat dihubungi atau queue belum pernah dipakai"
	} else {
		stats.Queue = &model.QueueInfo{
			Name:      info.Queue,
			Pending:   info.Pending,
			Active:    info.Active,
			Scheduled: info.Scheduled,
			Retry:     info.Retry,
			Archived:  info.Archived,
			Completed: info.Completed,
			Processed: info.Processed,
			Failed:    info.Failed,
		}
	}
	return helper.Success(c, fiber.StatusOK, "keadaan queue berhasil diambil", stats)
}

