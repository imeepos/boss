-- 物料/工具主档(师傅端领料与借还此前用 itemId 当名称占位,线上运营需主档驱动)。
BEGIN;

CREATE TABLE material_items (
    id   BIGSERIAL PRIMARY KEY,
    code VARCHAR(32) NOT NULL UNIQUE,       -- 物料编码 MI-*
    name VARCHAR(64) NOT NULL,
    spec VARCHAR(64) NOT NULL DEFAULT '',
    unit VARCHAR(8)  NOT NULL DEFAULT '件'
);

CREATE TABLE material_tools (
    id   BIGSERIAL PRIMARY KEY,
    code VARCHAR(32) NOT NULL UNIQUE,       -- 工具编码 TL-*
    name VARCHAR(64) NOT NULL
);

-- 起步档常用物料/工具(devseed 之外的运营基线,可由 admin 侧维护)
INSERT INTO material_items(code, name, spec, unit) VALUES
    ('MI-ONT', '光纤猫', 'XGSPON 千兆版', '台'),
    ('MI-FIBER', '皮线光纤', '单芯 150m', '卷'),
    ('MI-CONNECTOR', '冷接子', 'SC/UPC', '个'),
    ('MI-SPLITTER', '分光器', '1:8', '个');
INSERT INTO material_tools(code, name) VALUES
    ('TL-OTDR', 'OTDR 测试仪'),
    ('TL-FUSION', '光纤熔接机'),
    ('TL-POWER', '光功率计'),
    ('TL-LADDER', '绝缘梯');

COMMIT;
