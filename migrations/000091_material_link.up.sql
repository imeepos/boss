-- E13:物料/工具主档接线——000073 建了 material_items/material_tools 主档,
-- 但 worker_materials/worker_tools 仍只存自由文本 name,主档驱动未兑现:
-- 领料编码无从校验、按物料聚合做不出来。补 item_id/tool_id 强引用,
-- name 保留为展示快照(xxx_id + name 双列,fields.md 第 0 节规则)。
BEGIN;

ALTER TABLE worker_materials ADD COLUMN item_id BIGINT REFERENCES material_items(id);
ALTER TABLE worker_tools ADD COLUMN tool_id BIGINT REFERENCES material_tools(id);

-- 存量回填(宁缺勿错,仅唯一匹配):物料记录名是「主档名 + 空格 + 规格」
-- (师傅端 handler 拼接),按前缀匹配;工具名精确匹配。
UPDATE worker_materials wm SET item_id = (
    SELECT mi.id FROM material_items mi WHERE wm.name LIKE mi.name || '%'
  )
  WHERE (SELECT COUNT(*) FROM material_items mi WHERE wm.name LIKE mi.name || '%') = 1;

UPDATE worker_tools wt SET tool_id = (
    SELECT mt.id FROM material_tools mt WHERE wt.name = mt.name
  )
  WHERE (SELECT COUNT(*) FROM material_tools mt WHERE wt.name = mt.name) = 1;

CREATE INDEX idx_worker_materials_item ON worker_materials(item_id);
CREATE INDEX idx_worker_tools_tool ON worker_tools(tool_id);

COMMIT;
