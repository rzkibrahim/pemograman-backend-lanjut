-- reset.sql
-- Skrip reset data kelas untuk pengujian ulang load test

-- Reset enrollments
DELETE FROM enrollments;

-- Reset courses
UPDATE courses SET enrolled_count = 0, quota = 5, version = 1 WHERE id IN (1, 2, 3);
UPDATE courses SET enrolled_count = 0, quota = 40, version = 1 WHERE id IN (4, 5);

-- Reset supervisions
DELETE FROM supervisions;

-- Reset supervisors
UPDATE supervisors SET supervised_count = 0, quota = 3, version = 1 WHERE id IN (1, 2);
UPDATE supervisors SET supervised_count = 0, quota = 5, version = 1 WHERE id = 3;

-- Verifikasi
SELECT id, code, name, quota, enrolled_count, version FROM courses ORDER BY id;
SELECT id, name, quota, supervised_count, version FROM supervisors ORDER BY id;
