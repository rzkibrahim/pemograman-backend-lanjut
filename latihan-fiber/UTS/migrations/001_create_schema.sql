-- Migration 001: Skema Database SIAKAD Mini UTS
-- Membuat tabel users, students, courses, enrollments serta indeks dan batasan integritas.

-- 1. Tabel users
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('admin', 'mahasiswa')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Tabel students
CREATE TABLE IF NOT EXISTS students (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    nim VARCHAR(12) NOT NULL UNIQUE,
    nama VARCHAR(255) NOT NULL,
    prodi VARCHAR(100) NOT NULL,
    angkatan INT NOT NULL,
    ipk_terakhir NUMERIC(3, 2) NOT NULL DEFAULT 0.00 CHECK (ipk_terakhir >= 0.00 AND ipk_terakhir <= 4.00),
    deleted_at TIMESTAMPTZ DEFAULT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indeks untuk mendukung query pagination, search, dan soft-delete
CREATE INDEX IF NOT EXISTS idx_students_deleted_at ON students(deleted_at);
CREATE INDEX IF NOT EXISTS idx_students_prodi ON students(prodi) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_students_angkatan ON students(angkatan) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_students_nim ON students(nim);

-- 3. Tabel courses
CREATE TABLE IF NOT EXISTS courses (
    id SERIAL PRIMARY KEY,
    kode_mk VARCHAR(20) NOT NULL UNIQUE,
    nama_mk VARCHAR(255) NOT NULL,
    sks INT NOT NULL CHECK (sks > 0),
    semester INT NOT NULL CHECK (semester >= 1 AND semester <= 8),
    kuota INT NOT NULL CHECK (kuota >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_courses_semester ON courses(semester);
CREATE INDEX IF NOT EXISTS idx_courses_kode_mk ON courses(kode_mk);

-- 4. Tabel enrollments (KRS)
CREATE TABLE IF NOT EXISTS enrollments (
    id SERIAL PRIMARY KEY,
    student_id INT NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    course_id INT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    tahun_akademik VARCHAR(30) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_student_course_year UNIQUE (student_id, course_id, tahun_akademik)
);

CREATE INDEX IF NOT EXISTS idx_enrollments_student_year ON enrollments(student_id, tahun_akademik);
CREATE INDEX IF NOT EXISTS idx_enrollments_course_id ON enrollments(course_id);
