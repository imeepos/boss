#!/usr/bin/env bash
# 资产台账 CRUD 生产冒烟(P2-W1-T4 验收):建档→详情→受限编辑→守卫删除→报废拒硬删。
# 直连 102 部署环境(BOSS_SERVER 缺省 http://192.168.0.102:28080),admin key 取
# .agents/skills/bossctl-cli/test-accounts.json;造数带 ACC-E2E 前缀,可清理项当场清理。
set -u
cd "$(dirname "$0")/.."
export BOSS_SERVER="${BOSS_SERVER:-http://192.168.0.102:28080}"

if [ -x ./bossctl ]; then B=./bossctl; elif command -v bossctl >/dev/null 2>&1; then B=bossctl; else go build -o bossctl ./cmd/bossctl && B=./bossctl; fi
export BOSS_API_KEY="${BOSS_API_KEY:-$(python3 -c "import json;print(json.load(open('.agents/skills/bossctl-cli/test-accounts.json'))['admin']['apiKeys'][0]['key'])")}"

TS=$(date +%s)
CODE=ACC-E2E-$TS
CREATED=""
fail=0
step(){ echo "== $1 =="; }
idof(){ python3 -c "import json,sys;d=json.load(sys.stdin);print(str(d.get('id') or d.get('data',{}).get('id','')))" 2>/dev/null; }

step 'C-1 create asset'
OUT=$($B call POST /assets --data "{\"assetCode\":\"$CODE\",\"batchId\":1,\"type\":\"光猫\"}")
ID=$(printf '%s' "$OUT" | idof)
[ -n "$ID" ] || { echo "FAIL create: $OUT"; exit 1; }
CREATED="$CREATED $ID"
echo "created id=$ID code=$CODE"

step 'C-2 read detail'
$B call GET /assets/$ID | python3 -c "import json,sys;a=json.load(sys.stdin);assert str(a.get('assetId'))=='$ID',a;assert a.get('assetCode')=='$CODE',a;assert a.get('type')=='光猫',a;assert a.get('status')=='IN_STOCK',a;assert a.get('legalEntityName'),a;print('read ok: code/type/status/entity snapshot correct')" || fail=1

step 'C-3 bounded update'
$B call PUT /assets/$ID --data '{"type":"路由器"}' | python3 -c "import json,sys;a=json.load(sys.stdin);assert a.get('type')=='路由器',a;print('update ok type=路由器')" || fail=1

step 'C-4 guarded delete happy path'
$B call DELETE /assets/$ID >/dev/null && echo 'delete ok' || fail=1
if $B call GET /assets/$ID >/dev/null 2>&1; then echo 'FAIL: deleted asset still readable'; fail=1; else echo 'read-after-delete rejected as expected'; fi

step 'C-5 scrap fixture rejects hard delete'
OUT=$($B call POST /assets --data "{\"assetCode\":\"ACC-E2E-SCRAP-$TS\",\"batchId\":1,\"type\":\"ONU\"}")
ID2=$(printf '%s' "$OUT" | idof)
[ -n "$ID2" ] || { echo "FAIL create scrap fixture: $OUT"; exit 1; }
CREATED="$CREATED $ID2"
$B call POST /assets/$ID2/scrap --data '{"reason":"e2e-guard"}' >/dev/null && echo 'scrap ok'
if $B call DELETE /assets/$ID2 2>&1 | grep -q '40900'; then echo 'guard ok: scrapped hard-delete rejected 40900'; else echo 'FAIL: expected 40900 on scrapped hard delete'; fail=1; fi

step 'cleanup (non-scrapped fixtures)'
for id in $CREATED; do
  $B call DELETE /assets/$id >/dev/null 2>&1 && echo "cleaned $id" || echo "keep $id (blocked, expected for scrap fixture)"
done

step 'RESULT'
if [ $fail -eq 0 ]; then echo E2E-ASSET-CRUD-OK; else echo E2E-ASSET-CRUD-FAIL; exit 1; fi