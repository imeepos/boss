# S2 统一异常补偿与跨域对账（最小可验收）

## 交付范围

- `compensation_tasks` 责任队列表：失败来源、业务对象、失败原因、优先级、责任人、SLA、状态、重试次数和审计日志。
- Admin `comp-tasks` 工作台接口：列表、详情、领取、转派、重试、回放（已关闭任务重开）、关闭。
- 既有 `compensation-tasks` 聚合视图保留，并补充 Webhook、券、积分失败聚合；订单、支付/停复机、话单、税票、配置下发、对账批次继续复用已有表和重试入口。
- 每日 `recon-daily` 快照从六项扩展为十项：订单（2）、四码、资源、账务、GIS、税票、积分、GIS 投影、Webhook 投递。

## 自动化证据

| 检查 | 命令 | 结果 |
|---|---|---|
| report/domain 单测 | `/opt/homebrew/bin/go test ./internal/domain/report/` | 待门禁执行 |
| admin handler 单测 | `/opt/homebrew/bin/go test ./internal/httpapi/admin/` | 待门禁执行 |
| 全量 Go 测试 | `/opt/homebrew/bin/go test ./...` | 待门禁执行 |
| 契约同步 | `make check-contract-sync` | 待门禁执行 |

## API 与状态语义

- 优先级：`LOW` / `NORMAL` / `HIGH` / `URGENT`。
- 状态：`OPEN` → `CLAIMED` → `DOING` → `DONE` → `CLOSED`；关闭可从非 CLOSED 状态进入；回放只允许 `CLOSED`，回放后重置为 `OPEN`。
- 所有领取、转派、重试、回放、关闭写入 `audit_log`，HTTP 响应沿既有 envelope 返回业务码，不能以 HTTP 200 伪称业务成功。
- 每日对账异常仍进入既有 URGENT 通知入口；本提交未声称 102 真实环境验收通过。

## 未完成/风险

- 当前工作包提供责任队列核心模型和可验证 API；失败源自动写入队列的全量生产编排、各域真实回放适配、前端工作台和 102 部署验收仍需后续交付。
- 本文不构成 102 验收报告，不伪造外部税局、Webhook、积分或 GIS 生产数据结果。
