-- 自定义角色:内置 7 角色标记只读,允许新建派生角色(权限集可整组复制内置模板后单独增删)。
-- 决策记录:docs/notes/adopted/0007-custom-roles.md
BEGIN;

ALTER TABLE roles ADD COLUMN is_builtin BOOLEAN NOT NULL DEFAULT false;

UPDATE roles SET is_builtin = true
WHERE code IN ('customer', 'technician', 'asset_admin', 'resource_admin', 'ops', 'analyst', 'sysadmin');

COMMIT;
