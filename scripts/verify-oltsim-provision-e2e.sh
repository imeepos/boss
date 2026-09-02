#!/usr/bin/env bash
# 旧配置下发链路(telnet -> cmd/oltsim -> provision_tasks)E2E 回归验收。
# 流程: acc_ 隔离夹具(资源/端口/模板/订单) -> 既有 boss-provisioner(telnet driver)领取
#       preConfigOLT 任务 -> 宿主 systemd boss-oltsim 收到 provision apply ->
#       断言 task=DONE + provision_logs=SUCCESS + oltsim records/journal 留痕。
# 红线: 不停旧 provisioner;不改其配置;不启停/pkill 任何 oltsim;检测到其他 provisioner
#       容器(TL1 临时等)拒绝运行防抢队列;夹具全 acc_ 前缀,按实际 ID 精确清理,失败路径
#       trap 恢复现场。只回归旧 telnet 链路,不验证 TL1。
# 用法: scripts/verify-oltsim-provision-e2e.sh [BASE_URL]
# 环境变量: OLTSIM_HOST ADMIN_API_KEY USER_API_KEY OLTSIM_HTTP_PORT OLTSIM_TELNET_PORT
#           OLTSIM_EVID_DIR(默认 /tmp/verify-oltsim-e2e-<stamp>)
set -uo pipefail

BASE_URL="${1:-http://192.168.0.102:28080}"
API="$BASE_URL/api/admin/v1"
HOST="${OLTSIM_HOST:-imeepos@192.168.0.102}"
ADMIN_KEY="${ADMIN_API_KEY:-boss_a852c8434c1ae6370e454817dd6e497c}"
USER_KEY="${USER_API_KEY:-boss_100e5216b416080d4b95c7f3d648c6c1}"
SIM_HTTP="${OLTSIM_HTTP_PORT:-23333}"
SIM_TELNET="${OLTSIM_TELNET_PORT:-2323}"
STAMP="$(date +%s)"
PREFIX="acc_oltsim_$STAMP"
EVID_DIR="${OLTSIM_EVID_DIR:-/tmp/verify-oltsim-e2e-$STAMP}"
mkdir -p "$EVID_DIR" || exit 1

RESOURCE_ID=0; TEMPLATE_ID=0; TASK_ID=0; ORDER_ID=0; PORT_ID=0
RESERVED_PORT_ID=0; PORT_WAS_OURS=0; TASK_MODE="isolated"
ORDER_NO=""; DRV="unset"; PORT_ROW=""

fail() { echo "FAIL: $*" >&2; exit 1; }
step() { echo "==> $*"; }
info() { echo "  .. $*"; }

sql() { ssh "$HOST" "docker exec -i boss-infra-postgres-1 psql -q -v ON_ERROR_STOP=1 -U boss -d boss -Atc \"$1\""; }

api() {
  local m="$1" p="$2" b="${3:-}" out
  if [ -n "$b" ]; then
    out=$(curl -sS -f -X "$m" "$API$p" -H "X-API-Key: $ADMIN_KEY" -H 'Content-Type: application/json' -d "$b") || fail "api $m $p"
  else
    out=$(curl -sS -f -X "$m" "$API$p" -H "X-API-Key: $ADMIN_KEY") || fail "api $m $p"
  fi
  echo "$out" | jq -e '.code == 0 or .code == 200' >/dev/null || fail "api $m $p response=$out"
  echo "$out"
}

sim_records() { ssh "$HOST" "curl -sS --max-time 5 http://127.0.0.1:$SIM_HTTP/records" 2>/dev/null; }

provisioner_running() {
  [ "$(ssh "$HOST" "docker inspect -f '{{.State.Running}}' boss-provisioner 2>/dev/null")" = "true" ]
}

save_ids() {
  printf 'prefix=%s\norder_no=%s\norder_id=%s\ntask_id=%s\nresource_id=%s\nport_id=%s\nreserved_port_id=%s\ntemplate_id=%s\n' \
    "$PREFIX" "$ORDER_NO" "$ORDER_ID" "$TASK_ID" "$RESOURCE_ID" "$PORT_ID" "$RESERVED_PORT_ID" "$TEMPLATE_ID" > "$EVID_DIR/ids.txt"
}

# precheck: 既有 provisioner 在跑且队列空、无其他 provisioner 容器并跑、oltsim 存活可达。
precheck() {
  step "precheck provisioner/queue/oltsim"
  provisioner_running || fail "boss-provisioner 未运行"
  DRV=$(ssh "$HOST" "docker inspect boss-provisioner --format '{{range .Config.Env}}{{println .}}{{end}}'" | grep '^BOSS_PROVISION_DRIVER=' | cut -d= -f2)
  [ -n "$DRV" ] || DRV="log"
  echo "$DRV" > "$EVID_DIR/driver.txt"
  if [ "$DRV" != "telnet" ]; then
    echo "WARN: boss-provisioner driver=$DRV(期望 telnet)。commit cbdddb22 起 driver 需显式 BOSS_PROVISION_DRIVER=telnet(旧代码 OLT_ADDR 配置即走 telnet);log 桩只落 DONE 不真实下发,oltsim 断言将失败——这正是本脚本要暴露的旧链路回归。" >&2
  fi
  local others="" c ep dockerps
  dockerps=$(ssh "$HOST" "docker ps --format '{{.Names}}'") || fail "docker ps 失败"
  for c in $dockerps; do
    [ "$c" = "boss-provisioner" ] && continue
    ep=$(ssh "$HOST" "docker inspect '$c' --format '{{json .Config.Entrypoint}}' 2>/dev/null") || ep=""
    case "$ep" in *boss-provisioner*) others="$others $c" ;; esac
  done
  [ -z "$others" ] || fail "检测到其他 provisioner 容器在跑(拒绝并跑防抢队列):$others"
  local pend
  pend=$(sql "SELECT count(*) FROM provision_tasks WHERE status IN ('PENDING','DOING')")
  [ "$pend" = "0" ] || fail "下发队列非空 PENDING/DOING=$pend,拒绝注入任务"
  [ "$(ssh "$HOST" "systemctl is-active boss-oltsim")" = "active" ] || fail "boss-oltsim 服务非 active"
  ssh "$HOST" "timeout 3 bash -c 'exec 3<>/dev/tcp/127.0.0.1/$SIM_TELNET; head -c 6 <&3'" 2>/dev/null | grep -q login || fail "oltsim $SIM_TELNET 无 login 提示"
  sim_records | jq -e '.items' >/dev/null || fail "oltsim /records 不可读"
  info "provisioner=running driver=$DRV queue=empty oltsim=active(telnet $SIM_TELNET http $SIM_HTTP)"
}

# 隔离夹具: acc_ 资源/端口/模板(仅本次创建,清理按实际 ID)。
create_fixture() {
  step "create acc_ fixture prefix=$PREFIX"
  local res p
  res=$(api POST /provision/resources "{\"code\":\"OLT-$PREFIX\",\"name\":\"OLTSIM $PREFIX\",\"type\":\"OLT\",\"addressId\":290,\"legalEntityId\":1}")
  RESOURCE_ID=$(echo "$res" | jq -r '.data.id')
  case "$RESOURCE_ID" in ''|*[!0-9]*) fail "resource id 异常 response=$res" ;; esac
  p=$(api POST /provision/ports "{\"portCode\":\"P-$PREFIX-05\",\"resourceId\":$RESOURCE_ID,\"addressId\":290,\"legalEntityId\":1}")
  PORT_ID=$(echo "$p" | jq -r '.data.portId')
  case "$PORT_ID" in ''|*[!0-9]*) fail "port id 异常 response=$p" ;; esac
  TEMPLATE_ID=$(sql "INSERT INTO provision_templates(legal_entity_id,code,name,content,version,status) VALUES(1,'TPL-$PREFIX','OLTSIM $PREFIX',jsonb_build_object('bandwidth','100M','onuType','Internet','services',jsonb_build_object('internet',jsonb_build_object('svlan',1113,'cvlan',1,'uv',100,'scos',0,'ccos',0),'tr069',jsonb_build_object('cvlan',1000,'uv',1000,'scos',6,'ccos',6))),1,'ENABLED') RETURNING id")
  case "$TEMPLATE_ID" in ''|*[!0-9]*) fail "template 写入失败" ;; esac
  sql "UPDATE resources SET nms_oltid='$PREFIX-OLTID' WHERE id=$RESOURCE_ID" >/dev/null
  info "resource=$RESOURCE_ID port=$PORT_ID template=$TEMPLATE_ID"
}

# 用户下单 + 环节推进到预留(与 TL1 验收同款夹具口径 productId=101/addressId=290/channelId=102)。
create_order() {
  step "create order + check-resource + reserve"
  local out
  out=$(curl -sS -f -X POST "$BASE_URL/api/user/v1/orders" -H "X-API-Key: $USER_KEY" -H 'Content-Type: application/json' -d '{"productId":"101","addressId":"290","channelId":"102"}') || fail "创建订单失败"
  ORDER_NO=$(echo "$out" | jq -r '.data.orderNo')
  { [ -n "$ORDER_NO" ] && [ "$ORDER_NO" != "null" ]; } || fail "orderNo 异常 response=$out"
  ORDER_ID=$(sql "SELECT id FROM orders WHERE order_no='$ORDER_NO'")
  case "$ORDER_ID" in ''|*[!0-9]*) fail "order id 未落库 order_no=$ORDER_NO" ;; esac
  api POST "/orders/$ORDER_NO/check-resource" >/dev/null
  api POST "/orders/$ORDER_NO/reserve" >/dev/null
  info "order=$ORDER_NO id=$ORDER_ID"
}

# 端口改绑夹具资源 + 预建 PENDING 任务(隔离模板,自动化 CreateTask 同 task_no 幂等复用)。
bind_port_and_task() {
  step "bind port + pre-create PENDING preConfigOLT task"
  RESERVED_PORT_ID=$(sql "SELECT id FROM ports WHERE order_id=$ORDER_ID LIMIT 1")
  case "$RESERVED_PORT_ID" in ''|*[!0-9]*) fail "订单未预留端口" ;; esac
  if [ "$RESERVED_PORT_ID" = "$PORT_ID" ]; then
    PORT_WAS_OURS=1
  else
    PORT_ROW=$(sql "SELECT resource_id,COALESCE(pon_frame,-1),COALESCE(pon_slot,-1),COALESCE(pon_port,-1),COALESCE(onu_no,'~'),status FROM ports WHERE id=$RESERVED_PORT_ID")
  fi
  sql "UPDATE ports SET resource_id=$RESOURCE_ID,pon_frame=0,pon_slot=7,pon_port=5,onu_no=NULL WHERE id=$RESERVED_PORT_ID" >/dev/null
  local loid
  loid=$(sql "SELECT id FROM lo_accounts WHERE customer_id=(SELECT customer_id FROM orders WHERE id=$ORDER_ID) LIMIT 1")
  case "$loid" in
    ''|*[!0-9]*)
      TASK_MODE="fallback"
      ;;
    *)
      TASK_ID=$(sql "INSERT INTO provision_tasks(task_no,order_id,stage_event,lo_account_id,template_id,status) VALUES('PRV-O$ORDER_ID',$ORDER_ID,'preConfigOLT',$loid,$TEMPLATE_ID,'PENDING') RETURNING id")
      case "$TASK_ID" in ''|*[!0-9]*) fail "预建任务失败" ;; esac
      ;;
  esac
  info "reserved_port=$RESERVED_PORT_ID mode=$TASK_MODE task=$TASK_ID loid=$loid"
  save_ids
}

# 收费触发环节 7 自动化;等任务终态并断言 provision_logs=SUCCESS。
charge_and_wait() {
  step "charge -> automation -> wait task DONE"
  api POST "/orders/$ORDER_NO/charge" >/dev/null
  if [ "$TASK_MODE" = "fallback" ]; then
    TASK_ID=$(sql "SELECT id FROM provision_tasks WHERE order_id=$ORDER_ID AND stage_event='preConfigOLT' ORDER BY id DESC LIMIT 1")
    case "$TASK_ID" in ''|*[!0-9]*) fail "自动化未产生 preConfigOLT 任务" ;; esac
    sql "UPDATE provision_tasks SET template_id=$TEMPLATE_ID WHERE id=$TASK_ID AND status='PENDING'" >/dev/null
    save_ids
  fi
  local i s=""
  for i in $(seq 1 45); do
    s=$(sql "SELECT status FROM provision_tasks WHERE id=$TASK_ID")
    [ "$s" = "DONE" ] && break
    sleep 2
  done
  [ "$s" = "DONE" ] || fail "task=$TASK_ID 90s 未 DONE(最后 status=$s)"
  local lres
  lres=$(sql "SELECT result FROM provision_logs WHERE task_id=$TASK_ID ORDER BY id DESC LIMIT 1")
  [ "$lres" = "SUCCESS" ] || fail "provision_logs task=$TASK_ID result=$lres(期望 SUCCESS)"
  info "task=$TASK_ID DONE provision_logs=SUCCESS"
}

# oltsim 证据: records 出现本单 apply(result=OK,隔离模式下 template=夹具模板) + journal 留痕。
assert_sim_evidence() {
  step "assert oltsim received provision apply"
  local i item=""
  for i in $(seq 1 30); do
    item=$(sim_records | jq -c --arg t "PRV-O$ORDER_ID" '[.items[] | select(.taskNo == $t and .event == "preConfigOLT")][0] // empty')
    { [ -n "$item" ] && break; } || sleep 2
  done
  [ -n "$item" ] || fail "oltsim 60s 未收到 task=PRV-O$ORDER_ID 的 provision apply(driver=$DRV):旧链路未真实下发"
  echo "$item" | jq . > "$EVID_DIR/apply-record.json"
  [ "$(echo "$item" | jq -r '.result')" = "OK" ] || fail "oltsim apply result 非 OK: $item"
  if [ "$TASK_MODE" = "isolated" ]; then
    local got_tpl
    got_tpl=$(echo "$item" | jq -r '.template')
    [ "$got_tpl" = "$TEMPLATE_ID" ] || fail "oltsim apply template=$got_tpl 期望 $TEMPLATE_ID"
  fi
  ssh "$HOST" "journalctl -u boss-oltsim --no-pager -n 400 2>/dev/null | grep 'apply ok template=$TEMPLATE_ID task=PRV-O$ORDER_ID' | tail -3" > "$EVID_DIR/journal-apply.txt" || true
  sim_records > "$EVID_DIR/records-after.json"
  info "oltsim apply: $item"
  { [ -s "$EVID_DIR/journal-apply.txt" ] && info "journal: $(tail -1 "$EVID_DIR/journal-apply.txt")"; } || true
}

# 清理复核: 本单相关表逐项归零 + 队列空 + provisioner 仍 running。
verify_cleanup() {
  local bad="" pend
  [ "$(sql "SELECT count(*) FROM provision_tasks WHERE id=$TASK_ID")" = "0" ] || bad="$bad task"
  [ "$(sql "SELECT count(*) FROM provision_logs WHERE task_id=$TASK_ID")" = "0" ] || bad="$bad logs"
  [ "$(sql "SELECT count(*) FROM orders WHERE id=$ORDER_ID")" = "0" ] || bad="$bad order"
  [ "$(sql "SELECT count(*) FROM order_stages WHERE order_id=$ORDER_ID")" = "0" ] || bad="$bad stages"
  [ "$(sql "SELECT count(*) FROM dispatch_tickets WHERE order_id=$ORDER_ID")" = "0" ] || bad="$bad tickets"
  [ "$(sql "SELECT count(*) FROM provision_templates WHERE id=$TEMPLATE_ID")" = "0" ] || bad="$bad template"
  [ "$(sql "SELECT count(*) FROM resources WHERE id=$RESOURCE_ID")" = "0" ] || bad="$bad resource"
  [ "$(sql "SELECT count(*) FROM ports WHERE resource_id=$RESOURCE_ID")" = "0" ] || bad="$bad ports"
  [ "$(sql "SELECT count(*) FROM ports WHERE order_id=$ORDER_ID")" = "0" ] || bad="$bad port-ref"
  pend=$(sql "SELECT count(*) FROM provision_tasks WHERE status IN ('PENDING','DOING')")
  [ "$pend" = "0" ] || bad="$bad queue=$pend"
  provisioner_running || bad="$bad provisioner-down"
  if [ -n "$bad" ]; then echo "FAIL: 清理复核残留:$bad" >&2; return 1; fi
  info "cleanup verified: 订单/任务/日志/资源/端口/模板 归零,队列空,provisioner running"
}

# 恢复现场: 非夹具端口按快照还原,夹具数据按实际 ID 精确删除。
cleanup() {
  local rc=$?
  set +e
  echo "==> cleanup prefix=$PREFIX evid=$EVID_DIR"
  if [ "$RESERVED_PORT_ID" != "0" ] && [ "$PORT_WAS_OURS" != "1" ] && [ -n "$PORT_ROW" ]; then
    local or_res or_f or_s or_p or_o or_st
    IFS='|' read -r or_res or_f or_s or_p or_o or_st <<EOF
$PORT_ROW
EOF
    local f="NULL" s="NULL" p="NULL" o="NULL"
    [ "$or_f" = "-1" ] || f="$or_f"
    [ "$or_s" = "-1" ] || s="$or_s"
    [ "$or_p" = "-1" ] || p="$or_p"
    [ "$or_o" = "~" ] || o="'$or_o'"
    sql "UPDATE ports SET resource_id=$or_res,pon_frame=$f,pon_slot=$s,pon_port=$p,onu_no=$o,order_id=NULL,status='$or_st' WHERE id=$RESERVED_PORT_ID" >/dev/null
    info "restored foreign port $RESERVED_PORT_ID"
  fi
  if [ "$TASK_ID" != "0" ]; then
    sql "DELETE FROM provision_logs WHERE task_id=$TASK_ID; DELETE FROM admin_notification_reads WHERE notification_id IN (SELECT id FROM admin_notifications WHERE ref_type='provision' AND ref_id='$TASK_ID'); DELETE FROM admin_notifications WHERE ref_type='provision' AND ref_id='$TASK_ID'; DELETE FROM provision_tasks WHERE id=$TASK_ID;" >/dev/null
  fi
  if [ "$ORDER_ID" != "0" ]; then
    sql "DELETE FROM scan_logs WHERE order_id=$ORDER_ID; DELETE FROM dispatch_tickets WHERE order_id=$ORDER_ID; DELETE FROM activation_callbacks WHERE order_id=$ORDER_ID; DELETE FROM order_stages WHERE order_id=$ORDER_ID; DELETE FROM orders WHERE id=$ORDER_ID;" >/dev/null
  fi
  if [ "$RESOURCE_ID" != "0" ]; then
    sql "DELETE FROM pon_onu_alloc WHERE olt_resource_id=$RESOURCE_ID; DELETE FROM ports WHERE resource_id=$RESOURCE_ID; DELETE FROM resources WHERE id=$RESOURCE_ID;" >/dev/null
  fi
  [ "$TEMPLATE_ID" != "0" ] && sql "DELETE FROM provision_templates WHERE id=$TEMPLATE_ID;" >/dev/null
  verify_cleanup || rc=1
  exit $rc
}

trap cleanup EXIT
step "oltsim telnet 旧链路回归 target=$BASE_URL host=$HOST prefix=$PREFIX"
precheck
sim_records > "$EVID_DIR/records-before.json"
create_fixture
create_order
bind_port_and_task
charge_and_wait
assert_sim_evidence
save_ids
echo "PASS: 旧 telnet 链路(oltsim)回归通过 prefix=$PREFIX task=$TASK_ID order=$ORDER_NO evid=$EVID_DIR"
