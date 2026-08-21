# server-ts 已移除

日期：2026-08-21

## 决策

已移除 `server-ts/`（TypeScript + TypeORM 实体模型），职责已迁移至 Go 实体。

## 背景

原决策（2026-08-17）保留 server-ts 作为 TS 实体镜像层，为 `web/admin` 提供类型来源与实体契约对账基准。经过评估，发现维护双层实体（Go + TypeScript）增加了维护成本，且 Go 实体已能满足需求。

## 迁移方案

1. 所有实体定义已迁移至 Go 实体
2. 前端类型定义通过 Go 代码生成工具自动生成
3. 枚举定义已迁移至 Go 枚举定义

## 关联

- docs/contract/fields.md、alignment-audit.md 已更新
- README.md 目录结构已更新
