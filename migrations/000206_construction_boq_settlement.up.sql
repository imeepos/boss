-- 000206(原预分配 000204,随 000205 让号连号顺延): 工程量清单与工程结算(P-INFRA-1 W1)。
-- 1) construction_items 升级工程量清单:数量/单价可带,金额为 STORED 生成列
--    (round(quantity*unit_price,2)),后端计算由数据库兜底,任何写路径不可直写金额。
-- 2) construction_settlements 结算单:项目 ACCEPTED 且已指定承包商方可发起;
--    状态机 PENDING→SETTLED / PENDING|SETTLED→VOIDED(需原因);VOIDED 终态。
--    同项目同时最多一张有效结算单(部分唯一索引);作废后重开以新单表达,原单保留历史。
BEGIN;

ALTER TABLE construction_items
    ADD COLUMN quantity   NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    ADD COLUMN unit_price NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (unit_price >= 0),
    ADD COLUMN amount     NUMERIC(14,2) GENERATED ALWAYS AS (round(quantity * unit_price, 2)) STORED;

COMMENT ON COLUMN construction_items.quantity   IS '工程量(可空,默认 0=未定额;存量明细兼容)';
COMMENT ON COLUMN construction_items.unit_price IS '单价(可空,默认 0)';
COMMENT ON COLUMN construction_items.amount     IS '金额=round(quantity*unit_price,2),STORED 生成列,禁止直写(000206)';

CREATE TABLE construction_settlements (
    id              BIGSERIAL PRIMARY KEY,
    settlement_no   VARCHAR(32) NOT NULL UNIQUE,
    project_id      BIGINT NOT NULL REFERENCES construction_projects(id),
    project_no      VARCHAR(32) NOT NULL,
    contractor_id   BIGINT,
    contractor_name VARCHAR(128) NOT NULL DEFAULT '',
    total_amount    NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (total_amount >= 0),
    item_count      INTEGER NOT NULL DEFAULT 0,
    status          VARCHAR(16) NOT NULL DEFAULT 'PENDING'
                    CHECK (status IN ('PENDING','SETTLED','VOIDED')),
    void_reason     VARCHAR(255) NOT NULL DEFAULT '',
    created_by      BIGINT REFERENCES accounts(id),
    settled_by      BIGINT REFERENCES accounts(id),
    voided_by       BIGINT REFERENCES accounts(id),
    settled_at      TIMESTAMPTZ,
    voided_at       TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (project_id) REFERENCES construction_projects(id)
);
CREATE INDEX idx_construction_settlements_project ON construction_settlements(project_id);
CREATE INDEX idx_construction_settlements_status ON construction_settlements(status);
-- 同项目同时最多一张有效(非作废)结算单;作废行保留历史不限。
CREATE UNIQUE INDEX uq_construction_settlements_project_active
    ON construction_settlements(project_id) WHERE status <> 'VOIDED';

COMMENT ON TABLE  construction_settlements                 IS '工程结算单(项目 ACCEPTED 后汇总清单金额形成应付,000206)';
COMMENT ON COLUMN construction_settlements.contractor_id   IS '承包商软引用 → procurement_suppliers(id),名称快照;跨域不加 FK(adopted 2026-09-07)';
COMMENT ON COLUMN construction_settlements.total_amount    IS '应付金额=发起时 SUM(construction_items.amount);ACCEPTED 后明细锁定不漂移';
COMMENT ON COLUMN construction_settlements.void_reason     IS '作废原因(作废必填)';

COMMIT;