-- 回滚 ODN 施工项目表。
BEGIN;

DROP TABLE IF EXISTS construction_items;
DROP TABLE IF EXISTS construction_projects;

COMMIT;
