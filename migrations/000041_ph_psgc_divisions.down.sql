-- 回滚:移除菲律宾 PSGC 全量数据,并恢复 000038 的 PH 演示行。
BEGIN;
DELETE FROM geo_subdivision_i18n WHERE subdivision_code ~ '^PH-[01][0-9]{9}$';
DELETE FROM geo_subdivision WHERE code ~ '^PH-[01][0-9]{9}$';
INSERT INTO geo_subdivision (code, country_code, level, category, osm_admin_level) VALUES
    ('PH-NCR','PH',1,'region',4),
    ('PH-40','PH',1,'region',4)
ON CONFLICT (code) DO NOTHING;
INSERT INTO geo_subdivision_i18n (subdivision_code, locale, name) VALUES
    ('PH-NCR','zh-Hans','国家首都区'),('PH-NCR','en','National Capital Region'),
    ('PH-40','zh-Hans','卡拉巴松大区'),('PH-40','en','Calabarzon')
ON CONFLICT DO NOTHING;
COMMIT;
