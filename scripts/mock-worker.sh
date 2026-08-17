#!/usr/bin/env bash
# 师傅端 mock 守护循环:node api/mock/worker/server.js(默认 8091,MOCK_WORKER_PORT 覆盖)
set -u
LOG="${TMPDIR:-/tmp}/boss-mock-worker.log"
while true; do
  node api/mock/worker/server.js >> "$LOG" 2>&1
  echo "[$(date +%FT%T)] worker mock 崩溃(exit $?),3 秒后重启" >> "$LOG"
  sleep 3
done
