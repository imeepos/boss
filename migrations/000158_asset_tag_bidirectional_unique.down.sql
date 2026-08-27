-- 回滚:资产↔标签双向关联 DB 兜底唯一索引。
-- 注意:回滚前确保应用层双绑回填仍生效(internal/domain/asset/pg_write.go::CreateAsset/CreateTag)。

BEGIN;

DROP INDEX IF EXISTS uq_tags_bound_asset_notnull;
DROP INDEX IF EXISTS uq_assets_tag_notnull;

COMMIT;
