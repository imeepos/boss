-- 回滚 ODN 局点与核心链路设备。
BEGIN;
DROP TABLE IF EXISTS odn_device;
DROP TABLE IF EXISTS odn_site;
COMMIT;
