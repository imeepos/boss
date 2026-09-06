#!/usr/bin/env bash
# verify-asset-type-canonical-e2e.sh -- 资产类型归一+写入白名单端到端实测(P4-T2,102 真实部署,禁 mock)。
# 断言:
#   P0 造数: acc_atc- 前缀独立批次(幂等重跑)
#   P1 建档 type=光猫(已归一方言)→ HTTP 400 + code 42200
#   P2 建档 type=MI-ONU(e2e 残留方言)→ HTTP 400 + code 42200
#   P3 建档 type=ONU(权威码)→ 200 且 id>0
#   P4 详情回读: type=ONU
#   P5 编辑 type=ROUTER(白名单内)→ 200 且 type=ROUTER
#   P6 编辑 type=光猫 → HTTP 400(编辑通道同闸)
#   P7 SQL 直查 102:全库 assets.type 无「光猫」值(迁移 000191 生效后恒真)
#   P8 列表筛选 type=ONU 可用(total>=1)
#   RES 收尾清理造数(资产含建档轨迹行+批次)后残留断言为零(脚本可重复执行)
# 信号: 每条断言独立输出 "PASS: [编号] ..." / "FAIL: [编号] ...";收尾
#   "E2E-ASSET-TYPE-CANONICAL RESULT: ..." 可 grep。
# 用法: scripts/e2e/verify-asset-type-canonical-e2e.sh [BASE_URL]
# 环境: BASE_URL ADMIN_API_KEY SSH_HOST SKIP_CLEANUP=1
# 依赖: curl python3 ssh(102 免密);鉴权 X-API-Key(test-accounts.json admin key)。
# 注意: P1/P2/P6/P7 依赖白名单+迁移 000191 已随部署生效(Lead 统一收口后验收)。
set -u

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
source "$ROOT/scripts/ops/acceptance-lock.sh"
acquire_acceptance_lock || exit 1
trap release_acceptance_lock EXIT

env_or() { local v; v=$(printenv "$1" 2>/dev/null); if [ -n "$v" ]; then echo "$v"; else echo "$2"; fi; }

ARG1=""; if [ $# -ge 1 ]; then ARG1="$1"; fi
BASE_URL=$(env_or BASE_URL "http://192.168.0.102:28080")
case "$ARG1" in http*) BASE_URL="$ARG1" ;; esac
API="$BASE_URL/api/admin/v1"
KEY=$(env_or ADMIN_API_KEY "$(python3 -c "import json;print(json.load(open('$ROOT/.agents/skills/bossctl-cli/test-accounts.json'))['admin']['apiKeys'][0]['key'])")")
SSH_HOST=$(env_or SSH_HOST "imeepos@192.168.0.102")

sql() { ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST" "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q -tA"; }
sqlval() { sql < <(echo "$1") | tr -d '[:space:]'; }

FAIL_REASON=""
# req METHOD PATH [JSON] -> HTTP_CODE/BODY(业务失败也保留响应供断言,本脚本要断言 400)。
req() {
  local method="$1" path="$2" body="${3:-}" out
  out=$(curl -sS -m 30 -w $'\n%{http_code}' -X "$method" "$API$path" -H "X-API-Key: $KEY" \
    -H "Content-Type: application/json" ${body:+-d "$body"}) || { FAIL_REASON="curl $path"; return 1; }
  HTTP_CODE=$(printf '%s\n' "$out" | tail -n1 | tr -d '[:space:]')
  BODY=$(printf '%s\n' "$out" | sed '$d')
}

jf() { python3 -c "import json,sys;d=json.loads(sys.argv[1]);print(eval(sys.argv[2]))" "$1" "$2"; }

PASS_N=0; FAIL_N=0; FAILED_IDS=""
ok() { PASS_N=$((PASS_N+1)); echo "PASS: [$1] $2"; }
bad() {
  FAIL_N=$((FAIL_N+1)); FAILED_IDS="$FAILED_IDS $1"
  echo "FAIL: [$1] $2" >&2
  echo "[e2e-asset-type-canonical] ASSERTION FAILED id=$1 detail=$2" >&2
}
assert_eq() { if [ "$2" = "$3" ]; then ok "$1" "$2 ($4)"; else bad "$1" "期望=$2 实际=$3 上下文=$4"; fi; }

cleanup_data() {
  # 提示走 stderr:seed_data 经命令替换捕获返回值,stdout 混入会使 batchId 变脏。
  echo "清尾: 造数自清理(acc_atc- 前缀,幂等)" >&2
  echo "DELETE FROM asset_lifecycles WHERE asset_id IN (SELECT id FROM assets WHERE asset_code LIKE 'acc_atc-%');
DELETE FROM assets WHERE asset_code LIKE 'acc_atc-%';
DELETE FROM asset_batches WHERE code = 'acc_atc-batch';" | sql >/dev/null
}

seed_data() {
  local ent batch
  cleanup_data
  ent=$(sqlval "SELECT min(id) FROM legal_entities;")
  if [ -z "$ent" ]; then FAIL_REASON="no legal_entities row"; return 1; fi
  batch=$(sqlval "INSERT INTO asset_batches(legal_entity_id, code, name) VALUES ($ent, 'acc_atc-batch', '类型归一e2e批次') RETURNING id;")
  if [ -z "$batch" ]; then FAIL_REASON="seed batch insert"; return 1; fi
  echo "$batch"
}

echo "资产类型归一+白名单端到端实测 @ $BASE_URL"
BATCH=$(seed_data) || { bad "P0" "造数失败: $FAIL_REASON"; cleanup_data; exit 1; }
ok "P0" "造数基线 acc_atc- 批次=$BATCH"

# P1 建档 type=光猫 → 400(方言已归一,白名单外)
if req POST "/assets" '{"batchId": '"$BATCH"', "assetCode": "acc_atc-A-1", "type": "光猫"}'; then
  code=$(jf "$BODY" "d['code']")
  assert_eq "P1" "400|42200" "$HTTP_CODE|$code" "建档 type=光猫 拒绝"
else
  bad "P1" "请求失败: $FAIL_REASON"
fi

# P2 建档 type=MI-ONU → 400(e2e 残留方言不入类型体系)
if req POST "/assets" '{"batchId": '"$BATCH"', "assetCode": "acc_atc-A-2", "type": "MI-ONU"}'; then
  code=$(jf "$BODY" "d['code']")
  assert_eq "P2" "400|42200" "$HTTP_CODE|$code" "建档 type=MI-ONU 拒绝"
else
  bad "P2" "请求失败: $FAIL_REASON"
fi

# P3 建档 type=ONU → 200(权威码放行)
if req POST "/assets" '{"batchId": '"$BATCH"', "assetCode": "acc_atc-A-3", "type": "ONU"}'; then
  aid=$(jf "$BODY" "d['data']['id']")
  if [ "$HTTP_CODE" = "200" ] && [ "$aid" -gt 0 ] 2>/dev/null; then
    ok "P3" "建档 type=ONU 成功 id=$aid"
  else
    bad "P3" "http=$HTTP_CODE id=$aid(期望 200 且 id>0)"
  fi
else
  bad "P3" "请求失败: $FAIL_REASON"
fi

# P4 详情回读 type=ONU
if req GET "/assets/$aid"; then
  assert_eq "P4" "ONU" "$(jf "$BODY" "d['data']['type']")" "详情回读 type"
else
  bad "P4" "请求失败: $FAIL_REASON"
fi

# P5 编辑 type=ROUTER(白名单内)→ 200 且生效
if req PUT "/assets/$aid" '{"type": "ROUTER"}'; then
  assert_eq "P5" "ROUTER" "$(jf "$BODY" "d['data']['type']")" "编辑 type=ROUTER 放行"
else
  bad "P5" "请求失败: $FAIL_REASON"
fi

# P6 编辑 type=光猫 → 400(编辑通道同闸)
if req PUT "/assets/$aid" '{"type": "光猫"}'; then
  code=$(jf "$BODY" "d['code']")
  assert_eq "P6" "400|42200" "$HTTP_CODE|$code" "编辑 type=光猫 拒绝"
else
  bad "P6" "请求失败: $FAIL_REASON"
fi

# P7 SQL 直查 102:全库无「光猫」(迁移 000191 生效后恒真)
got=$(sqlval "SELECT count(*) FROM assets WHERE type = '光猫';")
assert_eq "P7" "0" "$got" "102 直查 assets.type 无光猫值"

# P8 列表筛选 type=ONU 可用
if req GET "/assets?type=ONU&limit=1"; then
  total=$(jf "$BODY" "d['data']['total']")
  if [ "$HTTP_CODE" = "200" ] && [ "$total" -ge 1 ] 2>/dev/null; then
    ok "P8" "列表筛选 type=ONU total=$total"
  else
    bad "P8" "http=$HTTP_CODE total=$total(期望 200 且 total>=1)"
  fi
else
  bad "P8" "请求失败: $FAIL_REASON"
fi

rc=0
if [ "$(env_or SKIP_CLEANUP 0)" != "1" ]; then
  cleanup_data
  res=$(sqlval "SELECT count(*) FROM assets WHERE asset_code LIKE 'acc_atc-%';")
  res2=$(sqlval "SELECT count(*) FROM asset_batches WHERE code = 'acc_atc-batch';")
  if [ "$res$res2" = "00" ]; then
    ok "RES" "造数清理后残留=0(assets/batches)"
  else
    bad "RES" "残留 assets=$res batches=$res2(应为零)"
    rc=1
  fi
fi

echo "断言明细: PASS=$PASS_N FAIL=$FAIL_N 失败编号=[$FAILED_IDS ]"
if [ "$FAIL_N" -eq 0 ] && [ "$rc" -eq 0 ]; then
  echo "E2E-ASSET-TYPE-CANONICAL RESULT: PASS pass=$PASS_N"
  exit 0
fi
echo "E2E-ASSET-TYPE-CANONICAL RESULT: FAIL pass=$PASS_N fail=$FAIL_N failed=[$FAILED_IDS ]"
exit 1
