#!/usr/bin/env bash
set -u
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
LOCK="${ACCEPTANCE_LOCK_PATH:-${1:-/tmp/boss-102-acceptance-lock-selftest.lock}}"
export ACCEPTANCE_LOCK_PATH="$LOCK"
source "$ROOT/scripts/ops/acceptance-lock.sh"
fail() { echo "FAIL: $*" >&2; rm -f "$LOCK" "$LOCK."*; exit 1; }
run_mutex() {
  rm -f "$LOCK" "$LOCK."*
  ACCEPTANCE_LOCK_PATH="$LOCK" SELFTEST_SLEEP=2 "$0" hold >"$LOCK.holder.log" 2>&1 &
  local holder=$! rc
  sleep 0.2
  set +e
  ACCEPTANCE_LOCK_PATH="$LOCK" "$0" hold >"$LOCK.contender.log" 2>&1
  rc=$?
  set -e
  wait "$holder" || fail "holder failed"
  grep -q "LOCK_BUSY holder=" "$LOCK.contender.log" || fail "missing LOCK_BUSY holder signal"
  [ "$rc" -ne 0 ] || fail "contender unexpectedly succeeded"
  echo "SELFTEST_MUTEX_PASS contender_rc=$rc"
  ACCEPTANCE_LOCK_PATH="$LOCK" "$0" hold >/dev/null 2>&1 || fail "post-release acquisition failed"
  rm -f "$LOCK" "$LOCK."*
}
run_ttl() {
  rm -f "$LOCK" "$LOCK."*
  printf "stale-selftest pid=999999 ts=2020-01-01T00:00:00Z\n" >"$LOCK"
  touch -t "$(date -v-16M +%Y%m%d%H%M.%S 2>/dev/null || date -d "16 minutes ago" +%Y%m%d%H%M.%S)" "$LOCK" || fail "cannot age lock"
  ACCEPTANCE_LOCK_PATH="$LOCK" SELFTEST_SLEEP=0 "$0" hold 2>"$LOCK.ttl.log" || fail "expired lock takeover failed"
  grep -q "LOCK_EXPIRED" "$LOCK.ttl.log" || fail "missing LOCK_EXPIRED signal"
  echo "SELFTEST_TTL_PASS age=16m"
  rm -f "$LOCK" "$LOCK."*
}
hold() {
  acquire_acceptance_lock || exit 1
  trap release_acceptance_lock EXIT
  sleep "${SELFTEST_SLEEP:-0.2}"
  echo "SELFTEST_PASS pid=$$"
}
case "${1:-all}" in
  hold) hold ;;
  mutex) run_mutex ;;
  ttl) run_ttl ;;
  all) run_mutex && run_ttl ;;
  *) fail "usage: $0 [all|mutex|ttl|hold]" ;;
esac
