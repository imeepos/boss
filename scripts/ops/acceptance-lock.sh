#!/usr/bin/env bash
# Shared mutex for destructive acceptance runs on 102.
ACCEPTANCE_LOCK_PATH="${ACCEPTANCE_LOCK_PATH:-/tmp/boss-102-acceptance.lock}"
ACCEPTANCE_LOCK_TTL_SECONDS=900
ACCEPTANCE_LOCK_HELD=0
ACCEPTANCE_LOCK_OWNER="$(basename "$0") pid=$$ ts=$(date -u +%Y-%m-%dT%H:%M:%SZ)"

acquire_acceptance_lock() {
  local now mtime age holder pid
  now=$(date +%s)
  if [ -e "$ACCEPTANCE_LOCK_PATH" ]; then
    holder=$(head -n 1 "$ACCEPTANCE_LOCK_PATH" 2>/dev/null || true)
    pid=$(printf "%s\n" "$holder" | sed -n "s/.* pid=\([0-9][0-9]*\) .*/\1/p")
    mtime=$(stat -c %Y "$ACCEPTANCE_LOCK_PATH" 2>/dev/null || stat -f %m "$ACCEPTANCE_LOCK_PATH")
    age=$((now-mtime))
    if { [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; } || [ "$age" -lt "$ACCEPTANCE_LOCK_TTL_SECONDS" ]; then
      echo "[acceptance-lock] LOCK_BUSY holder=${holder:-unknown} path=$ACCEPTANCE_LOCK_PATH age=${age}s (hint: 持锁排障请同时确认 tl1sim 在位;sim 缺位时验收断言按 FAIL 判死属预期环境暴露)" >&2
      return 1
    fi
    echo "[acceptance-lock] LOCK_EXPIRED replacing holder=${holder:-unknown} age=${age}s" >&2
    rm -f "$ACCEPTANCE_LOCK_PATH" || { echo "[acceptance-lock] RELEASE_FAILED path=$ACCEPTANCE_LOCK_PATH" >&2; return 1; }
  fi
  ( set -C; printf "%s\n" "$ACCEPTANCE_LOCK_OWNER" > "$ACCEPTANCE_LOCK_PATH" ) 2>/dev/null || {
    holder=$(head -n 1 "$ACCEPTANCE_LOCK_PATH" 2>/dev/null || true)
    echo "[acceptance-lock] LOCK_BUSY holder=${holder:-unknown} path=$ACCEPTANCE_LOCK_PATH (hint: 持锁排障请同时确认 tl1sim 在位;sim 缺位时验收断言按 FAIL 判死属预期环境暴露)" >&2
    return 1
  }
  ACCEPTANCE_LOCK_HELD=1
  echo "[acceptance-lock] LOCK_ACQUIRED owner=$ACCEPTANCE_LOCK_OWNER path=$ACCEPTANCE_LOCK_PATH" >&2
}

release_acceptance_lock() {
  if [ "$ACCEPTANCE_LOCK_HELD" = "1" ]; then
    rm -f "$ACCEPTANCE_LOCK_PATH" || echo "[acceptance-lock] RELEASE_FAILED path=$ACCEPTANCE_LOCK_PATH" >&2
    ACCEPTANCE_LOCK_HELD=0
    echo "[acceptance-lock] LOCK_RELEASED path=$ACCEPTANCE_LOCK_PATH" >&2
  fi
}
