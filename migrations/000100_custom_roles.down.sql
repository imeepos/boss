BEGIN;

ALTER TABLE roles DROP COLUMN IF EXISTS is_builtin;
-- 自定义角色行随列一并失效:角色删除由应用层引用检查把守,down 不物理清理派生行。

COMMIT;
