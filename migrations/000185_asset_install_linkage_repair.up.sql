-- 000185: 装机联动存量补账(P1-T1,adopted note 2026-09-06-asset-tag-p1-wave)。
-- 背景:此前扫码绑定只写 quad_links,资产停留 IN_STOCK(台账漂移源,G1);
-- 本迁移把「活跃链路指向但资产仍在库」的存量一次性置 DEPLOYED+绑地址+补轨迹行。
-- 口径:仅修 IN_STOCK(LINKED+MAINTENANCE/SCRAPPED 属业务态,留给巡检人工甄别);
-- 102 真库 2026-09-06 事务预演:LINKED 链路资产 5 条全部已是 DEPLOYED,本迁移预期 0 行命中
-- (迁移语义面向其他环境/未来回放);幂等,可重复执行。

BEGIN;

WITH targets AS (
  SELECT DISTINCT ON (q.asset_id) q.asset_id, q.address_id
  FROM quad_links q
  WHERE q.status = 'LINKED' AND q.asset_id IS NOT NULL
),
fix AS (
  UPDATE assets a SET status = 'DEPLOYED', address_id = t.address_id, updated_at = now()
  FROM targets t
  WHERE a.id = t.asset_id AND a.status = 'IN_STOCK'
  RETURNING a.id, t.address_id
)
INSERT INTO asset_lifecycles(asset_id, status, address_id, address_name, changed_at)
SELECT f.id, 'DEPLOYED', f.address_id, COALESCE((SELECT name FROM addresses ad WHERE ad.id = f.address_id), ''), now()
FROM fix f;

COMMIT;
