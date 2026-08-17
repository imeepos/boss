-- 阶段3:资产台账(状态轨迹 / 换新单 / 盘点任务)。
BEGIN;

CREATE TABLE asset_lifecycles (
    id           BIGSERIAL PRIMARY KEY,
    asset_id     BIGINT NOT NULL REFERENCES assets(id),
    status       VARCHAR(16) NOT NULL,  -- IN_STOCK/DEPLOYED/MAINTENANCE/SCRAPPED
    address_id   BIGINT,
    address_name VARCHAR(64),
    worker_id    BIGINT,
    worker_name  VARCHAR(64),
    changed_at   TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_asset_lifecycles_asset ON asset_lifecycles(asset_id, changed_at);

CREATE TABLE replacements (
    id                BIGSERIAL PRIMARY KEY,
    replacement_no    VARCHAR(32) NOT NULL UNIQUE,
    asset_id          BIGINT NOT NULL,          -- 故障资产(软引用 assets)
    legal_entity_id   BIGINT NOT NULL,
    legal_entity_name VARCHAR(128) NOT NULL,
    reason            VARCHAR(64) NOT NULL,
    priority          VARCHAR(8) NOT NULL,      -- HIGH/MEDIUM/LOW
    status            VARCHAR(16) NOT NULL DEFAULT 'PENDING' -- PENDING/DOING/DONE/FAILED
);

CREATE TABLE stocktakes (
    id              BIGSERIAL PRIMARY KEY,
    legal_entity_id BIGINT NOT NULL REFERENCES legal_entities(id),
    scope           VARCHAR(128) NOT NULL,  -- 盘点范围(区域 ltree 路径)
    progress        SMALLINT NOT NULL,      -- 0~100
    diff_count      INTEGER NOT NULL,       -- 差异条数
    status          VARCHAR(16) NOT NULL    -- DOING/DONE
);

COMMIT;
