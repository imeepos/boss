-- 000186 down:事件流为纯增量审计面,drop 即可(主表 tags/assets 不受影响)。
BEGIN;
DROP TABLE IF EXISTS tag_events;
COMMIT;
