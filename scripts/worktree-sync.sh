#!/usr/bin/env bash
# worktree 合并协议第 3 步机械化:对全部活跃 worktree 执行 merge main + 门禁。
# 用法: scripts/worktree-sync.sh [worktree目录...]   # 缺省=全部
# 冲突时停在当前 worktree,人工解决后重跑;不自动 commit 冲突结果。
set -uo pipefail

REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null) || { echo "not a git repo" >&2; exit 1; }
GATES_DIR="web/admin"

# 收集目标 worktree(排除主树,即 .git 文件所在仓库根);macOS bash 3.2 无 mapfile
MAIN_WT=$(git worktree list --porcelain | awk '/^worktree /{print $2}' | head -1)
TARGETS=()
if [ $# -gt 0 ]; then TARGETS=("$@"); else
  while IFS= read -r wt; do [ "$wt" != "$MAIN_WT" ] && TARGETS+=("$wt"); done \
    < <(git worktree list --porcelain | awk '/^worktree /{print $2}')
fi
[ ${#TARGETS[@]} -eq 0 ] && { echo "no active worktrees"; exit 0; }

fail=0
for wt in "${TARGETS[@]}"; do
  echo "=== $wt ==="
  branch=$(git -C "$wt" branch --show-current)
  # 未提交改动先挡住(协议要求写完立刻 commit)
  if [ -n "$(git -C "$wt" status --porcelain)" ]; then
    echo "SKIP: 未提交改动,先 commit"; fail=1; continue
  fi
  # 已完全合入 main 的 worktree 提示清理
  if [ "$(git -C "$REPO_ROOT" rev-list --count "main..$branch" 2>/dev/null)" = "0" ]; then
    echo "OK: 已全部进 main,可收尾(worktree remove + branch -d)"; continue
  fi
  if ! git -C "$wt" merge main --no-edit; then
    echo "CONFLICT: 在 $wt 内解决冲突后 commit,再重跑本脚本"; fail=1; continue
  fi
  if [ -d "$wt/$GATES_DIR" ]; then
    (cd "$wt/$GATES_DIR" && pnpm typecheck && pnpm test && pnpm build) \
      || { echo "GATE FAIL: $wt 门禁未过,勿合并"; fail=1; continue; }
  fi
  echo "SYNCED: $branch (门禁通过,可按收尾顺序合并)"
done
exit $fail
