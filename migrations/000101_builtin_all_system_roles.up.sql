-- 修复已应用 000100 的环境:partner_admin/partner_staff 等后加系统角色被误标
-- is_builtin=false,在角色管理页显示为"自定义"可编辑/可删除。凡非 custom_* 派生
-- 角色一律回置内置。
BEGIN;

UPDATE roles SET is_builtin = true WHERE is_builtin = false AND code NOT LIKE 'custom\_%';

COMMIT;
