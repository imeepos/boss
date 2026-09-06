-- 000190 down:完整回滚到 000188 现状(uq_assets_mac 恢复 mac 列值部分唯一形态)。
-- 归一 UPDATE 不逆转:上迁时存量 mac 非空 0 行,无可回滚值;且大写冒号规范形是
-- 裁定后的权威形态(docs/design/asset-tag-p4-plan.md §二-2),应用层此后仍只写
-- 规范形,按列值重建索引不会产生冲突。

BEGIN;

DROP INDEX IF EXISTS uq_assets_mac;

CREATE UNIQUE INDEX uq_assets_mac
    ON assets (mac)
    WHERE mac IS NOT NULL AND mac <> '';

COMMIT;
