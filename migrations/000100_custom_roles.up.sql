-- 自定义角色:既有系统角色全部标记只读,允许新建派生角色(权限集可整组复制内置模板后单独增删)。
-- 决策记录:docs/notes/adopted/2026-08-22-custom-roles.md
BEGIN;

ALTER TABLE roles ADD COLUMN is_builtin BOOLEAN NOT NULL DEFAULT false;

-- 迁移时已存在的角色一律视为系统内置(7 基础角色 + partner_admin/partner_staff 等
-- 后续种子;按码枚举会漏新种子角色,曾把 partner_* 误标为可删)。
UPDATE roles SET is_builtin = true;

COMMIT;
