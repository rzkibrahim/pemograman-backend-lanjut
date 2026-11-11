package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"

	"latihan-fiber/app/model"
	"latihan-fiber/config"
	"latihan-fiber/database"
)

// redeliver adalah alat bantu pengujian (test tool) untuk menyuntikkan N salinan task
// yang sama ke antrean broker Redis guna menguji idempotency worker (Uji C.5).
func main() {
	config.LoadEnv()

	outboxID := flag.Int64("outbox", 0, "ID baris outbox yang akan dikirim ulang")
	copies := flag.Int("copies", 1, "Jumlah salinan task yang akan di-enqueue ke Redis")
	flag.Parse()

	if *outboxID <= 0 {
		log.Fatal("Flag -outbox harus diisi dengan ID yang valid (> 0)")
	}
	if *copies <= 0 {
		log.Fatal("Flag -copies minimal bernilai 1")
	}

	ctx := context.Background()

	pool, err := database.NewPool(ctx)
	if err != nil {
		log.Fatalf("Gagal terhubung ke database: %v", err)
	}
	defer pool.Close()

	var eventType string
	var payload []byte
	err = pool.QueryRow(ctx, `SELECT event_type, payload FROM outbox WHERE id = $1`, *outboxID).Scan(&eventType, &payload)
	if err != nil {
		log.Fatalf("Gagal membaca row outbox ID %d: %v", *outboxID, err)
	}

	job := model.NotificationJob{
		OutboxID:  *outboxID,
		EventType: eventType,
		Payload:   payload,
	}

	payloadBytes, err := json.Marshal(job)
	if err != nil {
		log.Fatalf("Gagal marshal job: %v", err)
	}

	client := asynq.NewClient(config.RedisOpt())
	defer client.Close()

	fmt.Printf("Menyuntikkan %d salinan task untuk outbox ID %d (%s) ke queue Redis...\n", *copies, *outboxID, eventType)

	for i := 1; i <= *copies; i++ {
		// Menggunakan ID task unik per salinan agar tidak diblokir oleh ErrTaskIDConflict bawaan Redis,
		// sehingga benar-benar mensimulasikan kondisi at-least-once (multiple duplicate tasks in queue).
		taskID := fmt.Sprintf("redeliver-%d-copy-%d-%d", *outboxID, i, time.Now().UnixNano())
		task := asynq.NewTask(
			model.TaskSendNotification,
			payloadBytes,
			asynq.TaskID(taskID),
			asynq.Queue(model.QueueNotifications),
			asynq.MaxRetry(5),
			asynq.Timeout(30*time.Second),
			asynq.Retention(24*time.Hour),
		)

		info, err := client.EnqueueContext(ctx, task)
		if err != nil {
			log.Fatalf("Gagal enqueue salinan ke-%d: %v", i, err)
		}
		fmt.Printf("  [✓] Salinan #%d berhasil di-enqueue (TaskID: %s, Queue: %s)\n", i, info.ID, info.Queue)
	}

	fmt.Println("Selesai! Seluruh salinan task duplikat telah berada di broker Redis.")
}

