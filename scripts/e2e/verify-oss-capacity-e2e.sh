#!/usr/bin/env bash
# verify-oss-capacity-e2e.sh -- OSS 容量视图与阈值预警端到端实测(P5-W1,102 真实部署,禁 mock)。
# 性能契约(Lead 验收运行器 8s 硬超时):稳态重跑必须 <8s——
#   ①SQL 批处理:全程 6 次 ssh(造数 1/断言对 4/清尾+残留 1),单查询合并多计数;
#   ②P8 指纹增量:对 web/admin/src 做 git 内容指纹,指纹未变且既有 dist 含路由特征
#     则跳过重建直接断言(冷跑首建允许超时一次,重跑变热)。
# 断言:
#   P0 造数: acc_cap- 前缀两分光器(HIGH 8 端口 7 USED=87.50% / LOW 8 端口 1 USED=12.50%)
#        + OLT 4 端口 2 USED=50.00%(幂等重跑:单连接先清后造)
#   P1 聚合 API(dim=SPLITTER): HIGH total=8 used=7 rate=87.5,LOW rate=12.5
#   P2 倒序: items 按 usageRate 降序(独立重取,任务书:按使用率倒序)
#   P3 聚合 API(dim=OLT): OLT 行可见且不含 SPLITTER 行
#   P4 扫描一: HIGH 产生 1 条 WARNING OPEN 容量告警(open|warn|closed=1|1|0,仅一次)
#   P5 重跑幂等: 扫描二 → 仍 1|1|0(不重复)
#   P6 回落关闭: SQL 降使用率+扫描 → 0|0|1(回落自动关闭,状态变化)
#   P7 再越限: SQL 恢复 7 USED+扫描 → 1|1|1(状态变化才重复告警)
#   P8 前端构建产物含新页面路由 /oss/capacity(指纹增量)
#   RES 收尾清理造数(acc_cap- 资源/端口/容量告警)后残留=0|0|0(脚本可重复执行)
# 信号: 每条断言独立输出 "PASS: [编号] ..." / "FAIL: [编号] ...";收尾
#   "E2E-OSS-CAPACITY RESULT: ..." 可 grep。
# 用法: scripts/e2e/verify-oss-capacity-e2e.sh [BASE_URL]
# 环境: BASE_URL ADMIN_API_KEY SSH_HOST SKIP_CLEANUP=1 SKIP_FRONTEND=1
# 依赖: curl python3 ssh(102 免密);P8 冷跑需 pnpm;鉴权 X-API-Key(test-accounts.json)。
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

# sqlq 单连接批处理:一次 ssh 跑多条 SQL,只回显 SELECT 行(psql -q 抑制命令标签)。
sqlq() { ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST" "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q -tA"; }
sqlval() { sqlq < <(printf '%s\n' "$1") | tr -d '[:space:]'; }

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
# trio 高分光器告警三元组计数(open|warn|closed),单查询带回。
TRIO_SEL="count(*) FILTER (WHERE status = 'OPEN') || '|' || count(*) FILTER (WHERE status = 'OPEN' AND level = 'WARNING') || '|' || count(*) FILTER (WHERE status = 'CLOSED')"
trio() { sqlval "SELECT $TRIO_SEL FROM alarms WHERE source = 'capacity' AND resource_id = $HIGH_ID;"; }
run_scan() { req POST "/resources/capacity/alert-scan"; }

# HIGH_ID/LOW_ID/OLT_ID:造数资源 id(单连接造数落地,断言与清理共用)。
HIGH_ID=0; LOW_ID=0; OLT_ID=0

echo "OSS 容量视图与阈值预警端到端实测 @ $BASE_URL"
# P0 单连接造数:先清后造(幂等),回显三行 id|code
SEED_OUT=$(sqlq <<SQL
DELETE FROM alarms WHERE source='capacity' AND resource_id IN (SELECT id FROM resources WHERE code LIKE 'acc_cap-%');
DELETE FROM ports WHERE resource_id IN (SELECT id FROM resources WHERE code LIKE 'acc_cap-%');
DELETE FROM resources WHERE code LIKE 'acc_cap-%';
INSERT INTO resources(legal_entity_id, code, name, type, address_id)
SELECT (SELECT min(id) FROM legal_entities), 'acc_cap-SPL-HIGH', '容量e2e高分光器', 'SPLITTER', (SELECT min(id) FROM addresses);
INSERT INTO resources(legal_entity_id, code, name, type, address_id)
SELECT (SELECT min(id) FROM legal_entities), 'acc_cap-SPL-LOW', '容量e2e低分光器', 'SPLITTER', (SELECT min(id) FROM addresses);
INSERT INTO resources(legal_entity_id, code, name, type, address_id)
SELECT (SELECT min(id) FROM legal_entities), 'acc_cap-OLT-1', '容量e2eOLT', 'OLT', (SELECT min(id) FROM addresses);
INSERT INTO ports(port_code, quad_code, resource_id, legal_entity_id, legal_entity_name, address_id, region_id, region_name, status)
SELECT 'acc_cap-P-H-' || g, 'acc_cap-Q-H-' || g, r.id, r.legal_entity_id, '容量e2e', r.address_id, 0, '容量e2e', CASE WHEN g <= 7 THEN 'USED' ELSE 'IDLE' END FROM resources r, generate_series(1, 8) g WHERE r.code = 'acc_cap-SPL-HIGH';
INSERT INTO ports(port_code, quad_code, resource_id, legal_entity_id, legal_entity_name, address_id, region_id, region_name, status)
SELECT 'acc_cap-P-L-' || g, 'acc_cap-Q-L-' || g, r.id, r.legal_entity_id, '容量e2e', r.address_id, 0, '容量e2e', CASE WHEN g <= 1 THEN 'USED' ELSE 'IDLE' END FROM resources r, generate_series(1, 8) g WHERE r.code = 'acc_cap-SPL-LOW';
INSERT INTO ports(port_code, quad_code, resource_id, legal_entity_id, legal_entity_name, address_id, region_id, region_name, status)
SELECT 'acc_cap-P-O-' || g, 'acc_cap-Q-O-' || g, r.id, r.legal_entity_id, '容量e2e', r.address_id, 0, '容量e2e', CASE WHEN g <= 2 THEN 'USED' ELSE 'IDLE' END FROM resources r, generate_series(1, 4) g WHERE r.code = 'acc_cap-OLT-1';
SELECT id || '|' || code FROM resources WHERE code LIKE 'acc_cap-%' ORDER BY code;
SQL
)
HIGH_ID=$(printf '%s\n' "$SEED_OUT" | awk -F'|' '$2=="acc_cap-SPL-HIGH"{print $1}')
LOW_ID=$(printf '%s\n' "$SEED_OUT" | awk -F'|' '$2=="acc_cap-SPL-LOW"{print $1}')
OLT_ID=$(printf '%s\n' "$SEED_OUT" | awk -F'|' '$2=="acc_cap-OLT-1"{print $1}')
if [ -n "$HIGH_ID" ] && [ -n "$LOW_ID" ] && [ -n "$OLT_ID" ]; then
  ok "P0" "造数基线 acc_cap- HIGH=$HIGH_ID LOW=$LOW_ID OLT=$OLT_ID"
else
  bad "P0" "造数失败: out=$SEED_OUT $FAIL_REASON"
  printf '%s\n' "DELETE FROM alarms WHERE source='capacity' AND resource_id IN (SELECT id FROM resources WHERE code LIKE 'acc_cap-%'); DELETE FROM ports WHERE resource_id IN (SELECT id FROM resources WHERE code LIKE 'acc_cap-%'); DELETE FROM resources WHERE code LIKE 'acc_cap-%';" | sqlq >/dev/null
  exit 1
fi

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

# P4 扫描一: HIGH 产生且仅产生 1 条 WARNING OPEN 容量告警(单连接取三元组)
if run_scan && [ "$HTTP_CODE" = "200" ]; then
  assert_eq "P4" "1|1|0" "$(trio)" "HIGH WARNING OPEN 容量告警=1(仅一次)"
else
  bad "P4" "扫描请求失败: http=${HTTP_CODE:-} $FAIL_REASON"
fi

# P5 周期重跑幂等: 再扫一轮不重复产生
if run_scan && [ "$HTTP_CODE" = "200" ]; then
  assert_eq "P5" "1|1|0" "$(trio)" "重跑后仍 1|1|0(不重复)"
else
  bad "P5" "扫描请求失败: http=${HTTP_CODE:-} $FAIL_REASON"
fi

# P6 回落关闭: 使用率降为 0 → 扫描 → 0|0|1(恢复也是状态变化)
sqlval "UPDATE ports SET status = 'IDLE' WHERE resource_id = $HIGH_ID;" >/dev/null
if run_scan && [ "$HTTP_CODE" = "200" ]; then
  assert_eq "P6" "0|0|1" "$(trio)" "HIGH 回落后 OPEN=0 且存在 CLOSED"
else
  bad "P6" "扫描请求失败: http=${HTTP_CODE:-} $FAIL_REASON"
fi

# P7 再越限: 恢复 7 USED → 扫描 → 又产生 1 条 OPEN(状态变化才重复告警)
sqlval "UPDATE ports SET status = CASE WHEN right(port_code, 1) IN ('1','2','3','4','5','6','7') THEN 'USED' ELSE 'IDLE' END WHERE resource_id = $HIGH_ID;" >/dev/null
if run_scan && [ "$HTTP_CODE" = "200" ]; then
  assert_eq "P7" "1|1|1" "$(trio)" "再越限后 HIGH 重新产生 1 条 OPEN"
else
  bad "P7" "扫描请求失败: http=${HTTP_CODE:-} $FAIL_REASON"
fi

# P8 前端构建产物断言(指纹增量:src 指纹未变且既有 dist 含路由特征则跳过重建)。
# 冷跑首建允许超时一次,重跑变热(稳态 <8s 的关键)。SKIP_FRONTEND=1 仍可跳过。
hash_stdin() { if command -v md5sum >/dev/null 2>&1; then md5sum; else md5; fi; }
route_ok() { grep -rq '/oss/capacity' "$ROOT/web/admin/dist/assets/" 2>/dev/null; }
FP_FILE="$ROOT/web/admin/dist/.capacity-e2e-fp"
new_fp=$( (cd "$ROOT" && { git ls-files -s -- web/admin/src; git status --porcelain -- web/admin/src; } 2>/dev/null) | hash_stdin | tr -d '[:space:]')
if [ "$(env_or SKIP_FRONTEND 0)" = "1" ]; then
  ok "P8" "跳过(SKIP_FRONTEND=1)"
elif [ -f "$FP_FILE" ] && [ "$(cat "$FP_FILE" 2>/dev/null)" = "$new_fp" ] && [ -d "$ROOT/web/admin/dist/assets" ] && route_ok; then
  ok "P8" "构建产物含新页面路由 /oss/capacity"
elif command -v pnpm >/dev/null 2>&1; then
  (cd "$ROOT/web/admin" && pnpm build >/dev/null 2>&1) || bad "P8" "pnpm build 失败"
  if route_ok; then
    printf '%s' "$new_fp" > "$FP_FILE" 2>/dev/null
    ok "P8" "构建产物含新页面路由 /oss/capacity"
  else
    bad "P8" "构建产物未含 /oss/capacity"
  fi
else
  bad "P8" "pnpm 不可用(可 SKIP_FRONTEND=1 跳过)"
fi

rc=0
if [ "$(env_or SKIP_CLEANUP 0)" != "1" ]; then
  res=$(sqlval "DELETE FROM alarms WHERE source = 'capacity' AND resource_id IN (SELECT id FROM resources WHERE code LIKE 'acc_cap-%');
DELETE FROM ports WHERE resource_id IN (SELECT id FROM resources WHERE code LIKE 'acc_cap-%');
DELETE FROM resources WHERE code LIKE 'acc_cap-%';
SELECT (SELECT count(*) FROM resources WHERE code LIKE 'acc_cap-%') || '|' || (SELECT count(*) FROM ports WHERE port_code LIKE 'acc_cap-%') || '|' || (SELECT count(*) FROM alarms WHERE source = 'capacity' AND resource_id IN (SELECT id FROM resources WHERE code LIKE 'acc_cap-%'));")
  if [ "$res" = "0|0|0" ]; then
    ok "RES" "造数清理后残留=0|0|0(resources/ports/alarms)"
  else
    bad "RES" "残留 res=$res(应为 0|0|0)"
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