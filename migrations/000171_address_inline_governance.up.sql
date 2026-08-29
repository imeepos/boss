-- 开单内联建址治理标记(meeting-minutes/2026-08-29 §九):
-- needs_review:内联新建的 1-3 级节点强制 true,供地址治理页复核;4-5 级 false。
-- source:节点来源留痕,order-inline=开单流程内联建址;空=地址管理页/导入等常规来源。
BEGIN;
ALTER TABLE addresses
    ADD COLUMN needs_review BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN source TEXT NOT NULL DEFAULT '';
COMMENT ON COLUMN addresses.needs_review IS '治理复核标记:内联新建 1-3 级 true';
COMMENT ON COLUMN addresses.source IS '节点来源:order-inline=开单内联建址,空=常规';
COMMIT;
