#!/usr/bin/env bash
# 在 102 服务器上执行:从 daocloud 镜像源 pull 后 tag 到私有仓库 5000/boss/* 并 push。
# 用法: scp 本脚本到 102 后运行,或直接 ssh 执行。
set -uo pipefail

REPO="192.168.0.102:5000/boss"
MIRROR="docker.m.daocloud.io"

# 每行 "源镜像 目标短名:tag"
IMAGES=$(cat <<'EOF'
apache/apisix:3.10.0-debian                          apisix:3.10.0
bitnamilegacy/etcd:3.5.16                            etcd:3.5.16
prom/prometheus:v3.4.0                               prometheus:3.4.0
prom/alertmanager:v0.28.0                            alertmanager:0.28.0
grafana/grafana:11.5.2                               grafana:11.5.2
grafana/loki:3.3.2                                   loki:3.3.2
grafana/promtail:3.3.2                               promtail:3.3.2
jaegertracing/all-in-one:latest                      jaeger:latest
starrocks/allin1-ubuntu:latest                       starrocks:latest
apache/flink:1.20.1-scala_2.12-java11                flink:1.20.1-scala_2.12-java11
EOF
)

pull_push() {
  local src="$1" tgt="$2"
  echo "=== $src -> $REPO/$tgt ==="
  docker pull "$MIRROR/$src" || { echo "PULL FAIL: $src"; return 1; }
  docker tag "$MIRROR/$src" "$REPO/$tgt"
  docker push "$REPO/$tgt" || { echo "PUSH FAIL: $tgt"; return 1; }
  echo "DONE: $tgt"
}

while read -r src tgt; do
  [ -z "$src" ] && continue
  pull_push "$src" "$tgt" || true
done <<< "$IMAGES"

echo "=== ALL DONE ==="
