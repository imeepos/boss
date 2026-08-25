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
[ -n "$sha" ] || die "no artifact found in volume $VOL"

echo "fetching artifacts/$sha from $SSH_HOST ..."
out_dir="./apk-$sha"
mkdir -p "$out_dir"
holder="$(ssh_remote "docker create -v $VOL:/out $ALPINE true")"
remote_tmp="$(ssh_remote "mktemp -d /tmp/fetch-apk-XXXXXX")"
for f in boss-worker.apk boss-user.apk SHA256SUMS; do
  if ssh_remote "docker cp $holder:/out/$sha/$f $remote_tmp/$f" 2>/dev/null; then
    scp -q "$SSH_HOST:$remote_tmp/$f" "$out_dir/" 2>/dev/null || echo "warn: scp $f failed" >&2
  else
    echo "warn: $f not present for $sha" >&2
  fi
done
ssh_remote "rm -rf $remote_tmp; docker rm $holder >/dev/null" >/dev/null
if [ -f "$out_dir/SHA256SUMS" ]; then
  (cd "$out_dir" && shasum -a 256 -c SHA256SUMS 2>&1 | tail -1) || true
fi
echo "done: $out_dir"
ls -la "$out_dir"