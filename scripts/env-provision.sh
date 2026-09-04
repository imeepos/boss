#!/bin/sh
# 检出后自动补齐 .env:优先复制主 worktree 的 .env,否则由 .env.example 生成兜底模板。
# 幂等:目标已存在 .env 时直接退出,绝不覆盖。来源都缺失时输出 ALERT 留痕后仍返回 0,
# 不阻断检出(post-checkout 的退出码不影响检出结果)。
set -eu

root=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0

if [ -f "$root/.env" ]; then
    exit 0
fi

main=$(git worktree list --porcelain 2>/dev/null | sed -n 's/^worktree //p' | head -n 1)

if [ -n "$main" ] && [ "$main" != "$root" ] && [ -f "$main/.env" ]; then
    if cp "$main/.env" "$root/.env"; then
        echo "[env-provision] .env 已从主 worktree 复制: $main/.env -> $root/.env"
    else
        echo "[env-provision] ALERT 复制主 worktree .env 失败 src=$main/.env root=$root" >&2
    fi
    exit 0
fi

if [ -f "$root/.env.example" ]; then
    if cp "$root/.env.example" "$root/.env"; then
        echo "[env-provision] .env 不存在,已由 .env.example 生成兜底模板: $root/.env(请填入真实值)"
    else
        echo "[env-provision] ALERT 复制 .env.example 失败 root=$root" >&2
    fi
    exit 0
fi

echo "[env-provision] ALERT 未找到 .env 来源(主 worktree 与 .env.example 均缺),未创建 root=$root" >&2
exit 0
