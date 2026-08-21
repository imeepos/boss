-- 回滚 ODN 光缆段落与纤芯。
BEGIN;
DROP TABLE IF EXISTS odn_fiber;
DROP TABLE IF EXISTS odn_cable_segment;
COMMIT;
