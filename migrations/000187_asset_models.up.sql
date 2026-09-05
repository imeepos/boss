-- 000187: 资产型号字典(P1-T3,adopted note 2026-09-06-asset-tag-p1-wave)。
-- 形态依据 R3 调研(NetBox DeviceType/GLPI models/Snipe-IT models+fieldsets):
-- 型号层唯一约束防重(合并重复型号=改名重指,停用不物理删),规格收 spec JSONB
-- 不做动态物理列(Snipe-IT _snipeit_* 物理列方案弃用);assets.type 保留为展示冗余,
-- model_id 可空回填(映射不到的留 NULL 出待办,禁止程序猜测合并)。
-- 存量口径:按 DISTINCT type 播种(vendor='(存量未登记)',category=type 原值),
-- '光猫'与'ONU'是否归并为 category='ONU' 属业务裁定,裁定前种子并存、不合并。

BEGIN;

CREATE TABLE asset_models (
    id          BIGSERIAL PRIMARY KEY,
    vendor      VARCHAR(64)  NOT NULL DEFAULT '',
    model       VARCHAR(128) NOT NULL,
    category    VARCHAR(32)  NOT NULL,
    part_number VARCHAR(64)  NOT NULL DEFAULT '',
    spec        JSONB        NOT NULL DEFAULT '{}'::jsonb,
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_asset_models UNIQUE (vendor, model, category, part_number)
);
CREATE INDEX idx_asset_models_active ON asset_models(is_active);

ALTER TABLE assets ADD COLUMN model_id BIGINT REFERENCES asset_models(id);

-- 存量播种:每个既有 type 值一个字典行,并按精确匹配回填 model_id(空串已由 000184 清洗)。
INSERT INTO asset_models(vendor, model, category)
SELECT '(存量未登记)', t, t
  FROM (SELECT DISTINCT type AS t FROM assets WHERE type <> '') s
ON CONFLICT DO NOTHING;

UPDATE assets a SET model_id = m.id
  FROM asset_models m
 WHERE m.vendor = '(存量未登记)' AND m.model = a.type AND a.model_id IS NULL;

COMMIT;
