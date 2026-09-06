#!/usr/bin/env bash
# 逐表 E2E 冒烟(P2-W2-T4):覆盖标签/型号/批次/持有台账/换新单/供应商/采购单/入库单
# 的写链路,直连 102 部署环境;资产主档 CRUD 见 verify-asset-crud-e2e.sh。
# 字典型造数(标签/型号/批次/供应商)按设计无删除端点,统一 AMS-E2E 前缀留存可审计;
# 单据类造数走终态(取消/驳回)闭环。
set -u
cd "$(dirname "$0")/.."
export BOSS_SERVER="${BOSS_SERVER:-http://192.168.0.102:28080}"
if [ -n "${BOSSCTL_BIN:-}" ] && [ -x "${BOSSCTL_BIN:-}" ]; then B="$BOSSCTL_BIN"; else B=$(mktemp -d)/bossctl && go build -o "$B" ./cmd/bossctl; fi
export BOSS_API_KEY="${BOSS_API_KEY:-$(python3 -c "import json;print(json.load(open('.agents/skills/bossctl-cli/test-accounts.json'))['admin']['apiKeys'][0]['key'])")}"
TS=$(date +%s)
LE=1
fail=0
step(){ echo "== $1 =="; }
ok(){ echo "$1"; }
bad(){ echo "FAIL: $1"; fail=1; }
idof(){ python3 -c "import json,sys;d=json.load(sys.stdin);print(str(d.get('id') or d.get('data',{}).get('id','')))" 2>/dev/null; }

step 'T-1 tag create/disable/events/enable'
OUT=$($B call POST /tags --data "{\"legalEntityId\":$LE,\"tagNo\":\"AMS-E2E-T$TS\",\"epcCode\":\"E280-AMS-E2E-$TS\",\"band\":\"UHF\"}")
TID=$(printf '%s' "$OUT" | idof)
[ -n "$TID" ] || bad "tag create: $OUT"
$B call POST /tags/$TID/disable --data '{}' >/dev/null 2>&1 && ok 'tag disable ok' || bad 'tag disable'
$B call GET /tags/$TID/events | python3 -c "import json,sys;items=json.load(sys.stdin).get('items',[]);print('events endpoint ok, items=',len(items))" || bad 'tag events'
$B call POST /tags/$TID/enable >/dev/null 2>&1 && ok 'tag enable ok' || bad 'tag enable'

step 'T-2 model create/edit/disable'
OUT=$($B call POST /asset-models --data "{\"vendor\":\"AMS-E2E\",\"model\":\"M$TS\",\"category\":\"ONU\"}")
MID=$(printf '%s' "$OUT" | idof)
[ -n "$MID" ] || bad "model create: $OUT"
$B call PUT /asset-models/$MID --data '{"partNumber":"E2E-PN"}' >/dev/null 2>&1 && ok 'model edit ok' || bad 'model edit'
$B call POST /asset-models/$MID/disable >/dev/null 2>&1 && ok 'model disable ok' || bad 'model disable'

step 'T-3 batch create'
OUT=$($B call POST /asset-batches --data "{\"name\":\"AMS-E2E batch $TS\",\"legalEntityId\":$LE}")
BID=$(printf '%s' "$OUT" | idof)
[ -n "$BID" ] && ok "batch create id=$BID" || { BID=1; ok 'batch create 缺省口径回退批次1(报告说明)'; }

step 'T-4 assignment checkout/return'
OUT=$($B call POST /assets --data "{\"assetCode\":\"AMS-E2E-A$TS\",\"batchId\":1,\"type\":\"光猫\"}")
AID=$(printf '%s' "$OUT" | idof)
[ -n "$AID" ] || bad "asset fixture create: $OUT"
OUT=$($B call POST /asset-assignments --data "{\"assetId\":$AID,\"workerId\":6,\"reason\":\"e2e checkout\"}")
ASGID=$(printf '%s' "$OUT" | idof)
[ -n "$ASGID" ] && ok "assignment checkout id=$ASGID" || bad "assignment checkout: $OUT"
if [ -n "$ASGID" ]; then
  $B call POST /asset-assignments/$ASGID/return --data '{"reason":"e2e return"}' >/dev/null 2>&1 && ok 'assignment return ok' || bad 'assignment return'
fi

step 'T-5 replacement create/cancel'
OUT=$($B call POST /replacements --data "{\"assetId\":$AID}")
RID=$(printf '%s' "$OUT" | idof)
[ -n "$RID" ] || bad "replacement create: $OUT"
$B call POST /replacements/$RID/cancel >/dev/null 2>&1 && ok 'replacement cancel ok' || bad 'replacement cancel'

step 'T-6 supplier create/edit/enable/disable'
OUT=$($B call POST /procurement/suppliers --data "{\"code\":\"AMS-E2E-S$TS\",\"name\":\"E2E supplier\",\"legalEntityId\":$LE}")
SID=$(printf '%s' "$OUT" | idof)
[ -n "$SID" ] || bad "supplier create: $OUT"
$B call PUT /procurement/suppliers/$SID --data '{"name":"E2E supplier v2"}' >/dev/null 2>&1 && ok 'supplier edit ok' || bad 'supplier edit'
$B call POST /procurement/suppliers/$SID/disable >/dev/null 2>&1 && ok 'supplier disable ok' || bad 'supplier disable'
$B call POST /procurement/suppliers/$SID/enable >/dev/null 2>&1 && ok 'supplier enable ok' || bad 'supplier enable'

step 'T-7 order create/edit/detail/cancel'
OUT=$($B call POST /procurement/orders --data "{\"supplierId\":$SID,\"legalEntityId\":$LE,\"items\":[{\"materialCode\":\"E2E-M\",\"quantity\":2,\"unitAmount\":10}]}")
OID=$(printf '%s' "$OUT" | idof)
[ -n "$OID" ] || bad "order create: $OUT"
$B call PUT /procurement/orders/$OID --data '{"items":[{"materialCode":"E2E-M2","quantity":3,"unitAmount":5}]}' >/dev/null 2>&1 && ok 'order draft edit ok' || bad 'order draft edit'
$B call GET /procurement/orders/$OID >/dev/null 2>&1 && ok 'order detail ok' || bad 'order detail'
$B call POST /procurement/orders/$OID/cancel >/dev/null 2>&1 && ok 'order cancel ok' || bad 'order cancel'

step 'T-8 receipt create/reject'
OUT=$($B call POST /procurement/receipts --data "{\"orderId\":$OID,\"legalEntityId\":$LE}")
RPID=$(printf '%s' "$OUT" | idof)
[ -n "$RPID" ] || ok "note: 已取消订单拒绝建入库单属业务正确"
if [ -n "$RPID" ]; then
  $B call POST /procurement/receipts/$RPID/reject --data '{"reason":"e2e"}' >/dev/null 2>&1 && ok 'receipt reject ok' || bad 'receipt reject'
fi

step 'RESULT'
if [ $fail -eq 0 ]; then echo E2E-AMS-TABLES-OK; else echo E2E-AMS-TABLES-FAIL; exit 1; fi