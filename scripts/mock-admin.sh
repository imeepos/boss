#!/usr/bin/env bash
# 管理后台 mock 守护循环:node api/mock/admin/server.js(默认 8092,MOCK_ADMIN_PORT 覆盖)
set -u
LOG="${TMPDIR:-/tmp}/boss-mock-admin.log"
while true; do
  MOCK_ADMIN_PORT="${PORT:-8092}" node api/mock/admin/server.js >> "$LOG" 2>&1
  echo "[$(date +%FT%T)] admin mock 崩溃(exit $?),3 秒后重启" >> "$LOG"
  sleep 3
done
