#!/usr/bin/env bash
# 部署成功标记落盘(2026-09-06):由 deploy-102 workflow 尾步「Record deploy success
# marker」调用。job 容器文件系统易逝,故经 docker-run 挂宿主目录,把「最近一次成功
# 部署时间戳 + git sha」写到 102 宿主 /home/imeepos/boss-deploy-state/last-success.env;
# scripts/ops/deploy-guard-alert.sh(每日 cron)比对该标记年龄,超阈值输出
# [deploy-guard] ALERT——部署静默停摆(如 2026-09-06 runner 镜像被 prune 后 6 次 push
# 零部署且不可见)的兜底告警。本步排在容器/healthz/指纹/孤儿门禁全过之后,标记即
# 「最近一次确认健康的部署」。
# 用法: scripts/ops/deploy-marker-write.sh   (依赖 GITHUB_SHA,在 deploy-runner 容器内执行)
# 环境: DEPLOY_GUARD_STATE_DIR / DEPLOY_GUARD_WRITER_IMAGE 可覆盖(测试用)。
set -euo pipefail

STATE_DIR=/home/imeepos/boss-deploy-state
WRITER=192.168.0.102:5000/runner:bookworm
# printenv 探测:set -u 下直接引用未设 env 会 unbound,先探测再引用
if printenv DEPLOY_GUARD_STATE_DIR >/dev/null; then STATE_DIR="$DEPLOY_GUARD_STATE_DIR"; fi
if printenv DEPLOY_GUARD_WRITER_IMAGE >/dev/null; then WRITER="$DEPLOY_GUARD_WRITER_IMAGE"; fi
if ! printenv GITHUB_SHA >/dev/null || [ -z "$GITHUB_SHA" ]; then
  echo "[deploy-guard] MARKER WRITE FAILED: GITHUB_SHA is empty" >&2
  exit 2
fi

trap 'echo "[deploy-guard] MARKER WRITE FAILED rc=$? (deploy healthy but marker not persisted)" >&2' ERR

docker run --rm -v "$STATE_DIR:/state" -e MARK_SHA="$GITHUB_SHA" "$WRITER" sh -c \
  'set -e; ts=$(date +%s); { echo "# boss 最近一次成功部署(deploy-102 workflow 尾步写入,勿手改;巡检方=deploy-guard-alert.sh)"; echo "ts=$ts"; echo "iso=$(date -u +%Y-%m-%dT%H:%M:%SZ)"; echo "sha=$MARK_SHA"; } > /state/last-success.env; cat /state/last-success.env'

echo "[deploy-guard] marker written to $STATE_DIR/last-success.env sha=$GITHUB_SHA"
