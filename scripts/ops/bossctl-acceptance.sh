#!/usr/bin/env bash
set -u

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SERVER="${BOSS_SERVER:-http://192.168.0.102:28080}"

# 二进制解析优先级:BOSSCTL 环境变量 > PATH > 本机安装目录 > 源码临时编译。
# 禁止回落到技能 assets 里已过期的预编译产物(落后 102 路由会误判)。
BOSSCTL="${BOSSCTL:-}"
if [ -z "$BOSSCTL" ] && command -v bossctl >/dev/null 2>&1; then
  BOSSCTL="$(command -v bossctl)"
fi
if [ -z "$BOSSCTL" ] && [ -x "$HOME/bin/bossctl" ]; then
  BOSSCTL="$HOME/bin/bossctl"
fi
if [ -z "$BOSSCTL" ]; then
  GO="${GO:-$(command -v go 2>/dev/null || true)}"
  if [ -n "$GO" ]; then
    BOSSCTL="$(mktemp -t bossctl-acceptance.XXXXXX)"
    (cd "$ROOT" && "$GO" build -o "$BOSSCTL" ./cmd/bossctl) || { echo "FAIL: go build ./cmd/bossctl" >&2; exit 1; }
    echo "INFO: bossctl built from source at $BOSSCTL"
  fi
fi
ACCOUNTS="${BOSSCTL_TEST_ACCOUNTS:-$ROOT/.agents/skills/bossctl-cli/test-accounts.json}"

fail() { echo "FAIL: $*" >&2; exit 1; }
[ -x "$BOSSCTL" ] || fail "bossctl not executable: $BOSSCTL"
[ -r "$ACCOUNTS" ] || fail "test accounts unreadable: $ACCOUNTS"

KEY=$(python3 - "$ACCOUNTS" <<'PY'
import json
import sys
with open(sys.argv[1], encoding="utf-8") as f:
    print(json.load(f)["admin"]["apiKeys"][0]["key"])
PY
) || fail "cannot read admin API key"
[ -n "$KEY" ] || fail "admin API key is empty"

run_read() {
  local label="$1"
  shift
  local output status
  output=$("$BOSSCTL" --server "$SERVER" "$@" 2>&1)
  status=$?
  if [ "$status" -ne 0 ] || printf '%s' "$output" | grep -qiE 'invalid token|invalid_token'; then
    echo "FAIL: $label status=$status output=$(printf '%s' "$output" | head -n 1)" >&2
    exit 1
  fi
  printf '%s\n' "$output"
}

ME=$(run_read "auth/me" --api-key "$KEY" call GET /auth/me) || exit 1
echo "PASS: auth/me code=0"
printf '%s\n' "$ME" | head -n 1

ROUTES=$(run_read "admin routes" --api-key "$KEY" routes admin) || exit 1
COUNT=$(printf '%s\n' "$ROUTES" | awk '/^  (GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD) / {n++} END {print n+0}')
case "$COUNT" in
  508|509) echo "PASS: admin routes count=$COUNT (102-compatible)" ;;
  *) echo "FAIL: admin routes count=$COUNT (expected 508 or current 102-compatible count)" >&2; exit 1 ;;
esac

SAVED=$(run_read "saved api-key" --as admin call GET /auth/me) || exit 1
echo "PASS: saved api-key auth/me code=0 (no invalid token)"
printf '%s\n' "$SAVED" | head -n 1
