-- migrations/009_supervisors.sql
-- D.2 Pembimbing TA: Skema supervisors dan supervisions

-- 1. Tabel supervisors (pembimbing tugas akhir)
CREATE TABLE IF NOT EXISTS supervisors (
    id                SERIAL       PRIMARY KEY,
    name              VARCHAR(100) NOT NULL,
    quota             INT          NOT NULL DEFAULT 3 CHECK (quota > 0),
    supervised_count  INT          NOT NULL DEFAULT 0 CHECK (supervised_count >= 0),
    version           INT          NOT NULL DEFAULT 1,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    -- Constraint: supervised_count tidak boleh melebihi quota
    CONSTRAINT chk_supervised_within_quota CHECK (supervised_count <= quota)
);

-- 2. Tabel supervisions (relasi mahasiswa-pembimbing)
CREATE TABLE IF NOT EXISTS supervisions (
    student_id     INT NOT NULL UNIQUE,  -- satu mahasiswa hanya satu pembimbing
    supervisor_id  INT NOT NULL REFERENCES supervisors(id) ON DELETE CASCADE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (student_id, supervisor_id)
);

CREATE INDEX IF NOT EXISTS supervisions_supervisor_id_idx ON supervisions(supervisor_id);

-- 3. Seed data supervisors
INSERT INTO supervisors (name, quota, supervised_count)
VALUES
    ('Dr. Ahmad Fauzi', 3, 0),
    ('Dr. Budi Santoso', 3, 0),
    ('Dr. Citra Dewi', 5, 0)
ON CONFLICT DO NOTHING;

-- 4. Permission baru
INSERT INTO permissions (id, description)
VALUES
    ('supervision:manage', 'Mengelola penetapan pembimbing TA')
ON CONFLICT (id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
VALUES
    ('admin', 'supervision:manage'),
    ('staff', 'supervision:manage')
ON CONFLICT (role_id, permission_id) DO NOTHING;
