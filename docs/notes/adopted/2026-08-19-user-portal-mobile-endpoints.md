# 移动端门户缺失端点补齐:评审/落库映射/第三方网关姿态与演示种子

日期:2026-08-19

## 决策

1. **缺失端点全部补在 `api/openapi/user` 契约侧,复用现成域服务**,不新增重复读写路径。新端点集中在 `internal/httpapi/user/`:`POST /auth/logout`、`GET/POST /coupons`、`GET/POST /addresses`、`GET /products`、`GET /addons`(+subscribe/unsubscribe)、`GET/POST /orders`、`GET /orders/:orderNo`(时间轴)、`POST /orders/:orderNo/{cancel,urge,change-address}`、`GET/POST /orders/:orderNo/rate`、`GET/POST /plans/:planId/{cancel,change,move}`、`GET /bills`、`GET/POST /payments`、`GET /invoices`、`GET /complaints`。
2. **单文件超 300 行的旧 `trade.go/billing.go/portal_orders.go` 拆分为领域小文件**(trade/portal_orders/portal_plans/portal_addons/portal_catalog/portal_invoices/portal_gen),每文件 ≤300 行、每函数 ≤30 行,符合 AGENTS.md。
3. **`orders.address_id` 同时承载两套地址主键**:geo `addresses` 树节点 ID 与用户地址簿 `user_addresses.id`。这是既有迁移与用户侧下单的既有事实,新增 List/详情联表 `user_addresses` 恢复地址名(COALESCE(a.name, ua.detail))。
4. **订单评价落独立表 `order_ratings`**(order_no UNIQUE,stars/attitude/quality CHECK 1..5),不经订单主表;订单状态机不动。
5. **第三方网关(短信/支付/税局)保持"模拟在后端 DB"姿态**:短信验证码写 `portal_sms_codes`(无真实发送商,冒烟从 DB 读码);支付 `RecordPayment` 同事务插 `payments` + 置账单 PAID(无真实支付商);发票走既有 Tax 域(manual/leqi/bir_eis)不新增外部依赖。这延续此前 tax/支付"多属地网关并存、系统侧自持状态"的裁定。
6. **演示种子走迁移 `000054_portal_demo_seed`(幂等)**:ONLINE/AGENT 渠道、客户 213 钱包 88.00、示例地址/套餐/加值/优惠券/档位,供移动端端到端冒烟。

## Why

- 移动端(android 应用)与后端契约对齐的最低可用集,此前若干读写端点缺失或为内存桩,无法端到端验证。
- 评价/变更地址/套餐操作此前为响应用户交互的内存 map,不落库;本轮全部落到真实表,保证重启不丢、管理端可见。
- 演示种子以 migration 幂等内置(与 PSGC 行政区划内置同理),既免人工灌数据又可在 CI/冒烟重复执行。

## 放弃了什么

- 不为此新增独立 SMS/支付/税局外部供应商 SDK 与后台队列(阶段 7+ 再做);当前模拟姿态可支撑端到端验收,且不引入内网不可达的第三方依赖。
- 不在订单主表加评分列(避免与 stage/status 状态机耦合);评价独立表更易于扩展图文/追评。
- 不把 `orders.address_id` 强制收敛为单一外键(既有数据两套并存,收敛属破坏性迁移,另行评估)。

## 关联

- 评审闭环:`docs/architecture-review.md`(发现 3.x 移动端接口缺口)
- 契约:`api/openapi/user/*.yaml`
- 迁移:000053_portal_order_rating.up.sql、000054_portal_demo_seed.up.sql
- 既有禁令:`docs/notes/adopted/2026-08-18-tax-jurisdiction-china-shudian.md`(多属地网关并存)
