#!/usr/bin/env bash
# 部署静默停摆巡检(2026-09-06,ISSUE.md CI/deploy-102 根治项之二):比对「最近一次成功
# 部署标记」(deploy-102 workflow 尾步 deploy-marker-write.sh 写入 102 宿主)的年龄,
# 超阈值(默认 24h)即输出 "[deploy-guard] ALERT" 可 grep 日志并 exit 1;配套 102 每日
# cron 口径见 docs/ops/patrol-cron.md。判定三态:标记缺失/不可读/超龄任一 → ALERT;
# 新鲜 → OK。本巡检兜底「部署通道整体静默停摆」(push 持续发生但零成功部署);阈值取
# 舍依据:主线 push 常规节奏每天多次,24h 无健康部署即视为异常(Lead 裁定建议值)。
# 用法: scripts/ops/deploy-guard-alert.sh
# 环境: DEPLOY_GUARD_STATE_FILE / DEPLOY_GUARD_THRESHOLD_HOURS 可覆盖。
# 自测: --selftest 用临时标记验证 新鲜/超龄/缺失 三态判定与退出码,不读不写真实标记(A3)。
set -euo pipefail

STATE_FILE=/home/imeepos/boss-deploy-state/last-success.env
THRESHOLD_HOURS=24
# printenv 探测:set -u 下直接引用未设 env 会 unbound,先探测再引用
if printenv DEPLOY_GUARD_STATE_FILE >/dev/null; then STATE_FILE="$DEPLOY_GUARD_STATE_FILE"; fi
if printenv DEPLOY_GUARD_THRESHOLD_HOURS >/dev/null; then THRESHOLD_HOURS="$DEPLOY_GUARD_THRESHOLD_HOURS"; fi
SELFTEST=0
if [ $# -gt 0 ] && [ "$1" = "--selftest" ]; then SELFTEST=1; fi

judge() {
  # judge <state_file> <threshold_hours>:0=健康 1=告警(独立函数,供 --selftest 注入三态)
  f="$1"
  thr="$2"
  if [ ! -f "$f" ]; then
    echo "[deploy-guard] ALERT marker missing: $f (no healthy deploy ever recorded, or state dir wiped)"
    return 1
  fi
  ts=$(sed -n "/^ts=/s/^ts=//p" "$f")
  sha=$(sed -n "/^sha=/s/^sha=//p" "$f")
  case "$ts" in
    "") echo "[deploy-guard] ALERT marker unreadable (no ts line) in $f"; return 1 ;;
    *[!0-9]*) echo "[deploy-guard] ALERT marker unreadable (ts not numeric) in $f"; return 1 ;;
  esac
  now=$(date +%s)
  age_h=$(( (now - ts) / 3600 ))
  if [ "$age_h" -gt "$thr" ]; then
    echo "[deploy-guard] ALERT last healthy deploy sha=$sha age=$age_h h > $thr h threshold - deploy pipeline silently stalled? check gitea actions runs + gitea-runner logs"
    return 1
  fi
  echo "[deploy-guard] OK last healthy deploy sha=$sha age=$age_h h <= $thr h threshold"
  return 0
}

if [ "$SELFTEST" = 1 ]; then
  tmp=$(mktemp -d)
  trap 'rm -rf "$tmp"' EXIT
  now=$(date +%s)
  # 态1:新鲜标记(1h 前) → OK 退出 0
  { echo "ts=$(( now - 3600 ))"; echo "sha=fresh00"; } > "$tmp/fresh.env"
  if judge "$tmp/fresh.env" 24; then
    echo "[selftest] fresh -> OK: pass"
  else
    echo "[selftest] FAIL: fresh marker judged ALERT"; exit 1
  fi
  # 态2:超龄标记(25h 前) → ALERT 退出 1
  { echo "ts=$(( now - 25 * 3600 ))"; echo "sha=stale00"; } > "$tmp/stale.env"
  if judge "$tmp/stale.env" 24; then
    echo "[selftest] FAIL: stale marker judged OK"; exit 1
  else
    echo "[selftest] stale -> ALERT: pass"
  fi
  # 态3:标记缺失 → ALERT 退出 1
  if judge "$tmp/absent.env" 24; then
    echo "[selftest] FAIL: missing marker judged OK"; exit 1
  else
    echo "[selftest] missing -> ALERT: pass"
  fi
  echo "[selftest] PASS"
  exit 0
fi

judge "$STATE_FILE" "$THRESHOLD_HOURS"
