# 端口孤儿口径裁定：INSTALLING 计入在途（环节 12 才消费端口）

日期：2026-08-28

## 决策

1. **recon 口径**：`orphanReservedPorts`（每日对账）的"在途订单"集合从 `RESERVED`
   扩为 `RESERVED + INSTALLING`——端口预占后直到环节 12（UpdateMap）才转 USED，
   期间订单处于 INSTALLING 时端口保持 RESERVED 是设计合法，不算孤儿。
2. **僵尸单另有检查**：新增 recon 项 `installingStuck`（INSTALLING 超 7 天未完成），
   把"装维卡死"的信号从误报的孤儿端口里拆出来，用正确的指标暴露。
3. **存量处置**：已确认历史 daily_recon URGENT（20260822/20260823 孤儿端口 48/58）
   主体为 13900001234 测试客户的 E2E 僵尸单，按正式 `POST /orders/{orderNo}/cancel`
   取消清理（取消即释放 RESERVED 端口），不 SQL 直改。

## why

- 状态机事实：`internal/domain/order/pg_workflow.go` UpdateMap（环节 12）执行
  `UPDATE ports SET status='USED'`，注释明确"终态 DONE 后端口转在用（terms.md §4:
  IDLE→RESERVED→USED）"；取消/超时只回收 RESERVED。故 INSTALLING 单端口合法保留。
- 对照组实证：102 环境 DONE 订单端口 6/6 全部为 USED。
- 2026-08-25 客户中心审计把"端口 314 挂 INSTALLING 单 381"判为孤儿（是(314)），
  与本次裁定冲突——按决策代谢规则不改原文，加 Amended 指向本 note。

## 放弃了什么

- 放弃"INSTALLING 即应消费端口"方案（改环节推进复杂度高、且预占语义保留到激活
  更符合装维现场"先占端口再施工"的节奏）。
- 放弃 SQL 直改释放存量孤儿端口（绕过业务取消路径，丢审计留痕）。

## 关联

- `internal/domain/report/pg_recon_counts.go`（in-flight 集合 + installingStuck）
- `docs/notes/adopted/2026-08-25-customer-centric-data-audit.md`（Amended）
- `docs/ops/m0-nightwatch-evidence.md` §3（M0 守夜发现）