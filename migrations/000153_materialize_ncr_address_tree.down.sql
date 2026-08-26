-- 回滚 NCR 物化:仅删本迁移创建的子树(root path 前缀),不动其他树。
BEGIN;
DELETE FROM addresses WHERE path <@ 'ph1300000000'::ltree;
COMMIT;
