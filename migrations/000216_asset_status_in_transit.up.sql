-- 000216: assets.status 枚举放宽五态(terms.md §4 W8 修订):
-- IN_STOCK/IN_TRANSIT/DEPLOYED/MAINTENANCE/SCRAPPED。
-- IN_TRANSIT=出库在途(材料出库单 CONFIRMED 置此态,转固 ACTIVE 凭证生效即 DEPLOYED)。
-- 背景:F7「材料资产出库后从台账消失」——出库后仍记 IN_STOCK 会被库存盘点误捕,
-- 记 DEPLOYED 则虚增装网;独立在途态是台账连续性的最小表达。

BEGIN;

ALTER TABLE assets DROP CONSTRAINT ck_assets_status;
ALTER TABLE assets ADD CONSTRAINT ck_assets_status
  CHECK (status IN ('IN_STOCK','IN_TRANSIT','DEPLOYED','MAINTENANCE','SCRAPPED'));

COMMIT;
