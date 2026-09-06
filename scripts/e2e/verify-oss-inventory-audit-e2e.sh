#!/usr/bin/env bash
# verify-oss-inventory-audit-e2e.sh -- 资源台账稽核端到端实测(P5-W3,102 真实部署,禁 mock)。
# 断言:
#   P0 清残留+取基线: 三类基线计数(存量违例如实计入)
#   P1 造数三类违例各>=1: ownership+2 / state+2 / coding+2,端点全量识别且计数正确
#   P2 ?category= 按类过滤: counts 仅该类,total=该类计数
#   P3 六个检查码全部出现在明细(items)
#   P4 重跑稽核: 计数与 P1 一致(只读幂等,不重复计数)
#   RES 收尾清理造数(acc_w3ia-/ACCE2EW3 标记)后残留为零(脚本可重复执行)
# 造数: 分光器上游缺失(parent NULL)/端口引用不存在的分光器(session_replication_role
#   绕 FK 直插,模拟跨库迁移态脏数据)/USED 无四码 LINKED/RESERVED 72h 未推进(默认阈值
#   48h)/资源编码不合 OLT-*/SPL-*/端口编码不合 P-<设备码>-<序号> 两形态。
# 信号: 每条断言独立输出 PASS/FAIL;收尾 E2E-OSS-INVENTORY-AUDIT RESULT 可 grep。
# 用法: scripts/e2e/verify-oss-inventory-audit-e2e.sh [BASE_URL]
# 环境: BASE_URL ADMIN_API_KEY SSH_HOST SKIP_CLEANUP=1
# 依赖: curl python3 ssh(102 免密);鉴权 X-API-Key(test-accounts.json admin key)。
# 注意: 端点/迁移 000193 需已随部署生效(Lead 统一收口后验收)。
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

FAIL_REASON=""
# req METHOD PATH [JSON] -> HTTP_CODE/BODY(HTTPCODE 尾标切分)。
req() {
  local method="$1" path="$2" body="${3:-}" out
  if [ -n "$body" ]; then
    out=$(curl -sS -m 30 -w "HTTPCODE:%{http_code}" -X "$method" "$API$path" -H "X-API-Key: $KEY" -H "Content-Type: application/json" -d "$body") || { FAIL_REASON="curl $path"; return 1; }
  else
    out=$(curl -sS -m 30 -w "HTTPCODE:%{http_code}" -X "$method" "$API$path" -H "X-API-Key: $KEY") || { FAIL_REASON="curl $path"; return 1; }
  fi
  HTTP_CODE=$(printf '%s' "$out" | sed -e 's/.*HTTPCODE://')
  BODY=$(printf '%s' "$out" | sed -e 's/HTTPCODE:[0-9]*$//')
}

# jget EXPR: 用当前 BODY 算 python 表达式(json 信封取数)。
jget() {
  printf '%s' "$BODY" | python3 -c "import json,sys;d=json.loads(sys.stdin.read());print($1)"
}

# cat_count CATEGORY: 当前 BODY 的该类计数。
cat_count() {
  jget "[c['count'] for c in d['data']['counts'] if c['category']=='$1'][0]" 2>/dev/null || echo MISSING
}

PASS_N=0; FAIL_N=0; FAILED_IDS=""
ok() { PASS_N=$((PASS_N+1)); echo "PASS: [$1] $2"; }
bad() {
  FAIL_N=$((FAIL_N+1)); FAILED_IDS="$FAILED_IDS $1"
  echo "FAIL: [$1] $2" >&2
  echo "[e2e-oss-inventory-audit] ASSERTION FAILED id=$1 detail=$2" >&2
}
assert_eq() { if [ "$2" = "$3" ]; then ok "$1" "$2 ($4)"; else bad "$1" "期望=$2 实际=$3 上下文=$4"; fi; }

cleanup_data() {
  echo "清尾: 造数自清理(acc_w3ia-/ACCE2EW3 标记,幂等)" >&2
  sql < <(cat <<'CLEANSQL'
DELETE FROM port_change_history WHERE port_id IN (SELECT id FROM ports WHERE port_code IN ('P-SPLACCE2EW3-01','P-SPLACCE2EW3-02','acc_w3ia-P-BADCODE','acc_w3ia-P-ORPHAN'));
DELETE FROM ports WHERE port_code IN ('P-SPLACCE2EW3-01','P-SPLACCE2EW3-02','acc_w3ia-P-BADCODE','acc_w3ia-P-ORPHAN');
DELETE FROM resources WHERE code IN ('OLT-ACCE2EW3','SPL-ACCE2EW3','acc_w3ia-BAD');
CLEANSQL
  ) >/dev/null
}

seed_data() {
  # 单连接批量(Lead 提速要求): 清残留+六类造数+回执一次 ssh 完成,标量按行解析。
  SEEDRAW=$(sql < <(cat <<'SEEDSQL'
DELETE FROM port_change_history WHERE port_id IN (SELECT id FROM ports WHERE port_code IN ('P-SPLACCE2EW3-01','P-SPLACCE2EW3-02','acc_w3ia-P-BADCODE','acc_w3ia-P-ORPHAN'));
DELETE FROM ports WHERE port_code IN ('P-SPLACCE2EW3-01','P-SPLACCE2EW3-02','acc_w3ia-P-BADCODE','acc_w3ia-P-ORPHAN');
DELETE FROM resources WHERE code IN ('OLT-ACCE2EW3','SPL-ACCE2EW3','acc_w3ia-BAD');
INSERT INTO resources(legal_entity_id, code, name, type, parent_id, address_id, status)
SELECT p.legal_entity_id, 'OLT-ACCE2EW3', 'acc_w3ia e2e OLT', 'OLT', NULL, p.address_id, 'ONLINE' FROM ports p ORDER BY p.id LIMIT 1;
INSERT INTO resources(legal_entity_id, code, name, type, parent_id, address_id, status)
SELECT p.legal_entity_id, 'SPL-ACCE2EW3', 'acc_w3ia e2e SPL', 'SPLITTER', (SELECT id FROM resources WHERE code='OLT-ACCE2EW3'), p.address_id, 'ONLINE' FROM ports p ORDER BY p.id LIMIT 1;
INSERT INTO resources(legal_entity_id, code, name, type, parent_id, address_id, status)
SELECT p.legal_entity_id, 'acc_w3ia-BAD', 'acc_w3ia e2e bad SPL', 'SPLITTER', NULL, p.address_id, 'ONLINE' FROM ports p ORDER BY p.id LIMIT 1;
INSERT INTO ports(port_code, quad_code, resource_id, legal_entity_id, legal_entity_name, address_id, region_id, region_name, order_id, status)
SELECT 'P-SPLACCE2EW3-01','P-SPLACCE2EW3-01', (SELECT id FROM resources WHERE code='SPL-ACCE2EW3'), p.legal_entity_id, p.legal_entity_name, p.address_id, p.region_id, p.region_name, CASE WHEN EXISTS(SELECT 1 FROM orders WHERE status='DONE') THEN (SELECT max(id) FROM orders WHERE status='DONE') ELSE NULL END, 'USED' FROM ports p ORDER BY p.id LIMIT 1;
INSERT INTO ports(port_code, quad_code, resource_id, legal_entity_id, legal_entity_name, address_id, region_id, region_name, order_id, status)
SELECT 'P-SPLACCE2EW3-02','P-SPLACCE2EW3-02', (SELECT id FROM resources WHERE code='SPL-ACCE2EW3'), p.legal_entity_id, p.legal_entity_name, p.address_id, p.region_id, p.region_name, CASE WHEN EXISTS(SELECT 1 FROM orders WHERE status='INSTALLING') THEN (SELECT max(id) FROM orders WHERE status='INSTALLING') ELSE NULL END, 'RESERVED' FROM ports p ORDER BY p.id LIMIT 1;
INSERT INTO port_change_history(port_id, status, order_id, changed_at)
SELECT id, 'RESERVED', NULL, now() - interval '72 hours' FROM ports WHERE port_code='P-SPLACCE2EW3-02';
INSERT INTO ports(port_code, quad_code, resource_id, legal_entity_id, legal_entity_name, address_id, region_id, region_name, order_id, status)
SELECT 'acc_w3ia-P-BADCODE','acc_w3ia-P-BADCODE', (SELECT id FROM resources WHERE code='SPL-ACCE2EW3'), p.legal_entity_id, p.legal_entity_name, p.address_id, p.region_id, p.region_name, NULL, 'IDLE' FROM ports p ORDER BY p.id LIMIT 1;
SET session_replication_role = replica;
INSERT INTO ports(port_code, quad_code, resource_id, legal_entity_id, legal_entity_name, address_id, region_id, region_name, order_id, status)
SELECT 'acc_w3ia-P-ORPHAN','acc_w3ia-P-ORPHAN', 999999999, p.legal_entity_id, p.legal_entity_name, p.address_id, p.region_id, p.region_name, NULL, 'IDLE' FROM ports p ORDER BY p.id LIMIT 1;
SET session_replication_role = DEFAULT;
SELECT 'ORPHAN=' || count(*) FROM ports WHERE port_code = 'acc_w3ia-P-ORPHAN';
SEEDSQL
  ) )
  ORPHAN=$(printf '%s\n' "$SEEDRAW" | sed -n 's/^ORPHAN=//p' | tr -d '[:space:]')
  if [ "$ORPHAN" != "1" ]; then FAIL_REASON="orphan port seed failed(FK bypass)"; return 1; fi
  echo "seeded orphan=$ORPHAN"
}
echo "资源台账稽核端到端实测 @ $BASE_URL"

# P0 基线(先清残留再取,幂等重跑同一基线)
cleanup_data
if req GET "/inventory-audit"; then
  if [ "$HTTP_CODE" = "200" ]; then
    ok "P0" "基线 own=$(cat_count ownership) state=$(cat_count state) coding=$(cat_count coding)"
  else
    bad "P0" "GET /inventory-audit http=$HTTP_CODE body=$BODY(端点未部署?)"
  fi
else
  bad "P0" "请求失败: $FAIL_REASON"
fi
OWN_BASE=$(cat_count ownership); STATE_BASE=$(cat_count state); CODING_BASE=$(cat_count coding)
if [ "$HTTP_CODE" != "200" ]; then cleanup_data; echo "E2E-OSS-INVENTORY-AUDIT RESULT: FAIL(pass=$PASS_N fail=$FAIL_N failed=[$FAILED_IDS ])"; exit 1; fi
# P1 造数三类违例各>=1 并断言计数
SEED_OUT=$(seed_data) || { bad "P1" "造数失败: $FAIL_REASON"; cleanup_data; echo "E2E-OSS-INVENTORY-AUDIT RESULT: FAIL(pass=$PASS_N fail=$FAIL_N failed=[$FAILED_IDS ])"; exit 1; }
echo "P1 $SEED_OUT"
if req GET "/inventory-audit" && [ "$HTTP_CODE" = "200" ]; then
  assert_eq "P1" "$((OWN_BASE+2))" "$(cat_count ownership)" "归属断裂 +2"
  assert_eq "P1" "$((STATE_BASE+2))" "$(cat_count state)" "状态机违例 +2"
  assert_eq "P1" "$((CODING_BASE+2))" "$(cat_count coding)" "编码违例 +2"
else
  bad "P1" "GET /inventory-audit http=$HTTP_CODE"
fi

# P2 按类过滤
if req GET "/inventory-audit?category=ownership" && [ "$HTTP_CODE" = "200" ]; then
  assert_eq "P2" "$((OWN_BASE+2))" "$(jget "d['data']['total']")" "过滤 ownership total"
  assert_eq "P2" "1" "$(jget "len(d['data']['counts'])")" "counts 仅一类"
  assert_eq "P2" "ownership" "$(jget "d['data']['counts'][0]['category']")" "类别=ownership"
else
  bad "P2" "GET ?category=ownership http=$HTTP_CODE"
fi

# P3 六检查码全部出现在明细
if req GET "/inventory-audit" && [ "$HTTP_CODE" = "200" ]; then
  CHECKS="PORT_SPLITTER_MISSING SPLITTER_UPSTREAM_MISSING USED_PORT_NO_QUAD_LINK RESERVED_PORT_STALE RESOURCE_CODE_BAD PORT_CODE_BAD"
  GOT=$(jget "','.join(sorted(set(i['check'] for i in d['data']['items'])))")
  MISS=""
  for c in $CHECKS; do
    case ",$GOT," in *$c,*) ;; *) MISS="$MISS $c" ;; esac
  done
  assert_eq "P3" "" "$MISS" "六检查码全部命中"
else
  bad "P3" "GET /inventory-audit http=$HTTP_CODE"
fi

# P3A 造数标记作用域: 每个检查码都命中至少一条带 acc_w3ia-/ACCE2EW3 标记的造数明细。
# 只对采样上限(50)内的检查做标记断言;PORT_CODE_BAD 存量 101>50,明细样本
# 不保证含造数行,其造数证据由 P1 类别差值(coding +2)承载。
if req GET "/inventory-audit" && [ "$HTTP_CODE" = "200" ]; then
  MARKHIT=$(jget "','.join(sorted(set(i['check'] for i in d['data']['items'] if i['code'].startswith('acc_w3ia-') or 'ACCE2EW3' in i['code'])))")
  MARKED_CHECKS="PORT_SPLITTER_MISSING SPLITTER_UPSTREAM_MISSING USED_PORT_NO_QUAD_LINK RESERVED_PORT_STALE RESOURCE_CODE_BAD"
  MISS2=""
  for c in $MARKED_CHECKS; do
    case ",$MARKHIT," in *$c,*) ;; *) MISS2="$MISS2 $c" ;; esac
  done
  assert_eq "P3A" "" "$MISS2" "标记作用域五码全命中(PORT_CODE_BAD 走 P1 差值)"
else
  bad "P3A" "GET /inventory-audit http=$HTTP_CODE"
fi

# P4 重跑幂等(只读,不重复计数)
if req GET "/inventory-audit" && [ "$HTTP_CODE" = "200" ]; then
  assert_eq "P4" "$((OWN_BASE+2))/$((STATE_BASE+2))/$((CODING_BASE+2))" "$(cat_count ownership)/$(cat_count state)/$(cat_count coding)" "重跑计数不变"
else
  bad "P4" "GET /inventory-audit http=$HTTP_CODE"
fi

rc=0
if [ "$(env_or SKIP_CLEANUP 0)" != "1" ]; then
  # 单连接批量: 清理+残留校验一次 ssh 完成。
  echo "清尾: 造数自清理(acc_w3ia-/ACCE2EW3 标记,幂等)" >&2
  res=$(sql < <(cat <<'RESSQL'
DELETE FROM port_change_history WHERE port_id IN (SELECT id FROM ports WHERE port_code IN ('P-SPLACCE2EW3-01','P-SPLACCE2EW3-02','acc_w3ia-P-BADCODE','acc_w3ia-P-ORPHAN'));
DELETE FROM ports WHERE port_code IN ('P-SPLACCE2EW3-01','P-SPLACCE2EW3-02','acc_w3ia-P-BADCODE','acc_w3ia-P-ORPHAN');
DELETE FROM resources WHERE code IN ('OLT-ACCE2EW3','SPL-ACCE2EW3','acc_w3ia-BAD');
SELECT 'RES=' || (SELECT count(*) FROM ports WHERE port_code IN ('P-SPLACCE2EW3-01','P-SPLACCE2EW3-02','acc_w3ia-P-BADCODE','acc_w3ia-P-ORPHAN')) || '/' || (SELECT count(*) FROM resources WHERE code IN ('OLT-ACCE2EW3','SPL-ACCE2EW3','acc_w3ia-BAD'));
RESSQL
  ) | sed -n 's/^RES=//p' | tr -d '[:space:]')
  assert_eq "RES" "0/0" "$res" "造数清理后残留"
  if [ "$res" != "0/0" ]; then rc=1; fi
fi

echo "断言明细: PASS=$PASS_N FAIL=$FAIL_N 失败编号=[$FAILED_IDS ]"
if [ "$FAIL_N" -eq 0 ] && [ "$rc" -eq 0 ]; then
  echo "E2E-OSS-INVENTORY-AUDIT RESULT: PASS pass=$PASS_N"
  exit 0
fi
echo "E2E-OSS-INVENTORY-AUDIT RESULT: FAIL pass=$PASS_N fail=$FAIL_N failed=[$FAILED_IDS ]"
exit 1