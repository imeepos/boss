# admin OpenAPI 契约 vs internal/app HTTP 实现差异清单

> 状态更新：本清单中的契约偏差已于后续修订中全部对齐代码（参数/envelope/路径参数名；org 与 addresses 裸数组 envelope；未实现路由统一标注"未实现 planned"；/olt-devices 并入 /device/metrics）。本文件保留作为修订依据的原始差异记录。

复核范围：`api/openapi/admin*.yaml`（含 `api/openapi/admin/*.yaml`）与 `internal/app/http*.go`，逐路由比对路径/参数/envelope。**一切以代码为准**。复核方法：4 组并行 agent 逐路由比对 + 人工抽查关键证据行（均已验证）。

## 系统性差异（影响所有列表路由）

1. **List envelope 裸数组 vs {items}**：`schemas.yaml` 的 `List` 组件声明 `data` 为裸数组，且自注"前端请求层适配 {items}"；代码统一返回 `data:{items:[...]}`（`internal/app/http.go:21-23` respond + 各 list handler）。涉及几乎所有 `$ref List` 路由。代码为准：应改 schemas.yaml。
2. **stats 字段普遍缺失**：`billing.yaml /bills /arrears`、`provision.yaml /provision-tasks`、`oss.yaml /ports`、`asset.yaml /assets` 均声明 `data.stats` 汇总；代码一律只返回 `{items}`，无 stats（如 `http_billing.go:24`、`http_provision.go:26`、`http_resource.go:38`、`http_asset.go:22`）。

## YAML 声明、代码未实现（缺失）

| 路由 | 方法 | 证据 |
|---|---|---|
| /auth/logout | POST | auth.yaml:35（自注未实现）；http.go:69-107 无 |
| /alarms/batch-retest | POST | alarm.yaml:24-47；http_device.go 全文无（envelope 亦用 code/message 而非 code/msg） |
| /stop-resume-tasks/{taskId}/retry | POST | billing.yaml:80-88 |
| /reconciliations, /reconciliations/{batchNo}/settle | GET/POST | billing.yaml:90-109 |
| /products/{productId}/price-history | POST | customer.yaml:46-62 |
| /dispatch/pool、/dispatch/pool/{ticketNo}/assign、/dispatch/my-tickets、/dispatch/transfers、/dispatch/tickets/{ticketNo}/transfer | GET/POST | order.yaml:68-127 |
| /dismantles (GET/POST)、/complaints、/complaints/{ticketNo}/close、/activation-callbacks、/activation-callbacks/{callbackId}/retry | GET/POST | order.yaml:129-189 |
| /legal-entities (POST)、/legal-entities/{id} (PUT) | POST/PUT | org.yaml:11-28；http_org.go 仅 GET |
| /menu-perms、/data-scopes | GET | org.yaml:64-101（menu-perms 自注未实现） |
| /dashboard | GET | dashboard.yaml；internal grep 无 |
| /reports (POST)、/reports/{reportId}/send | POST | intel.yaml（自注未实现） |
| /provision-tasks/{taskNo}/retry、/provision-templates (POST) | POST | provision.yaml:18,35 |
| /workers/{workerId}、/workers/{workerId}/settings | GET/PUT | worker.yaml:22-49 |
| /worker-feedbacks/{feedbackId}/review、/worker-messages (POST)、/notices (GET/POST)、/notices/{id}/toggle、/faqs (GET/POST)、/faqs/{id}/toggle、/device-maintenances、/hall-items、/service-messages、/asset-returns/{id}/confirm | 各 | worker.yaml:75-248 |
| /accounts (GET/POST)、/accounts/{id} (PUT/DELETE)、/roles、/params、/params/{key}、/audit-logs、/import-tasks | 各 | sys.yaml；http.go:109-123 注册清单无 |
| /olt-devices | GET | oss.yaml:97-111；设备指标实际为 /device/metrics |
| userdata.yaml 全部 42 个操作（/users、/coupons、/user-balances、/addons 等） | 各 | userdata.yaml:1-233；yaml 自注实体存于 api/mock |

## 代码实现、YAML 未声明（多余）

| 路由 | 方法 | 证据 |
|---|---|---|
| /tickets/{ticketNo}/scan-bind | POST | http_scan.go:50（注释指契约在 worker/scan.yaml） |
| /tickets/{ticketNo}/dismantle/scan | POST | http_scan.go:82（注释指 worker/asset.yaml） |
| /addresses、/addresses/import | GET/POST | http_org.go:63-93；admin yaml 无（body 为 `{rows:[{path,name}]}`，返回 `{imported:n}`） |

## 参数差异（代码为准）

| 路由 | YAML 声明 | 代码实际 | 证据 |
|---|---|---|---|
| GET /cdrs、/auth-logs | keyword | 读 `loid` (string) | http_aaa.go:21,30 |
| GET /alarms | level 枚举 | 读 `resourceId` (int64) | http_device.go:14 |
| POST /alarms/{alarmId}/ack | alarmId: string | 按 int64 ParseInt，非数字静默 0 | http_device.go:23 |
| POST /alarms ack envelope | data 未定义 | data 恒 `{ok:true}` | http_device.go:28 |
| GET /assets、/replacements | keyword,status | 不读任何参数，全量 | http_asset.go:16-17,116 |
| GET /assets/assignments | 无参数 | 读 `assetId` (int64) | http_asset.go:42 |
| POST /stocktakes | body 无约束 | 必填 legalEntityId≠0、scope 非空；status 缺省 DOING；返回 `{id}` | http_asset.go:60-74 |
| POST /stocktakes/{taskId}/diff-handle | taskId: string | 按 int64 解析 | http_asset.go:77 |
| POST /replacements | body 无约束 | 必填 assetId；replacementNo/status 自动补；返回 `{id,replacementNo}` | http_asset.go:86-103 |
| GET /bills | 无参数 | 读 `customerId` | http_billing.go:19 |
| GET /payments | keyword | 读 `billId` (int64) | http_billing.go:28 |
| GET /stop-resume-tasks | keyword,status | 读 `customerId` (int64) | http_billing.go:46 |
| GET /customers | 仅 keyword | 另读 phone,status,limit,offset | http_customer.go:16-22 |
| GET /customers/{customerId}/verify-logs、/products/{productId}/price-history | 路径参数 customerId/productId | gin 注册为 `:id` | http_customer.go:30,64 |
| GET /products | 无参数 | 读 `legalEntityId` | http_customer.go:42 |
| GET /orders envelope | 列表项含 statusLabel、无 createdAt | 无 statusLabel，有 createdAt | http_order.go:19-29 |
| GET /legal-entities | keyword | 不读参数 | http_org.go:28 |
| GET /regions | keyword | 读 `parentPath` | http_org.go:55 |
| GET /lo-accounts | keyword,status | 不读参数 | http_aaa.go:11-18 |
| GET /gis/drill | level + keyword | level(1-8) + 未声明 `parentId`；keyword 未用 | http_gis.go:19-31 |
| GET /analytics/indicators | region,brand | 不读任何参数 | http_analytics.go:19-26 |
| GET /reports | keyword | 不读参数 | http_analytics.go:51 |
| GET /reports/latest | 无参数 | 读 `period`（缺省 daily） | http_analytics.go:60-62 |
| GET /ports | 无参数；stats+items | 读 `resourceId`；仅 items | http_resource.go:32-39 |
| GET /reserves | keyword,status | 读 `portId` (int64) | http_resource.go:61-68 |
| POST /reserves/{reserveId}/release | reserveId: string | 按 int64 解析 | http_resource.go:51-59 |
| GET /transfers | status,type 枚举 | 不读参数 | http_resource.go:110 |
| POST /transfers | body 任意 object | 必填 resourceId,fromRegionId,toRegionId；返回 `{id,transferNo}` | http_resource.go:71-90 |
| GET /expansions | keyword,status | 不读参数 | http_resource.go:118 |
| POST /expansions | body 任意 object | 必填 legalEntityId,regionId；返回 `{id,expansionNo}` | http_resource.go:126-145 |
| GET /expansions/qos-templates | 无参数 | 读 `legalEntityId` | http_resource.go:147-154 |
| GET /device/metrics | 无参数 | 读 `resourceId` | http_device.go:31-38 |
| GET /quad-links | code (required) | 不读任何参数 | http_quadlink.go:11-12 |
| GET /quad-links/by-address | `addrId` | 参数名为 `addressId` | http_quadlink.go:48 vs quad.yaml:86 |
| GET /quad-links/by-asset/by-customer/by-port/by-address envelope | List(data 为数组) | data 为单个 QuadLink 对象 | http_quadlink.go:26,35,44,53 |
| GET /scan-logs | keyword | 读 `orderId` (int64) | http_scan.go:103 |
| GET /workers | keyword | 读 `groupId` (int64) | http_worker.go:21 |
| GET /worker-performances 等七个列表 | 无参数 | 均读 `workerId` (int64) | http_worker.go:30-84 |

## 结论

- 路由数：admin YAML 声明约 173 个操作，代码实现约 108 个；userdata.yaml（42 操作）、sys.yaml、order.yaml 派单/拆机/投诉段、worker.yaml 运营段基本未实现（多数 yaml 已自注"未实现"或指向 mock）。
- 最大系统性偏差：List envelope（裸数组 vs `{items}`）、stats 汇总缺失、查询参数名大量不匹配（keyword vs 实际业务过滤字段）、路径参数 string vs int64。
- 建议后续：以本清单为准修订 schemas.yaml 与各 yaml 参数段；缺失路由要么补实现、要么在 yaml 标注 planned。
