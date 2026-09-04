#!/bin/sh
# env-provision 机制机械自测:沙箱仓库内验证四个场景,全程不接触真实 .env。
# 用法: sh scripts/env-provision-selftest.sh ;任一断言失败退出码非 0。
set -eu

repo_root=$(git rev-parse --show-toplevel)
tmp=$(mktemp -d /tmp/env-provision-selftest.XXXXXX)
trap 'rm -rf "$tmp"' EXIT
pass_n=0
fail_n=0

ok() { pass_n=$((pass_n+1)); echo "PASS: $1"; }
ko() { fail_n=$((fail_n+1)); echo "FAIL: $1"; }
assert_files_eq() {
    if cmp -s "$2" "$3"; then ok "$1"; else ko "$1"; fi
}
assert_grep() {
    if grep -q "$2" "$3"; then ok "$1"; else ko "$1"; fi
}
assert_absent() {
    if [ ! -e "$2" ]; then ok "$1"; else ko "$1"; fi
}

make_sandbox() {
    d=$1
    git init -q "$d"
    git -C "$d" config user.email selftest@localhost
    git -C "$d" config user.name selftest
    mkdir -p "$d/scripts" "$d/.githooks"
    cp "$repo_root/scripts/env-provision.sh" "$d/scripts/"
    cp "$repo_root/scripts/env-hooks-init.sh" "$d/scripts/"
    cp "$repo_root/.githooks/post-checkout" "$d/.githooks/"
    chmod +x "$d/scripts/env-provision.sh" "$d/scripts/env-hooks-init.sh" "$d/.githooks/post-checkout"
    printf 'FAKE_EXAMPLE_KEY=from-example\n' > "$d/.env.example"
    printf '.env\n' > "$d/.gitignore"
    git -C "$d" add -A
    git -C "$d" commit -qm "selftest fixture"
    git -C "$d" config core.hooksPath .githooks
}

# 场景1:主 worktree 有 .env 时,新 worktree 自动复制且主 .env 原样
case_worktree_copy() {
    sb=$tmp/c1
    make_sandbox "$sb"
    printf 'FAKE_MAIN_KEY=from-main\n' > "$sb/.env"
    git -C "$sb" worktree add -q "$tmp/c1-wt" -b c1-branch >/dev/null 2>&1
    assert_files_eq "场景1 新 worktree 自动复制主 .env" "$tmp/c1-wt/.env" "$sb/.env"
    assert_grep "场景1 主 worktree .env 未被改动" 'FAKE_MAIN_KEY=from-main' "$sb/.env"
}

# 场景2:主 worktree 无 .env 时新 worktree 由 .env.example 兜底;分支往返检出不得覆盖既有 .env
case_fallback_and_no_overwrite() {
    sb=$tmp/c2
    make_sandbox "$sb"
    git -C "$sb" worktree add -q "$tmp/c2-wt" -b c2-branch >/dev/null 2>&1
    assert_files_eq "场景2 无主 .env 时由 .env.example 兜底" "$tmp/c2-wt/.env" "$sb/.env.example"
    printf 'SENTINEL_EXISTING=1\n' > "$tmp/c2-wt/.env"
    git -C "$tmp/c2-wt" checkout -q -b c2-other >/dev/null 2>&1
    git -C "$tmp/c2-wt" checkout -q c2-branch >/dev/null 2>&1
    assert_grep "场景2 分支往返检出后既有 .env 未被覆盖" 'SENTINEL_EXISTING=1' "$tmp/c2-wt/.env"
}

# 场景3:全新 clone 不带钩子配置;运行一次性引导后立即由 example 兜底出 .env
case_clone_bootstrap() {
    sb=$tmp/c3
    make_sandbox "$sb"
    printf 'FAKE_MAIN_KEY=from-main\n' > "$sb/.env"
    git -C "$sb" clone -q "$sb" "$tmp/c3-clone" >/dev/null 2>&1
    assert_absent "场景3 克隆后未引导时无 .env(钩子未启用,配置不随克隆传播)" "$tmp/c3-clone/.env"
    ( cd / && sh "$tmp/c3-clone/scripts/env-hooks-init.sh" >/dev/null )
    assert_files_eq "场景3 引导后 .env 由 example 兜底" "$tmp/c3-clone/.env" "$tmp/c3-clone/.env.example"
}

# 场景4:provision 幂等,重复执行不覆盖已有 .env
case_idempotent() {
    sb=$tmp/c4
    make_sandbox "$sb"
    printf 'FAKE_MAIN_KEY=from-main\n' > "$sb/.env"
    git -C "$sb" worktree add -q "$tmp/c4-wt" -b c4-branch >/dev/null 2>&1
    sh "$tmp/c4-wt/scripts/env-provision.sh" >/dev/null
    sh "$tmp/c4-wt/scripts/env-provision.sh" >/dev/null
    assert_grep "场景4 重复执行不覆盖已有 .env" 'FAKE_MAIN_KEY=from-main' "$tmp/c4-wt/.env"
}

case_worktree_copy
case_fallback_and_no_overwrite
case_clone_bootstrap
case_idempotent

echo "---"
echo "env-provision selftest: pass=$pass_n fail=$fail_n"
[ "$fail_n" -eq 0 ]
