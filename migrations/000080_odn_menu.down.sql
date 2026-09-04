-- 回滚 000080:撤销 ODN 无源网络菜单权限及其角色授权。
BEGIN;
DELETE FROM role_permissions rp
USING permissions p
WHERE rp.permission_id = p.id AND p.code = 'menu:odn';

DELETE FROM permissions WHERE code = 'menu:odn';
COMMIT;
