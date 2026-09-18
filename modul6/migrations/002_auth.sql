-- 002_auth.sql
-- Migrasi untuk modul 5: Authentication & Security

-- 1. Tabel users (jika belum ada)
CREATE TABLE IF NOT EXISTS users (
    id         SERIAL       PRIMARY KEY,
    username   VARCHAR(50)  NOT NULL,
    email      VARCHAR(255) NOT NULL,
    password   VARCHAR(255) NOT NULL,
    role       VARCHAR(20)  NOT NULL DEFAULT 'user',
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Indeks unik username case-insensitive
CREATE UNIQUE INDEX IF NOT EXISTS users_username_lower_key
    ON users (LOWER(username));

-- Indeks pencarian email
CREATE INDEX IF NOT EXISTS users_email_lower_idx
    ON users (LOWER(email));

-- Tambahkan kolom role jika tabel users sudah ada sebelumnya tanpa kolom role
ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'user';

-- 2. Kosongkan tabel users dari password plaintext lama (Sesuai Landasan Teori A.1)
TRUNCATE TABLE users CASCADE;

-- 3. Tabel refresh_tokens untuk menyimpan hash SHA-256 dari refresh token
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         SERIAL       PRIMARY KEY,
    user_id    INT          NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64)  NOT NULL,
    expires_at TIMESTAMPTZ  NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS refresh_tokens_token_hash_idx
    ON refresh_tokens (token_hash);

CREATE INDEX IF NOT EXISTS refresh_tokens_user_id_idx
    ON refresh_tokens (user_id);
