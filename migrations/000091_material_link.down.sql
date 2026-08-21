BEGIN;
DROP INDEX IF EXISTS idx_worker_tools_tool;
DROP INDEX IF EXISTS idx_worker_materials_item;
ALTER TABLE worker_tools DROP COLUMN IF EXISTS tool_id;
ALTER TABLE worker_materials DROP COLUMN IF EXISTS item_id;
COMMIT;
