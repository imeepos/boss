-- W3 设施类型字典扩展(P-INFRA-1 W3):核心链路设备新增 OBD/SBD 箱内部件类型。
-- 依据《Suniway ODN 基础设施资源编码规范》V1.0 口径裁定(2.2 编码格式/2.4 箱内部件归属;
-- 3.1 同址扩容后缀),裁定全文见 docs/notes/adopted/2026-09-07-odn-box-types-import-chain.md:
-- ODF/OCC/ODB/SDB 既有设备类型不变;OBD(一级分光器,归属 ODB)/SBD(二级分光器,归属 SDB)为新增;
-- 导入域(模板批量导入)箱体设备允许无城市(prv/city 可空,模板不带城市码,红线:不猜填),
-- SNW/OLT/PRT/TBP 仍强制城市;城市设备与导入域设备经部分唯一索引分域,存量数据零改动。
BEGIN;

ALTER TABLE odn_device DROP CONSTRAINT IF EXISTS odn_device_code_check;
ALTER TABLE odn_device DROP CONSTRAINT IF EXISTS odn_device_kind_check;
ALTER TABLE odn_device ALTER COLUMN prv_code DROP NOT NULL;
ALTER TABLE odn_device ALTER COLUMN city_prefix DROP NOT NULL;
ALTER TABLE odn_device ADD CONSTRAINT odn_device_code_check
    CHECK (code ~ '^(SNW|OLT|ODF|OCC|ODB|OBD|SDB|SBD|PRT|TBP)[0-9]{3}(-([2-9]|[1-9][0-9]+))?$');
ALTER TABLE odn_device ADD CONSTRAINT odn_device_kind_check
    CHECK (kind IN ('SNW','OLT','ODF','OCC','ODB','OBD','SDB','SBD','PRT','TBP'));
ALTER TABLE odn_device ADD CONSTRAINT odn_device_city_required_check
    CHECK (kind NOT IN ('SNW','OLT','PRT','TBP') OR (prv_code IS NOT NULL AND city_prefix IS NOT NULL));
CREATE UNIQUE INDEX uq_odn_device_box ON odn_device (code) WHERE prv_code IS NULL;

COMMIT;
