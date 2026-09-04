#!/bin/sh
# 一次性引导:启用 .githooks 钩子目录(仓库级配置,本仓库全部 worktree 生效),
# 并为脚本所在 worktree 立即补齐 .env。全新 clone 后执行: sh scripts/env-hooks-init.sh
# 仓库按脚本自身位置解析,从任意 cwd 调用都安全。
set -eu

script_dir=$(cd "$(dirname "$0")" && pwd)
root=$(git -C "$script_dir/.." rev-parse --show-toplevel)
git -C "$root" config core.hooksPath .githooks
echo "[env-hooks-init] 已配置 core.hooksPath=.githooks(仓库级,对本仓库全部 worktree 生效)"
cd "$root"
exec "$root/scripts/env-provision.sh"
