-- 崩溃日志管理端查看页 menu:crashlogs 权限(基础配置·崩溃日志)
-- 与 web/admin menu.def key 一一对应(check-contract-sync E 项要求 key↔perm 同名);
-- 授予 sysadmin。
BEGIN;
INSERT INTO permissions (code, name) VALUES
    ('menu:crashlogs', '基础配置·崩溃日志')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'menu:crashlogs'
WHERE r.code = 'sysadmin'
ON CONFLICT DO NOTHING;
COMMIT;