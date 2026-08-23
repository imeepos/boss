# Q1 CS/AR 基础交付契约

## 域边界

- **CS（客服工单）**拥有 `complaints` 及 `cs_*`：受理、报障/投诉分类、状态 `OPEN/PROCESSING/CLOSED`、SLA、升级、知识库、回访和评价。
- **AR（应收信用）**拥有 `arrears` 及 `ar_*`：账龄快照、信用等级、规则停机线、催收任务、承诺还款和核销。
- 订单、客户、资源、告警和装维任务是只读关联事实；跨域不直接调用实现，仅通过 ID 和事件/任务回放衔接。
- 停复机仍由既有 AAA 编排执行，AR 只生成策略结果和 `stop_resume_tasks`，人工可接管。

## 核心模型

`cs_ticket_extensions` 扩展既有工单：优先级、升级级别、首次响应、SLA 截止和解决码；`cs_ticket_events` 是状态与动作审计轨迹。知识库和回访评价分别落 `cs_knowledge_articles`、`cs_callbacks`。

AR 每日按客户生成 `ar_aging_snapshots`（当前、1-30、31-60、61-90、90+），`ar_credit_profiles` 保存可解释等级/风险分/停机线，`ar_collection_tasks` 提供人工催收队列，`ar_payment_promises` 与 `ar_writeoffs` 留承诺和核销审计。

## 指标口径

`service_metric_snapshots` 按日保存分子、分母和值（迁移 000118），至少支持：`first_response_rate`、`first_contact_resolution_rate`、`sla_overdue_rate`、`repeat_ticket_rate`、`arrears_recovery_rate`。没有样本时值为 0，禁止用模拟数据补齐。

## 主链路回放

1. 客户报障 → `complaints(OPEN)` → 首响/处理 → `cs_ticket_events` → `CLOSED` → 回访评价。
2. 账单逾期 → 账龄快照 → 信用策略生成催收任务 → 承诺还款或停机任务 → 缴费后复机；每一步均可按 ID 查询和重放。
3. 工单详情通过 `customer_id/order_id` 互查订单、客户与装维任务；资源/告警通过既有关联查询，不复制事实。

明确不建设复杂智能催收和独立呼叫中心。
