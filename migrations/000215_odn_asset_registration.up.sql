-- 000215: ODN 资产化转固(W8,docs/plan/infra-buildout-plan.md 顺序二;F7 资产转固阻塞)。
-- 关联裁定(adopted 2026-09-07-odn-asset-capitalization):odn×asset 用桥表软引用,
-- 不加跨域 FK、不动 assets/odn_facility/odn_device 既有 schema;桥表行=资产凭证
-- (登记号/价值/来源/项目溯源),冲销留历史不删(同 construction_settlements VOIDED 先例)。
-- 材料出库单:资产出库至施工项目工地的台账连续性载体(F7:出库后从台账消失);
-- 出库确认置 assets.status=IN_TRANSIT(000216 同批放宽枚举),转固 ACTIVE 凭证生效即 DEPLOYED。

BEGIN;

CREATE TABLE odn_asset_registrations (
    id          BIGSERIAL PRIMARY KEY,
    registration_no    VARCHAR(32) UNIQUE NOT NULL,     -- ZG-YYYYMMDD-NNNNN 后端生成兜底
    entity_kind VARCHAR(16) NOT NULL CHECK (entity_kind IN ('FACILITY','DEVICE')),
    facility_code VARCHAR(8),                            -- entity_kind=FACILITY 时必填 → odn_facility(code)
    device_id     BIGINT,                                -- entity_kind=DEVICE 时必填 → odn_device(id),含 W3 导入域箱体
    asset_id    BIGINT NOT NULL,                         -- 软引用 assets,跨域不加 FK(adopted 裁定)
    source_kind VARCHAR(16) NOT NULL DEFAULT 'DIRECT'
      CHECK (source_kind IN ('PROCUREMENT','CONSTRUCTION','DIRECT')),
    construction_project_id BIGINT,                      -- 软引用 construction_projects,施工建成来源溯源
    batch_id    BIGINT,                                  -- 采购溯源快照(登记时自 assets.batch_id 回填,可空)
    value_amount NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (value_amount >= 0),
    status      VARCHAR(16) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','REVERSED')),
    reverse_reason VARCHAR(255),
    remark      VARCHAR(255),
    registered_by BIGINT,
    registered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reversed_by BIGINT,
    reversed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((entity_kind = 'FACILITY' AND facility_code IS NOT NULL AND device_id IS NULL)
        OR (entity_kind = 'DEVICE'   AND device_id IS NOT NULL   AND facility_code IS NULL))
);

-- 一对象同时至多一张有效凭证(冲销后可重新登记,历史保留)
CREATE UNIQUE INDEX uq_odn_asset_reg_facility_active
    ON odn_asset_registrations (facility_code)
    WHERE entity_kind = 'FACILITY' AND status = 'ACTIVE';
CREATE UNIQUE INDEX uq_odn_asset_reg_device_active
    ON odn_asset_registrations (device_id)
    WHERE entity_kind = 'DEVICE' AND status = 'ACTIVE';
-- 一资产同时至多挂一张有效凭证(转固后资产即 DEPLOYED,不得重复登记)
CREATE UNIQUE INDEX uq_odn_asset_reg_asset_active
    ON odn_asset_registrations (asset_id)
    WHERE status = 'ACTIVE';
CREATE INDEX idx_odn_asset_reg_asset ON odn_asset_registrations (asset_id);

CREATE TABLE odn_material_issues (
    id          BIGSERIAL PRIMARY KEY,
    issue_no    VARCHAR(32) UNIQUE NOT NULL,             -- MI-YYYYMMDD-NNNNN 后端生成兜底
    project_id  BIGINT NOT NULL,                         -- 软引用 construction_projects(跨域出库对象)
    project_no  VARCHAR(32) NOT NULL,                    -- 单号快照
    status      VARCHAR(16) NOT NULL DEFAULT 'OPEN'
      CHECK (status IN ('OPEN','CONFIRMED','CANCELLED')),
    remark      VARCHAR(255),
    created_by  BIGINT,
    issued_by   BIGINT,                                  -- 出库确认人
    issued_at   TIMESTAMPTZ,
    cancelled_by BIGINT,
    cancelled_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE odn_material_issue_items (
    id          BIGSERIAL PRIMARY KEY,
    issue_id    BIGINT NOT NULL REFERENCES odn_material_issues(id) ON DELETE CASCADE,
    asset_id    BIGINT NOT NULL,                         -- 软引用 assets;逐台出库(材料类散料不在 assets 台账)
    UNIQUE (issue_id, asset_id)
);
CREATE INDEX idx_odn_material_issue_items_asset ON odn_material_issue_items (asset_id);

COMMIT;
