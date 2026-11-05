-- Migration 009: Memperbarui Idempotency Key pada tabel notifications untuk mendukung Multi-Recipient Broadcast
-- psql -U postgres -d praktikum_backend -f migrations/009_notifications_idempotency.sql

-- Hapus constraint UNIQUE lama yang hanya berbasis (outbox_id)
ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_outbox_unique;

-- Buat composite UNIQUE constraint baru pada (outbox_id, user_id).
-- Idempotency key ini menjamin:
-- 1. Untuk event satu mahasiswa (enroll, drop, swap): tetap idempoten karena user_id tunggal.
-- 2. Untuk event broadcast (course.quota_changed): satu baris outbox dapat menghasilkan notifikasi
--    ke banyak mahasiswa, namun setiap mahasiswa dijamin hanya menerima TEPAT SATU notifikasi
--    berapa kali pun task di-retry.
ALTER TABLE notifications ADD CONSTRAINT notifications_outbox_user_unique UNIQUE (outbox_id, user_id);

