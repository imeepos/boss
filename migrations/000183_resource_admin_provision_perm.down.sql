-- 回滚 000183:回收 resource_admin 的 menu:provision 授权(对齐 up 的授对范围)。
BEGIN;
DELETE FROM role_permissions rp
USING roles r, permissions p
WHERE rp.role_id = r.id
  AND rp.permission_id = p.id
  AND r.code = 'resource_admin'
  AND p.code = 'menu:provision';
COMMIT;
