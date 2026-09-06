#!/usr/bin/env bash
# verify-patrol-extended.sh -- P4-T3 验收:巡检两查 + 造数残留清理(102 真实环境,禁 mock)。
# 断言:
#   PA 巡检两查输出可 grep:db-patrol-gate.sh --asset 输出含固定前缀
#      [db-patrol] ASSET-EPC-INVALID count=N(含 SAMPLE 行)与
#      [db-patrol] ASSET-TYPE-UNKNOWN count=N(含 BREAKDOWN 行)
#   PB 引用暴露逻辑生效:造 1 行带生命周期历史的 MI-ONU 目标资产(批内自清),
#      清理脚本须 SKIP+REMAINING=1(不误删有历史行);移除历史行后再清理=DELETED
#   PC 残留为零:目标两类(SMOKE-P3-* / A-RK-E2E-001-MI-ONU-*)SQL 直查=0
#   PD 幂等:清理脚本重跑 DELETED 0 / REMAINING 0,退出码恒 0
# 信号: 每条断言独立输出 "PASS: [编号] ..." / "FAIL: [编号] ...";收尾
#   "PATROL-EXTENDED RESULT: ..." 可 grep;真实退出码 0/1。
# 用法: scripts/ops/verify-patrol-extended.sh
# 环境: SSH_HOST(同 db-patrol-gate.sh 回退逻辑) 依赖: curl ssh psql(经 102 容器)
set -u

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
source "$ROOT/scripts/ops/acceptance-lock.sh"
acquire_acceptance_lock || exit 1
trap release_acceptance_lock EXIT

env_or() { local v; v=$(printenv "$1" 2>/dev/null); if [ -n "$v" ]; then echo "$v"; else echo "$2"; fi; }
SSH_HOST=$(env_or SSH_HOST "imeepos@192.168.0.102")

sql() { ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST" "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q -tA"; }
sqlval() { sql < <(echo "$1") | tr -d '[:space:]'; }

PASS_N=0; FAIL_N=0
ok() { PASS_N=$((PASS_N+1)); echo "PASS: [$1] $2"; }
bad() { FAIL_N=$((FAIL_N+1)); echo "FAIL: [$1] $2" >&2; }

# VRFY 造数(批内自清,幂等):先清后插
teardown_vrfy() {
  echo "收尾: VRFY 造数自清(幂等)"
  echo "DELETE FROM asset_lifecycles WHERE asset_id IN (SELECT id FROM assets WHERE asset_code LIKE 'A-RK-E2E-001-VRFY-%');
DELETE FROM assets WHERE asset_code LIKE 'A-RK-E2E-001-VRFY-%';
DELETE FROM asset_batches WHERE code = 'acc_patrol-batch';" | sql >/dev/null
}
teardown_vrfy

echo "P4-T3 验收: 巡检两查 + 残留清理 @ ssh=$SSH_HOST"

# PA 两查输出可 grep(固定前缀+计数)
PATROL_OUT=$(bash "$ROOT/scripts/ops/db-patrol-gate.sh" --asset 2>/dev/null)
rc=$?
if [ $rc -eq 0 ] \
   && printf '%s' "$PATROL_OUT" | grep -q '\[db-patrol\] ASSET-EPC-INVALID count=[0-9]' \
   && printf '%s' "$PATROL_OUT" | grep -q '\[db-patrol\] ASSET-TYPE-UNKNOWN count=[0-9]'; then
  ok "PA" "两查输出可 grep(ASSET-EPC-INVALID/ASSET-TYPE-UNKNOWN 均带 count=)"
else
  bad "PA" "巡检输出缺固定前缀或退出码=$rc: $(printf '%s' "$PATROL_OUT" | head -2)"
fi

# PB 引用暴露逻辑:带生命周期历史的 MI-ONU 目标行必须 SKIP 不误删
ent=$(sqlval "SELECT min(id) FROM legal_entities;")
batch=$(sqlval "INSERT INTO asset_batches(legal_entity_id, code, name) VALUES ($ent, 'acc_patrol-batch', '巡检验收批次') RETURNING id;")
vrfy=$(sqlval "INSERT INTO assets(asset_code, batch_id, legal_entity_id, legal_entity_name, type, status)
  VALUES ('A-RK-E2E-001-VRFY-999', $batch, $ent, '巡检验收主体', 'MI-ONU', 'IN_STOCK') RETURNING id;")
sql < <(echo "INSERT INTO asset_lifecycles(asset_id, status, changed_at) VALUES ($vrfy, 'IN_STOCK', now()), ($vrfy, 'DEPLOYED', now());") >/dev/null
CLEAN1=$(bash "$ROOT/scripts/ops/clean-asset-type-residue.sh" 2>/dev/null)
if printf '%s' "$CLEAN1" | grep -q 'SKIP id=' && printf '%s' "$CLEAN1" | grep -q 'REMAINING 1' \
   && [ "$(sqlval "SELECT count(*) FROM assets WHERE asset_code='A-RK-E2E-001-VRFY-999';")" = "1" ]; then
  ok "PB" "引用暴露逻辑生效(生命周期历史行命中→SKIP+REMAINING 1,不误删)"
else
  bad "PB" "暴露逻辑未生效: $(printf '%s' "$CLEAN1" | grep -E 'SKIP|REMAINING' | tr '\n' ' ')"
fi

# PB2 移除历史行后再清理=删除
sql < <(echo "DELETE FROM asset_lifecycles WHERE asset_id=$vrfy AND status='DEPLOYED';") >/dev/null
CLEAN2=$(bash "$ROOT/scripts/ops/clean-asset-type-residue.sh" 2>/dev/null)
if printf '%s' "$CLEAN2" | grep -q 'DELETED 1'; then
  ok "PB2" "引用解除后可删(DELETED 1)"
else
  bad "PB2" "解除引用后未删除: $(printf '%s' "$CLEAN2" | grep -E 'DELETED|REMAINING' | tr '\n' ' ')"
fi

# PC 残留为零(SMOKE-P3-* / A-RK-E2E-001-MI-ONU-* 两目标类,SQL 直查)
remain=$(sqlval "SELECT count(*) FROM assets a WHERE (a.type='SMOKE' AND a.asset_code LIKE 'SMOKE-%') OR (a.type='MI-ONU' AND a.asset_code LIKE 'A-RK-E2E-001-%');")
if [ "$remain" = "0" ]; then
  ok "PC" "残留资产目标类=0(SMOKE-P3-*/A-RK-E2E-001-MI-ONU-*)"
else
  bad "PC" "残留目标类=$remain(应为 0)"
fi

# PD 幂等:重跑 DELETED 0 / REMAINING 0 / exit 0
CLEAN3=$(bash "$ROOT/scripts/ops/clean-asset-type-residue.sh" 2>/dev/null); rc=$?
if [ $rc -eq 0 ] && printf '%s' "$CLEAN3" | grep -q 'DELETED 0' && printf '%s' "$CLEAN3" | grep -q 'REMAINING 0'; then
  ok "PD" "清理脚本幂等(重跑 DELETED 0/REMAINING 0,exit 0)"
else
  bad "PD" "幂等不成立 rc=$rc: $(printf '%s' "$CLEAN3" | grep -E 'DELETED|REMAINING' | tr '\n' ' ')"
fi

teardown_vrfy
leftover=$(sqlval "SELECT count(*) FROM assets WHERE asset_code LIKE 'A-RK-E2E-001-VRFY-%';")
if [ "$leftover" != "0" ]; then bad "RES" "VRFY 造数残留=$leftover"; else ok "RES" "VRFY 造数自清=0"; fi

echo "断言明细: PASS=$PASS_N FAIL=$FAIL_N"
if [ "$FAIL_N" -eq 0 ]; then
  echo "PATROL-EXTENDED RESULT: PASS pass=$PASS_N"
  exit 0
fi
echo "PATROL-EXTENDED RESULT: FAIL pass=$PASS_N fail=$FAIL_N"
exit 1
