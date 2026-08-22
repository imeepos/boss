#!/usr/bin/env bash
# 102 生产库备份:pg_dump 自定义格式(压缩)+ 完整性校验 + 保留期轮转。
# 用法: scripts/ops/backup-102.sh          (保留最近 KEEP 份,默认 14)
# 产物: 102:~/backups/pg/boss_YYYYmmdd_HHMMSS.dump + 同名 .sha256
set -euo pipefail

SSH_HOST="${SSH_HOST:-imeepos@192.168.0.102}"
PG_CONTAINER="${PG_CONTAINER:-boss-infra-postgres-1}"
PG_USER="${PG_USER:-boss}"
PG_DB="${PG_DB:-boss}"
KEEP="${KEEP:-14}"
REMOTE_DIR="\$HOME/backups/pg"

ssh "$SSH_HOST" bash -s "$PG_CONTAINER" "$PG_USER" "$PG_DB" "$KEEP" "$REMOTE_DIR" <<'REMOTE'
set -euo pipefail
container="$1"; user="$2"; db="$3"; keep="$4"; dir="$5"
mkdir -p "$dir"
ts=$(date +%Y%m%d_%H%M%S)
out="$dir/boss_${ts}.dump"

# 自定义格式压缩备份
docker exec "$container" pg_dump -U "$user" -Fc -d "$db" > "$out"

# 校验:pg_restore 目录清单可读 + 非空
size=$(stat -c%s "$out")
[ "$size" -gt 1024 ] || { echo "备份过小($size B),疑似失败"; rm -f "$out"; exit 1; }
docker exec -i "$container" pg_restore -U "$user" --list < "$out" >/dev/null

sha256sum "$out" > "${out}.sha256"
echo "OK $out ($(numfmt --to=iec "$size" 2>/dev/null || echo ${size}B))"

# 轮转:只保留最近 keep 份 dump(+对应 sha256)
ls -1t "$dir"/boss_*.dump | tail -n +$((keep+1)) | while read -r f; do
  rm -f "$f" "${f}.sha256"
  echo "expired $(basename "$f")"
done
REMOTE
