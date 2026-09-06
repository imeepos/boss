-- ODN P2 物理端口占用态(路线图 T11):分光器/终端盒端口级资源,订单预占的物理落地。
-- order_id 为订单软引用(独立命名空间裁定 E10/E11,不加 FK);RETIRED 设备端口随设备退役不可用。
BEGIN;

CREATE TABLE odn_port (
    id         BIGSERIAL PRIMARY KEY,
    device_id  BIGINT      NOT NULL REFERENCES odn_device(id),
    port_no    SMALLINT    NOT NULL CHECK (port_no BETWEEN 1 AND 99),
    status     VARCHAR(16) NOT NULL DEFAULT 'IDLE'
               CHECK (status IN ('IDLE','RESERVED','IN_SERVICE')),
    order_id   BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (device_id, port_no)
);
CREATE INDEX idx_odn_port_status ON odn_port (device_id, status);

COMMIT;
