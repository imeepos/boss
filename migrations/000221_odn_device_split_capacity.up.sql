-- 000221(P-INFRA-1 W5,审查 F1 分光比建模): odn_device_split_capacity 设备分光容量模型。
-- odn_resource_chain 暂存的一级/二级分光比(W3,000210)经回写端点幂等重建为设备级容量事实:
-- ratio=分光比分母(端口容量),used_ports=链行端口标签去重占用,has_secondary 供户级口径
-- 区分「下挂二级链的一级器」(端口不直接到户,不计户级潜在户数)与直达户场景。
-- 口径与裁定: adopted 2026-09-07-split-capacity-investment-depth;fields.md 1.5.15。
BEGIN;

CREATE TABLE odn_device_split_capacity (
    device_id     BIGINT PRIMARY KEY REFERENCES odn_device(id) ON DELETE CASCADE,
    split_level   SMALLINT    NOT NULL CHECK (split_level IN (1,2)),
    ratio         SMALLINT    NOT NULL CHECK (ratio BETWEEN 2 AND 128),
    chain_rows    INTEGER     NOT NULL DEFAULT 0,
    used_ports    INTEGER     NOT NULL DEFAULT 0 CHECK (used_ports >= 0),
    has_secondary BOOLEAN     NOT NULL DEFAULT FALSE,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE  odn_device_split_capacity IS '设备分光容量模型(W5,000221):资源链暂存分光比回写,幂等重建';
COMMENT ON COLUMN odn_device_split_capacity.split_level   IS '1=一级分光器(OBD) 2=二级分光器(SBD)';
COMMENT ON COLUMN odn_device_split_capacity.ratio         IS '分光比分母=端口容量(2~128)';
COMMENT ON COLUMN odn_device_split_capacity.used_ports    IS '已用端口占用=链行端口标签去重计数';
COMMENT ON COLUMN odn_device_split_capacity.has_secondary IS '一级器下挂二级分光链(端口不到户,户级口径跳过)';

COMMIT;
