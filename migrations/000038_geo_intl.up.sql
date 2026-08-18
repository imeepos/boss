-- 国际化地理基础数据:ISO 3166 国家 + ISO 3166-2 行政区划 + 多语言译名 + 时区/货币/电话码关联。
-- 依据:docs/contract/fields.md 1.5.1;调研结论(ISO 3166/UN M49/CLDR/GeoNames/TMF673)。
-- 原则:国家主键=alpha-2,区划主键=完整 ISO 3166-2 码;层级用自引用树(各国深度不同);
--       多语言一律独立译名表;停用码软删除保留(is_active=false),不物理删除。
BEGIN;

-- 国家(ISO 3166-1 + UN M49 + CLDR 关联属性)
CREATE TABLE geo_country (
    alpha2         CHAR(2) PRIMARY KEY,          -- 'PH'/'CN',全局外联主键
    alpha3         CHAR(3) NOT NULL UNIQUE,      -- 'PHL'
    numeric_code   CHAR(3) NOT NULL UNIQUE,      -- '608',=UN M49 国家码,存字符串保前导零
    short_name     VARCHAR(128) NOT NULL,        -- English short name
    full_name      VARCHAR(256),                 -- 全称,可空
    status         VARCHAR(16) NOT NULL DEFAULT 'INDEPENDENT',  -- INDEPENDENT/DISCONTINUED
    continent_code CHAR(2) NOT NULL,             -- AS/EU/NA/SA/AF/OC/AN
    m49_region     CHAR(3),                      -- M49 洲/子区域码,如 '142'(亚洲)
    postal_regex   VARCHAR(128),                 -- 邮编正则模板,地址录入校验
    is_active      BOOLEAN NOT NULL DEFAULT TRUE,-- 停用码保留:false
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_geo_country_status CHECK (status IN ('INDEPENDENT','DISCONTINUED')),
    CONSTRAINT chk_geo_country_continent CHECK (continent_code IN ('AS','EU','NA','SA','AF','OC','AN'))
);

-- 国家译名(CLDR territories + GeoNames alternateNames 语料)
CREATE TABLE geo_country_i18n (
    country_code CHAR(2) NOT NULL REFERENCES geo_country(alpha2),
    locale       VARCHAR(16) NOT NULL,           -- 'zh-Hans'/'en'/'ja'
    name         VARCHAR(256) NOT NULL,
    name_type    VARCHAR(16) NOT NULL DEFAULT 'STANDARD',  -- STANDARD/SHORT/ALIAS/HISTORIC
    CONSTRAINT uq_geo_country_i18n UNIQUE (country_code, locale, name_type),
    CONSTRAINT chk_geo_country_i18n_type CHECK (name_type IN ('STANDARD','SHORT','ALIAS','HISTORIC'))
);

-- 行政区划(ISO 3166-2 完整码 + OSM admin_level + GeoNames 挂接)
CREATE TABLE geo_subdivision (
    code            VARCHAR(16) PRIMARY KEY,     -- 'PH-NCR'/'CN-BJ'/'US-CA'
    country_code    CHAR(2) NOT NULL REFERENCES geo_country(alpha2),
    parent_code     VARCHAR(16) REFERENCES geo_subdivision(code),  -- 自引用树,各国深度不同
    level           SMALLINT NOT NULL,           -- 1=一级行政区,2=二级,以此类推
    category        VARCHAR(32) NOT NULL,        -- state/province/region/municipality/district...
    osm_admin_level SMALLINT,                    -- OSM 行政层级标尺 2~10
    geonameid       BIGINT,                      -- GeoNames 挂接(坐标/别名/人口)
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_geo_subdiv_level CHECK (level BETWEEN 1 AND 4)
);
CREATE INDEX idx_geo_subdiv_country ON geo_subdivision (country_code);
CREATE INDEX idx_geo_subdiv_parent  ON geo_subdivision (parent_code);

-- 区划译名
CREATE TABLE geo_subdivision_i18n (
    subdivision_code VARCHAR(16) NOT NULL REFERENCES geo_subdivision(code),
    locale           VARCHAR(16) NOT NULL,
    name             VARCHAR(256) NOT NULL,
    name_type        VARCHAR(16) NOT NULL DEFAULT 'STANDARD',  -- STANDARD/SHORT/ALIAS/HISTORIC/PINYIN/ROMANIZED
    CONSTRAINT uq_geo_subdiv_i18n UNIQUE (subdivision_code, locale, name_type),
    CONSTRAINT chk_geo_subdiv_i18n_type CHECK (name_type IN ('STANDARD','SHORT','ALIAS','HISTORIC','PINYIN','ROMANIZED'))
);

-- 国家关联属性:一律一对多(US/RU 多时区,EUR 多国共用,+1 一码多国)
CREATE TABLE country_time_zone (
    country_code CHAR(2) NOT NULL REFERENCES geo_country(alpha2),
    tz_name      VARCHAR(64) NOT NULL,           -- IANA 时区,如 'Asia/Manila'
    PRIMARY KEY (country_code, tz_name)
);

CREATE TABLE country_currency (
    country_code CHAR(2) NOT NULL REFERENCES geo_country(alpha2),
    currency     CHAR(3) NOT NULL,               -- ISO 4217,如 'PHP'
    is_primary   BOOLEAN NOT NULL DEFAULT TRUE,
    minor_unit   SMALLINT NOT NULL DEFAULT 2,    -- 小数位:JPY=0,USD=2
    PRIMARY KEY (country_code, currency)
);

CREATE TABLE country_calling_code (
    country_code CHAR(2) NOT NULL REFERENCES geo_country(alpha2),
    calling_code VARCHAR(8) NOT NULL,            -- E.164 国家码,如 '63'
    PRIMARY KEY (country_code, calling_code)
);

-- addresses 国际化挂接:补国家与行政区锚点(path 权威不变,见 ADR-002)
ALTER TABLE addresses
    ADD COLUMN country_code CHAR(2) REFERENCES geo_country(alpha2),      -- 归属国家,空=历史数据未挂
    ADD COLUMN admin_code   VARCHAR(16) REFERENCES geo_subdivision(code);-- 一级行政区锚点,如 'PH-NCR'
CREATE INDEX idx_addresses_country ON addresses (country_code);

COMMIT;

-- 种子:最小国际基线(PH 主营 + CN/US 演示),幂等。数据源 ISO 3166/CLDR。
BEGIN;
INSERT INTO geo_country (alpha2, alpha3, numeric_code, short_name, full_name, status, continent_code, m49_region) VALUES
    ('PH','PHL','608','Philippines','Republic of the Philippines','INDEPENDENT','AS','035'),
    ('CN','CHN','156','China','People''s Republic of China','INDEPENDENT','AS','142'),
    ('US','USA','840','United States','United States of America','INDEPENDENT','NA','021')
ON CONFLICT (alpha2) DO NOTHING;

INSERT INTO geo_country_i18n (country_code, locale, name) VALUES
    ('PH','zh-Hans','菲律宾'),('PH','en','Philippines'),
    ('CN','zh-Hans','中国'),  ('CN','en','China'),
    ('US','zh-Hans','美国'),  ('US','en','United States')
ON CONFLICT DO NOTHING;

INSERT INTO geo_subdivision (code, country_code, level, category, osm_admin_level) VALUES
    ('PH-NCR',   'PH', 1, 'region',            4),  -- 国家首都区(马尼拉都会区)
    ('PH-40',    'PH', 1, 'region',            4),  -- Calabarzon 大区(吕宋南部)
    ('CN-BJ',    'CN', 1, 'municipality',      4),
    ('CN-SH',    'CN', 1, 'municipality',      4),
    ('US-CA',    'US', 1, 'state',             4),
    ('US-NY',    'US', 1, 'state',             4)
ON CONFLICT (code) DO NOTHING;

INSERT INTO geo_subdivision_i18n (subdivision_code, locale, name) VALUES
    ('PH-NCR','zh-Hans','国家首都区'),('PH-NCR','en','National Capital Region'),
    ('PH-40','zh-Hans','卡拉巴松大区'),('PH-40','en','Calabarzon'),
    ('CN-BJ', 'zh-Hans','北京'),      ('CN-BJ', 'en','Beijing'),
    ('CN-SH', 'zh-Hans','上海'),      ('CN-SH', 'en','Shanghai'),
    ('US-CA', 'zh-Hans','加利福尼亚州'),('US-CA','en','California'),
    ('US-NY', 'zh-Hans','纽约州'),    ('US-NY','en','New York')
ON CONFLICT DO NOTHING;

INSERT INTO country_time_zone (country_code, tz_name) VALUES
    ('PH','Asia/Manila'),('CN','Asia/Shanghai'),
    ('US','America/New_York'),('US','America/Chicago'),('US','America/Los_Angeles')
ON CONFLICT DO NOTHING;

INSERT INTO country_currency (country_code, currency, is_primary, minor_unit) VALUES
    ('PH','PHP',TRUE,2),('CN','CNY',TRUE,2),('US','USD',TRUE,2)
ON CONFLICT DO NOTHING;

INSERT INTO country_calling_code (country_code, calling_code) VALUES
    ('PH','63'),('CN','86'),('US','1')
ON CONFLICT DO NOTHING;

COMMIT;
