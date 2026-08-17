GO ?= go
MODULE := github.com/ymm-001/boss

.PHONY: infra-up infra-down migrate-up migrate-down run test lint proto docker-build

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

lint:
	$(GO) vet ./...

## 从 proto 生成 gRPC 代码
proto:
	protoc --go_out=. --go_opt=module=$(MODULE) \
	       --go-grpc_out=. --go-grpc_opt=module=$(MODULE) \
	       api/proto/.../*.proto

## 构建全部部署物
docker-build:
	docker build -t boss-server -f deployments/docker/server.Dockerfile .
