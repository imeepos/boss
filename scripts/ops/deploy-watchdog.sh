#!/usr/bin/env bash
# deploy-watchdog: 102 部署通道看门狗(C1,2026-09-08,方案全文 docs/ops/deploy-watchdog.md)。
# 缺陷:act_runner 0.2.11 网络抖动时拉到 deploy 任务却无 job 容器、无错误行,任务卡死
# 需人工 retrigger(ISSUE.md CI/deploy-102,W4/W7 两次阻断在案)。
# 检测(只读):gitea 库 action_run_job JOIN action_run 中 deploy job waiting(5)/running(6)
# 超阈值 且 宿主无 GITEA-ACTIONS-TASK-<task_id> 容器 → 判「已领取无容器」卡死。
# 动作白名单(红线:绝不 docker restart/stop runner,防杀进行中构建,W7 在案):
#   ①Gitea API workflow_dispatch 对 main 重触发(concurrency cancel-in-progress 自动
#     取消卡死旧 run,Classify 与实际部署镜像 sha 比对,语义安全);
#   ②[deploy-watchdog] 可 grep 告警(行内含 task_id 与所采取动作)。
# token 文件缺失自动降级纯告警态(只响铃不动手)。
# 用法: scripts/ops/deploy-watchdog.sh [--dry-run|--selftest]
# cron: */5 * * * * cd ~/boss && ./scripts/ops/deploy-watchdog.sh >> /tmp/deploy-watchdog.log 2>&1
set -euo pipefail

# ---- 配置常量(批复要求:全部可经环境覆盖便于调参) ----
THRESHOLD_SEC=900          # job 无容器超此秒数判卡死(正常部署实测 86~172s)
COOLDOWN_SEC=1800          # 两次 retrigger 最小间隔
DAILY_CAP=4                # 24h 内 retrigger 上限,超限转告警
STATE_DIR=/home/imeepos/boss-deploy-state
TOKEN_FILE=/home/imeepos/gitea/runner/watchdog-token
GITEA_BASE=http://192.168.0.102:3001
REPO=sker/boss
REPO_ID=83
JOB_NAME=deploy
WORKFLOW=deploy-102.yml
REF=refs/heads/main
DB_CONTAINER=gitea-postgres
DB_USER=gitea
DB_NAME=gitea
for k in THRESHOLD_SEC COOLDOWN_SEC DAILY_CAP STATE_DIR TOKEN_FILE GITEA_BASE REPO \
         REPO_ID JOB_NAME WORKFLOW REF DB_CONTAINER DB_USER DB_NAME; do
  if printenv "DEPLOY_WATCHDOG_$k" >/dev/null; then eval "$k=\$DEPLOY_WATCHDOG_$k"; fi
done

DRY_RUN=0
SELFTEST=0
for a in "$@"; do
  case "$a" in
    --dry-run) DRY_RUN=1 ;;
    --selftest) SELFTEST=1 ;;
    *) echo "[deploy-watchdog] ALERT unknown arg=$a action=NONE"; exit 2 ;;
  esac
done

# ---- 自测夹具:离线注入,不碰真实 docker/psql/token/state ----
FIXTURE_DIR=""
if [ "$SELFTEST" = 1 ]; then
  FIXTURE_DIR=$(mktemp -d)
  STATE_DIR="$FIXTURE_DIR/state"
  TOKEN_FILE="$FIXTURE_DIR/token"
  mkdir -p "$STATE_DIR"
fi
[ -d "$STATE_DIR" ] || mkdir -p "$STATE_DIR"
HANDLED_DIR="$STATE_DIR/watchdog-handled"
RETRIG_DIR="$STATE_DIR/watchdog-retriggers"

log() { echo "[deploy-watchdog] $*"; }

# ---- 探测原语(selftest 模式读夹具文件,生产模式读真实环境) ----
query_jobs() {
  if [ -n "$FIXTURE_DIR" ]; then cat "$FIXTURE_DIR/jobs.tsv" 2>/dev/null || true; return 0; fi
  local sql="SELECT j.id, j.task_id, j.status, j.created, j.started, r.status FROM action_run_job j JOIN action_run r ON r.id=j.run_id WHERE j.repo_id=$REPO_ID AND j.name='$JOB_NAME' AND j.status IN (5,6) AND r.status IN (5,6) AND r.ref='$REF' ORDER BY j.id"
  # 分隔符用 |:字段全为整数,且避免 tab 作为 IFS 空白被 read 折叠(NULL task_id 会串位)
  docker exec -i "$DB_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -At -F '|' -c "$sql"
}

list_containers() {
  if [ -n "$FIXTURE_DIR" ]; then cat "$FIXTURE_DIR/containers.txt" 2>/dev/null || true; return 0; fi
  docker ps -a --format '{{.Names}}'
}

container_state() {
  # container_state <task_id>:输出 running|notrunning|absent
  local task_id="$1" name
  name=$(list_containers | grep -E "^GITEA-ACTIONS-TASK-${task_id}(\$|_)" || true)
  if [ -z "$name" ]; then echo absent; return 0; fi
  if [ -n "$FIXTURE_DIR" ]; then
    grep -qxF "$name" "$FIXTURE_DIR/running.txt" 2>/dev/null && echo running || echo notrunning
    return 0
  fi
  [ "$(docker inspect -f '{{.State.Status}}' "$name" 2>/dev/null || echo unknown)" = running ] \
    && echo running || echo notrunning
}

dispatch_retrigger() {
  if [ -n "$FIXTURE_DIR" ]; then echo "MOCK dispatch $REPO/$WORKFLOW ref=${REF#refs/heads/}"; return 0; fi
  local code
  code=$(curl -s -o /dev/null -w '%{http_code}' -m 20 -X POST \
    "$GITEA_BASE/api/v1/repos/$REPO/actions/workflows/$WORKFLOW/dispatches" \
    -H "Authorization: token $(cat "$TOKEN_FILE")" \
    -H 'Content-Type: application/json' -d "{\"ref\":\"${REF#refs/heads/}\"}")
  [ "$code" = 204 ] || [ "$code" = 200 ] || { echo "http=$code"; return 1; }
}

# ---- 幂等/限流状态 ----
is_handled() { [ -f "$HANDLED_DIR/$1" ]; }
mark_handled() { [ "$DRY_RUN" = 1 ] || { mkdir -p "$HANDLED_DIR"; echo "$2" > "$HANDLED_DIR/$1"; }; }
last_retrigger_ts() {
  local f="$RETRIG_DIR/last" m=0
  [ -f "$f" ] && m=$(cat "$f" 2>/dev/null || echo 0)
  echo "${m:-0}"
}
recent_retrigger_count() {
  local now="$1" n=0 f
  [ -d "$RETRIG_DIR" ] || { echo 0; return 0; }
  for f in "$RETRIG_DIR"/[0-9]*; do
    [ -f "$f" ] || continue
    [ $(( now - $(basename "$f") )) -lt 86400 ] && n=$((n+1))
  done
  echo "$n"
}
record_retrigger() {
  [ "$DRY_RUN" = 1 ] && return 0
  mkdir -p "$RETRIG_DIR"
  local now; now=$(date +%s)
  echo "$now" > "$RETRIG_DIR/last"
  : > "$RETRIG_DIR/$now"
  # 清理 48h 前限流记录
  for f in "$RETRIG_DIR"/[0-9]*; do
    [ -f "$f" ] || continue
    [ $(( now - $(basename "$f") )) -gt 172800 ] && rm -f "$f"
  done
  return 0
}

# ---- 判定与动作(单 job) ----
# judge <job_id> <task_id> <age> <now>;stuck 前提由调用方保证(无容器)
judge() {
  local job_id="$1" task_id="$2" age="$3" now="$4" act
  if [ "$task_id" -le 0 ]; then
    log "ALERT task_id=$task_id job_id=$job_id age=${age}s action=NONE reason=unclaimed runner_never_created_task"
    return 1
  fi
  if is_handled "$job_id"; then
    log "OK task_id=$task_id job_id=$job_id age=${age}s action=SUPPRESS reason=already_handled"
    return 0
  fi
  if [ $(( now - $(last_retrigger_ts) )) -lt "$COOLDOWN_SEC" ]; then
    log "OK task_id=$task_id job_id=$job_id age=${age}s action=SUPPRESS reason=cooldown"
    return 0
  fi
  if [ "$(recent_retrigger_count "$now")" -ge "$DAILY_CAP" ]; then
    log "ALERT task_id=$task_id job_id=$job_id age=${age}s action=NONE reason=daily_cap_reached cap=$DAILY_CAP"
    return 1
  fi
  if [ "$DRY_RUN" = 1 ]; then
    log "STUCK task_id=$task_id job_id=$job_id age=${age}s action=RETRIGGER mode=dry-run dispatch_skipped"
    return 0
  fi
  if [ ! -s "$TOKEN_FILE" ]; then
    log "ALERT task_id=$task_id job_id=$job_id age=${age}s action=NONE reason=no_token degraded_alert_only token_file=$TOKEN_FILE"
    return 1
  fi
  local out rc=0
  out=$(dispatch_retrigger) || rc=1
  if [ "$rc" = 0 ]; then
    mark_handled "$job_id" "$task_id"
    record_retrigger
    log "STUCK task_id=$task_id job_id=$job_id age=${age}s action=RETRIGGER dispatched=1 ${out:+detail=$out}"
    return 0
  fi
  log "ALERT task_id=$task_id job_id=$job_id age=${age}s action=NONE reason=dispatch_failed detail=$out"
  return 1
}

run() {
  local now; now=$(date +%s)
  local rows rc=0 checked=0 line jid tid jst created started rst age cstate
  rows=$(query_jobs) || {
    log "ALERT task_id=- job_id=- age=- action=NONE reason=db_probe_failed (gitea 库不可读,部署通道观测中断)"
    return 1
  }
  while IFS='|' read -r jid tid jst created started rst; do
    [ -n "$jid" ] || continue
    tid=${tid:-0}; created=${created:-0}; started=${started:-0}
    # 纵深防御:run 非 waiting/running 一律跳过(blocked 残留等,102 有真实样本)
    { [ "$rst" = 5 ] || [ "$rst" = 6 ]; } || continue
    [ "$jst" = 5 ] || [ "$jst" = 6 ] || continue
    checked=$((checked+1))
    [ "${started:-0}" -gt 0 ] && age=$(( now - started )) || age=$(( now - created ))
    [ "$age" -gt "$THRESHOLD_SEC" ] || continue
    cstate=$(container_state "$tid")
    case "$cstate" in
      running)  log "OK task_id=$tid job_id=$jid age=${age}s action=NONE reason=container_running job_in_progress" ;;
      notrunning) log "ALERT task_id=$tid job_id=$jid age=${age}s action=NONE reason=container_present_but_not_running" ; rc=1 ;;
      absent)   judge "$jid" "$tid" "$age" "$now" || rc=1 ;;
    esac
  done <<< "$rows"
  log "OK repo=$REPO checked=$checked stuck_gate=threshold=${THRESHOLD_SEC}s cooldown=${COOLDOWN_SEC}s daily_cap=$DAILY_CAP dry_run=$DRY_RUN" \
    || true
  if [ "$checked" = 0 ]; then log "OK repo=$REPO checked=0 no waiting/running deploy job"; fi
  return "$rc"
}

# ---- 自测:六用例注入(离线零副作用,批复验收规则 3) ----
selftest() {
  local fails=0 now; now=$(date +%s)
  fx() { printf '%s\n' "$@" > "$FIXTURE_DIR/jobs.tsv"; }
  # 用例1:卡死(已领取无容器) → 判定卡死,token 缺失 → ALERT 降级态
  fx "$(printf '%s|%s|%s|%s|%s|%s' 7196 4894 6 $((now-1200)) $((now-1200)) 6)"
  : > "$FIXTURE_DIR/containers.txt"
  : > "$FIXTURE_DIR/running.txt"
  if out=$(run 2>&1); then echo "[selftest] FAIL case1 expect ALERT exit1"; fails=$((fails+1));
  else echo "$out" | grep -q "task_id=4894 .*action=NONE reason=no_token" \
       && echo "[selftest] case1 stuck+no_token -> ALERT degraded: pass" \
       || { echo "[selftest] FAIL case1 $out"; fails=$((fails+1)); }; fi
  # 用例2:同 job 已 handled → 抑制幂等
  mkdir -p "$HANDLED_DIR"; echo 4894 > "$HANDLED_DIR/7196"
  if out=$(run 2>&1) && echo "$out" | grep -q "action=SUPPRESS reason=already_handled"; then
    echo "[selftest] case2 handled -> SUPPRESS: pass"; else echo "[selftest] FAIL case2 $out"; fails=$((fails+1)); fi
  # 用例3:容器在跑 → OK 零动作
  fx "$(printf '%s|%s|%s|%s|%s|%s' 7197 4895 6 $((now-1200)) $((now-1200)) 6)"
  echo "GITEA-ACTIONS-TASK-4895_WORKFLOW-deploy-102_JOB-deploy" > "$FIXTURE_DIR/containers.txt"
  echo "GITEA-ACTIONS-TASK-4895_WORKFLOW-deploy-102_JOB-deploy" > "$FIXTURE_DIR/running.txt"
  if out=$(run 2>&1) && echo "$out" | grep -q "task_id=4895 .*reason=container_running"; then
    echo "[selftest] case3 container present -> OK: pass"; else echo "[selftest] FAIL case3 $out"; fails=$((fails+1)); fi
  # 用例4:run 终态残留(1=success) → SQL 纵深过滤,零动作
  fx "$(printf '%s|%s|%s|%s|%s|%s' 7087 4850 7 1788837790 0 2)"
  if out=$(run 2>&1) && ! echo "$out" | grep -q "task_id=4850"; then
    echo "[selftest] case4 blocked-residue filtered -> zero action: pass"; else echo "[selftest] FAIL case4 $out"; fails=$((fails+1)); fi
  # 用例5:task_id=0 未领取 → ALERT(不 retrigger)
  fx "$(printf '%s|%s|%s|%s|%s|%s' 7198 0 5 $((now-4000)) 0 6)"
  if out=$(run 2>&1) || true; then echo "$out" | grep -q "task_id=0 .*reason=unclaimed" \
       && echo "[selftest] case5 unclaimed -> ALERT only: pass" \
       || { echo "[selftest] FAIL case5 $out"; fails=$((fails+1)); }; fi
  # 用例6:token 就位 → MOCK dispatch 成功 + handled + 限流落盘;冷却期内第二次抑制
  fx "$(printf '%s|%s|%s|%s|%s|%s' 7199 4896 6 $((now-1200)) $((now-1200)) 6)"
  echo "dummy-token" > "$TOKEN_FILE"
  if out=$(run 2>&1) && echo "$out" | grep -q "action=RETRIGGER dispatched=1"; then
    echo "[selftest] case6a dispatch path -> RETRIGGER: pass"; else echo "[selftest] FAIL case6a $out"; fails=$((fails+1)); fi
  if [ -f "$HANDLED_DIR/7199" ] && [ "$(recent_retrigger_count "$now")" = 1 ]; then
    echo "[selftest] case6b state persisted(handled+rate): pass"; else echo "[selftest] FAIL case6b state not persisted"; fails=$((fails+1)); fi
  fx "$(printf '%s|%s|%s|%s|%s|%s' 7200 4897 6 $((now-1200)) $((now-1200)) 6)"
  if out=$(run 2>&1) && echo "$out" | grep -q "action=SUPPRESS reason=cooldown"; then
    echo "[selftest] case6c cooldown -> SUPPRESS: pass"; else echo "[selftest] FAIL case6c $out"; fails=$((fails+1)); fi
  rm -rf "$FIXTURE_DIR"
  [ "$fails" = 0 ] && { echo "[selftest] PASS"; return 0; }
  echo "[selftest] FAIL $fails case(s)"; return 1
}

if [ "$SELFTEST" = 1 ]; then selftest; else run; fi
