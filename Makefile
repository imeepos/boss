GO ?= go
MODULE := github.com/ymm-001/boss

.PHONY: infra-up infra-down migrate-up migrate-down run test lint check proto docker-build

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
		$(GO) vet ./... && test -z "$$(gofmt -l .)"; \
	fi

## check:CI 等价门禁(本地一键复现 .github/workflows/ci.yml)
check: test lint
	$(GO) build ./...

## 从 proto 生成 gRPC 代码
proto:
	protoc --go_out=. --go_opt=module=$(MODULE) \
	       --go-grpc_out=. --go-grpc_opt=module=$(MODULE) \
	       api/proto/boss/*/v1/*.proto

## 构建全部部署物
docker-build:
	docker build -t boss-server -f deployments/docker/server.Dockerfile .
