-- ODN P1 覆盖关联:地址 ↔ 服务设施/核心设备 + 可装状态。
-- 决策:adopted/2026-09-06-odn-business-linkage.md("网络规划是业务基础",P1 可查可判不做下单硬校验)。
-- 让号说明:000196 已被 feat/aaa-a5-per-nas-vsa(aaa_nas_clients)占用,按迁移编号规则顺延。
-- status 语义:SERVED=已覆盖可装机 / PENDING=规划在建(已挂规划设施) / UNSERVED=未覆盖。
BEGIN;

CREATE TABLE address_coverage (
    id            BIGSERIAL PRIMARY KEY,
    address_id    BIGINT      NOT NULL REFERENCES addresses(id),
    facility_code VARCHAR(8)  REFERENCES odn_facility(code), -- 服务设施(ODB/SDB/分纤点),可空
    device_id     BIGINT      REFERENCES odn_device(id),     -- 服务核心链路设备,可空
    status        VARCHAR(16) NOT NULL DEFAULT 'UNSERVED'
                  CHECK (status IN ('SERVED','PENDING','UNSERVED')),
    note          VARCHAR(256) NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT address_coverage_addr_uq UNIQUE (address_id),
    CONSTRAINT address_coverage_target_chk
        CHECK (status = 'UNSERVED' OR facility_code IS NOT NULL OR device_id IS NOT NULL)
);
CREATE INDEX idx_address_coverage_facility ON address_coverage (facility_code) WHERE facility_code IS NOT NULL;
CREATE INDEX idx_address_coverage_device   ON address_coverage (device_id)   WHERE device_id IS NOT NULL;

COMMIT;
