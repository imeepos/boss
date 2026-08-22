-- 回滚 000098:删入驻申请表 + 入驻角色及其权限绑定。
-- 注意:审核通过后创建的 legal_entities/accounts 行不级联回滚(业务数据,如需清理另行处理)。
BEGIN;
DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE code IN
    ('menu:partner', 'menu:partner-home', 'menu:partner-staff', 'menu:partner-orders'));
DELETE FROM permissions WHERE code IN
    ('menu:partner', 'menu:partner-home', 'menu:partner-staff', 'menu:partner-orders');
DELETE FROM role_permissions WHERE role_id IN (SELECT id FROM roles WHERE code IN ('partner_admin', 'partner_staff'));
DELETE FROM roles WHERE code IN ('partner_admin', 'partner_staff');
DROP TABLE IF EXISTS partner_applications;
COMMIT;
