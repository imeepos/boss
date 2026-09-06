-- 000189: 标签事件流历史回填(P3-T4,docs/design/asset-tag-p3-plan.md)。
-- 事件流(000186)上线前,存量资产无 CREATE 事件、在绑标签无当前绑定对的 BIND 事件,
-- 时间轴从中间开始。回填口径:
--   - 对没有任何 tag_events 事件的资产各补一条 CREATE 事件(事件行 tag_id=0 哨兵,
--     CREATE 无标签维度;append-only 流保持统一时间轴);
--   - 对处于绑定中(BOUND 且 bound_asset_id 非空)但当前绑定对 (tag,asset) 无 BIND
--     事件的标签各补一条 BIND 事件;
--   - changed 带来源标记 backfill=000189(事件可溯源,down 仅删回填行);
--   - 幂等:NOT EXISTS 守卫,重跑零新增。

BEGIN;

INSERT INTO tag_events(tag_id, asset_id, action, detail, changed)
SELECT 0, a.id, 'CREATE', 'backfill: asset created before event stream',
       jsonb_build_object('backfill', '000189', 'source', 'migration')
  FROM assets a
 WHERE NOT EXISTS (SELECT 1 FROM tag_events e WHERE e.asset_id = a.id);

INSERT INTO tag_events(tag_id, asset_id, action, detail, changed)
SELECT t.id, t.bound_asset_id, 'BIND', 'backfill: binding predates event stream',
       jsonb_build_object('bound_asset_id', t.bound_asset_id,
                          'backfill', '000189', 'source', 'migration')
  FROM tags t
 WHERE t.status = 'BOUND' AND t.bound_asset_id IS NOT NULL
   AND NOT EXISTS (SELECT 1 FROM tag_events e
                    WHERE e.tag_id = t.id AND e.asset_id = t.bound_asset_id
                      AND e.action = 'BIND');

COMMIT;
