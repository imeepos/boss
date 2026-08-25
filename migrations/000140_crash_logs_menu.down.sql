DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code = 'menu:crash_logs');
DELETE FROM permissions WHERE code = 'menu:crash_logs';