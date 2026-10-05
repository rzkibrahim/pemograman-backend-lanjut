-- migrations/003_rbac.sql
-- Migrasi Langkah 1: Skema RBAC (Role-Based Access Control)
-- Membuat tabel roles, permissions, dan role_permissions dengan composite primary key
-- serta menambahkan Foreign Key pada users.role.

-- 1. Tabel roles
CREATE TABLE IF NOT EXISTS roles (
    id          VARCHAR(20)  PRIMARY KEY,
    name        VARCHAR(50)  NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- 2. Tabel permissions
CREATE TABLE IF NOT EXISTS permissions (
    id          VARCHAR(50)  PRIMARY KEY,
    description TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- 3. Tabel role_permissions (Composite Primary Key mencegah duplikasi)
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id       VARCHAR(20) NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id VARCHAR(50) NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, permission_id)
);

-- 4. Indeks untuk mempercepat pencarian relasi
CREATE INDEX IF NOT EXISTS role_permissions_role_id_idx ON role_permissions(role_id);
CREATE INDEX IF NOT EXISTS role_permissions_permission_id_idx ON role_permissions(permission_id);

-- 5. Seed data Roles awal
INSERT INTO roles (id, name, description)
VALUES 
    ('admin', 'Administrator', 'Akses penuh ke seluruh modul administratif dan sistem'),
    ('staff', 'Staf Akademik', 'Akses operasional data dan pembacaan umum'),
    ('user',  'Pengguna Reguler', 'Akses mandiri ke data pribadi dan kepemilikan data')
ON CONFLICT (id) DO NOTHING;

-- 6. Seed data Permissions awal untuk domain User
INSERT INTO permissions (id, description)
VALUES
    ('user:list',        'Melihat daftar seluruh user'),
    ('user:read:any',    'Melihat data profil user mana pun'),
    ('user:update:any',  'Mengubah profil data user mana pun'),
    ('user:delete',      'Menghapus akun user dari sistem'),
    ('role:assign',      'Mengubah role akun milik user lain')
ON CONFLICT (id) DO NOTHING;

-- 7. Hubungkan Permission ke Role (role_permissions)
-- Admin memiliki seluruh permission di atas
INSERT INTO role_permissions (role_id, permission_id)
VALUES
    ('admin', 'user:list'),
    ('admin', 'user:read:any'),
    ('admin', 'user:update:any'),
    ('admin', 'user:delete'),
    ('admin', 'role:assign'),
    -- Staff memiliki hak melihat daftar dan detail user mana pun
    ('staff', 'user:list'),
    ('staff', 'user:read:any')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- 8. Normalisasi data lama pada tabel users sebelum menambahkan constraint foreign key
-- WAJIB: UPDATE data lama terlebih dahulu agar ALTER TABLE tidak melanggar constraint
UPDATE users 
SET role = 'user' 
WHERE role IS NULL OR role NOT IN ('admin', 'staff', 'user');

-- 9. Tambahkan FOREIGN KEY constraint pada users.role
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_users_role'
    ) THEN
        ALTER TABLE users 
        ADD CONSTRAINT fk_users_role 
        FOREIGN KEY (role) REFERENCES roles(id) ON UPDATE CASCADE;
    END IF;
END $$;
