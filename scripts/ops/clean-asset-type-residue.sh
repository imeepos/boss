#!/usr/bin/env bash
# clean-asset-type-residue: 资产类型 e2e 残留定向清理(P4-T3,Lead 裁定 2026-09-06)。
# 目标(仅命中以下两类,其余方言不经此脚本;102 实查 2026-09-06 合计 6 行):
#   type='SMOKE'  AND asset_code LIKE 'SMOKE-%'        —— 冒烟残留(1 行 SMOKE-P3-*)
#   type='MI-ONU' AND asset_code LIKE 'A-RK-E2E-001-%' —— 采购域 e2e 造数残留(5 行)
# 逐行引用守卫(命中任一=保留并输出 [asset-residue] SKIP 暴露行,不删):
#   标签绑定 tags.bound_asset_id / 领用台账 asset_assignments / 换新单 replacements /
#   盘点明细 stocktake_items / 四码关联 quad_links(与 DeleteAsset 守卫同口径);
#   生命周期历史 asset_lifecycles:建档首行不算历史(000184 起建档即留痕),
#   第 2 行起=真实流转历史,命中即保留。无引用行的建档首行随主档同删
#   (自有子数据,FK 硬约束必先清,同 DeleteAsset 口径)。
# 幂等:清理后重跑目标 0 行,DELETED 0,恒 exit 0;SQL 单事务,失败即回滚。
# 用法: scripts/ops/clean-asset-type-residue.sh
# 环境: SSH_HOST/PG_CONTAINER/PG_USER/PG_DB 可覆盖(同 db-patrol-gate.sh 回退逻辑)。
set -u

SSH_HOST="${SSH_HOST:-imeepos@192.168.0.102}"
PG_CONTAINER="${PG_CONTAINER:-boss-infra-postgres-1}"
PG_USER="${PG_USER:-boss}"
PG_DB="${PG_DB:-boss}"

psql_run() {
  if [ -n "$SSH_HOST" ]; then
    if out=$(ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST" \
        "docker exec -i $PG_CONTAINER psql -U $PG_USER -d $PG_DB -v ON_ERROR_STOP=1 -q -tA" 2>/dev/null); then
      printf '%s' "$out"
      return 0
    fi
    echo "[asset-residue] ALERT sql via local docker exec (ssh $SSH_HOST unavailable)" >&2
  fi
  docker exec -i "$PG_CONTAINER" psql -U "$PG_USER" -d "$PG_DB" -v ON_ERROR_STOP=1 -q -tA
}

psql_run <<'SQL'
DROP TABLE IF EXISTS _asset_residue_tmp;
CREATE TEMP TABLE _asset_residue_tmp AS
SELECT a.id, a.asset_code, a.type,
  (SELECT count(*) FROM tags x WHERE x.bound_asset_id = a.id)
  + (SELECT count(*) FROM asset_assignments x WHERE x.asset_id = a.id)
  + (SELECT count(*) FROM replacements x WHERE x.asset_id = a.id)
  + (SELECT count(*) FROM stocktake_items x WHERE x.asset_id = a.id)
  + (SELECT count(*) FROM quad_links x WHERE x.asset_id = a.id)
  + GREATEST((SELECT count(*) FROM asset_lifecycles x WHERE x.asset_id = a.id) - 1, 0) AS refs
FROM assets a
WHERE (a.type = 'SMOKE' AND a.asset_code LIKE 'SMOKE-%')
   OR (a.type = 'MI-ONU' AND a.asset_code LIKE 'A-RK-E2E-001-%');

SELECT '[asset-residue] TARGET ' || count(*) FROM _asset_residue_tmp;
SELECT '[asset-residue] SKIP id=' || id || ' code=' || asset_code || ' type=' || type || ' refs=' || refs
  FROM _asset_residue_tmp WHERE refs > 0 ORDER BY id;

BEGIN;
DELETE FROM asset_lifecycles WHERE asset_id IN (SELECT id FROM _asset_residue_tmp WHERE refs = 0);
DELETE FROM assets WHERE id IN (SELECT id FROM _asset_residue_tmp WHERE refs = 0);
SELECT '[asset-residue] DELETED ' || count(*) FROM _asset_residue_tmp WHERE refs = 0;
COMMIT;

SELECT '[asset-residue] KEPT ' || count(*) FROM _asset_residue_tmp WHERE refs > 0;
DROP TABLE IF EXISTS _asset_residue_tmp;
SELECT '[asset-residue] REMAINING ' || count(*) FROM assets a
WHERE (a.type = 'SMOKE' AND a.asset_code LIKE 'SMOKE-%')
   OR (a.type = 'MI-ONU' AND a.asset_code LIKE 'A-RK-E2E-001-%');
SQL
rc=$?
if [ "$rc" -ne 0 ]; then
  echo "[asset-residue] ALERT cleanup sql failed rc=$rc (事务已回滚,目标行未动)" >&2
  exit "$rc"
fi
echo ""
echo "[asset-residue] RESULT: OK(暴露=SKIP 行,清理=DELETED 行,残留=REMAINING)"
exit 0
