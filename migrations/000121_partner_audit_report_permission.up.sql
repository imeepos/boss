-- 渠道审计报表权限：复用 audit_logs 原始审计，不复制审计存储。
BEGIN;
INSERT INTO permissions(code, name) VALUES ('menu:partner-audit', '企业工作台·渠道审计报表') ON CONFLICT (code) DO NOTHING;
INSERT INTO role_permissions(role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code='menu:partner-audit'
WHERE r.code IN ('partner_admin','partner_staff') ON CONFLICT DO NOTHING;
COMMIT;
