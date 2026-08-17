-- 阶段5:订单渠道目录(下单来源,REQ-ORD-006 必填不可改)。
-- 字段权威:server-ts/src/entities/customer.ts(Channel)。
BEGIN;

CREATE TABLE channels (
    id     BIGSERIAL PRIMARY KEY,
    code   VARCHAR(32) NOT NULL UNIQUE,   -- HALL营业厅/ONLINE线上/AGENT代理商
    name   VARCHAR(64) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'ACTIVE'  -- ACTIVE启用/DISABLED停用
);

COMMIT;
