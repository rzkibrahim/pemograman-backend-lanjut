-- migrations/004_student_permissions.sql
-- Migrasi Tugas Mandiri: Skema Permission Mahasiswa & Kolom Kepemilikan (owner_id)
-- Menutup lubang keamanan agar endpoint students tidak bisa diakses sembarangan
-- dan mendukung otorisasi berbasis kepemilikan (Ownership Access Control).

-- 1. Tambahkan permission untuk domain student
INSERT INTO permissions (id, description)
VALUES
    ('student:list',        'Melihat daftar seluruh mahasiswa'),
    ('student:read:any',    'Melihat data mahasiswa mana pun'),
    ('student:create',      'Menambahkan data mahasiswa baru'),
    ('student:update:any',  'Mengubah data mahasiswa milik siapa pun'),
    ('student:delete',      'Menghapus data mahasiswa')
ON CONFLICT (id) DO NOTHING;

-- 2. Hubungkan permission student ke role (role_permissions)
-- Admin: memiliki seluruh permission student
INSERT INTO role_permissions (role_id, permission_id)
VALUES
    ('admin', 'student:list'),
    ('admin', 'student:read:any'),
    ('admin', 'student:create'),
    ('admin', 'student:update:any'),
    ('admin', 'student:delete'),
    -- Staff: hanya list, read:any, dan create (tidak boleh update data orang lain atau delete)
    ('staff', 'student:list'),
    ('staff', 'student:read:any'),
    ('staff', 'student:create')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Catatan untuk role 'user':
-- Role 'user' sengaja TIDAK diberikan permission ':any' atau 'student:list'.
-- Hak user mengalir dari KEPEMILIKAN (owner_id == caller_id) untuk melihat/mengubah
-- data mahasiswa miliknya sendiri (ownership check di service layer).

-- 3. Tambahkan kolom owner_id pada tabel students
ALTER TABLE students ADD COLUMN IF NOT EXISTS owner_id INTEGER;

-- 4. Penanganan data lama yang belum memiliki owner_id
-- Mengaitkan data student lama yang sudah ada ke user pertama yang ada di sistem
-- agar integritas referensial terjaga sebelum batasan NOT NULL dan FOREIGN KEY dipasang.
DO $$
DECLARE
    default_user_id INT;
BEGIN
    SELECT id INTO default_user_id FROM users ORDER BY id ASC LIMIT 1;
    IF default_user_id IS NOT NULL THEN
        UPDATE students 
        SET owner_id = default_user_id 
        WHERE owner_id IS NULL;
    END IF;
END $$;

-- 5. Tambahkan constraint FOREIGN KEY ke tabel users
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_students_owner'
    ) THEN
        ALTER TABLE students 
        ADD CONSTRAINT fk_students_owner 
        FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE;
    END IF;
END $$;

-- 6. Buat indeks pada owner_id untuk mempercepat pencarian data berdasarkan pemilik
CREATE INDEX IF NOT EXISTS students_owner_id_idx ON students (owner_id);
