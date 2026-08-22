-- Q2 四码冲突 4 小时清零率:quad_links 补冲突发现/清零时间戳。
-- Reconcile 标 CONFLICT 时记 conflict_at(已在冲突态不重置时钟),
-- ResolveConflict 置回 UNLINKED 时记 cleared_at;两列共同构成冲突
-- 事件时间线,清零率 = (cleared_at-conflict_at<=4h) / 发现数。
BEGIN;

ALTER TABLE quad_links
    ADD COLUMN conflict_at TIMESTAMPTZ,
    ADD COLUMN cleared_at  TIMESTAMPTZ;

CREATE INDEX idx_quad_links_conflict_at ON quad_links(conflict_at);

COMMIT;
