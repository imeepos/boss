-- 订单归属由安装地址判定(adopted note 2026-08-20-order-legal-entity-by-address):
-- 地址挂经营区域(addresses.region_id),区域挂运营主体覆盖(regions.legal_entity_id,可只挂上层节点,
-- 下单时沿 path 向上取最近非空祖先)。两列均可空:未配置区域或区域未覆盖 → 下单拒单。
BEGIN;

ALTER TABLE regions
    ADD COLUMN legal_entity_id BIGINT REFERENCES legal_entities(id); -- 覆盖运营主体,空=继承祖先
CREATE INDEX idx_regions_legal_entity ON regions (legal_entity_id);

ALTER TABLE addresses
    ADD COLUMN region_id BIGINT REFERENCES regions(id); -- 装机地址所在经营区域
CREATE INDEX idx_addresses_region ON addresses (region_id);

COMMIT;
