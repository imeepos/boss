-- addresses 国际化挂接闭环(承接 000038 新增列):根节点锚点约束 + 继承视图 + 存量回填。
-- 原则(docs/contract/fields.md 1.5.1 / ADR-002):path 权威不变;锚点只挂 level=1 根节点,
-- 子节点经 v_addresses_geo 视图按 path 根前缀继承;admin_code 可空,但必须与国家前缀一致。
BEGIN;

-- 1. 锚点只允许出现在根节点(level=1)
ALTER TABLE addresses ADD CONSTRAINT chk_addresses_geo_root
    CHECK ((country_code IS NULL AND admin_code IS NULL) OR level = 1);

-- 2. 区划锚必须归属同国家(ISO 3166-2 码前缀即国家码)
ALTER TABLE addresses ADD CONSTRAINT chk_addresses_geo_match
    CHECK (admin_code IS NULL
        OR (country_code IS NOT NULL AND admin_code LIKE country_code || '-%'));

-- 3. 继承视图:任意节点 → 所在树根的锚点(subpath(path,0,1) 恒为根,自包含无递归)
CREATE VIEW v_addresses_geo AS
SELECT a.id, a.path, a.level, a.name, a.parent_id,
       r.country_code, r.admin_code
FROM addresses a
JOIN addresses r ON r.path = subpath(a.path, 0, 1);

-- 4. 未挂接工作清单索引(回填工作台查询)
CREATE INDEX idx_addresses_geo_unlinked ON addresses (id)
    WHERE level = 1 AND country_code IS NULL;

-- 5. 存量回填:现有地址树均为中国口径;可识别的省级 ISO 码补 admin_code,其余留空进后台清单
UPDATE addresses SET country_code = 'CN'
WHERE level = 1 AND country_code IS NULL;

UPDATE addresses SET admin_code = 'CN-BJ' WHERE level = 1 AND path::text = 'bj';
UPDATE addresses SET admin_code = 'CN-SH' WHERE level = 1 AND path::text = 'sh';

COMMIT;
