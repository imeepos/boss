# Q1 CS/AR 主链路回放用例

## CS 报障到评价

1. `POST /api/user/v1/faults` 创建 `complaints`，状态为 `OPEN`，并关联 `customer_id/order_id`。
2. 客服工作台 `GET /api/admin/v1/complaints` 查询工单，按 `ticketNo` 追踪订单与客户；装维任务由既有 `dispatch_tickets` 查询，资源/告警从关联订单与设备域读取。
3. `POST /api/admin/v1/complaints/{ticketNo}/close` 办结，写 `closed_at`，指标端点 `GET /complaint-metrics` 反映 SLA 和平均办结时长。
4. 用户端回访/评价写入 `cs_callbacks`，评价结果与工单 ID 可追溯。

## AR 逾期到复机

1. `POST /api/admin/v1/dunning-runs` 按规则把账单标记为 `OVERDUE`，生成 `arrears` 快照。
2. `GET /api/admin/v1/ar-metrics` 校验总额、客户数、停机数和账龄桶。
3. 规则引擎将任务写入 `ar_collection_tasks`；`GET /collection-tasks?status=PENDING` 按优先级和到期时间返回队列。
4. 人工通过 `POST /collection-tasks/{taskId}/status` 写入 `DOING/DONE/FAILED`、结果和备注；不使用智能催收或自动外呼。
5. 欠费客户走既有 `/arrears/{customerId}/stop` 与 `/resume`，停复机结果落 `stop_resume_tasks`，认证状态可回放验证。

每一步使用真实 102 部署数据或真实集成测试数据，不使用 mock 代替业务链路；响应均遵循 `{code,msg,data}` 信封。
