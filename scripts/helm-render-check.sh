#!/usr/bin/env bash
# Helm 渲染门禁:依赖外接/自建两种模式必须全绿(真集群部署前置检查)。
# 用法: scripts/helm-render-check.sh
set -euo pipefail
cd "$(dirname "$0")/../deployments/helm/boss"

helm lint . >/dev/null
echo "lint(external deps): OK"

out=$(helm template boss .)
echo "$out" | grep -q "kind: Deployment" && [ "$(echo "$out" | grep -c 'name: boss-postgresql')" -eq 0 ] \
  || { echo "FAIL: external 模式不应渲染依赖"; exit 1; }
echo "template(external deps): OK, no subcharts rendered"

out=$(helm template boss . --set postgresql.enabled=true --set kafka.enabled=true --set redis.enabled=true)
echo "$out" | grep -q "name: boss-postgresql" || { echo "FAIL: postgresql 子 chart 未渲染"; exit 1; }
echo "$out" | grep -q "name: boss-kafka" || { echo "FAIL: kafka 子 chart 未渲染"; exit 1; }
echo "$out" | grep -q "name: boss-redis" || { echo "FAIL: redis 子 chart 未渲染"; exit 1; }
echo "$out" | grep -A1 "name: BOSS_DATABASE_DSN" | grep -q 'boss-postgresql' \
  || { echo "FAIL: DSN 未自动接线子 chart postgres"; exit 1; }
echo "$out" | grep -A1 "name: BOSS_KAFKA_BROKERS" | grep -q 'boss-kafka:9092' \
  || { echo "FAIL: Kafka brokers 未自动接线子 chart"; exit 1; }
echo "template(bundled deps): OK, DSN/brokers auto-wired"

# 显式 env 优先于自动接线。
out=$(helm template boss . --set postgresql.enabled=true \
  --set env.BOSS_DATABASE_DSN="host=external port=5432")
echo "$out" | grep -q 'host=external' || { echo "FAIL: 显式 DSN 未优先生效"; exit 1; }
echo "template(explicit env precedence): OK"
