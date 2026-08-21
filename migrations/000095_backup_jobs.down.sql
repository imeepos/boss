-- 回滚 000095:摘除 backup_jobs 表与 menu:backup 权限(归档文件不在此清理)。
BEGIN;
DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code = 'menu:backup');
DELETE FROM permissions WHERE code = 'menu:backup';
DROP TABLE IF EXISTS backup_jobs;
COMMIT;
