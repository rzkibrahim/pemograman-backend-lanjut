package model

import (
	"encoding/json"
	"time"
)

// Nama-nama event yang dicatat ke outbox. Ditulis sebagai konstanta,
// bukan string yang diketik ulang di banyak tempat, karena salah ketik pada
// nama event tidak ditangkap compiler maupun database.
const (
	EventEnrollmentCreated   = "enrollment.created"
	EventEnrollmentDropped   = "enrollment.dropped"
	EventEnrollmentSwapped   = "enrollment.swapped"
	EventCourseQuotaChanged  = "course.quota_changed" // Event untuk Tugas Mandiri D.2: perubahan kuota kelas
)

// TaskSendNotification adalah nama JENIS task (task type); worker memakainya untuk
// memilih handler. QueueNotifications adalah nama QUEUE tempat task
// itu menunggu; worker hanya mengambil task dari queue yang ia dengarkan.
const (
	TaskSendNotification = "notification:send"
	QueueNotifications   = "notifications"
)

// EnrollmentEvent adalah isi sebuah event KRS.
//
// Nama dan kode kelas DISALIN ke dalam event, bukan hanya id-nya.
// Event adalah catatan tentang apa yang terjadi PADA SAAT ITU. Bila nama
// kelas diubah esok hari, notifikasi untuk event hari ini tetap memakai
// nama yang dilihat mahasiswa ketika ia mendaftar.
type EnrollmentEvent struct {
	UserID         int    `json:"user_id"`
	CourseID       int    `json:"course_id"`
	CourseCode     string `json:"course_code"`
	CourseName     string `json:"course_name"`
	FromCourseCode string `json:"from_course_code,omitempty"`
	FromCourseName string `json:"from_course_name,omitempty"`
}

// CourseQuotaChangedEvent adalah isi event ketika admin mengubah kuota kelas (Tugas Mandiri D.2).
// Memuat data kelas, kuota baru, serta daftar user_id mahasiswa yang sedang terdaftar di kelas tersebut.
type CourseQuotaChangedEvent struct {
	CourseID   int    `json:"course_id"`
	CourseCode string `json:"course_code"`
	CourseName string `json:"course_name"`
	NewQuota   int    `json:"new_quota"`
	UserIDs    []int  `json:"user_ids"`
}

// OutboxEvent adalah representasi satu baris pada tabel outbox di database.
type OutboxEvent struct {
	ID        int64     `json:"id"`
	EventType string    `json:"event_type"`
	Payload   []byte    `json:"payload"`
	CreatedAt time.Time `json:"created_at"`
}

// NotificationJob adalah isi task yang dikirim ke queue Redis.
//
// OutboxID ikut dibawa karena dialah idempotency key: worker memakainya
// untuk mengetahui apakah event ini sudah pernah dikirimkan.
//
// Payload dibiarkan sebagai JSON mentah (json.RawMessage). Relay cukup
// meneruskannya apa adanya tanpa perlu tahu bentuk setiap jenis event;
// yang membaca isinya adalah worker, berdasarkan EventType.
type NotificationJob struct {
	OutboxID  int64           `json:"outbox_id"`
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
}

// Message adalah representasi pesan yang siap dikirim (email/notifikasi),
// independen terhadap media atau penyedia transport pengirimannya.
type Message struct {
	To      string
	Subject string
	Body    string
}

// Notification adalah entitas notifikasi yang sudah berhasil disimpan/dikirim,
// seperti yang ditampilkan kepada mahasiswa pada endpoint GET /notifications/mine.
type Notification struct {
	ID       int64     `json:"id"`
	OutboxID int64     `json:"outbox_id"`
	UserID   int       `json:"user_id"`
	Channel  string    `json:"channel"`
	Subject  string    `json:"subject"`
	Body     string    `json:"body"`
	SentAt   time.Time `json:"sent_at"`
}

// QueueStats adalah ringkasan status outbox dan queue Asynq untuk endpoint pemantauan admin.
type QueueStats struct {
	OutboxPending     int        `json:"outbox_pending"`
	OutboxPublished   int        `json:"outbox_published"`
	OldestPendingAge  string     `json:"oldest_pending_age"`
	NotificationsSent int        `json:"notifications_sent"`
	Queue             *QueueInfo `json:"queue"`
	QueueError        string     `json:"queue_error,omitempty"`
}

// QueueInfo adalah salinan metrik queue dari Asynq. Ditulis ulang secara mandiri
// di sini agar package model tetap bersih tanpa bergantung langsung pada library luar.
type QueueInfo struct {
	Name      string `json:"name"`
	Pending   int    `json:"pending"`
	Active    int    `json:"active"`
	Scheduled int    `json:"scheduled"`
	Retry     int    `json:"retry"`
	Archived  int    `json:"archived"`
	Completed int    `json:"completed"`
	Processed int    `json:"processed_today"`
	Failed    int    `json:"failed_today"`
}

