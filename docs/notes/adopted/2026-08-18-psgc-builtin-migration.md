# 菲律宾行政区划以 migration 全量内置

日期：2026-08-18（来源提交 06:01，PSA PSGC 2025-07-31，43769 节点）

## 决策

菲律宾行政区划（PSGC）43769 节点以 golang-migrate migration 全量内置入库，不做运行时拉取，也不做首次启动 seed。

## why

业务部署地在菲律宾，区划字典是订单/地址/派单的硬依赖，缺失即主链路不可用；PSG C官方数据年更一次，churn 极低。内置保证 fresh-clone 即可用、环境间零差异。落地过程中的教训已沉淀在 self-evolving notes（远端库优先/断点续传/数据真实性三层校验）。

## 放弃了什么

- 运行时拉取 PSGC API：引入外网依赖，离线部署（102 服务器内网）不可行。
- 首次启动 seed 脚本：与 migration 双轨，新鲜度无法由迁移版本号表达。

## 关联

- geo 域（geo/pg_country.go、pg_import.go 仍保留增量导入接口供后续年度更新）
- ADR-002（ltree path 权威，subdivision 层级遵循同一模型）
