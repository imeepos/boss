# audit_logs 按月分区策略（ops）

> 2026-09-05 固化｜来源：2026-09-04 表数量对账纪要行动项 4（消除 2 vs 6 口径差）｜权威源：migrations/000001 + internal/pkg/audit/partitions.go

## 结构

- 父表 `audit_logs`：RANGE(created_at) 分区（000001 创建），业务表计数时按 1 张父表计。
- 迁移期子表 2 张：`audit_logs_2025_08`（首月）+ `audit_logs_default`（兜底）。
- 运行期子表滚动：服务启动时 `internal/app/wiring_events.go` 调用 `EnsurePartitions(ctx, time.Now(), 2)`，
  幂等预建「当月 + 未来 2 个月」共 3 个月度子表（`CREATE TABLE IF NOT EXISTS ... PARTITION OF`）。
  因此任意时点库内通常可见 4~6 张 `audit_logs_YYYY_MM` 子表（相邻两次启动窗口的并集）。

## default 分区搬移（PG 23514 自愈）

若目标月数据已积压在 default 分区，直接建分区会被 PG 以 23514（分区越界）拒绝。
`ensureMonthPartition` 在事务内原子执行：建 `_stage` 临时表 → 搬出该月行 → 删 default 中该月行 → 建分区 →
放回 → 删 stage。任一步失败整体回滚，启动不会崩溃循环（修复 0001-postmortem / architecture-review 1.1、1.4、E14）。

## 对账口径（硬性）

1. 统计表数量时，`audit_logs_*` 整族（父表除外）一律剔除；父表 `audit_logs` 计 1 张业务表。
2. 迁移建表口径 = 2 张迁移期子表；库存口径 = 2 + 运行期滚动 N 张。两口径差是预期行为，不是漂移。
3. `schema_migrations`（工具自建）与 `spatial_ref_sys`（PostGIS）同样不计入业务表。

## 保留与清理

- 当前无自动清理/归档策略，历史分区与 default 分区长期保留；如需按月归档另行立项。
- 分区缺失的故障表现为启动失败（EnsurePartitions 出错即 fail-fast，随 wiring 中断启动），属显式可观测信号。
