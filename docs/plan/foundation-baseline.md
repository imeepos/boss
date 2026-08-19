# 第一阶段奠基基线

> 目的：把项目整理到可持续开发的起点。本文只记录工程骨架、门禁和边界，不定义业务实现和大架构。

## 当前工程骨架

- `cmd/`：可执行入口，入口只负责装配和进程生命周期。
- `internal/`：Go 内部实现，业务域和基础设施的代码边界。
- `api/`：OpenAPI、Protobuf、mock 和契约对账工具。
- `migrations/`：数据库迁移。
- `deployments/`：部署与环境编排。
- `scripts/`：可重复执行的生成、检查、开发和压测脚本。
- `docs/`：契约、计划、决策记录和评审资料。
- `web/admin/`：管理端 Web 工程。
- `web/desktop/`：管理端 Tauri 桌面壳工程，包装现有 Web 产物，不复制业务页面。
- `mobile/worker/`：师傅端 Android、iOS、H5 独立 app 目录。
- `mobile/user/`：用户端 Android、iOS、H5 独立 app 目录。

## 本地门禁

### Go

```bash
make test          # race 测试
make lint          # golangci-lint 或 go vet + gofmt
make contract-sync # 路由、OpenAPI、命名和行数对账
make check         # Go build + 上述 Go 门禁
```

### 管理端 Web

```bash
make web-admin-check
```

等价于 `web/admin` 下的 `pnpm typecheck`、`pnpm test`、`pnpm build`。

### CI

`.github/workflows/ci.yml` 必须覆盖：

- Go build、vet、race、golangci-lint；
- contract-sync；
- admin Web typecheck、test、build。

## 第一阶段边界

本阶段只处理：

1. 目录和入口职责清晰；
2. 本地命令与 CI 门禁可对应；
3. 契约、决策记录和目录规范可追溯；
4. 新增 app 有明确的独立目录，不与其他 app 共享页面实现；
5. 生成物、依赖目录和本地构建产物不进入版本控制。

本阶段暂不处理：

- worker/user 的业务功能实现；
- Android、iOS、H5 的同步协议实现；
- 大规模域拆分、微服务化和新的状态机设计；
- 各端 UI 重写。

## 完成判定

- `make check` 通过；
- `make web-admin-check` 通过；
- 目录规范、README 和实际目录保持一致；
- 不把其他并行开发改动混入本阶段提交。
