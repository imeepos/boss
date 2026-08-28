-- 000168: 施工回单接口收权配套——授 ops menu:install-board。
-- GET /install-logs 此前无 permCode 无数据范围,任意登录角色(含 technician)可读
-- 全部施工回单(2026-08-28 权限实测暴露面)。接口按 menu:install-board(000165)收权后,
-- ops(履约域,施工看板页使用方)补授保持看板可用;technician 回单归 worker 端通道;
-- sysadmin 全量不受影响。顺带消解 ops 在 FE/BE diff 中 install-board 的"前端有后端无"漂移。
BEGIN;
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'menu:install-board'
WHERE r.code = 'ops'
ON CONFLICT DO NOTHING;
COMMIT;
