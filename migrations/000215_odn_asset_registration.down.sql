-- 000215 down:桥表与出库单均为登记事实表,回滚即整体移除;
-- 出库在途资产需先经业务归位(转固/报废/退库)再回滚,down 不代做业务决策。

BEGIN;

DROP TABLE IF EXISTS odn_material_issue_items;
DROP TABLE IF EXISTS odn_material_issues;
DROP INDEX IF EXISTS idx_odn_asset_reg_asset;
DROP INDEX IF EXISTS uq_odn_asset_reg_asset_active;
DROP INDEX IF EXISTS uq_odn_asset_reg_device_active;
DROP INDEX IF EXISTS uq_odn_asset_reg_facility_active;
DROP TABLE IF EXISTS odn_asset_registrations;

COMMIT;
