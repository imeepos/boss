#!/usr/bin/env bash
# 取件 helper:从 102 宿主 docker 命名卷 boss-apk-artifacts/<sha>/ 拉取
# android-apk CI 产出的 worker/user APK + SHA256SUMS 到本地当前目录。
#
# 用法:
#   scripts/fetch-apk.sh [sha]            # 指定 commit sha(前缀可),无参取最新一份
#   scripts/fetch-apk.sh --list          # 列出卷里所有 sha 目录
#   scripts/fetch-apk.sh --latest         # 取最近一份
#
# 依赖:ssh 免密 imeepos@192.168.0.102(BatchMode 可),102 docker socket 可用。
# 与 .gitea/workflows/android-apk.yml 归档卷一致;后者完成时打印同一卷路径。
# 102 docker 通过 `ssh HOST <cmd>` 调用(本机无 docker CLI)。
# 注意:ssh 把 argv 按空格拼接给远端 shell,含管道/重定向的命令必须整体作为
# 一个字符串参数传递,否则远端 shell 会按字面管道解析(踩坑:sh -c "ls | head"
# 在 argv 形式下管道被远端 shell 接管,导致输出异常)。

set -euo pipefail

SSH_HOST="${SSH_HOST:-imeepos@192.168.0.102}"
ALPINE="docker.m.daocloud.io/library/alpine:3"
VOL="boss-apk-artifacts"

die() { echo "fetch-apk: $*" >&2; exit 1; }

# ssh_remote <shell-string>  整体作为远端命令发送。
ssh_remote() { ssh -o BatchMode=yes "$SSH_HOST" "$1"; }

list_remote() {
  ssh_remote "docker run --rm -v $VOL:/out $ALPINE ls -1 /out"
}

case "${1:-}" in
  --list|"-l") list_remote; exit 0 ;;
  --help|-h)
    sed -n '2,9p' "$0"
    exit 0
    ;;
esac

sha="${1:-}"
if [ -z "$sha" ] || [ "$sha" = "--latest" ]; then
  sha="$(ssh_remote "docker run --rm -v $VOL:/out $ALPINE sh -c 'ls -1t /out | head -n1'")"
fi
# 前缀展开(用法承诺"前缀可"但旧版从未实现,短前缀直拼路径必 No such file——
# 2026-08-25 fetch 全空排查半天,根因即此):前缀唯一命中补全,多命中/未命中报错。
full="$(ssh_remote "docker run --rm -v $VOL:/out $ALPINE sh -c 'ls -1 /out | grep ^$sha'")"
case "$(echo "$full" | grep -c .)" in
  1) sha="$full" ;;
  0) die "no artifact dir matching '$sha' in volume $VOL" ;;
  *) die "prefix '$sha' ambiguous: $(echo $full | tr '\n' ' ')" ;;
esac

# 取件改"docker run cat 流式 stdout 直落本地":单命令单连接,不经
# holder 容器/mktemp/scp 三段接力(实测 docker create 与 CI 并发时偶发失败,
# 旧版把一切 cp 错误误报成 "not present",排查被带偏——worker 包其实一直在卷里)。
# 重试 2 次骑过归档竞态:CI 侧 mkdir sha 目录与逐文件 docker cp 之间有窗口,
# 恰好撞上会三个文件全 "No such file"(f237fe9a 实例),几秒后重取即成功。
echo "fetching artifacts/$sha from $SSH_HOST ..."
out_dir="./apk-$sha"
mkdir -p "$out_dir"
got=0
fetch_one() {
  local f="$1" try
  for try in 1 2 3; do
    if ssh_remote "docker run --rm -v $VOL:/out $ALPINE cat /out/$sha/$f" > "$out_dir/$f" 2>/tmp/fetch-apk-err; then
      return 0
    fi
    [ "$try" -lt 3 ] && sleep 3
  done
  rm -f "$out_dir/$f"
  echo "warn: fetch $f failed for $sha: $(tr '\n' ' ' </tmp/fetch-apk-err)" >&2
  return 1
}
for f in boss-worker.apk boss-user.apk SHA256SUMS; do
  fetch_one "$f" && got=$((got+1))
done
[ "$got" -gt 0 ] || die "no artifact fetched for $sha (check volume: scripts/fetch-apk.sh --list)"
if [ -f "$out_dir/SHA256SUMS" ]; then
  (cd "$out_dir" && shasum -a 256 -c SHA256SUMS 2>&1 | tail -1) || true
fi
echo "done: $out_dir ($got files)"
ls -la "$out_dir"