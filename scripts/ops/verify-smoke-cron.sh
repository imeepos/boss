#!/usr/bin/env bash
# verify-smoke-cron.sh -- 资产冒烟 cron + runner 凭据卷持久化 落地检查(102 宿主)。
# --check: 三项实测:1) crontab 含 asset-smoke 行 2) runner compose 含凭据卷挂载
#          3) 冒烟日志可写;任一失败退出非零。
#          双模式:102 本机直跑(默认 compose 路径存在)= local 读本机;
#          其它主机自动转 remote,经 ssh 取 102 实况检查,两种模式对同一
#          事实给出同一结论;ssh 不可达显式 HINT 并退出 1。
# --selftest: 离线自检:对临时 fixture 双路径(命中/缺失)自测三项检查函数,
#          不读不写宿主真实状态,可在任意主机执行。
# 无参数: 先 --selftest 后 --check,两段都过退出 0。
# 用法: scripts/ops/verify-smoke-cron.sh [--check|--selftest]
# 环境: SMOKE_RUNNER_COMPOSE 覆盖 compose 路径(local 模式为本地路径;remote 模式为 102 上路径;默认 /home/imeepos/gitea/compose.yml)
set -u

CRON_MARK="asset-smoke-e2e-daily"
LOG_FILE="/tmp/asset-smoke-e2e.log"
COMPOSE_FILE="$(printenv SMOKE_RUNNER_COMPOSE || true)"
if [ -z "$COMPOSE_FILE" ]; then COMPOSE_FILE="/home/imeepos/gitea/compose.yml"; fi
CRED_SRC_MARK="runner/docker-config.json"
CRED_DST_MARK="/root/.docker/config.json"
REMOTE_HOST="imeepos@192.168.0.102"

# --- 三项检查函数(输入显式传参,--check 与 --selftest 共用) ---

check_cron_line() { # $1=crontab 全文 -> 0=含 smoke 行
  echo "$1" | grep -q "$CRON_MARK"
}

check_compose_text() { # $1=compose 全文 -> 0=含凭据卷挂载源与目标两段
  echo "$1" | grep -q "$CRED_SRC_MARK" || return 1
  echo "$1" | grep -q "$CRED_DST_MARK"
}

check_compose_mount() { # $1=compose 路径 -> 0=文件存在且含凭据卷挂载源与目标两段
  [ -f "$1" ] || return 1
  check_compose_text "$(cat "$1")"
}

check_log_writable() { # $1=日志路径 -> 0=可写(缺失则试建)
  touch "$1" 2>/dev/null || return 1
  [ -w "$1" ]
}

run_check() {
  local st=0 cron_text compose_text log_rc
  if [ -f "$COMPOSE_FILE" ]; then
    echo "[verify-smoke] mode=local(102 本机)"
    cron_text="$(crontab -l 2>/dev/null || true)"
    compose_text="$(cat "$COMPOSE_FILE" 2>/dev/null || true)"
    check_log_writable "$LOG_FILE"
    log_rc=$?
  else
    echo "[verify-smoke] mode=remote via ssh $REMOTE_HOST"
    if ! ssh -o ConnectTimeout=10 -o BatchMode=yes "$REMOTE_HOST" true 2>/dev/null; then
      echo "[verify-smoke] HINT: ssh $REMOTE_HOST 不可达,远端检查无法进行"
      return 1
    fi
    cron_text="$(ssh -o ConnectTimeout=10 -o BatchMode=yes "$REMOTE_HOST" "crontab -l 2>/dev/null" || true)"
    compose_text="$(ssh -o ConnectTimeout=10 -o BatchMode=yes "$REMOTE_HOST" "cat $COMPOSE_FILE" 2>/dev/null || true)"
    ssh -o ConnectTimeout=10 -o BatchMode=yes "$REMOTE_HOST" "touch '$LOG_FILE' 2>/dev/null && [ -w '$LOG_FILE' ]"
    log_rc=$?
  fi
  if check_cron_line "$cron_text"; then
    echo "[verify-smoke] check1 OK: crontab 含 $CRON_MARK 行"
  else
    echo "[verify-smoke] check1 FAILED: crontab 缺 $CRON_MARK 行"
    st=1
  fi
  if [ -n "$compose_text" ] && check_compose_text "$compose_text"; then
    echo "[verify-smoke] check2 OK: $COMPOSE_FILE 含凭据卷挂载 $CRED_SRC_MARK -> $CRED_DST_MARK"
  else
    echo "[verify-smoke] check2 FAILED: $COMPOSE_FILE 缺凭据卷挂载(或不可读)"
    st=1
  fi
  if [ "$log_rc" -eq 0 ]; then
    echo "[verify-smoke] check3 OK: $LOG_FILE 可写"
  else
    echo "[verify-smoke] check3 FAILED: $LOG_FILE 不可写"
    st=1
  fi
  return $st
}

run_selftest() {
  local tmp st=0 total=0 passed=0
  tmp="$(mktemp -d)" || return 1
  assert() { # $1=期望 rc $2=实际 rc $3=用例名
    total=$((total+1))
    if [ "$1" = "$2" ]; then
      passed=$((passed+1))
    else
      echo "[verify-smoke] selftest FAILED: $3 expect=$1 got=$2"
      st=1
    fi
  }
  local cron_hit cron_miss
  cron_hit="30 7 * * * cd /home/imeepos/boss && ./scripts/ops/asset-smoke-cron.sh >> /tmp/asset-smoke-e2e.log 2>&1 # asset-smoke-e2e-daily"
  cron_miss="10 8 * * * cd /home/imeepos/boss && ./scripts/ops/db-patrol-gate.sh # boss-patrol-gate"
  assert 0 "$(check_cron_line "$cron_hit"; echo $?)" cron-hit
  assert 1 "$(check_cron_line "$cron_miss"; echo $?)" cron-miss
  local compose_hit="$tmp/compose.hit" compose_miss="$tmp/compose.miss"
  {
    echo "services:"
    echo "  runner:"
    echo "    volumes:"
    echo "      - ./runner/docker-config.json:/root/.docker/config.json:ro"
  } > "$compose_hit"
  echo "services:" > "$compose_miss"
  assert 0 "$(check_compose_mount "$compose_hit"; echo $?)" compose-hit
  assert 1 "$(check_compose_mount "$compose_miss"; echo $?)" compose-miss
  assert 1 "$(check_compose_mount "$tmp/no-such-compose.yml"; echo $?)" compose-missing-file
  assert 0 "$(check_log_writable "$tmp/smoke.log"; echo $?)" log-writable-new-file
  assert 1 "$(check_log_writable "$tmp/no-such-dir/sub/smoke.log"; echo $?)" log-unwritable-path
  rm -rf "$tmp"
  echo "[verify-smoke] selftest $passed/$total cases passed"
  return $st
}

MODE="all"
if [ $# -ge 1 ]; then
  case "$1" in
    --check) MODE="check" ;;
    --selftest) MODE="selftest" ;;
    *) echo "[verify-smoke] FAILED unknown arg: $1 (usage: $0 [--check|--selftest])" >&2
       exit 2 ;;
  esac
fi

st=0
if [ "$MODE" = "selftest" ] || [ "$MODE" = "all" ]; then
  run_selftest || st=1
fi
if [ "$MODE" = "check" ] || [ "$MODE" = "all" ]; then
  run_check || st=1
fi
exit $st
