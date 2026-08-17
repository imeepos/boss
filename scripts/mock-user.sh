#!/usr/bin/env bash
# 用户端 mock 守护循环:node api/mock/combined.js(用户+师傅合一,默认 8090,MOCK_PORT 覆盖)
set -u
LOG="${TMPDIR:-/tmp}/boss-mock-user.log"
while true; do
  node api/mock/combined.js >> "$LOG" 2>&1
  echo "[$(date +%FT%T)] mock 崩溃(exit $?),3 秒后重启" >> "$LOG"
  sleep 3
done
