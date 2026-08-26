# 2026-09-03 导入任务登记幂等键(clientKey)与下单幂等键(requestId)语义区分

## 裁定

导入任务登记(`POST /import-tasks`)新增**可选客户端幂等键 `clientKey`**(migration 000146),
与下单幂等键 `requestId`(000116)是**两套正交的实现**,命名不同、语义不同,不合并、不混用。

| 维度 | 下单 requestId | 导入登记 clientKey |
|---|---|---|
| 载体列 | orders.request_id | import_tasks.client_key |
| 唯一索引 | 部分唯一 `(customer_id, request_id) WHERE request_id <> ''` | 部分唯一 `(client_key) WHERE client_key IS NOT NULL` |
| 作用域 | 单客户内(维度复合) | 全局单列(无需维度复合) |
| 幂等语义 | 重放**返回已有记录**(查询式,同 orderNo) | 重复登记**覆盖统计数**(upsert,更新 imported/failed/skipped) |
| 空值行为 | `''` 不键控,行为不变 | `NULL` 不键控,允许重复登记(旧地址/Geo 兼容) |
| 前端载体 | 下单请求体透传 | EntityImportPanel 每次 run 生成随机键 |

## 为什么不做统一

- 业务不同:下单幂等是"防重复下单并返回既得订单";导入登记幂等是"防同一
  client 批量任务重复计数、登记可覆盖重试"。前者查询式、后者写入式,语义不可混用。
- 作用域不同:下单需 customer 维度复合防跨客户碰撞;导入登记是 sys 域全局单列,
  无客户维度。
- 复用复用型方案会破坏既有 000116 契约与 000146 迁移独立性,收益低于成本。

## 放弃了什么

- 让 import_tasks 复用 orders.request_id 字段:表/域不同,跨表复用是反模式。
- 把 clientKey 做成必填:破坏旧地址(与 Geo 服务端登记)的无键调用,违背兼容原则。

## 关联

- 迁移: migrations/000146_import_task_idempotency.up.sql
- 实现: internal/domain/user/import_task.go(upsert)、internal/httpapi/admin/sys.go
- 前端: web/admin/src/pages/base/importer/EntityImportPanel.tsx
- 对照: adopted/2026-08-23-order-submit-idempotency.md(requestId)
