#!/usr/bin/env bash
# verify-asset-pagination-e2e.sh -- 资产/标签列表服务端分页端到端实测(P3-T1,102 真实部署,禁 mock)。
# 断言:
#   P0 造数: acc_pg- 前缀 210 资产(其中 0101-0120 DEPLOYED)+ 210 标签(独立批次,幂等重跑)
#   P1 默认分页: 不带参数 items<=50 且有 total(assets/tags)
#   P2 total 正确: q=acc_pg 过滤后 total==210(SQL COUNT 同口径)
#   P3 翻页不重不漏: 第1/2页 assetId 无重复,并集+第3页=过滤后全集 210
#   P4 limit 钳制: limit=99999 实际返回 200(210>200),HTTP 200 不报错(assets/tags)
#   P5 排序白名单: sort=battery / tags sort=asset_code -> HTTP 400 + code 42200
#   P6 合法排序: assets sort=asset_code DESC 生效(首行=acc_pg-A-0210)
#   P7 q 前缀+status 组合: q=acc_pg-A-01&status=DEPLOYED 命中 20(SQL 同口径)
#   P8 tags status 过滤: UNBOUND=210 / BOUND=0
#   P9 默认排序: created_at DESC, id DESC tie-breaker(items[0]=0210,同秒插入序)
#   RES 收尾清理造数后残留断言为零(脚本可重复执行)
# 信号: 每条断言独立输出 "PASS: [编号] ..." / "FAIL: [编号] ...";收尾 "E2E-ASSET-PAGINATION RESULT: ..." 可 grep。
# 用法: scripts/e2e/verify-asset-pagination-e2e.sh [BASE_URL]
# 环境: BASE_URL ADMIN_API_KEY SSH_HOST SKIP_CLEANUP=1
# 依赖: curl python3 ssh(102 免密);鉴权 X-API-Key(test-accounts.json admin key)。
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

# sql: 102 断言/造数 SQL(stdin 传 SQL 防叠引号)。
sql() { ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST" "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q -tA"; }

FAIL_REASON=""
# req PATH -> HTTP_CODE/BODY;业务失败也保留响应供断言(本脚本要断言 400)。
req() {
  local out
  out=$(curl -sS -m 30 -w $'\n%{http_code}' "$API$1" -H "X-API-Key: $KEY") || { FAIL_REASON="curl $1"; return 1; }
  HTTP_CODE=$(printf '%s\n' "$out" | tail -n1 | tr -d '[:space:]')
  BODY=$(printf '%s\n' "$out" | sed '$d')
}

# jf BODY EXPR -> python 求值(表达式基于解析后的 d)。
jf() { python3 -c "import json,sys;d=json.loads(sys.argv[1]);print(eval(sys.argv[2]))" "$1" "$2"; }

PASS_N=0; FAIL_N=0; FAILED_IDS=""
ok() { PASS_N=$((PASS_N+1)); echo "PASS: [$1] $2"; }
bad() {
  FAIL_N=$((FAIL_N+1)); FAILED_IDS="$FAILED_IDS $1"
  echo "FAIL: [$1] $2" >&2
  echo "[e2e-asset-pagination] ASSERTION FAILED id=$1 detail=$2" >&2
}
assert_eq() { if [ "$2" = "$3" ]; then ok "$1" "$2 ($4)"; else bad "$1" "期望=$2 实际=$3 上下文=$4"; fi; }

sqlval() { sql < <(echo "$1") | tr -d '[:space:]'; }

cleanup_data() {
  echo "清尾: 造数自清理(acc_pg- 前缀,幂等)"
  echo "DELETE FROM assets WHERE asset_code LIKE 'acc_pg-%';
DELETE FROM tags WHERE tag_no LIKE 'acc_pg-%';
DELETE FROM asset_batches WHERE code = 'acc_pg-batch';" | sql >/dev/null
}

seed_data() { # P0 造数(先清再插,幂等)并回读校验
  local ent batch n_asset n_tag
  cleanup_data
  ent=$(sqlval "SELECT min(id) FROM legal_entities;")
  if [ -z "$ent" ]; then FAIL_REASON="no legal_entities row"; return 1; fi
  batch=$(sqlval "INSERT INTO asset_batches(legal_entity_id, code, name) VALUES ($ent, 'acc_pg-batch', '分页e2e批次') RETURNING id;")
  if [ -z "$batch" ]; then FAIL_REASON="seed batch insert"; return 1; fi
  local dml
  dml="INSERT INTO assets(asset_code, batch_id, legal_entity_id, legal_entity_name, type, status)"
  dml="$dml SELECT 'acc_pg-A-' || lpad(g::text, 4, '0'), $batch, $ent, '分页e2e主体', 'ONU',"
  dml="$dml CASE WHEN g BETWEEN 101 AND 120 THEN 'DEPLOYED' ELSE 'IN_STOCK' END FROM generate_series(1, 210) g;"
  dml="$dml INSERT INTO tags(legal_entity_id, tag_no, epc_code, band, status, battery)"
  dml="$dml SELECT $ent, 'acc_pg-T-' || lpad(g::text, 4, '0'), 'ACCPG-EPC-' || lpad(g::text, 4, '0'), 'UHF', 'UNBOUND', '100%' FROM generate_series(1, 210) g;"
  sql < <(echo "$dml") >/dev/null || { FAIL_REASON="seed bulk insert"; return 1; }
  n_asset=$(sqlval "SELECT count(*) FROM assets WHERE asset_code LIKE 'acc_pg-%';")
  n_tag=$(sqlval "SELECT count(*) FROM tags WHERE tag_no LIKE 'acc_pg-%';")
  assert_eq "P0" "210|210" "$n_asset|$n_tag" "造数基线 acc_pg- 210 资产+210 标签"
}

echo "资产/标签列表服务端分页端到端实测 @ $BASE_URL"
seed_data || { bad "P0" "造数失败: $FAIL_REASON"; cleanup_data; exit 1; }

# P1 默认分页(不带参数): items<=50 且 total 存在(assets/tags)
for res in assets tags; do
  if req "/$res"; then
    n=$(jf "$BODY" "len(d['data']['items'])")
    total=$(jf "$BODY" "d['data']['total']")
    if [ "$HTTP_CODE" = "200" ] && [ "$n" -le 50 ] && [ -n "$total" ] && [ "$total" -ge 1 ]; then
      ok "P1-$res" "默认分页 items=$n(<=50) total=$total"
    else
      bad "P1-$res" "http=$HTTP_CODE items=$n total=$total(期望 items<=50 且 total>=1)"
    fi
  else
    bad "P1-$res" "请求失败: $FAIL_REASON"
  fi
done

# P2 过滤后 total 与 SQL 同口径(assets/tags = 210)
if req "/assets?q=acc_pg"; then
  total=$(jf "$BODY" "d['data']['total']")
  assert_eq "P2-assets" "210" "$total" "q 前缀过滤 total"
else
  bad "P2-assets" "请求失败: $FAIL_REASON"
fi
if req "/tags?q=acc_pg"; then
  total=$(jf "$BODY" "d['data']['total']")
  assert_eq "P2-tags" "210" "$total" "q 前缀过滤 total"
else
  bad "P2-tags" "请求失败: $FAIL_REASON"
fi

# P3 翻页不重不漏: limit=100 两页 200 个 id 无重复,加第 3 页共 210 全覆盖
req "/assets?q=acc_pg&limit=100&offset=0" >/dev/null
IDS1=$(jf "$BODY" "','.join(str(x['assetId']) for x in d['data']['items'])")
req "/assets?q=acc_pg&limit=100&offset=100" >/dev/null
IDS2=$(jf "$BODY" "','.join(str(x['assetId']) for x in d['data']['items'])")
req "/assets?q=acc_pg&limit=100&offset=200" >/dev/null
IDS3=$(jf "$BODY" "','.join(str(x['assetId']) for x in d['data']['items'])")
VERIFY=$(python3 -c "
import sys
a='$(printf '%s' "$IDS1")'.split(','); b='$(printf '%s' "$IDS2")'.split(','); c='$(printf '%s' "$IDS3")'.split(',')
all_ids=a+b+c
dup=len(all_ids)-len(set(all_ids))
print('%d %d %d' % (len(a), dup, len(all_ids)))")
assert_eq "P3" "100 0 210" "$VERIFY" "第1页100条/跨页无重复/三页并集210"

# P4 limit 钳制: 99999 -> 实际 200(HTTP 200,不报错)
for res in assets tags; do
  if req "/$res?q=acc_pg&limit=99999"; then
    n=$(jf "$BODY" "len(d['data']['items'])")
    if [ "$HTTP_CODE" = "200" ] && [ "$n" -le 200 ]; then
      ok "P4-$res" "limit=99999 钳制后 items=$n(<=200) http=200"
    else
      bad "P4-$res" "http=$HTTP_CODE items=$n(期望钳制到 200)"
    fi
  else
    bad "P4-$res" "请求失败: $FAIL_REASON"
  fi
done

# P5 排序白名单: 越界值 HTTP 400 + code 42200(assets/tags)
if req "/assets?sort=battery"; then
  code=$(jf "$BODY" "d['code']")
  assert_eq "P5-assets" "400|42200" "$HTTP_CODE|$code" "sort=battery 拒绝"
else
  bad "P5-assets" "请求失败: $FAIL_REASON"
fi
if req "/tags?sort=asset_code"; then
  code=$(jf "$BODY" "d['code']")
  assert_eq "P5-tags" "400|42200" "$HTTP_CODE|$code" "tags sort=asset_code 不在白名单"
else
  bad "P5-tags" "请求失败: $FAIL_REASON"
fi

# P6 合法排序: sort=asset_code DESC 生效(零填充字典序,首行=0210)
if req "/assets?q=acc_pg&sort=asset_code&limit=5"; then
  first=$(jf "$BODY" "d['data']['items'][0]['assetCode']")
  assert_eq "P6" "acc_pg-A-0210" "$first" "sort=asset_code DESC 首行"
else
  bad "P6" "请求失败: $FAIL_REASON"
fi

# P7 q 前缀+status 组合: acc_pg-A-01 前缀命中 0100-0199;叠加 DEPLOYED(0101-0120)为 20
want=$(sqlval "SELECT count(*) FROM assets WHERE asset_code LIKE 'acc_pg-A-01%' AND status='DEPLOYED';")
if req "/assets?q=acc_pg-A-01&status=DEPLOYED"; then
  total=$(jf "$BODY" "d['data']['total']")
  assert_eq "P7" "$want" "$total" "q+status 组合过滤(SQL 口径=$want)"
else
  bad "P7" "请求失败: $FAIL_REASON"
fi

# P8 tags status 过滤: UNBOUND=210 / BOUND=0
if req "/tags?q=acc_pg&status=UNBOUND"; then
  total=$(jf "$BODY" "d['data']['total']")
  assert_eq "P8-unbound" "210" "$total" "status=UNBOUND"
else
  bad "P8-unbound" "请求失败: $FAIL_REASON"
fi
if req "/tags?q=acc_pg&status=BOUND"; then
  total=$(jf "$BODY" "d['data']['total']")
  assert_eq "P8-bound" "0" "$total" "status=BOUND(造数全 UNBOUND)"
else
  bad "P8-bound" "请求失败: $FAIL_REASON"
fi

# P9 默认排序: 同秒插入 created_at DESC 后按 id DESC tie-breaker,过滤后首行=0210
if req "/assets?q=acc_pg"; then
  first=$(jf "$BODY" "d['data']['items'][0]['assetCode']")
  assert_eq "P9" "acc_pg-A-0210" "$first" "默认排序 created_at DESC, id DESC"
else
  bad "P9" "请求失败: $FAIL_REASON"
fi

rc=0
if [ "$(env_or SKIP_CLEANUP 0)" != "1" ]; then
  cleanup_data
  res=$(sqlval "SELECT count(*) FROM assets WHERE asset_code LIKE 'acc_pg-%';")
  res2=$(sqlval "SELECT count(*) FROM tags WHERE tag_no LIKE 'acc_pg-%';")
  res3=$(sqlval "SELECT count(*) FROM asset_batches WHERE code = 'acc_pg-batch';")
  if [ "$res$res2$res3" = "000" ]; then
    ok "RES" "造数清理后残留=0(assets/tags/batches)"
  else
    bad "RES" "残留 assets=$res tags=$res2 batches=$res3(应为零)"
    rc=1
  fi
fi

echo "断言明细: PASS=$PASS_N FAIL=$FAIL_N 失败编号=[$FAILED_IDS ]"
if [ "$FAIL_N" -eq 0 ] && [ "$rc" -eq 0 ]; then
  echo "E2E-ASSET-PAGINATION RESULT: PASS pass=$PASS_N"
  exit 0
fi
echo "E2E-ASSET-PAGINATION RESULT: FAIL pass=$PASS_N fail=$FAIL_N failed=[$FAILED_IDS ]"
exit 1
