-- 000163: 采购-库存域新建(procurement 域增量挂靠,不开阶段 10)。
-- 4 张表 + asset_batches 字段补丁 + GIS 仓库点坐标。
-- 决策依据:docs/notes/adopted/2026-08-28-procurement-install-gis-linkage.md §决策 1。
-- 数据分层:procurement.* 在 L2-L3 之间(供应商 L1.5 / 采购单 L2 / 入库单 L4)。

-- 供应商(procurement.suppliers):公司自定义基础数据,挂 legal_entity。
CREATE TABLE procurement_suppliers (
    id              BIGSERIAL PRIMARY KEY,
    code            VARCHAR(64)  NOT NULL UNIQUE,
    name            VARCHAR(128) NOT NULL,
    contact_name    VARCHAR(64),
    contact_phone   VARCHAR(32),
    legal_entity_id BIGINT NOT NULL REFERENCES legal_entities(id),
    status          VARCHAR(16) NOT NULL DEFAULT 'ENABLED'
                    CHECK (status IN ('ENABLED','DISABLED')),
    remark          VARCHAR(255),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_procurement_suppliers_entity ON procurement_suppliers(legal_entity_id);

-- 采购单头(procurement.procurement_orders):
-- 状态机 DRAFT→SUBMITTED→PARTIAL→RECEIVED;任意点 CANCELLED。
CREATE TABLE procurement_orders (
    id                BIGSERIAL PRIMARY KEY,
    procurement_no    VARCHAR(32) NOT NULL UNIQUE,
    legal_entity_id   BIGINT NOT NULL REFERENCES legal_entities(id),
    legal_entity_name VARCHAR(128) NOT NULL,
    supplier_id       BIGINT NOT NULL REFERENCES procurement_suppliers(id),
    supplier_name     VARCHAR(128) NOT NULL,
    status            VARCHAR(16) NOT NULL DEFAULT 'DRAFT'
                      CHECK (status IN ('DRAFT','SUBMITTED','PARTIAL','RECEIVED','CANCELLED')),
    total_amount      NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (total_amount >= 0),
    expected_date     DATE,
    remark            VARCHAR(255),
    created_by        BIGINT REFERENCES accounts(id),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    submitted_at      TIMESTAMPTZ,
    received_at       TIMESTAMPTZ,
    cancelled_at      TIMESTAMPTZ
);
CREATE INDEX idx_procurement_orders_entity ON procurement_orders(legal_entity_id);
CREATE INDEX idx_procurement_orders_supplier ON procurement_orders(supplier_id);
CREATE INDEX idx_procurement_orders_status ON procurement_orders(status);

-- 采购单明细(procurement.procurement_order_items):
-- 物料类型对齐 material_items.code(全局主档,L0,见 data-layers §1)。
CREATE TABLE procurement_order_items (
    id              BIGSERIAL PRIMARY KEY,
    order_id        BIGINT NOT NULL REFERENCES procurement_orders(id) ON DELETE CASCADE,
    material_code   VARCHAR(64) NOT NULL,
    spec            VARCHAR(128),
    quantity        INTEGER NOT NULL CHECK (quantity > 0),
    received_qty    INTEGER NOT NULL DEFAULT 0 CHECK (received_qty >= 0),
    unit_amount     NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (unit_amount >= 0),
    remark          VARCHAR(255),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (received_qty <= quantity)
);
CREATE INDEX idx_procurement_order_items_order ON procurement_order_items(order_id);

-- 到货入库单(procurement.procurement_receipts):
-- 同事务建 asset_batches + 逐台 assets IN_STOCK;receipt.status 走 CONFIRMED 后才落台账。
-- batch_id 入库后回填,事前 NULL。
CREATE TABLE procurement_receipts (
    id              BIGSERIAL PRIMARY KEY,
    receipt_no      VARCHAR(32) NOT NULL UNIQUE,
    order_id        BIGINT NOT NULL REFERENCES procurement_orders(id),
    order_no        VARCHAR(32) NOT NULL,
    batch_id        BIGINT REFERENCES asset_batches(id),
    legal_entity_id BIGINT NOT NULL REFERENCES legal_entities(id),
    legal_entity_name VARCHAR(128) NOT NULL,
    received_by     BIGINT REFERENCES accounts(id),
    received_at     TIMESTAMPTZ,
    status          VARCHAR(16) NOT NULL DEFAULT 'DRAFT'
                    CHECK (status IN ('DRAFT','CONFIRMED','REJECTED')),
    remark          VARCHAR(255),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_procurement_receipts_order ON procurement_receipts(order_id);
CREATE INDEX idx_procurement_receipts_status ON procurement_receipts(status);

-- GIS 仓库点坐标(决策 1 §GIS 图层①):仓库挂可空 lat/lng 即可,无坐标行过滤。
-- 仓库实体最小化:不另建 warehouses 表,只补 asset_batches 坐标列(L4 业务主单+GIS 投影共用)。
ALTER TABLE asset_batches
    ADD COLUMN IF NOT EXISTS warehouse_lat DOUBLE PRECISION
        CHECK (warehouse_lat IS NULL OR warehouse_lat BETWEEN -90 AND 90),
    ADD COLUMN IF NOT EXISTS warehouse_lng DOUBLE PRECISION
        CHECK (warehouse_lng IS NULL OR warehouse_lng BETWEEN -180 AND 180);

COMMENT ON TABLE  procurement_suppliers        IS '供应商(L1.5 公司自定义基础数据)';
COMMENT ON TABLE  procurement_orders          IS '采购单头(procurement 域主单,L2)';
COMMENT ON TABLE  procurement_order_items     IS '采购单明细(L3)';
COMMENT ON TABLE  procurement_receipts        IS '到货入库单(L4,与 asset_batches/资产双写)';
COMMENT ON COLUMN asset_batches.warehouse_lat IS '仓库 GPS 纬度(GIS 库存分布图层用,可空)';
COMMENT ON COLUMN asset_batches.warehouse_lng IS '仓库 GPS 经度(GIS 库存分布图层用,可空)';
