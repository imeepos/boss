-- 000221 down: 撤销设备分光容量模型(暂存列 odn_resource_chain 不受影响,可重放回写)。
BEGIN;
DROP TABLE IF EXISTS odn_device_split_capacity;
COMMIT;
