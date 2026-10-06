-- Migration 006: Composite Index untuk Cursor Pagination pada tabel students
-- Pasangan kolom: (created_at DESC, id DESC)
-- Alasan:
-- 1. created_at mengurutkan dari data terbaru ke terlama.
-- 2. id (Primary Key) menjamin tie-breaker deterministik yang unik mutlak,
--    mencegah data terlewat atau terulang saat ada timestamp yang identik.
-- 3. Arah DESC pada kedua kolom memungkinkan PostgreSQL menggunakan Index Scan (Forward)
--    langsung melompat ke tuple (created_at, id) < (cursor_created_at, cursor_id).

CREATE INDEX IF NOT EXISTS students_created_at_id_desc_idx
    ON students (created_at DESC, id DESC);
