-- 000208: 施工项目承包商两列补丁(修复 000206 漏列,2026-09-07 负责人打回)。
-- 000206 只落了 construction_items 工程量列与 construction_settlements 表,
-- construction_projects.contractor_id/contractor_name 被遗漏,读路径 42703 全 50000。
-- 000206 已应用(102/远端),迁移不可变,fix-forward 新开号;列口径=fields.md 1.5.8。
ALTER TABLE construction_projects
    ADD COLUMN contractor_id   BIGINT,
    ADD COLUMN contractor_name VARCHAR(128) NOT NULL DEFAULT '';

COMMENT ON COLUMN construction_projects.contractor_id   IS '承包商软引用 → procurement_suppliers(id),NULL=未指定(存量兼容,000208)';
COMMENT ON COLUMN construction_projects.contractor_name IS '承包商名称快照(指定时落,000208)';