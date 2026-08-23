# Q1 CS/AR 真实回放结果（2026-08-24）

环境：`http://192.168.0.102:28080`，使用真实 admin JWT，未使用 mock。

## 指标与队列端点

| 端点 | 结果 |
|:---|:---|
| `GET /api/admin/v1/complaint-metrics` | code=0；返回 7 个 CS SLA/状态指标 |
| `GET /api/admin/v1/ar-metrics` | code=0；返回欠费总额、客户数、停机数、逾期账单数、账龄桶、最近运行时间 |
| `GET /api/admin/v1/collection-tasks` | code=0；返回 `items` 队列 |
| `GET /api/admin/v1/complaints` | code=0；真实返回 4 个 OPEN 工单 |
| `GET /api/admin/v1/arrears` | code=0；真实返回客户 213 欠费 500、账龄 40 天、状态 COLLECTING |

## 互查证据

- 客户聚合 `/users/213` 同时返回投诉、账单、地址、余额等服务事实。
- `/orders` 返回真实订单状态；工单记录保留 `customerId/orderId` 关联字段。
- `/dispatch/my-tickets` 返回装维派单记录。
- `/resources` 返回真实网络资源；`/alarms` 返回真实告警，告警带 `resourceId`，可沿资源链路定位。
- 所有接口均使用 `{code,msg,data}` 响应信封。

## 边界结论

主链路已具备可查询、可回放的 ID 关联边界；本次环境内没有待处理催收任务，因此没有执行任务状态写回。复杂智能催收和独立呼叫中心未建设，符合 Q1 明确不做范围。
