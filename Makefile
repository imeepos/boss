GO ?= $(or $(shell command -v go 2>/dev/null),/opt/homebrew/bin/go)
GOFMT ?= $(or $(shell command -v gofmt 2>/dev/null),/opt/homebrew/bin/gofmt)
MODULE := github.com/ymm-001/boss

.PHONY: infra-up infra-down migrate-up migrate-down run test lint check contract-sync web-admin-check proto docker-build load bossctl bossctl-routes

## 构建 bossctl CLI 工具(操作全部 API 接口,支持免登录 API key 认证)
bossctl:
	$(GO) build -ldflags="-s -w" -o bossctl ./cmd/bossctl

## 由 api/openapi 重新生成 bossctl 三端路由目录(routes_*.go,契约变更后执行)
bossctl-routes:
	node scripts/gen-bossctl-routes.mjs

## W11 压测:种子压测账号 → 起服务 → k6 → 摘服务(真实 PG 需 BOSS_DATABASE_DSN;端口可经 BOSS_HTTP_PORT 覆盖)
load:
	BOSS_LOAD_PASSWORD="Load-test-123" $(GO) run scripts/load/seed_account.go "$${BOSS_DATABASE_DSN:?BOSS_DATABASE_DSN 未设置}"
	$(GO) build -o /tmp/boss-server ./cmd/server
	PORT="$${BOSS_HTTP_PORT:-18080}"; \
	BOSS_DATABASE_DSN="$${BOSS_DATABASE_DSN}" BOSS_HTTP_ADDR=":$$PORT" BOSS_GRPC_ADDR=":$$((PORT+1000))" /tmp/boss-server > /tmp/boss-server.log 2>&1 & \
	  SERVER_PID=$$!; \
	  for i in $$(seq 1 20); do curl -sf "http://127.0.0.1:$$PORT/healthz" >/dev/null && break; sleep 1; done; \
	  k6 run -e BASE="http://127.0.0.1:$$PORT" scripts/load/api-load.js; \
	  RC=$$?; kill $$SERVER_PID; exit $$RC

## 基础设施:PG/Redis/Kafka/Nacos/MinIO/Temporal/VM
infra-up:
	docker compose -f deployments/docker-compose.infra.yml up -d

infra-down:
	docker compose -f deployments/docker-compose.infra.yml down

## 数据库迁移
migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

## 本地运行业务单体
run:
	$(GO) run ./cmd/server

## 测试 / 静态检查
test:
	$(GO) test ./... -race -count=1

## lint:优先用 golangci-lint(如已安装),否则回退 go vet + gofmt check
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		$(GO) vet ./... && test -z "$$($(GOFMT) -l .)"; \
	fi

## check:CI 等价门禁(本地一键复现 .github/workflows/ci.yml)
check: test lint contract-sync
	$(GO) build ./...

## 契约同步门禁:路由<->OpenAPI 对账 + json tag 命名 + 文件行数红线
contract-sync:
	$(GO) run ./scripts/check-contract-sync -root .

## 管理端 Web 门禁:类型检查、单测、生产构建
web-admin-check:
	pnpm --dir web/admin typecheck
	pnpm --dir web/admin test
	pnpm --dir web/admin build
	node scripts/check-ds-adoption.js

## 从 proto 生成 gRPC 代码
proto:
	protoc --go_out=. --go_opt=module=$(MODULE) \
	       --go-grpc_out=. --go-grpc_opt=module=$(MODULE) \
	       api/proto/boss/*/v1/*.proto

## 构建全部部署物
docker-build:
	docker build -t boss-server -f deployments/docker/server.Dockerfile .
