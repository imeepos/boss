#!/usr/bin/env bash
# verify-oss-capacity-e2e.sh -- OSS 容量视图与阈值预警端到端实测(P5-W1,102 真实部署,禁 mock)。
# 断言:
#   P0 造数: acc_cap- 前缀两分光器(HIGH 8 端口 7 USED=87.50% / LOW 8 端口 1 USED=12.50%)
#        + OLT 4 端口 2 USED=50.00%(幂等重跑:先清后造)
#   P1 聚合 API(dim=SPLITTER): HIGH total=8 used=7 rate=87.5,LOW rate=12.5
#   P2 倒序: items 按 usageRate 降序(任务书:按使用率倒序)
#   P3 聚合 API(dim=OLT): OLT 行可见且不含 SPLITTER 行
#   P4 扫描一: POST alert-scan → HIGH 产生 1 条 WARNING OPEN 容量告警(仅一次)
#   P5 重跑幂等: 扫描二 → HIGH 告警仍 1 条(不重复)
#   P6 回落关闭: SQL 降 HIGH 使用率 → 扫描三 → OPEN=0 且出现 CLOSED(状态变化)
#   P7 再越限: SQL 恢复 7 USED → 扫描四 → HIGH 又 1 条 OPEN(状态变化才重复告警)
#   P8 前端构建产物含新页面路由 /oss/capacity
#   RES 收尾清理造数(acc_cap- 资源/端口/容量告警)后残留断言为零(脚本可重复执行)
# 信号: 每条断言独立输出 "PASS: [编号] ..." / "FAIL: [编号] ...";收尾
#   "E2E-OSS-CAPACITY RESULT: ..." 可 grep。
# 用法: scripts/e2e/verify-oss-capacity-e2e.sh [BASE_URL]
# 环境: BASE_URL ADMIN_API_KEY SSH_HOST SKIP_CLEANUP=1 SKIP_FRONTEND=1
# 依赖: curl python3 ssh(102 免密);P8 需 pnpm(SKIP_FRONTEND=1 可跳过);
#   鉴权 X-API-Key(test-accounts.json admin key)。
# 注意: P4-P7 依赖容量端点与迁移 000192 已随部署生效(Lead 统一收口后验收)。
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
# req METHOD PATH [JSON] -> HTTP_CODE/BODY(业务失败也保留响应供断言)。
req() {
  local method="$1" path="$2" body="${3:-}" out
  out=$(curl -sS -m 30 -w $'\n%{http_code}' -X "$method" "$API$path" -H "X-API-Key: $KEY" \
    -H "Content-Type: application/json" ${body:+-d "$body"}) || { FAIL_REASON="curl $path"; return 1; }
  HTTP_CODE=$(printf '%s\n' "$out" | tail -n1 | tr -d '[:space:]')
  BODY=$(printf '%s\n' "$out" | sed '$d')
}

jf() { python3 -c "import json,sys;d=json.loads(sys.argv[1]);print(eval(sys.argv[2]))" "$1" "$2"; }
# capfield FIELD CODE -> 当前 BODY 中指定编码对象的字段值;取不到回退 <err>(不刷栈)。
capfield() { jf "$BODY" "[x['$1'] for x in d['data']['items'] if x['code']=='$2'][0]" 2>/dev/null || echo "<err>"; }

PASS_N=0; FAIL_N=0; FAILED_IDS=""
ok() { PASS_N=$((PASS_N+1)); echo "PASS: [$1] $2"; }
bad() {
  FAIL_N=$((FAIL_N+1)); FAILED_IDS="$FAILED_IDS $1"
  echo "FAIL: [$1] $2" >&2
  echo "[e2e-oss-capacity] ASSERTION FAILED id=$1 detail=$2" >&2
}
assert_eq() { if [ "$2" = "$3" ]; then ok "$1" "$2 ($4)"; else bad "$1" "期望=$2 实际=$3 上下文=$4"; fi; }
scan_alerts() { # scan_alerts STATUS -> HIGH 对象指定状态容量告警条数
  sqlval "SELECT count(*) FROM alarms WHERE source='capacity' AND resource_id = $HIGH_ID AND status = '$1';"
}
run_scan() { req POST "/resources/capacity/alert-scan"; }

# HIGH_ID/LOW_ID/OLT_ID:造数资源 id(seed_data 落地,断言与清理共用)。
HIGH_ID=0; LOW_ID=0; OLT_ID=0
cleanup_data() {
  echo "清尾: 造数自清理(acc_cap- 前缀,幂等)" >&2
  echo "DELETE FROM alarms WHERE source='capacity' AND resource_id IN (SELECT id FROM resources WHERE code LIKE 'acc_cap-%');
DELETE FROM ports WHERE resource_id IN (SELECT id FROM resources WHERE code LIKE 'acc_cap-%');
DELETE FROM resources WHERE code LIKE 'acc_cap-%';" | sql >/dev/null
}

seed_data() {
  local ent addr
  cleanup_data
  ent=$(sqlval "SELECT min(id) FROM legal_entities;")
  addr=$(sqlval "SELECT min(id) FROM addresses;")
  if [ -z "$ent" ] || [ -z "$addr" ]; then FAIL_REASON="no legal_entities/addresses row"; return 1; fi
  HIGH_ID=$(sqlval "INSERT INTO resources(legal_entity_id, code, name, type, address_id) VALUES ($ent, 'acc_cap-SPL-HIGH', '容量e2e高分光器', 'SPLITTER', $addr) RETURNING id;")
  LOW_ID=$(sqlval "INSERT INTO resources(legal_entity_id, code, name, type, address_id) VALUES ($ent, 'acc_cap-SPL-LOW', '容量e2e低分光器', 'SPLITTER', $addr) RETURNING id;")
  OLT_ID=$(sqlval "INSERT INTO resources(legal_entity_id, code, name, type, address_id) VALUES ($ent, 'acc_cap-OLT-1', '容量e2eOLT', 'OLT', $addr) RETURNING id;")
  if [ -z "$HIGH_ID" ] || [ -z "$LOW_ID" ] || [ -z "$OLT_ID" ]; then FAIL_REASON="seed resources insert"; return 1; fi
  sqlval "INSERT INTO ports(port_code, quad_code, resource_id, legal_entity_id, legal_entity_name, address_id, region_id, region_name, status)
SELECT 'acc_cap-P-H-' || g, 'acc_cap-Q-H-' || g, $HIGH_ID, $ent, '容量e2e', $addr, 0, '容量e2e', CASE WHEN g <= 7 THEN 'USED' ELSE 'IDLE' END FROM generate_series(1, 8) g;
INSERT INTO ports(port_code, quad_code, resource_id, legal_entity_id, legal_entity_name, address_id, region_id, region_name, status)
SELECT 'acc_cap-P-L-' || g, 'acc_cap-Q-L-' || g, $LOW_ID, $ent, '容量e2e', $addr, 0, '容量e2e', CASE WHEN g <= 1 THEN 'USED' ELSE 'IDLE' END FROM generate_series(1, 8) g;
INSERT INTO ports(port_code, quad_code, resource_id, legal_entity_id, legal_entity_name, address_id, region_id, region_name, status)
SELECT 'acc_cap-P-O-' || g, 'acc_cap-Q-O-' || g, $OLT_ID, $ent, '容量e2e', $addr, 0, '容量e2e', CASE WHEN g <= 2 THEN 'USED' ELSE 'IDLE' END FROM generate_series(1, 4) g;" >/dev/null
}

echo "OSS 容量视图与阈值预警端到端实测 @ $BASE_URL"
seed_data || { bad "P0" "造数失败: $FAIL_REASON"; cleanup_data; exit 1; }
ok "P0" "造数基线 acc_cap- HIGH=$HIGH_ID LOW=$LOW_ID OLT=$OLT_ID"

# P1 聚合数值(SPLITTER 维度): HIGH 8/7/87.5,LOW 12.5(口径 USED/(USED+IDLE))
if req GET "/resources/capacity?dim=SPLITTER" && [ "$HTTP_CODE" = "200" ]; then
  got="$(capfield totalPorts acc_cap-SPL-HIGH)|$(capfield usedPorts acc_cap-SPL-HIGH)|$(capfield usageRate acc_cap-SPL-HIGH)|$(capfield usageRate acc_cap-SPL-LOW)"
  assert_eq "P1" "8|7|87.5|12.5" "$got" "SPLITTER 维度 HIGH/LOW 数值"
else
  bad "P1" "请求失败: http=${HTTP_CODE:-} $FAIL_REASON"
fi

# P2 按使用率倒序(独立重取,不依赖 P1 的 BODY)
if req GET "/resources/capacity?dim=SPLITTER" && [ "$HTTP_CODE" = "200" ]; then
  sorted_ok=$(jf "$BODY" "all(a['usageRate'] >= b['usageRate'] for a, b in zip(d['data']['items'], d['data']['items'][1:]))" 2>/dev/null || echo "False")
  assert_eq "P2" "True" "$sorted_ok" "items 按 usageRate 降序"
else
  bad "P2" "请求失败: http=${HTTP_CODE:-} $FAIL_REASON"
fi

# P3 OLT 维度: OLT 行可见且不含 SPLITTER 行
if req GET "/resources/capacity?dim=OLT" && [ "$HTTP_CODE" = "200" ]; then
  all_olt=$(jf "$BODY" "all(x['type'] == 'OLT' for x in d['data']['items'])" 2>/dev/null || echo "False")
  assert_eq "P3" "4|50|True" "$(capfield totalPorts acc_cap-OLT-1)|$(capfield usageRate acc_cap-OLT-1)|$all_olt" "OLT 维度过滤与数值"
else
  bad "P3" "请求失败: http=${HTTP_CODE:-} $FAIL_REASON"
fi

# P4 扫描一: HIGH 产生且仅产生 1 条 WARNING OPEN 容量告警
if run_scan && [ "$HTTP_CODE" = "200" ]; then
  assert_eq "P4" "1" "$(scan_alerts OPEN)" "HIGH WARNING OPEN 容量告警=1(仅一次)"
else
  bad "P4" "扫描请求失败: http=${HTTP_CODE:-} $FAIL_REASON"
fi

# P5 周期重跑幂等: 再扫一轮不重复产生
if run_scan && [ "$HTTP_CODE" = "200" ]; then
  assert_eq "P5" "1" "$(scan_alerts OPEN)" "重跑后 HIGH OPEN 容量告警仍=1"
else
  bad "P5" "扫描请求失败: http=${HTTP_CODE:-} $FAIL_REASON"
fi

# P6 回落关闭: 使用率降为 0 → 扫描 → OPEN=0 且出现 CLOSED(恢复也是状态变化)
sqlval "UPDATE ports SET status = 'IDLE' WHERE resource_id = $HIGH_ID;" >/dev/null
if run_scan && [ "$HTTP_CODE" = "200" ]; then
  closed_n=$(scan_alerts CLOSED)
  closed_s=no; [ "$closed_n" -ge 1 ] 2>/dev/null && closed_s=yes
  assert_eq "P6" "0|yes" "$(scan_alerts OPEN)|$closed_s" "HIGH 回落后 OPEN=0 且存在 CLOSED"
else
  bad "P6" "扫描请求失败: http=${HTTP_CODE:-} $FAIL_REASON"
fi

# P7 再越限: 恢复 7 USED → 扫描 → 又产生 1 条 OPEN(状态变化才重复告警)
sqlval "UPDATE ports SET status = CASE WHEN right(port_code, 1) IN ('1','2','3','4','5','6','7') THEN 'USED' ELSE 'IDLE' END WHERE resource_id = $HIGH_ID;" >/dev/null
if run_scan && [ "$HTTP_CODE" = "200" ]; then
  assert_eq "P7" "1" "$(scan_alerts OPEN)" "再越限后 HIGH 重新产生 1 条 OPEN"
else
  bad "P7" "扫描请求失败: http=${HTTP_CODE:-} $FAIL_REASON"
fi

# P8 前端构建产物含新页面路由(可用 SKIP_FRONTEND=1 跳过)
if [ "$(env_or SKIP_FRONTEND 0)" = "1" ]; then
  ok "P8" "跳过(SKIP_FRONTEND=1)"
elif command -v pnpm >/dev/null 2>&1; then
  (cd "$ROOT/web/admin" && pnpm build >/dev/null 2>&1) || bad "P8" "pnpm build 失败"
  if grep -rq '/oss/capacity' "$ROOT/web/admin/dist/assets/" 2>/dev/null; then
    ok "P8" "构建产物含新页面路由 /oss/capacity"
  else
    bad "P8" "构建产物未含 /oss/capacity"
  fi
else
  bad "P8" "pnpm 不可用(可 SKIP_FRONTEND=1 跳过)"
fi

rc=0
if [ "$(env_or SKIP_CLEANUP 0)" != "1" ]; then
  cleanup_data
  res1=$(sqlval "SELECT count(*) FROM resources WHERE code LIKE 'acc_cap-%';")
  res2=$(sqlval "SELECT count(*) FROM ports WHERE port_code LIKE 'acc_cap-%';")
  res3=$(sqlval "SELECT count(*) FROM alarms WHERE source='capacity' AND resource_id IN (SELECT id FROM resources WHERE code LIKE 'acc_cap-%');")
  if [ "$res1$res2$res3" = "000" ]; then
    ok "RES" "造数清理后残留=0(resources/ports/alarms)"
  else
    bad "RES" "残留 resources=$res1 ports=$res2 alarms=$res3(应为零)"
    rc=1
  fi
fi

echo "断言明细: PASS=$PASS_N FAIL=$FAIL_N 失败编号=[$FAILED_IDS ]"
if [ "$FAIL_N" -eq 0 ] && [ "$rc" -eq 0 ]; then
  echo "E2E-OSS-CAPACITY RESULT: PASS pass=$PASS_N"
  exit 0
fi
echo "E2E-OSS-CAPACITY RESULT: FAIL pass=$PASS_N fail=$FAIL_N failed=[$FAILED_IDS ]"
exit 1