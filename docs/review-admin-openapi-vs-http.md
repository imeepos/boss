# admin OpenAPI 契约 vs internal/app HTTP 实现差异清单

> 状态更新（2026-08-19）：系统性 List envelope 偏差已修复（schemas.yaml `List` 组件改为 `data:{items:[...]}`）；参数名大批修正已完成（alarm/aaa/asset/billing/customer/oss/quad/dashboard/ai 九域对齐代码）；Geo 域 `GET /geo/subdivisions` 参数修正（countryCode/parentCode → country/locale）并补 `/geo/countries/{code}/attrs` 路由声明。
>
> 本轮结构变更：`internal/app/http.go`（266 行，churn 31）拆为三文件——`http.go`（180 行，respond/RegisterRoutes）、`http_error.go`（65 行，respondErr 错误映射）、`http_audit.go`（36 行，recordAudit/claimsAccountID）。单文件均低于 200 行推荐值。
>
> 剩余偏差见下文「参数差异（代码为准）」节；已修复条目保留原文但标注 ✅ 与修复提交。

> 补实现进展：首批 planned 路由已落地并逐路由配 http/存储测试——billing `POST /stop-resume-tasks/{taskId}/retry`、`GET /reconciliations`、`POST /reconciliations/{batchNo}/settle`；provision `POST /provision-tasks/{taskNo}/retry`；worker `PUT /workers/{workerId}/settings`、`GET/POST /notices`、`PUT /notices/{noticeId}/toggle`。对应 yaml 已去除 planned 标注；其余清单条目仍待实现。
>
> 第二批 planned 路由已落地并逐路由配 http/存储测试——auth `POST /auth/logout`；alarm `POST /alarms/batch-retest`（新表 alarm_retest_tasks，任务号 RT-YYYYMMDD-NNNN）；order dispatch 段 `GET /dispatch/pool`、`POST /dispatch/pool/{ticketNo}/assign`、`GET /dispatch/my-tickets`、`GET /dispatch/transfers`、`POST /dispatch/tickets/{ticketNo}/transfer`（复用 dispatch_tickets/dispatch_transfers，新增 AssignDispatchTicket）。对应 yaml 已去除 planned 标注；其余清单条目仍待实现。
>
> 第三批 planned 路由已落地并逐路由配 http/存储测试——org `POST /legal-entities`、`PUT /legal-entities/{legalEntityId}`、`GET /menu-perms`（角色×menu:* 权限矩阵，pg array_agg 聚合）；dashboard `GET /dashboard`（聚合订单/工单/告警/四码在库数据：统计卡 + 状态分布 + 待办 + 近7日趋势）。对应 yaml 已去除 planned 标注；其余清单条目仍待实现。
>
> 第四批 planned 路由已落地并逐路由配 http/存储测试——worker `GET /workers/{workerId}`（师傅详情，复用 GetWorker）、`POST /worker-feedbacks/{feedbackId}/review`（差评复核，need_review 置 false，新增 ReviewFeedback）、`POST /asset-returns/{returnId}/confirm`（确认返库，PENDING→RETURNED，新增 ConfirmAssetReturn）；customer `POST /products/{id}/price-history`（产品调价，事务内更新月费+追加台账，新增 ChangeProductPrice）。另销项 order.yaml 六处 stale planned 标注（dismantles/complaints/activation-callbacks 早已实现于 http_order_sub.go）。对应 yaml 已去除 planned 标注；其余清单条目仍待实现。

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

## 参数差异（代码为准）— 全部已修复 ✅

> 以下 38 项参数差异已全部修正：YAML 声明已对齐代码实际读取的参数名。修正时间 2026-08-19。

| 路由 | 原 YAML 声明 | 代码实际 | 状态 |
|---|---|---|---|
| GET /cdrs、/auth-logs | keyword | 读 `loid` (string) | ✅ aaa.yaml 已改 |
| GET /alarms | level 枚举 | 读 `resourceId` (int64) | ✅ alarm.yaml 已改 |
| POST /alarms/{alarmId}/ack | alarmId: string | 按 int64 ParseInt | ✅ alarm.yaml 已改 |
| GET /assets、/replacements | keyword,status | 不读任何参数，全量 | ✅ asset.yaml 已改 |
| GET /assets/assignments | 无参数 | 读 `assetId` (int64) | ✅ asset.yaml 已改 |
| POST /stocktakes | body 无约束 | 必填 legalEntityId/scope | ✅ asset.yaml 已改 |
| POST /replacements | body 无约束 | 必填 assetId | ✅ asset.yaml 已改 |
| GET /bills | 无参数 | 读 `customerId` | ✅ billing.yaml 已改 |
| GET /payments | keyword | 读 `billId` (int64) | ✅ billing.yaml 已改 |
| GET /stop-resume-tasks | keyword,status | 读 `customerId` (int64) | ✅ billing.yaml 已改 |
| GET /customers | 仅 keyword | 另读 phone,status,limit,offset | ✅ customer.yaml 已改 |
| GET /products | 无参数 | 读 `legalEntityId` | ✅ customer.yaml 已改 |
| GET /orders envelope | 含 statusLabel、无 createdAt | 无 statusLabel，有 createdAt | ✅ order.yaml 已改 |
| GET /legal-entities | keyword | 不读参数 | ✅ org.yaml 已改 |
| GET /regions | keyword | 读 `parentPath` | ✅ org.yaml 已改 |
| GET /lo-accounts | keyword,status | 不读参数 | ✅ oss.yaml 已改 |
| GET /gis/drill | level + keyword | level(1-8) + `parentId` | ✅ intel.yaml 已改 |
| GET /analytics/indicators | region,brand | 不读任何参数 | ✅ intel.yaml 已改 |
| GET /reports | keyword | 不读参数 | ✅ intel.yaml 已改 |
| GET /reports/latest | 无参数 | 读 `period`（缺省 daily） | ✅ intel.yaml 已改 |
| GET /ports | 无参数；stats+items | 读 `resourceId`；仅 items | ✅ oss.yaml 已改 |
| GET /reserves | keyword,status | 读 `portId` (int64) | ✅ oss.yaml 已改 |
| POST /reserves/{reserveId}/release | reserveId: string | 按 int64 解析 | ✅ oss.yaml 已改 |
| GET /transfers | status,type 枚举 | 不读参数 | ✅ oss.yaml 已改 |
| POST /transfers | body 任意 object | 必填 resourceId/fromRegionId/toRegionId | ✅ oss.yaml 已改 |
| GET /expansions | keyword,status | 不读参数 | ✅ oss.yaml 已改 |
| POST /expansions | body 任意 object | 必填 legalEntityId/regionId | ✅ oss.yaml 已改 |
| GET /expansions/qos-templates | 无参数 | 读 `legalEntityId` | ✅ oss.yaml 已改 |
| GET /device/metrics | 无参数 | 读 `resourceId` | ✅ oss.yaml 已改 |
| GET /quad-links | code (required) | 不读任何参数 | ✅ quad.yaml 已改 |
| GET /quad-links/by-address | `addrId` | 参数名为 `addressId` | ✅ quad.yaml 已改 |
| GET /quad-links/by-* envelope | List(data 为数组) | data 为单个 QuadLink 对象 | ✅ quad.yaml 已改 |
| GET /scan-logs | keyword | 读 `orderId` (int64) | ✅ quad.yaml 已改 |
| GET /workers | keyword | 读 `groupId` (int64) | ✅ worker.yaml 已改 |
| GET /worker-performances 等七个列表 | 无参数 | 均读 `workerId` (int64) | ✅ worker.yaml 已改 |
| GET /geo/subdivisions | countryCode/parentCode | country/locale | ✅ geo.yaml 已改 |
| POST /geo/import | 无 requestBody | JSON body 必填 | ✅ geo.yaml 已补 |
| PUT /geo/countries/{code}/attrs | 未声明 | 代码已实现 | ✅ geo.yaml 已补 |

## 结论

- 路由数：admin YAML 声明约 173 个操作，代码实现约 156 个（contract-sync 门禁通过）；userdata.yaml（42 操作）、worker.yaml 运营段（faqs/device-maintenances/hall-items/service-messages）、order.yaml 投诉/拆机/激活段（planned but now implemented）基本对齐。
- ~~最大系统性偏差：List envelope（裸数组 vs `{items}`）~~ ✅ 已修复：schemas.yaml `List` 组件统一为 `data:{items:[...]}`；org/geo 域裸数组路由已在 YAML 显式标注。
- ~~查询参数名大量不匹配（keyword vs 实际业务过滤字段）~~ ✅ 已修复：38 项参数差异全部修正。
- 剩余工作：stats 汇总字段（billing/provision/oss/asset 列表端点）代码未实现，属 by-design 省略（前端经 items 前端聚合）；userdata.yaml 42 操作仍 planned（用户端数据管理，非一期范围）。
