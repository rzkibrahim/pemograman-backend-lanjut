-- Register review moderation as an explicit RBAC permission.
INSERT INTO permissions (id, description)
VALUES ('review:moderate', 'Menghapus ulasan milik pengguna lain')
ON CONFLICT (id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
VALUES ('admin', 'review:moderate')
ON CONFLICT (role_id, permission_id) DO NOTHING;