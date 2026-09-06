-- 000189 down:仅删除回填行(changed->>backfill 标记定位,不伤真实事件)。
BEGIN;
DELETE FROM tag_events WHERE changed->>'backfill' = '000189';
COMMIT;
