# complaints.type → 故障类型中文标签映射

> 权威源：OpenAPI `api/openapi/worker/schemas.yaml::TicketDetail.faultTypeLabel` + 用户端落库口径（2026-08-30 补登字面）
> 用途：师傅端工单详情页 Card1 工单头，报障单显示故障类型 + SLA 级别；admin 用户详情 faults/complaints 段展示。
> 新增 `complaints.type` 值时必回写本表。

## 1. 装维域故障码（管理后台工单域口径）

> 权威依据：全案 3.3 报障 + `internal/domain/order/sub.go` complaintTypeLabels / complaintSlaHours。

| complaints.type | faultTypeLabel (中文) | SLA 小时 | 权威依据 |
|---|---|---|---|
| `SINGLE_OUTAGE` | 单户断网（紧急 SLA ≤4h） | 4 | 全案 3.3 报障 |
| `PARTIAL_OUTAGE` | 部分业务不可用（常规 SLA ≤8h） | 8 | 全案 3.3 |
| `SLOW_NET` | 网速慢（一般 SLA ≤24h） | 24 | 全案 3.3 |
| `WIFI_ISSUE` | Wi-Fi 信号问题（一般 SLA ≤24h） | 24 | 全案 3.3 |
| `DEVICE_FAULT` | 设备故障（紧急 SLA ≤4h） | 4 | 全案 3.3 |
| `OTHER` | 其他问题（一般 SLA ≤24h） | 24 | 全案 3.3 |

> 未知 type 值 fallback："报障（待分类）", SLA 默认 24h。

## 2. 用户端口径（前缀落库，2026-08-30 补登）

> 用户端报障/投诉不裸落故障码，而是带前缀落库：`complaints.type = '用户报障: ' + faultType`
> （`internal/httpapi/user/service_handlers.go:portalFaultTypeName`）与
> `'用户投诉: ' + req.Type`（`internal/httpapi/user/complaint_handlers.go:portalCreateComplaintFlow`）。
> 展示侧一律 strip 前缀（`service.go:portalFaultTypeLabelFromStored` / `userdata/pg_aggregate.go` faults 段
> `replace(type, '用户报障: ', '')`）；admin 用户详情按 `type NOT LIKE '用户投诉:%'` 归 faults 段、
> `type LIKE '用户投诉:%'` 归 complaints 段（2026-09 收口，fields.md §2.1.1）。
> 用户端提交进参白名单由 binding oneof 强制，白名单外 42200。

| 存储值（complaints.type 完整前缀串） | strip 后值 | 中文标签 | binding 白名单 | 权威依据 |
|---|---|---|---|---|
| `用户报障: no_internet` | no_internet | 无法上网 | faultType: `no_internet/slow/ont_fault/other` | service_handlers.go:62 |
| `用户报障: slow` | slow | 网速慢 | 同上 | 同上 |
| `用户报障: ont_fault` | ont_fault | 光猫故障 | 同上 | 同上 |
| `用户报障: other` | other | 其他 | 同上 | 同上 |
| `用户投诉: attitude` | attitude | 服务态度 | type: `attitude/quality/billing/suggestion/other` | complaint_handlers.go:131 |
| `用户投诉: quality` | quality | 服务质量 | 同上 | 同上 |
| `用户投诉: billing` | billing | 计费问题 | 同上 | 同上 |
| `用户投诉: suggestion` | suggestion | 意见建议 | 同上 | 同上 |
| `用户投诉: other` | other | 其他 | 同上 | 同上 |

> 真库实测（102 库，2026-08-30）：`SELECT DISTINCT type FROM complaints` 现有取值 =
> `用户投诉: suggestion` / `用户报障: no_internet` / `用户报障: slow` 三值；
> 装维域故障码（SINGLE_OUTAGE 等）当前无生产样本（仅单测/测试种子），契约保留待用。
> 前端文案键：faults 段 `dFault*`、complaints 段 `dCpn*` 见
> `web/admin/src/pages/bss/user/detail-view.ts` FAULT_TYPE / COMPLAINT_CATEGORY。
