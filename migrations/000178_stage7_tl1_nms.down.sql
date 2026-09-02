-- 迁移 000178 回滚:逆序撤销(先分配器,再端口列,再 OLT 列,最后端点表)。
-- 与 up 完全对称:pon_onu_alloc 先于 resources 外键依赖而存在,故先 DROP。
BEGIN;

DROP TABLE IF EXISTS pon_onu_alloc;

ALTER TABLE ports DROP COLUMN IF EXISTS onu_no;
ALTER TABLE ports DROP COLUMN IF EXISTS pon_port;
ALTER TABLE ports DROP COLUMN IF EXISTS pon_slot;
ALTER TABLE ports DROP COLUMN IF EXISTS pon_frame;

ALTER TABLE resources DROP COLUMN IF EXISTS nms_oltid;

DROP TABLE IF EXISTS provision_nms;

COMMIT;
