-- ODN 无源物理层(网格分区 + 电杆/人井/铁塔/接头盒/终端盒),依据《Suniway ODN 地理空间编码规范》V1.0 第 4 章。
-- E6~E9 落地;PRV/城市前缀字典见 000075。红线 2:网格须备案;红线 3:5 位数字;编码报废永久锁定(软删)。
BEGIN;

CREATE TABLE odn_grid (
    prv_code     CHAR(6)     NOT NULL,
    city_prefix  VARCHAR(5)  NOT NULL,
    grid_code    SMALLINT    NOT NULL CHECK (grid_code BETWEEN 1 AND 99),
    name         VARCHAR(64) NOT NULL DEFAULT '',
    coverage     VARCHAR(255) NOT NULL DEFAULT '',
    status       VARCHAR(16) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','RESERVED','RETIRED')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (prv_code, city_prefix, grid_code),
    FOREIGN KEY (prv_code, city_prefix) REFERENCES odn_city_code (prv_code, city_prefix)
);

-- 单城市网格数 01~99,>=90 触发细分评估由应用层预警(规范 4.7)。

CREATE TABLE odn_facility (
    code        VARCHAR(8)  PRIMARY KEY CHECK (code ~ '^(P|MH|TW|CLS|TBX)[0-9]{5}$'), -- 红线 3:5 位数字
    kind        VARCHAR(3)  NOT NULL CHECK (kind IN ('P','MH','TW','CLS','TBX')),
    prv_code    CHAR(6)     NOT NULL,
    city_prefix VARCHAR(5)  NOT NULL,
    grid_code   SMALLINT, -- P/MH=所属网格 01~99;TW/CLS/TBX=NULL(市域顺序,FK 对 NULL 不生效)
    name        VARCHAR(64) NOT NULL DEFAULT '',
    lat         DOUBLE PRECISION,
    lng         DOUBLE PRECISION,
    status      VARCHAR(16) NOT NULL DEFAULT 'IN_USE' CHECK (status IN ('IN_USE','RETIRED')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (prv_code, city_prefix) REFERENCES odn_city_code (prv_code, city_prefix),
    FOREIGN KEY (prv_code, city_prefix, grid_code) REFERENCES odn_grid (prv_code, city_prefix, grid_code)
);
CREATE INDEX idx_odn_facility_grid ON odn_facility (prv_code, city_prefix, grid_code) WHERE grid_code IS NOT NULL;
CREATE INDEX idx_odn_facility_kind ON odn_facility (kind, status);

-- code 主键即全网唯一(P/MH 编码含网格段,天然不跨网格重号);TW/CLS/TBX 市域顺序由主键保证。

-- 报废设施编码永久锁定:status=RETIRED 保留行,禁止改回/复用(应用层校验)。

COMMIT;
