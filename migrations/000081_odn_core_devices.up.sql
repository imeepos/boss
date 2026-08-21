-- ODN 局点(NodeCode 序号段)与核心链路设备(《Suniway ODN 基础设施资源编码规范》第 2/3 章)。
-- NodeCode = 城市前缀 + 3 位局点序号(如 MNL001);设备码 = 类型前缀 + 3 位编号,
-- 同址扩容后缀 -N(N>=2,禁 -1);编号隔离域:SNW 全网唯一,OLT/ODF/OCC 市域唯一,
-- ODB 归属 OCC / SDB 归属 ODB / PRT 归属 SDB / TBP 归属 PRT(parent 链)。
BEGIN;

CREATE TABLE odn_site (
    prv_code     CHAR(6)     NOT NULL,
    city_prefix  VARCHAR(5)  NOT NULL,
    site_no      SMALLINT    NOT NULL CHECK (site_no BETWEEN 1 AND 999), -- NodeCode 序号段
    name         VARCHAR(64) NOT NULL DEFAULT '',
    lat          DOUBLE PRECISION,
    lng          DOUBLE PRECISION,
    status       VARCHAR(16) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','RETIRED')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (prv_code, city_prefix, site_no),
    FOREIGN KEY (prv_code, city_prefix) REFERENCES odn_city_code (prv_code, city_prefix)
);

CREATE TABLE odn_device (
    id          BIGSERIAL PRIMARY KEY,
    code        VARCHAR(12) NOT NULL CHECK (code ~ '^(SNW|OLT|ODF|OCC|ODB|SDB|PRT|TBP)[0-9]{3}(-([2-9]|[1-9][0-9]+))?$'),
    kind        VARCHAR(3)  NOT NULL CHECK (kind IN ('SNW','OLT','ODF','OCC','ODB','SDB','PRT','TBP')),
    prv_code    CHAR(6)     NOT NULL,
    city_prefix VARCHAR(5)  NOT NULL,
    site_no     SMALLINT    CHECK (site_no IS NULL OR site_no BETWEEN 1 AND 999), -- 归属局点,可空(市域设备)
    parent_id   BIGINT      REFERENCES odn_device(id), -- ODB→OCC/SDB→ODB/PRT→SDB/TBP→PRT;顶层 NULL
    name        VARCHAR(64) NOT NULL DEFAULT '',
    status      VARCHAR(16) NOT NULL DEFAULT 'IN_USE' CHECK (status IN ('IN_USE','RETIRED')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (prv_code, city_prefix) REFERENCES odn_city_code (prv_code, city_prefix)
);
-- 编号隔离域(规范 2.2):市域内唯一;SNW 全网唯一。
CREATE UNIQUE INDEX uq_odn_device_city ON odn_device (prv_code, city_prefix, code);
CREATE UNIQUE INDEX uq_odn_device_snw  ON odn_device (code) WHERE kind = 'SNW';
CREATE INDEX idx_odn_device_parent ON odn_device (parent_id);
CREATE INDEX idx_odn_device_kind ON odn_device (kind, status);

-- 报废设备编码永久锁定(资产编码规范红线 2),应用层校验禁复用。

COMMIT;
