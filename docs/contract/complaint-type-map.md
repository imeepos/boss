# complaints.type → 故障类型中文标签映射

> 权威源：OpenAPI `api/openapi/worker/schemas.yaml::TicketDetail.faultTypeLabel`
> 用途：师傅端工单详情页 Card1 工单头，报障单显示故障类型 + SLA 级别。
> 新增 `complaints.type` 值时必回写本表。

| complaints.type | faultTypeLabel (中文) | SLA 小时 | 权威依据 |
|---|---|---|---|
| `SINGLE_OUTAGE` | 单户断网（紧急 SLA ≤4h） | 4 | 全案 3.3 报障 |
| `PARTIAL_OUTAGE` | 部分业务不可用（常规 SLA ≤8h） | 8 | 全案 3.3 |
| `SLOW_NET` | 网速慢（一般 SLA ≤24h） | 24 | 全案 3.3 |
| `WIFI_ISSUE` | Wi-Fi 信号问题（一般 SLA ≤24h） | 24 | 全案 3.3 |
| `DEVICE_FAULT` | 设备故障（紧急 SLA ≤4h） | 4 | 全案 3.3 |
| `OTHER` | 其他问题（一般 SLA ≤24h） | 24 | 全案 3.3 |

> 未知 type 值 fallback："报障（待分类）", SLA 默认 24h。
