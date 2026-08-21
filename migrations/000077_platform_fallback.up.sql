-- 总公司兜底(用户裁定 2026-08-20):区域未覆盖的订单交给平台(总公司)处理。
-- 机制:legal_entities 增 is_platform 标志(全库唯一),root 根区域挂总公司覆盖;
-- 子公司在更深层节点覆盖时优先命中(最近祖先),全链未覆盖/地址未挂区域 → 总公司 admin。
BEGIN;

ALTER TABLE legal_entities
    ADD COLUMN is_platform BOOLEAN NOT NULL DEFAULT false;

-- 平台总公司全局唯一
CREATE UNIQUE INDEX uq_legal_entities_platform
    ON legal_entities (is_platform) WHERE is_platform;

-- 种子:平台总公司(幂等)
INSERT INTO legal_entities (code, name, is_platform)
VALUES ('LEG-PLAT', '平台总公司', true)
ON CONFLICT (code) DO NOTHING;

-- root 根区域挂总公司覆盖:未被子公司覆盖的区域沿祖先链兜底到这里
UPDATE regions
SET legal_entity_id = (SELECT id FROM legal_entities WHERE is_platform)
WHERE path = 'root' AND legal_entity_id IS NULL;

COMMIT;
