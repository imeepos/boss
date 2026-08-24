#!/usr/bin/env bash
# worktree 建树包装:git worktree add + 自动补 Android local.properties(不入库,从主树拷)。
# 用法: scripts/worktree-add.sh <path> <new-branch> [base(缺省 main)]
# 背景:local.properties 被 .gitignore 忽略,新 worktree 缺它时 gradlew 直接报
# "SDK location not found"(2026-08-24 两轮 android 开发均手工补拷)。
set -euo pipefail

[ $# -ge 2 ] || { echo "usage: $0 <path> <new-branch> [base]" >&2; exit 1; }
WT_PATH=$1; BRANCH=$2; BASE=${3:-main}

MAIN_WT=$(git worktree list --porcelain | awk '/^worktree /{print $2}' | head -1)
git worktree add "$WT_PATH" -b "$BRANCH" "$BASE"

for d in mobile/worker/android mobile/user/android; do
  if [ -f "$MAIN_WT/$d/local.properties" ] && [ -d "$WT_PATH/$d" ]; then
    cp "$MAIN_WT/$d/local.properties" "$WT_PATH/$d/local.properties"
    echo "copied local.properties: $d"
  fi
done
echo "worktree ready: $WT_PATH ($BRANCH from $BASE)"
