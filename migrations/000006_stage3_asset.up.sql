-- 阶段3:资产域(入库批次 / 电子标签 / 资产台账)。
-- 字段权威:docs/contract/fields.md §4.1 + server-ts/src/entities/asset.ts。
BEGIN;

CREATE TABLE asset_batches (
    id              BIGSERIAL PRIMARY KEY,
    legal_entity_id BIGINT NOT NULL REFERENCES legal_entities(id),
    code            VARCHAR(32) NOT NULL UNIQUE,  -- 批次编码,如 RK-202607-01
    name            VARCHAR(64) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tags (
    id              BIGSERIAL PRIMARY KEY,
    legal_entity_id BIGINT NOT NULL REFERENCES legal_entities(id),
    tag_no          VARCHAR(32) NOT NULL UNIQUE,
    epc_code        VARCHAR(32) NOT NULL UNIQUE,
    band            VARCHAR(8) NOT NULL,          -- 频段,如 UHF
    bound_asset_id  BIGINT,                       -- 预绑定资产,空=未绑定
    status          VARCHAR(16) NOT NULL DEFAULT 'UNBOUND', -- UNBOUND/BOUND/DISABLED
    battery         VARCHAR(8) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_tags_entity ON tags(legal_entity_id);

CREATE TABLE assets (
    id                BIGSERIAL PRIMARY KEY,
    asset_code        VARCHAR(32) NOT NULL UNIQUE, -- 资产编码,如 A-20260001
    batch_id          BIGINT NOT NULL REFERENCES asset_batches(id),
    legal_entity_id   BIGINT NOT NULL,             -- 企业归属快照
    legal_entity_name VARCHAR(128) NOT NULL,
    tag_id            BIGINT,                      -- 绑定标签,空=未绑定
    address_id        BIGINT,                      -- 部署地址,空=未部署
    region_id         INTEGER,                     -- 经营区域快照
    region_name       VARCHAR(64),
    type              VARCHAR(32) NOT NULL,        -- 光猫/ONU/路由器
    status            VARCHAR(16) NOT NULL,        -- IN_STOCK/DEPLOYED/MAINTENANCE/SCRAPPED
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_assets_batch ON assets(batch_id);
CREATE INDEX idx_assets_tag ON assets(tag_id);
CREATE INDEX idx_assets_address ON assets(address_id);

COMMIT;
