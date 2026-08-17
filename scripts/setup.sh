#!/usr/bin/env bash
# 本地开发环境一键初始化
set -euo pipefail
docker compose -f deployments/docker-compose.infra.yml up -d
sleep 5
migrate -path migrations -database "postgres://boss:boss@localhost:5432/boss?sslmode=disable" up
echo "done. run: make run"
