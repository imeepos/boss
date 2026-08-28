# 运维批量改进 · 验收证据

> 日期：2026-08-28｜四项 ops 改进（commit 51d40a95），全部 102 真实环境验证。

## 1. 骨灰链路脚本风控共存

- 改动：验收前检测并临时关停 `risk.direct.enabled`，结束后恢复。
- 实测：`mainchain-acceptance.sh 3` → **3/3 = 100% 成功率**（26 秒），ORD-576/577/578，无 42300 拦截。

## 2. P1 待办 resolve API

- 新增 `POST /notifications/resolve`（refType+refID），复用域层 Resolve（000115 办结+SLA 时间戳）。
- 实测：
  - resolve `daily_recon/20260822` + `daily_recon/20260823` → 两笔均 code:0
  - SLA stats 变化：`overdueOpen 2→0`，`resolvedLate=2`（超期办结正确计数）
- 审计：`notification.resolve` 自动留痕。

## 3. 注册来源统计端点

- 新增 `GET /customer-registrations/source-stats`（按 source 聚合 total/approved）。
- 实测：
  ```json
  {"source":"未携带","total":6,"approved":5}
  {"source":"landing-m4","total":1,"approved":0}
  ```
- 与 M4 注册追踪数据一致（landing-m4 为 M4 3.4 演练注册单，仍 PENDING）。

## 4. 渠道佣金归属（前项补验）

- `partner_entity_id` 列已部署（000162）；C 案下单实测 `legal_entity_id=6, partner_entity_id=15`。
- 佣金 SQL `COALESCE(partner_entity_id, legal_entity_id)` 已部署 + 集成测试 PASS。
- 渠道订单推到终态（自举资源）→ 佣金 ACCRUED → settle 列入放量运营阶段。

## 全部通过门禁

- `go test ./...` 64 包全绿
- `check-contract-sync` A/B/C/D/E 全 OK
- 102 部署 healthy + healthz 200