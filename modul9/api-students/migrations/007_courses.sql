-- migrations/007_courses.sql
-- Skema kelas & KRS untuk Modul 9 (Transactions & Concurrency)

-- 1. Tabel courses (mata kuliah / kelas)
CREATE TABLE IF NOT EXISTS courses (
    id              SERIAL       PRIMARY KEY,
    code            VARCHAR(20)  NOT NULL UNIQUE,
    name            VARCHAR(100) NOT NULL,
    credits         INT          NOT NULL DEFAULT 3 CHECK (credits > 0 AND credits <= 6),
    quota           INT          NOT NULL DEFAULT 5 CHECK (quota > 0),
    enrolled_count  INT          NOT NULL DEFAULT 0 CHECK (enrolled_count >= 0),
    version         INT          NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    -- Constraint: enrolled_count tidak boleh melebihi quota
    CONSTRAINT chk_enrolled_within_quota CHECK (enrolled_count <= quota)
);

-- 2. Tabel enrollments (KRS / pendaftaran mahasiswa ke kelas)
CREATE TABLE IF NOT EXISTS enrollments (
    id         SERIAL      PRIMARY KEY,
    student_id INT         NOT NULL,
    course_id  INT         NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Satu mahasiswa tidak boleh mendaftar ke kelas yang sama dua kali
    CONSTRAINT uq_enrollment UNIQUE (student_id, course_id)
);

-- 3. Index untuk mempercepat pencarian enrollment berdasarkan student
CREATE INDEX IF NOT EXISTS enrollments_student_id_idx ON enrollments(student_id);
CREATE INDEX IF NOT EXISTS enrollments_course_id_idx ON enrollments(course_id);

-- 4. Seed data courses awal (3 kelas, kuota kecil untuk pengujian race condition)
INSERT INTO courses (code, name, credits, quota, enrolled_count)
VALUES
    ('IF101', 'Pemrograman Dasar', 3, 5, 0),
    ('IF102', 'Basis Data', 3, 5, 0),
    ('IF103', 'Jaringan Komputer', 4, 5, 0),
    ('IF104', 'Sistem Operasi', 3, 40, 0),
    ('IF105', 'Kecerdasan Buatan', 3, 40, 0)
ON CONFLICT (code) DO NOTHING;

-- 5. Permission baru untuk course management
INSERT INTO permissions (id, description)
VALUES
    ('course:list',       'Melihat daftar seluruh mata kuliah'),
    ('course:enroll',     'Mendaftar ke mata kuliah'),
    ('course:drop',       'Membatalkan pendaftaran mata kuliah'),
    ('course:swap',       'Menukar kelas mata kuliah'),
    ('course:update',     'Mengubah kuota mata kuliah (admin)')
ON CONFLICT (id) DO NOTHING;

-- 6. Hubungkan permission ke role
INSERT INTO role_permissions (role_id, permission_id)
VALUES
    ('admin', 'course:list'),
    ('admin', 'course:enroll'),
    ('admin', 'course:drop'),
    ('admin', 'course:swap'),
    ('admin', 'course:update'),
    ('staff', 'course:list'),
    ('staff', 'course:enroll'),
    ('staff', 'course:drop'),
    ('staff', 'course:swap'),
    ('user',  'course:list'),
    ('user',  'course:enroll'),
    ('user',  'course:drop'),
    ('user',  'course:swap')
ON CONFLICT (role_id, permission_id) DO NOTHING;
