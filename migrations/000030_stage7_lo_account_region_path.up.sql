-- 阶段7:lo_accounts 补 region_path(区域 ltree 路径快照),承接区域调价覆盖(region_offers)与 GIS 下钻。
-- 字段权威:见 docs/plan/3-month-roadmap D7 与 billing.GenerateBills 注释;region_id/region_name 已有,补 path 消除区域价覆盖 join 缺口。
BEGIN;
ALTER TABLE lo_accounts ADD COLUMN region_path VARCHAR(128);  -- 区域 ltree 路径,如 root.luzon.ncr.manila;null=未挂区域
COMMIT;
