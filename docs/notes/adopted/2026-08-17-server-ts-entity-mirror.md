# server-ts 作为 TS 实体镜像层保留

日期：2026-08-17（补记于 2026-08-18，来源提交 5e7d4dc）

## 决策

保留 `server-ts/`（TypeScript + TypeORM 实体模型）作为 Go 实体的 TS 镜像层，职责单一：为 `web/admin` 提供类型来源与实体契约对账基准，不承载业务逻辑、不部署。

## why

三端页面（admin/user/worker）是 TS，需要一份与 migrations 对齐的实体类型；对齐审计台账（alignment-audit.md 的 A/B/D/E/G 各类 finding）以"Go 模型 / TS 实体 / migrations 三方对账"的方式发现并销掉了大量漂移，证明该层有真实防漂移价值。

## 放弃了什么

- 单一 Go 模型 + 手写 TS 类型：三端字段漂移只能靠人肉 review（D1 角色枚举混入 dispatcher 即此类漂移实例）。
- server-ts 直接跑业务（TypeORM 入库）：双写路径必然漂移，明确否决——server-ts 不连生产库。

## 关联

- docs/contract/data-layers.md、alignment-audit.md
- 后续：字段对账应由机械门禁（scripts/check-contract-sync）执行，server-ts 是其输入之一
