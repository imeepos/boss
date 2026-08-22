# 2026-08-23 下单幂等键方案(requestId 可选键)

## 裁定

下单(环节1)幂等采用**方案A:可选客户端幂等键 `requestId`**,
拒绝**方案B:同客户同地址存在非终态订单即拒单**。

- 载体:orders.request_id(TEXT, nullable)+ 部分唯一索引
  `uq_orders_customer_request ON (customer_id, request_id) WHERE request_id <> ''`
  (migration 000116)。
- 语义:requestId 为空 → 行为不变(不键控);非空 → 重放返回已有订单
  (同 orderNo),并发撞唯一索引回读兜底。admin 与 user 门户两端透传。
- 契约影响:新增可选字段,向后兼容,不冻结破坏。

## 为什么放弃方案B

- 同客户同地址第二单是合法业务(二装/商铺分户),拒单语义误伤;
- 拒单把幂等问题变成业务限制,客户端无法通过重试语义自愈;
- B 无法区分"双击重放"与"真实第二单",A 用显式键精确区分。

## 放弃了什么

- 全局唯一 requestId(不带 customer 维度):防止跨客户键碰撞语义混乱,
  键的作用域=单客户内;不同客户撞同键各自建单,互不干扰。
- 强制 requestId(必填):三端老客户端立即破坏,违背 Q1 冻结期兼容原则。

## 关联

- 审计: docs/acceptance/2026-08-23-write-idempotency-audit.md(覆盖率 90%→100%)
- 回归: internal/app/e2e_submit_idem_test.go
