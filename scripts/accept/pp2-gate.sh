#!/usr/bin/env bash
# PP2 波次机械验收门禁:web/admin 三步门禁 + 域级红线 grep
# 用法: pp2-gate.sh [--adoption] [--no-inline PATH]... PATH...
#   --adoption        额外跑 node scripts/check-ds-adoption.js(须全绿)
#   --no-inline PATH  该路径下 grep -rn "style={{" 必须零命中
#   PATH...           每个路径做裸 alert( 与 <select 红线 grep(零命中才过)
set -u
ADOPTION=0
declare -a NO_INLINE=()
declare -a PATHS=()
while [ $# -gt 0 ]; do
  case "$1" in
    --adoption) ADOPTION=1 ;;
    --no-inline) shift; NO_INLINE+=("$1") ;;
    *) PATHS+=("$1") ;;
  esac
  shift
done

pnpm --dir web/admin typecheck || exit 10
TZ=Asia/Shanghai pnpm --dir web/admin test || exit 11
pnpm --dir web/admin build || exit 12

for p in " web/admin/src/components" "${PATHS[@]:-}"; do
  [ -n "$p" ] || continue
  git grep -qn "<select" -- "$p" && { echo "FAIL native select: $p"; exit 20; }
  git grep -qnE "(^|[^.a-zA-Z])alert[(]" -- "$p" && { echo "FAIL bare alert: $p"; exit 21; }
done

for p in "${NO_INLINE[@]:-}"; do
  [ -n "$p" ] || continue
  grep -rn "style={{" "$p" >/dev/null 2>&1 && { echo "FAIL inline style: $p"; exit 22; }
done

if [ "$ADOPTION" -eq 1 ]; then
  node scripts/check-ds-adoption.js || exit 30
fi

echo "PP2 GATE PASS"
