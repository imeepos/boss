-- 回滚 ODN 无源物理层。
BEGIN;
DROP TABLE IF EXISTS odn_facility;
DROP TABLE IF EXISTS odn_grid;
COMMIT;
