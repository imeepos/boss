#!/usr/bin/env bash
# shellcheck disable=SC2029  # 远端命令串在本侧组装注入参数属设计意图(sql/start_sim 等 helper)
# TL1 102 主链路 E2E 验收:验收对象是 102 已部署主链路(docker-compose.102.app.yml 的
# boss-provisioner,BOSS_PROVISION_DRIVER=tl1,TL1_* 指向 tl1sim 演练地址)。
# 红线: 不停/不改/不重建已部署 provisioner;不启动隔离 provisioner 容器(会抢队列);
#       只拉起/停止自己的 tl1sim 实例(pidfile+cmdline 双重校验,不碰他人进程);
#       夹具全 acc_tl1_ 前缀,收尾按前缀全量清扫(本轮+历史遗留,造数不过夜)。
# B 口径: 三笔隔离订单验证 UP 正向/Power-Off 负向/UP 恢复正向;断言 provision_logs
#       .driver=tl1 + result + 负向订单停留 INSTALLING + tl1sim record 留痕。
# 用法: scripts/verify-tl1-e2e.sh [BASE_URL]   # 默认 http://192.168.0.102:28080
# 环境变量: TL1_HOST ADMIN_API_KEY USER_API_KEY TL1_SIM_BIN TL1_SIM_USER/TL1_SIM_PASS
#           TL1_EVID_DIR(默认 /tmp/verify-tl1-e2e-<stamp>);sim 端口取 provisioner 的
#           BOSS_PROVISION_TL1_ADDR,演练地址与主链路一致,不另起炉灶。
# 2026-09-04 失败轮(acc_tl1_1788458558)三修复:R1 record 目录未在 102 预建,tl1sim
#   OpenFile 即死,端口被野实例顶替答话,任务 SUCCESS 但证据断裂 → 目录预创建+端口归属
#   自检+LOGIN 探测落盘自检;R2 清理 SQL 类型/外键/顺序错 → 前缀全量清扫;R3 EXIT trap
#   覆盖原始退出码造成验收假绿 → trap 传参入 cleanup 保真。
set -uo pipefail

BASE_URL="${1:-http://192.168.0.102:28080}"
API="${BASE_URL}/api/admin/v1"
HOST="${TL1_HOST:-imeepos@192.168.0.102}"
ADMIN_KEY="${ADMIN_API_KEY:-boss_a852c8434c1ae6370e454817dd6e497c}"
USER_KEY="${USER_API_KEY:-boss_100e5216b416080d4b95c7f3d648c6c1}"
SIM_BIN="${TL1_SIM_BIN:-/home/imeepos/bin/tl1sim}"
SIM_USER="${TL1_SIM_USER:-admin}"
SIM_PASS="${TL1_SIM_PASS:-admin}"
STAMP="$(date +%s)"
PREFIX="acc_tl1_${STAMP}"
EVID_DIR="${TL1_EVID_DIR:-/tmp/verify-tl1-e2e-${STAMP}}"
PIDFILE=/tmp/tl1sim-mainchain.pid
SIM_LOG=/tmp/tl1sim-mainchain.log

ORDERS=()
RESOURCE_ID=0; TEMPLATE_ID=0; SIM_PORT=

fail() { echo "FAIL: $*" >&2; exit 1; }
step() { echo "==> $*"; }
info() { echo "  .. $*"; }

sql() { local q="${1}"; ssh "${HOST}" "docker exec -i boss-infra-postgres-1 psql -q -v ON_ERROR_STOP=1 -U boss -d boss -Atc \"${q}\""; }
# 清理段 SQL 逐条独立执行: 失败输出可 grep 的 FAIL 行并置非零,不中断收尾(防一错串断)。
sqln() { local q="${1}"; sql "${q}" >/dev/null || { echo "FAIL: cleanup SQL: ${q:0:100}" >&2; return 1; } }
collect_ids() { sql "${1}" | tr -d '[:space:]'; }

api() {
  local m="${1}" p="${2}" b="${3:-}" out
  if [ -n "${b}" ]; then
    out=$(curl -sS -f -X "${m}" "${API}${p}" -H "X-API-Key: ${ADMIN_KEY}" -H 'Content-Type: application/json' -d "${b}") || fail "api ${m} ${p}"
  else
    out=$(curl -sS -f -X "${m}" "${API}${p}" -H "X-API-Key: ${ADMIN_KEY}") || fail "api ${m} ${p}"
  fi
  echo "${out}" | jq -e '.code == 0 or .code == 200' >/dev/null || fail "api ${m} ${p} response=${out}"
  echo "${out}"
}

provisioner_env() { ssh "${HOST}" "docker inspect boss-provisioner --format '{{range .Config.Env}}{{println .}}{{end}}'"; }
provisioner_running() {
  [ "$(ssh "${HOST}" "docker inspect -f '{{.State.Running}}' boss-provisioner 2>/dev/null")" = "true" ]
}

# 停自己拉起的 tl1sim(pidfile+cmdline 双重校验),绝不 pkill 他人实例;失败由调用方处置。
stop_sim() {
  ssh "${HOST}" "if test -s ${PIDFILE}; then pid=\$(cat ${PIDFILE}); if test -r /proc/\$pid/cmdline && tr '\0' ' ' < /proc/\$pid/cmdline | grep -F -- ${SIM_BIN} >/dev/null; then kill \$pid >/dev/null 2>&1 || true; fi; rm -f ${PIDFILE}; fi"
}
# R1 自检一: sim 进程存活 + 演练端口归属本实例。2026-09-04 事故即本实例启动即死后端口
# 被无 record 的野实例顶替答话,任务 SUCCESS 而证据断裂,此断言让该形态早失败。
assert_sim_owner() {
  local pid owner
  pid=$(ssh "${HOST}" "cat ${PIDFILE} 2>/dev/null" | tr -d '[:space:]')
  case "${pid}" in ''|*[!0-9]*) fail 'sim pidfile 无有效 pid(启动即死?查 /tmp/tl1sim-mainchain.log)' ;; esac
  ssh "${HOST}" "kill -0 ${pid} 2>/dev/null" || fail "sim pid=${pid} 已死:$(ssh "${HOST}" "tail -3 ${SIM_LOG} 2>/dev/null")"
  owner=$(ssh "${HOST}" "ss -tlnpt \"sport = :${SIM_PORT}\" 2>/dev/null" | grep -o "pid=${pid},")
  [ -n "${owner}" ] || fail "端口 ${SIM_PORT} 非本实例(pid=${pid})监听:被他人占用,先协调停占口进程再重跑"
  info "sim pid=${pid} port=${SIM_PORT} owner-ok"
}
# R1 自检二: LOGIN 探测——102 本机 /dev/tcp 发标准 LOGIN 帧,读响应验证通道并触发留痕。
sim_login_probe() {
  ssh "${HOST}" "bash -c 'exec 3<>/dev/tcp/127.0.0.1/${SIM_PORT} && printf \"LOGIN:::PCHK::UN=${SIM_USER},PWD=${SIM_PASS};\" >&3 && read -t 3 -n 40 _resp <&3; rc=\$?; exec 3<&-; exit \$rc'"
}
# 每场景重启自己的 tl1sim 并切换 ONU oper-state;主链路 provisioner 的 TL1 Manager 断线
# 自动重连,下一任务即打到新实例。record 目录必须先在 102 预建(R1 根因: tl1sim
# OpenFile 不建父目录,目录缺失启动即死)。
start_sim() {
  local state="${1}" record="${2}" rdir
  stop_sim
  ssh "${HOST}" "test -x ${SIM_BIN}" || fail "missing ${SIM_BIN}"
  rdir=$(dirname "${record}")
  ssh "${HOST}" "mkdir -p ${rdir} && rm -f ${record} ${SIM_LOG} ${PIDFILE}" || fail "sim 前置失败(record 目录预创建 ${rdir})"
  ssh "${HOST}" "nohup ${SIM_BIN} -addr 0.0.0.0:${SIM_PORT} -user ${SIM_USER} -pass ${SIM_PASS} -onu-oper-state ${state} -record ${record} >>${SIM_LOG} 2>&1 & echo \$! > ${PIDFILE}" || fail "start sim state=${state}"
  sleep 1
  assert_sim_owner
}
wait_task() {
  local id="${1}" want="${2}" label="${3}" s=""
  for _ in $(seq 1 45); do
    s=$(sql "SELECT status FROM provision_tasks WHERE id=${id}"|tr -d '[:space:]')
    [ "${s}" = "${want}" ] && return 0
    sleep 2
  done
  fail "${label} task=${id} expected=${want} got=${s}"
}
# precheck: 主链路部署假设逐项硬断言,任一不过拒绝注入任务(不碰现场)。
precheck() {
  step "precheck deployed provisioner / driver=tl1 / queue / tl1sim"
  provisioner_running || fail "boss-provisioner 未运行:102 主链路未部署或已宕"
  local env drv addr pend nms occ pre_rec others="" c ep dockerps logs
  env=$(provisioner_env)
  drv=$(echo "${env}" | grep '^BOSS_PROVISION_DRIVER=' | cut -d= -f2)
  [ "${drv}" = "tl1" ] || fail "boss-provisioner driver=${drv:-<unset>} 非 tl1:TL1 主链路未生效;按 docs/ops/tl1-driver-cutover.md §4 重建 provisioner 后重跑"
  addr=$(echo "${env}" | grep '^BOSS_PROVISION_TL1_ADDR=' | cut -d= -f2)
  case "${addr}" in *:*) SIM_PORT="${addr##*:}" ;; *) fail "BOSS_PROVISION_TL1_ADDR=${addr} 缺端口,无法对齐 tl1sim 监听" ;; esac
  case "${SIM_PORT}" in ''|*[!0-9]*) fail "TL1_ADDR 端口非法:${addr}" ;; esac
  nms=$(sql "SELECT count(*) FROM provision_nms WHERE legal_entity_id=1")
  [ "${nms}" = "0" ] || fail "legal_entity 1 已有 provision_nms 行(${nms}):端点取表行而非演练 env 兜底;按 docs/ops/tl1-driver-cutover.md §3.1 处置后重跑"
  dockerps=$(ssh "${HOST}" "docker ps --format '{{.Names}}'") || fail "docker ps 失败"
  for c in ${dockerps}; do
    [ "${c}" = "boss-provisioner" ] && continue
    ep=$(ssh "${HOST}" "docker inspect '${c}' --format '{{json .Config.Entrypoint}}' 2>/dev/null") || ep=""
    case "${ep}" in *boss-provisioner*) others="${others} ${c}" ;; esac
  done
  [ -z "${others}" ] || fail "检测到其他 provisioner 容器在跑(拒绝并跑防抢队列):${others}"
  pend=$(sql "SELECT count(*) FROM provision_tasks WHERE status IN ('PENDING','DOING')"|tr -d '[:space:]')
  [ "${pend}" = "0" ] || fail "下发队列非空 PENDING/DOING=${pend},拒绝注入任务"
  logs=$(ssh "${HOST}" "docker logs --tail 5000 boss-provisioner 2>&1" 2>/dev/null || true)
  echo "${logs}" | grep -q 'provisioner: telnet executor' && fail "容器日志仍见 telnet executor:驱动未切净,拒绝验收"
  echo "${logs}" | grep -q 'provisioner: tl1 executor' && info "startup log: tl1 executor confirmed" || info "startup log 无 tl1 executor 行(可能轮转),以 env driver=tl1 为准"
  ssh "${HOST}" "test -x ${SIM_BIN}" || fail "missing ${SIM_BIN}(tl1sim 演练二进制)"
  # R1 自检三: 演练端口必须空闲(LISTEN 即他人占用);起 sim 发 LOGIN 验证 record 落盘,
  # 设备侧证据链不通即早失败,拒绝注入任务。
  stop_sim
  occ=$(ssh "${HOST}" "ss -tlnpt \"sport = :${SIM_PORT}\" 2>/dev/null" | grep -o 'pid=[0-9]*' | head -1)
  [ -z "${occ}" ] || fail "演练端口 ${SIM_PORT} 已被进程 ${occ} 占用(顶替答话=证据断裂):协调停占口进程后重跑"
  pre_rec="${EVID_DIR}/tl1sim-precheck.jsonl"
  start_sim "UP" "${pre_rec}"
  sim_login_probe || fail "precheck LOGIN 探测无响应(port=${SIM_PORT}):sim 通道异常"
  ssh "${HOST}" "test -s ${pre_rec} && grep -q LOGIN ${pre_rec}" || fail "precheck record 未落盘:${pre_rec}(R1 证据链自检不过)"
  stop_sim
  info "provisioner=running driver=tl1 addr=${addr} sim_port=${SIM_PORT} queue=empty record-ok"
}
# 隔离夹具: acc_tl1_<stamp> 资源/3 端口/模板(admin API)+ nms_oltid 就位。
create_fixture() {
  step "create acc_ fixture prefix=${PREFIX}"
  local res p n
  res=$(api POST /provision/resources "\"{\"code\":\"OLT-${PREFIX}\",\"name\":\"TL1 ${PREFIX}\",\"type\":\"OLT\",\"addressId\":290,\"legalEntityId\":1}")
  RESOURCE_ID=$(echo "${res}" | jq -r '.data.id')
  case "${RESOURCE_ID}" in ''|*[!0-9]*) fail "resource id 异常 response=${res}" ;; esac
  for n in 5 6 7; do
    p=$(api POST /provision/ports "\"{\"portCode\":\"P-${PREFIX}-0${n}\",\"resourceId\":${RESOURCE_ID},\"addressId\":290,\"legalEntityId\":1}")
    echo "${p}" | jq -e '.data.portId' >/dev/null || fail "port 0${n} id 异常 response=${p}"
  done
  TEMPLATE_ID=$(sql "INSERT INTO provision_templates(legal_entity_id,code,name,content,version,status) VALUES(1,'TPL-${PREFIX}','TL1 ${PREFIX}',jsonb_build_object('bandwidth','100M','onuType','Internet','services',jsonb_build_object('internet',jsonb_build_object('svlan',1113,'cvlan',1,'uv',100,'scos',0,'ccos',0),'tr069',jsonb_build_object('cvlan',1000,'uv',1000,'scos',6,'ccos',6))),1,'ENABLED') RETURNING id"|tr -d '[:space:]')
  case "${TEMPLATE_ID}" in ''|*[!0-9]*) fail "template 写入失败" ;; esac
  sql "UPDATE resources SET nms_oltid='${PREFIX}-OLTID' WHERE id=${RESOURCE_ID};" >/dev/null
  info "resource=${RESOURCE_ID} template=${TEMPLATE_ID} ports=P-${PREFIX}-0{5,6,7}"
}
# 主链路 TL1 留痕断言: result 符合预期 + driver=tl1 + 指令非 telnet 形态。
assert_tl1_log() {
  local task="${1}" label="${2}" want="${3}" res drv bad
  res=$(sql "SELECT result FROM provision_logs WHERE task_id=${task} ORDER BY id DESC LIMIT 1")
  if [ "${want}" = "SUCCESS" ]; then
    [ "${res}" = "SUCCESS" ] || fail "${label} task=${task} provision_logs.result=${res}(期望 SUCCESS)"
  else
    case "${res}" in FAILED:*) ;; *) fail "${label} task=${task} provision_logs.result=${res:-<null>}(期望 FAILED: 前缀)" ;; esac
  fi
  drv=$(sql "SELECT driver FROM provision_logs WHERE task_id=${task} ORDER BY id DESC LIMIT 1")
  [ "${drv}" = "tl1" ] || fail "${label} task=${task} provision_logs.driver=${drv:-<null>}(期望 tl1)"
  bad=$(sql "SELECT count(*) FROM provision_logs WHERE task_id=${task} AND commands LIKE '%provision apply%'"|tr -d '[:space:]')
  [ "${bad}" = "0" ] || fail "${label} task=${task} commands 含 telnet 形态 provision apply(仍是旧链路)"
  info "${label} task=${task} log result=${res} driver=tl1"
}
# tl1sim 证据断言: record 留痕 LOGIN/ADD-ONU/ADD-PONVLAN(LOGIN 属会话层,不进 provision_logs)。
assert_sim_record() {
  local record="${1}" label="${2}" pat
  ssh "${HOST}" "test -s ${record}" || fail "${label} tl1sim record 空:${record}"
  for pat in LOGIN ADD-ONU ADD-PONVLAN; do
    ssh "${HOST}" "grep -q ${pat} ${record}" || fail "${label} tl1sim record 缺 ${pat}:${record}"
  done
  info "${label} record tail: $(ssh "${HOST}" "grep -E 'LOGIN|ADD-ONU|ADD-PONVLAN|LST-ONUSTATE' ${record} | tail -8")"
}
# 单场景: 拉起指定 oper-state 的 tl1sim → 夹具订单推进到 charge → 由已部署主链路
# provisioner 领取执行。预建 PENDING preConfigOLT(task_no=PRV-O<oid>)保证夹具模板必达设备。
run_order() {
  local state="${1}" label="${2}" port_no="${3}"
  local record="${EVID_DIR}/tl1sim-${label}.jsonl" order no oid port task loid status
  step "order ${label} state=${state} port=0${port_no}"
  start_sim "${state}" "${record}"
  order=$(curl -sS -f -X POST "${BASE_URL}/api/user/v1/orders" -H "X-API-Key: ${USER_KEY}" -H 'Content-Type: application/json' -d '{"productId":"101","addressId":"290","channelId":"102"}') || fail "create ${label} order"
  no=$(echo "${order}" | jq -r '.data.orderNo')
  oid=$(sql "SELECT id FROM orders WHERE order_no='${no}'"|tr -d '[:space:]')
  case "${oid}" in ''|*[!0-9]*) fail "${label} order id 未落库 order_no=${no}" ;; esac
  ORDERS+=("${oid}")
  api POST "/orders/${no}/check-resource" >/dev/null
  api POST "/orders/${no}/reserve" >/dev/null
  port=$(sql "SELECT id FROM ports WHERE order_id=${oid} LIMIT 1"|tr -d '[:space:]')
  case "${port}" in ''|*[!0-9]*) fail "${label} reserved port" ;; esac
  sql "UPDATE ports SET resource_id=${RESOURCE_ID},pon_frame=0,pon_slot=7,pon_port=${port_no},onu_no=NULL WHERE id=${port};" >/dev/null
  loid=$(sql "SELECT id FROM lo_accounts WHERE customer_id=(SELECT customer_id FROM orders WHERE id=${oid}) ORDER BY id LIMIT 1"|tr -d '[:space:]')
  case "${loid}" in ''|*[!0-9]*) fail "${label} 无 lo_account,无法预建隔离任务" ;; esac
  task=$(sql "INSERT INTO provision_tasks(task_no,order_id,stage_event,lo_account_id,template_id,status) VALUES('PRV-O${oid}',${oid},'preConfigOLT',${loid},${TEMPLATE_ID},'PENDING') RETURNING id"|tr -d '[:space:]')
  case "${task}" in ''|*[!0-9]*) fail "${label} 预建 preConfigOLT 任务失败" ;; esac
  api POST "/orders/${no}/charge" >/dev/null
  wait_task "${task}" DONE "${label} preConfig"
  assert_tl1_log "${task}" "${label} preConfig" SUCCESS
  echo "  PASS ${label} preConfig order=${no} task=${task}"
  task=$(sql "INSERT INTO provision_tasks(task_no,order_id,stage_event,lo_account_id,template_id,status) VALUES('T6-${label}',${oid},'activateUser',${loid},${TEMPLATE_ID},'PENDING') RETURNING id"|tr -d '[:space:]')
  case "${task}" in ''|*[!0-9]*) fail "${label} 建 activateUser 任务失败" ;; esac
  if [ "${state}" = "Power-Off" ]; then
    wait_task "${task}" FAILED "${label} activate"
    assert_tl1_log "${task}" "${label} activate" FAILED
    status=$(sql "SELECT status FROM orders WHERE id=${oid}"|tr -d '[:space:]')
    [ "${status}" = "INSTALLING" ] || fail "${label} order moved to ${status}(负向应停留 INSTALLING)"
    echo "  PASS ${label} activate FAILED(负向不推进) order=${no} task=${task} status=${status}"
  else
    wait_task "${task}" DONE "${label} activate"
    assert_tl1_log "${task}" "${label} activate" SUCCESS
    status=$(sql "SELECT status FROM orders WHERE id=${oid}"|tr -d '[:space:]')
    echo "  PASS ${label} activate DONE order=${no} task=${task} status=${status}"
  fi
  assert_sim_record "${record}" "${label}"
}
# 订单集合: 前缀资源/模板反查历史与本轮订单,并入本轮数组(端口/任务未落时兜底)。
order_id_set() {
  local ids id
  ids=$(collect_ids "SELECT string_agg(DISTINCT t.id::text,',') FROM (SELECT order_id AS id FROM ports WHERE resource_id IN (SELECT id FROM resources WHERE code LIKE 'OLT-acc_tl1_%') UNION SELECT order_id FROM provision_tasks WHERE template_id IN (SELECT id FROM provision_templates WHERE code LIKE 'TPL-acc_tl1_%')) t WHERE t.id IS NOT NULL") || return 1
  for id in ${ORDERS[@]+"${ORDERS[@]}"}; do [ -n "${id}" ] && ids="${ids:+${ids},}${id}"; done
  echo "${ids}"
}
# R2 订单链清扫: ref_id 按文本比较(实测 text=bigint 报错),orders 8 张外键子表先删。
sweep_orders() {
  local oids bad=0
  oids=$(order_id_set) || return 1
  [ -n "${oids}" ] || return 0
  sqln "DELETE FROM admin_notification_reads WHERE notification_id IN (SELECT id FROM admin_notifications WHERE ref_type='provision' AND ref_id IN (SELECT id::text FROM provision_tasks WHERE order_id IN (${oids})))" || bad=1
  sqln "DELETE FROM admin_notifications WHERE ref_type='provision' AND ref_id IN (SELECT id::text FROM provision_tasks WHERE order_id IN (${oids}))" || bad=1
  sqln "DELETE FROM provision_logs WHERE task_id IN (SELECT id FROM provision_tasks WHERE order_id IN (${oids}))" || bad=1
  sqln "DELETE FROM order_stages WHERE order_id IN (${oids}); DELETE FROM dispatch_tickets WHERE order_id IN (${oids}); DELETE FROM scan_logs WHERE order_id IN (${oids}); DELETE FROM activation_callbacks WHERE order_id IN (${oids})" || bad=1
  sqln "DELETE FROM complaints WHERE order_id IN (${oids}); DELETE FROM dismantles WHERE order_id IN (${oids}); DELETE FROM install_logs WHERE order_id IN (${oids}); DELETE FROM partner_commission_ledger WHERE order_id IN (${oids})" || bad=1
  sqln "DELETE FROM orders WHERE id IN (${oids})" || bad=1
  return "${bad}"
}
# R2 夹具清扫: 任务链(logs→tasks)先于模板(实测 provision_tasks 挡模板删除);
# 资源路含 resource_assignments 与 parent_id 子资源(外键图)。
sweep_fixtures() {
  local rids tids bad=0
  rids=$(collect_ids "SELECT string_agg(id::text,',') FROM resources WHERE code LIKE 'OLT-acc_tl1_%'") || return 1
  tids=$(collect_ids "SELECT string_agg(id::text,',') FROM provision_templates WHERE code LIKE 'TPL-acc_tl1_%'") || return 1
  sqln "DELETE FROM provision_logs WHERE task_id IN (SELECT id FROM provision_tasks WHERE template_id IN (SELECT id FROM provision_templates WHERE code LIKE 'TPL-acc_tl1_%'))" || bad=1
  sqln "DELETE FROM provision_tasks WHERE template_id IN (SELECT id FROM provision_templates WHERE code LIKE 'TPL-acc_tl1_%')" || bad=1
  if [ -n "${rids}" ]; then
    sqln "DELETE FROM pon_onu_alloc WHERE olt_resource_id IN (${rids})" || bad=1
    sqln "DELETE FROM resource_assignments WHERE resource_id IN (${rids})" || bad=1
    sqln "DELETE FROM ports WHERE resource_id IN (${rids})" || bad=1
    sqln "DELETE FROM resources WHERE parent_id IN (${rids})" || bad=1
    sqln "DELETE FROM resources WHERE id IN (${rids})" || bad=1
  fi
  if [ -n "${tids}" ]; then
    sqln "DELETE FROM offer_provision_bindings WHERE template_id IN (${tids})" || bad=1
    sqln "DELETE FROM provision_templates WHERE id IN (${tids})" || bad=1
  fi
  return "${bad}"
}
# A2 口径复核: acc_tl1_ 前缀资源/模板及其级联任务/日志/订单/通知全量归零(含历史遗留)。
assert_clean() {
  local bad=0 c
  c=$(collect_ids "SELECT count(*) FROM resources WHERE code LIKE 'OLT-acc_tl1_%' UNION ALL SELECT count(*) FROM provision_templates WHERE code LIKE 'TPL-acc_tl1_%' UNION ALL SELECT count(*) FROM provision_tasks WHERE template_id IN (SELECT id FROM provision_templates WHERE code LIKE 'TPL-acc_tl1_%') OR order_id IN (SELECT order_id FROM ports WHERE resource_id IN (SELECT id FROM resources WHERE code LIKE 'OLT-acc_tl1_%')) UNION ALL SELECT count(*) FROM provision_logs WHERE task_id IN (SELECT id FROM provision_tasks WHERE template_id IN (SELECT id FROM provision_templates WHERE code LIKE 'TPL-acc_tl1_%') OR order_id IN (SELECT order_id FROM ports WHERE resource_id IN (SELECT id FROM resources WHERE code LIKE 'OLT-acc_tl1_%'))) UNION ALL SELECT count(*) FROM orders WHERE id IN (SELECT order_id FROM ports WHERE resource_id IN (SELECT id FROM resources WHERE code LIKE 'OLT-acc_tl1_%') UNION SELECT order_id FROM provision_tasks WHERE template_id IN (SELECT id FROM provision_templates WHERE code LIKE 'TPL-acc_tl1_%')) UNION ALL SELECT count(*) FROM admin_notifications WHERE ref_type='provision' AND ref_id IN (SELECT id::text FROM provision_tasks WHERE template_id IN (SELECT id FROM provision_templates WHERE code LIKE 'TPL-acc_tl1_%'))")
  case "${c}" in "000000") ;; *) echo "FAIL: 清扫复核残留(resources/templates/tasks/logs/orders/notifications)=${c:-query-error}" >&2; bad=1 ;; esac
  return "${bad}"
}
# R3 退出码保真: EXIT trap 把原始退出码传参入 cleanup;清理段自身故障置 1,原始失败码
# 优先透传(2026-09-04 实测 trap 覆盖原始码造成验收假绿)。
cleanup() {
  local orig="${1:-1}" bad=0 pend
  set +e
  echo "==> cleanup prefix=${PREFIX} evid=${EVID_DIR}"
  stop_sim
  sweep_orders || bad=1
  sweep_fixtures || bad=1
  assert_clean || bad=1
  pend=$(collect_ids "SELECT count(*) FROM provision_tasks WHERE status IN ('PENDING','DOING')")
  [ "${pend}" = "0" ] || { echo "FAIL: 清理后队列非空:${pend}" >&2; bad=1; }
  provisioner_running || { echo 'FAIL: boss-provisioner not running after cleanup' >&2; bad=1; }
  if [ "${orig}" -ne 0 ]; then exit "${orig}"; fi
  exit "${bad}"
}
trap 'cleanup "$?"' EXIT

step "TL1 102 main-chain E2E target=${BASE_URL} host=${HOST} prefix=${PREFIX}"
precheck
create_fixture
run_order UP positive-a 5
run_order Power-Off negative-poweroff 6
run_order UP positive-c 7
echo "PASS: TL1 102 主链路 E2E 三订单(UP 正向/Power-Off 负向/UP 恢复)通过 driver=tl1"
