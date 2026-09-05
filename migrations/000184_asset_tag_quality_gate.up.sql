-- 000184: 资产/标签数据质量闸门——状态枚举 CHECK + 存量脏行清洗。
-- 背景(2026-09-06 真库实证 102):assets 1 条 status=''/type=''(fixture 残留 #190),
-- tags 2 条 status=''(#161/#186)。terms.md §4 枚举为契约权威,但 DB 层无约束,
-- 脏值可静默入库——成熟 ITAM/CMDB 惯例:枚举列必须有 DB CHECK 兜底(应用层校验可绕过)。
-- 清洗裁定(adopted note 2026-09-06-asset-tag-quality-gate):
--   fixture 残留行按默认态归一 assets.status→IN_STOCK / assets.type→ONU /
--   tags.status→UNBOUND,不做业务推断;'光猫'与'ONU'并存是存量口径,字典化归一见
--   docs/design/asset-tag-research-mature-designs.md P1,本次不合并。

BEGIN;

-- 存量清洗(幂等:仅命中非法值)
UPDATE assets SET status = 'IN_STOCK'
 WHERE status NOT IN ('IN_STOCK','DEPLOYED','MAINTENANCE','SCRAPPED');
UPDATE assets SET type = 'ONU' WHERE type = '';
UPDATE tags SET status = 'UNBOUND'
 WHERE status NOT IN ('UNBOUND','BOUND','DISABLED');

-- 枚举闸门(terms.md §4:asset.status 四态 / tag.status 三态)
ALTER TABLE assets ADD CONSTRAINT ck_assets_status
  CHECK (status IN ('IN_STOCK','DEPLOYED','MAINTENANCE','SCRAPPED'));
-- type 仅拦空串;受控字典(asset_models)为 P1 路线,放开增删不改约束
ALTER TABLE assets ADD CONSTRAINT ck_assets_type_notblank CHECK (type <> '');
ALTER TABLE tags ADD CONSTRAINT ck_tags_status
  CHECK (status IN ('UNBOUND','BOUND','DISABLED'));

COMMIT;
