-- 资产↔标签双向关联 DB 层兜底:部分唯一索引,应用层遗漏时也能拦截隐性双绑。
-- 背景:internal/domain/asset/pg_write.go::CreateAsset/CreateTag 应用层已做双绑回填
-- + ErrBindingConflict 拦截(744abd23),但应用层 bug / 未来回归仍可能落孤儿。
-- DB 唯一索引作最后防线。
--
-- 设计:部分索引 WHERE col IS NOT NULL,允许空(NULL=未绑定),非空时全表唯一。
-- 风格沿用 000056/000048(部分唯一索引),跨大版本兼容 PG 11+。
--
-- 预检:本迁移前必须先清存量孤儿(migration 前置 SQL 不在此,见
-- adopted note 2026-08-27 §五);本次提交前已在 102 真库清掉 A 端 1 条 +
-- 回填 B 端 124 条 → 一致性 192/192。

BEGIN;

-- tags.bound_asset_id 非空时唯一:一资产最多绑一个标签。
CREATE UNIQUE INDEX uq_tags_bound_asset_notnull
  ON tags(bound_asset_id) WHERE bound_asset_id IS NOT NULL;

-- assets.tag_id 非空时唯一:一标签最多绑一个资产。
CREATE UNIQUE INDEX uq_assets_tag_notnull
  ON assets(tag_id) WHERE tag_id IS NOT NULL;

COMMIT;
