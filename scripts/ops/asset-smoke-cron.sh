#!/usr/bin/env bash
# asset-smoke-cron.sh -- 资产域每日 e2e 冒烟跑批(102 宿主 crontab 用,串行四脚本)。
# 串行执行资产域四个 e2e 脚本;任一失败输出可 grep 的
# "[asset-smoke] ALERT <script> rc=<n>" 并以退出码 1 结束;全绿输出
# "[asset-smoke] OK smoke-suite all-green"。日志由 cron 行重定向追加到
# /tmp/asset-smoke-e2e.log;口径登记见 docs/ops/patrol-cron.md。
# 落地检查: scripts/ops/verify-smoke-cron.sh(--check 三项实测 + --selftest 离线自检)。
# 用法: scripts/ops/asset-smoke-cron.sh   环境: 无(依赖同树 ../e2e 四脚本)
set -u

E2E_DIR="$(cd "$(dirname "$0")/../e2e" && pwd)"
SCRIPTS="verify-asset-pagination-e2e.sh verify-asset-identity-epc-e2e.sh verify-tag-event-backfill.sh verify-asset-scrap-confirm-e2e.sh"

FAILED=0
for name in $SCRIPTS; do
  if [ ! -f "$E2E_DIR/$name" ]; then
    echo "[asset-smoke] ALERT $name missing at $E2E_DIR rc=127"
    FAILED=1
    continue
  fi
  echo "[asset-smoke] RUN $name start=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  if bash "$E2E_DIR/$name"; then
    echo "[asset-smoke] PASS $name rc=0"
  else
    rc=$?
    echo "[asset-smoke] ALERT $name rc=$rc"
    FAILED=1
  fi
done

if [ "$FAILED" -ne 0 ]; then
  echo "[asset-smoke] ALERT smoke-suite finished with failures (see RESULT lines above)"
  exit 1
fi
N=$(echo $SCRIPTS | wc -w | tr -d " ")
echo "[asset-smoke] OK smoke-suite all-green scripts=$N"
