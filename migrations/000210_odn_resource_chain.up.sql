-- W3 ODN 资源链实体(P-INFRA-1 W3;需求源 docs/ODN网络资源表模板.xlsx):
-- 一行=模板一条完整资源链(23 列),导入时展开为系统关系数据(箱体设备落 odn_device,迁移 000209);
-- 指纹唯一约束承载说明页规则⑥(一行一条资源关系,重复行去重);
-- 枚举列存系统状态码(枚举 sheet 为校验字典,口径 terms.md/fields.md 1.5.12)。
BEGIN;

CREATE TABLE odn_resource_chain (
    id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    batch_no         VARCHAR(40)  NOT NULL,
    line_no          INTEGER      NOT NULL, -- 模板行号(可观测定位,含示例行偏移)
    lifecycle_status VARCHAR(16)  NOT NULL DEFAULT 'PLANNED'
        CHECK (lifecycle_status IN ('PLANNED','IN_BUILD','IN_SERVICE','RETIRED')),
    site_code        VARCHAR(32)  NOT NULL DEFAULT '',
    site_name        VARCHAR(64)  NOT NULL DEFAULT '',
    olt_code         VARCHAR(40)  NOT NULL DEFAULT '',
    odf_code         VARCHAR(24),
    odf_port         VARCHAR(16),
    occ_code         VARCHAR(12),
    odb_code         VARCHAR(12),
    obd_code         VARCHAR(12),
    split1_ratio     SMALLINT,
    split1_port      VARCHAR(16),
    sdb_code         VARCHAR(12),
    sbd_code         VARCHAR(12),
    split2_ratio     SMALLINT,
    split2_port      VARCHAR(16),
    total_split      INTEGER,
    fiber_code       VARCHAR(64),
    fr_to            VARCHAR(128),
    port_status      VARCHAR(16)
        CHECK (port_status IS NULL OR port_status IN ('IDLE','RESERVED','USED','DISABLED')),
    laying_method    VARCHAR(16)
        CHECK (laying_method IS NULL OR laying_method IN ('AERIAL','UNDERGROUND','SUBMARINE','MICROTRENCH','INDOOR')),
    row_status       VARCHAR(16)
        CHECK (row_status IS NULL OR row_status IN ('NOT_STARTED','PENDING','APPROVED','EXPIRED','NA')),
    pece_status      VARCHAR(16)
        CHECK (pece_status IS NULL OR pece_status IN ('PENDING_SIGN','SIGNED','STAMPED','NA')),
    remark           VARCHAR(255),
    fingerprint      CHAR(64)     NOT NULL, -- 23 列归一化指纹(重复行去重)
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_odn_chain_fingerprint UNIQUE (fingerprint)
);
CREATE INDEX idx_odn_chain_batch ON odn_resource_chain (batch_no);
CREATE INDEX idx_odn_chain_occ ON odn_resource_chain (occ_code);
CREATE INDEX idx_odn_chain_odb ON odn_resource_chain (odb_code);
CREATE INDEX idx_odn_chain_sdb ON odn_resource_chain (sdb_code);

COMMIT;
