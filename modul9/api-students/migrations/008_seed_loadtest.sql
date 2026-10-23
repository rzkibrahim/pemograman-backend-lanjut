-- migrations/008_seed_loadtest.sql
-- Seed 60 akun uji beban untuk pengujian race condition

-- Password untuk semua akun uji: "TestPass123"
-- bcrypt hash dari "TestPass123" dengan cost 12
DO $$
DECLARE
    hashed TEXT := '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewJPHyDVSIQBLbPy';
    i INT;
    uname TEXT;
    mail TEXT;
BEGIN
    FOR i IN 1..60 LOOP
        uname := 'user' || LPAD(i::text, 2, '0');
        mail  := uname || '@test.local';
        INSERT INTO users (username, email, password, role, is_active)
        VALUES (uname, mail, hashed, 'user', true)
        ON CONFLICT DO NOTHING;
    END LOOP;
END $$;
