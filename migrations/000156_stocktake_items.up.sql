-- 盘点差异明细(S10 流程补全):任务建单时冻结资产快照,扫码回填实盘,差异逐条处置。
BEGIN;

CREATE TABLE stocktake_items (
    id              BIGSERIAL PRIMARY KEY,
    task_id         BIGINT NOT NULL REFERENCES stocktakes(id),
    asset_id        BIGINT NOT NULL,              -- 软引用 assets(对齐 replacements 口径)
    expected_status VARCHAR(16),                  -- 建单快照状态;NULL=计划外(多扫 EXTRA)
    scanned_status  VARCHAR(16),                  -- 实盘状态;NULL=未扫
    scanned_at      TIMESTAMPTZ,
    kind            VARCHAR(16) NOT NULL DEFAULT 'PENDING', -- PENDING/OK/MISMATCH/MISSING/EXTRA
    resolution      VARCHAR(16) NOT NULL DEFAULT 'OPEN',    -- OPEN/CONFIRMED/FIXED/ESCALATED
    handled_by      BIGINT,                       -- 处置人账号
    handled_at      TIMESTAMPTZ,
    note            VARCHAR(255)
);
CREATE INDEX idx_stocktake_items_task ON stocktake_items(task_id, kind);
CREATE UNIQUE INDEX uq_stocktake_items_task_asset ON stocktake_items(task_id, asset_id);

COMMIT;
