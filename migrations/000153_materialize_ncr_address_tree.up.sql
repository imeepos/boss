-- NCR(国家首都区)地址层级物化:geo_subdivision PSGC → addresses 权威树(ADR-002)。
-- 层级映射:NCR大区→L1(锚点 PH/PH-1300000000) · 17市→L2 · 全部 Barangay→L3
--   (PSGC 里挂 sub-municipality 下的 Barangay 拍平到市,PH 邮政口径不用副市级区)。
-- path 标签 = PSGC 码去横杠小写(ltree 标签合法且全局唯一);名称取 i18n,
--   根用 zh-Hans 通行译名,市/Barangay 用 en 官方名(000041 口径)。
-- parent_id/level 服务层派生口径同 ImportAddresses:level=nlevel(path)、
--   parent_id=path 父段反查;ON CONFLICT DO NOTHING 幂等可重放。
BEGIN;

-- L1 根:NCR 大区,挂国际锚点
INSERT INTO addresses(path, level, name, parent_id, country_code, admin_code)
SELECT 'ph1300000000'::ltree, 1,
       COALESCE((SELECT i.name FROM geo_subdivision_i18n i
                  WHERE i.subdivision_code = s.code AND i.locale = 'zh-Hans'), s2.en_name),
       NULL, 'PH', 'PH-1300000000'
FROM geo_subdivision s
LEFT JOIN LATERAL (SELECT i.name AS en_name FROM geo_subdivision_i18n i
                    WHERE i.subdivision_code = s.code AND i.locale = 'en') s2 ON TRUE
WHERE s.code = 'PH-1300000000'
ON CONFLICT (path) DO NOTHING;

-- L2:NCR 直辖市/市(parent 反查根)
INSERT INTO addresses(path, level, name, parent_id)
SELECT ('ph1300000000.' || replace(lower(s.code), '-', ''))::ltree, 2,
       i.name,
       (SELECT a.id FROM addresses a WHERE a.path = 'ph1300000000'::ltree)
FROM geo_subdivision s
JOIN geo_subdivision_i18n i ON i.subdivision_code = s.code AND i.locale = 'en'
WHERE s.parent_code = 'PH-1300000000'
ON CONFLICT (path) DO NOTHING;

-- L3:全部 Barangay(含挂 sub-municipality 的),城市归属沿 parent_code 上溯到 level=2
WITH RECURSIVE up AS (
    SELECT s.code, s.parent_code, s.level, s.code AS origin
    FROM geo_subdivision s
    WHERE s.code LIKE 'PH-13%'
      AND s.category = 'barangay'
    UNION ALL
    SELECT p.code, p.parent_code, p.level, u.origin
    FROM geo_subdivision p
    JOIN up u ON p.code = u.parent_code
    WHERE p.level >= 2
)
INSERT INTO addresses(path, level, name, parent_id)
SELECT ('ph1300000000.' || replace(lower(city.code), '-', '')
        || '.' || replace(lower(b.code), '-', ''))::ltree, 3,
       i.name,
       (SELECT a.id FROM addresses a
         WHERE a.path = ('ph1300000000.' || replace(lower(city.code), '-', ''))::ltree)
FROM (SELECT DISTINCT ON (origin) origin, code AS city_code FROM up WHERE level = 2
      ORDER BY origin) m
JOIN geo_subdivision b ON b.code = m.origin
JOIN geo_subdivision city ON city.code = m.city_code
JOIN geo_subdivision_i18n i ON i.subdivision_code = b.code AND i.locale = 'en'
ON CONFLICT (path) DO NOTHING;

COMMIT;
