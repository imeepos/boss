# 0001-2026-08-21-audit-partition-boot-loop

## 现象

102 环境 `boss-server` 容器崩溃循环（重启 13 次），28080 端口无响应，
boss-admin 全部接口不可用。日志反复输出：

```
wiring: audit partitions: audit: ensure partition audit_logs_2026_08:
ERROR: updated partition constraint for default partition "audit_logs_default"
would be violated by some row (SQLSTATE 23514)
```

## 根因

E14 修复（`EnsurePartitions`，随 wiring 启动预建分区）上线前，旧镜像无任何
分区预建逻辑，000001 迁移只建了 `audit_logs_2025_08` + default，于是
2026-08-17～08-21 共 691 条审计写入全部落入 `audit_logs_default`。

新镜像首次启动时 `EnsurePartitions` 执行
`CREATE TABLE audit_logs_2026_08 PARTITION OF ...`，而 PostgreSQL 规定：
default 分区中存在属于新分区范围的行时拒绝创建（23514）。wiring 返回错误
→ 进程退出 → docker restart policy 无限拉起 → 服务全断。

机理链：**积压在 default 的历史数据 + 启动期硬失败分区维护 = 无自愈的崩溃循环**。
同一模式会在任何"default 已有目标月数据"的场景复现（长期停机跨月、时钟
回拨、手工写入、新环境接旧库）。

## 修复

- 102 止血（当天已完成）：事务内把 default 中 691 条 8 月数据暂存 → 建
  `audit_logs_2026_08` 分区 → 数据放回，服务自动恢复（healthy），wiring
  随后预建了 2026_08/09/10。
- 代码根治：`internal/pkg/audit/partitions.go` 的
  `ensureMonthPartition` 捕获 23514 后走 `migrateDefaultRows` —— 事务内
  暂存搬出 → 建分区 → 放回 → 删暂存表，失败整体回滚。服务从"遇积压
  崩溃"变为"遇积压自愈"。

## 防回归

- `TestPGWriter_EnsurePartitions_DefaultHasRows`（pgxmock 模拟 23514 →
  断言搬移事务语句序列 → 分区建成），随本 fix 同提交。

## 是否需要 Amended 决策 note

否。未推翻任何既有裁定；E14 的"预建 + default 兜底"设计维持，只是补上
default 积压场景的自愈路径。
