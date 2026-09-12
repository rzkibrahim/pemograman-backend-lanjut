-- migrations/001_create_students.sql
-- Migrasi pertama: membuat tabel students beserta indeks-indeksnya.

CREATE TABLE IF NOT EXISTS students (
    id         SERIAL        PRIMARY KEY,
    nim        VARCHAR(20)   NOT NULL,
    name       VARCHAR(100)  NOT NULL,
    grade      NUMERIC(5,2)  NOT NULL DEFAULT 0,
    is_active  BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- Keunikan NIM tanpa membedakan huruf besar dan kecil.
-- Ini menggantikan pemeriksaan manual isNIMDuplicate() di pertemuan 2.
-- Batasan di basis data lebih kuat: tidak ada celah race condition
-- yang bisa terjadi bila dua permintaan datang bersamaan.
CREATE UNIQUE INDEX IF NOT EXISTS students_nim_lower_key
    ON students (LOWER(nim));

-- Indeks pada LOWER(name) mempercepat pencarian ILIKE pada kolom name.
-- Tanpa indeks ini, setiap pencarian harus memindai seluruh tabel.
CREATE INDEX IF NOT EXISTS students_name_lower_idx
    ON students (LOWER(name));

-- Indeks pada grade mempercepat penyaringan berdasarkan rentang nilai
-- (grade_min dan grade_max).
CREATE INDEX IF NOT EXISTS students_grade_idx
    ON students (grade);

create table if not exists nilai(
    id_nilai serial primary key,
    id_student serial not null,
    mata_kuliah varchar(100) not null,
    nilai numeric(5,2) not null default 0
);