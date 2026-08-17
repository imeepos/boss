-- 阶段3:资产持有台账(每次领用/部署/归还的时间段,历史归属不随当前值漂移)。
BEGIN;

CREATE TABLE asset_assignments (
    id                  BIGSERIAL PRIMARY KEY,
    asset_id            BIGINT NOT NULL REFERENCES assets(id),
    worker_id           BIGINT,
    worker_name         VARCHAR(64),
    address_id          BIGINT,
    address_name        VARCHAR(64),
    reason              VARCHAR(128),
    operator_account_id BIGINT,
    effective_from      TIMESTAMPTZ NOT NULL,
    effective_to        TIMESTAMPTZ   -- null=至今
);
CREATE INDEX idx_asset_assignments_asset ON asset_assignments(asset_id, effective_from);

COMMIT;
