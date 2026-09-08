#!/usr/bin/env bash
# 复盘整改 Phase C1 验收:补走体验打磨期间因缺数据未真提交的三条动作链。
# 102 真实环境;造数 acc_ 前缀,跑完即清;输出 ACCEPTANCE PASS/FAIL。
# 链1 师傅调队(全真):建组B→建师傅→调队→断言台账→复位→软删组B。
# 链2 派单指派(契约探针):无存量 PENDING 工单,assign 对不存在单号的拒绝信封。
# 链3 催收执行(契约探针):列表形状 + status 对不存在任务的拒绝信封。
# 附:Phase A 新端点 POST /storage-config/test 存活冒烟。
# 用法: scripts/ops/debt-acceptance.sh
set -uo pipefail
BASE=${BASE:-http://192.168.0.102:28080}
API="$BASE/api/admin/v1"

TOKEN=$(curl -sf -m 10 "$API/auth/login" -X POST -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')
[ -n "$TOKEN" ] || { echo "LOGIN FAILED"; exit 1; }
AH="Authorization: Bearer $TOKEN"
CT='Content-Type: application/json'
fails=0
api() { curl -sf -m 15 "$API$1" -X "${2:-GET}" -H "$AH" -H "$CT" ${3:+-d "$3"}; }
codeof() { printf '%s' "$1" | sed -n 's/.*"code":\([0-9]*\).*/\1/p'; }

# ── 链1 师傅调队:全真(动态取真实班组→建组B→建师傅→调队→复位→清理) ──
STAMP=$(date +%s)
GROUP_MAIN=$(api /worker-groups | sed -n 's/.*"items":\[{"id":\([0-9]*\).*/\1/p')
[ -n "$GROUP_MAIN" ] || { echo "FAIL 无既有班组可挂师傅(需先建一组)"; fails=$((fails+1)); }
GROUP_B=$(api /worker-groups POST "{\"name\":\"acc_验收组B$STAMP\",\"code\":\"accB$STAMP\",\"legalEntityId\":1}" | sed -n 's/.*"id":\([0-9]*\).*/\1/p')
[ -n "$GROUP_B" ] && echo "OK 建组B id=$GROUP_B" || { echo "FAIL 建组B"; fails=$((fails+1)); }

WORKER=$(api /workers POST "{\"name\":\"acc_验收师傅$STAMP\",\"staffNo\":\"acc$STAMP\",\"phone\":\"09171234567\",\"groupId\":$GROUP_MAIN,\"regionId\":1,\"regionIds\":[1],\"password\":\"accPass$STAMP\"}" | sed -n 's/.*"data":{"id":\([0-9]*\).*/\1/p')
[ -n "$WORKER" ] && echo "OK 建师傅 id=$WORKER(主组=$GROUP_MAIN)" || { echo "FAIL 建师傅"; fails=$((fails+1)); }

if [ -n "$GROUP_B" ] && [ -n "$WORKER" ]; then
  # 查现班组(调队前),用于复位
  BEFORE=$(api "/workers/$WORKER" | sed -n 's/.*"groupId":\([0-9]*\).*/\1/p')
  R=$(api "/workers/$WORKER/transfer" POST "{\"groupId\":$GROUP_B,\"reason\":\"acc_验收调队\"}")
  [ "$(codeof "$R")" = "0" ] && echo "OK 调队提交 code=0" || { echo "FAIL 调队 code=$(codeof "$R")"; fails=$((fails+1)); }
  AFTER=$(api "/workers/$WORKER" | sed -n 's/.*"groupId":\([0-9]*\).*/\1/p')
  [ "$AFTER" = "$GROUP_B" ] && echo "OK 调队生效 groupId=$AFTER" || { echo "FAIL 调队后 groupId=$AFTER want $GROUP_B"; fails=$((fails+1)); }
  GN=$(api "/workers/$WORKER" | sed -n 's/.*"groupName":"\([^"]*\)".*/\1/p')
  case "$GN" in acc_*) echo "OK groupName 回显=$GN";; *) echo "FAIL groupName 回显=$GN"; fails=$((fails+1));; esac
  # 复位到原班组
  R=$(api "/workers/$WORKER/transfer" POST "{\"groupId\":$BEFORE,\"reason\":\"acc_验收复位\"}")
  [ "$(codeof "$R")" = "0" ] && echo "OK 复位 code=0" || echo "WARN 复位 code=$(codeof "$R")(师傅留 acc_ 标记,人工可清)"
  # 师傅无删除接口:置离职+改名保留 acc_ 前缀痕迹
  api "/workers/$WORKER" PUT "{\"name\":\"acc_已离职验收$STAMP\",\"status\":0}" >/dev/null
  # 软删组B(复位后无在职成员才允许)
  D=$(api "/worker-groups/$GROUP_B" DELETE)
  [ "$(codeof "$D")" = "0" ] && echo "OK 组B软删" || echo "WARN 组B软删 code=$(codeof "$D")(有成员时 40900 属预期保护)"
else
  echo "SKIP 调队链(种子失败,不产生半程数据)"
fi

# ── 链2 派单指派:契约探针(不存在单号 → 干净拒绝信封) ─────────────
R=$(api "/dispatch/pool/acc-NO-SUCH-TICKET/assign" POST '{"workerId":1}')
C=$(codeof "$R"); case "$C" in 40400|40401|42200|40900) echo "OK 指派探针 code=$C(干净拒绝)";; *) echo "FAIL 指派探针 code=$C body=$R"; fails=$((fails+1));; esac

# ── 链3 催收执行:列表形状 + status 契约探针 ───────────────────────
R=$(api "/collection-tasks")
[ "$(codeof "$R")" = "0" ] && echo "OK 催收列表信封" || { echo "FAIL 催收列表 code=$(codeof "$R")"; fails=$((fails+1)); }
R=$(api "/collection-tasks/acc-NO-SUCH/status" POST '{"status":"DONE","note":"acc_探针"}')
C=$(codeof "$R"); case "$C" in 40400|40401|42200|40900) echo "OK 催收探针 code=$C";; *) echo "FAIL 催收探针 code=$C body=$R"; fails=$((fails+1));; esac

# ── 附:Phase A 新端点存活冒烟(真探 102 MinIO,预期 ok=true) ───────
R=$(api "/storage-config/test" POST)
[ "$(codeof "$R")" = "0" ] && echo "OK storage/test code=0 → $(printf '%s' "$R" | head -c 160)" || echo "WARN storage/test code=$(codeof "$R")(CI 未部署新 main 时 404 属预期)"

[ "$fails" = 0 ] && echo "ACCEPTANCE PASS" || { echo "ACCEPTANCE FAIL($fails)"; exit 1; }
