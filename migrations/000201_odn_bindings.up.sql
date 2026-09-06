-- ODN P3 逻辑-物理绑定(路线图 T13):装机激活后,订单/逻辑端口 ↔ ODN 物理端口 的绑定事实。
-- 一口一绑定(port_id UNIQUE);resource_port_id 为逻辑资源口软引用(独立命名空间裁定,不加 FK)。
BEGIN;

CREATE TABLE odn_bindings (
    id                BIGSERIAL PRIMARY KEY,
    port_id           BIGINT      NOT NULL REFERENCES odn_port(id),
    order_id          BIGINT      NOT NULL,
    resource_port_id  BIGINT,
    note              VARCHAR(255) NOT NULL DEFAULT '',
    bound_by          BIGINT REFERENCES accounts(id),
    bound_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (port_id)
);
CREATE INDEX idx_odn_bindings_order ON odn_bindings (order_id);

COMMIT;
