# 支付链路收口后续收尾:隧道驻留 / 演示门户 / 缴费口径 / 余额与合成客户裁定

日期:2026-09-03

## 1. 范围

支付链路收口(2026-08-26 已上线)之后的 6 项验收收尾:
- P0-1 隧道自动化驻留 + 存活告警
- P1-1 演示门户 locale.js 污染 + 信封解封
- P1-2 门户缴费记录口径(SUCCESS only)
- P2-1 充值余额"预存"语义裁定
- P2-2 admin-web 支付配置页浏览器实测
- P2-3 合成客户充值边界

## 2. 关键决策

### 2.1 隧道告警通道:不复用微信/SMS 通道,改走后台提醒中心

**why**:隧道告警是后台运维事项,非用户级通知;与既有
`internal/app/stripe_webhook_guard.go::emitStripeWebhookAlert`
(refType=stripe_webhook_guard)复用同一通道(refType=stripe_tunnel),
提醒中心按(refType, refID, category)幂等聚合,同源不刷屏。

新端点 `POST /api/admin/v1/ops/notify-emit`(account 主体 API key 调用,
refType 白名单)仅扩展最小攻击面,无菜单 UI 入口。

### 2.2 演示门户 locale.js 拆分:3 语区 × 单文件 = 953 行 → 322/296/296 + 73 行引擎

**why**:单文件 953 行超 300 行红线且不便维护。引擎与词条分离后:
- 引擎 locale.js 仅依赖 window.LOCALES(子文件填充),启动时按 localStorage 选语区;
- 子文件 locale.{zh-CN,en-US,ms-MY}.js 各 < 322 行;
- 切换语区无需改引擎,仅替换子文件。

### 2.3 缴费记录仅 SUCCESS

**why**:SUCCESS 是用户级"缴费完成"事件;FAILED 仅留痕排查,
REFUNDED 由财务冲账——都不向终端用户默认展示。admin `/payments`
保留全量供审计。`?include=failed,refunded` 显式 opt-in。

### 2.4 余额 = 预存,不是可用余额

详见 2026-09-03-portal-wallet-balance-semantic.md。

本期不动代码:grep 实证 AdjustBalance 接口签名虽允许负 delta,但
production 无任何调用方(仅测试用例)。未来接消费时需补
`portal_wallet_transactions` 流水表 + CHECK balance >= 0。

### 2.5 合成客户(隔离空间负数 ID)拒绝充值

详见 2026-09-03-synthetic-customer-recharge-boundary.md。

裁定拒绝(走 42200)而非放开:`payments.customer_id` 是硬 FK
(000068 起),负数 ID 在 customers 表无对应行触发 23503。合成客户
不在真实收费场景,不动 FK、不建旁路表。

## 3. 落地 commit(共 6 条,自 main 69c74dda 起)

| commit | 用途 |
|---|---|
| cd26fce9 feat(ops) | 隧道 URL 上报 + 容器 DOWN 告警 |
| db276358 fix(docs/user) | 演示门户 locale.js 污染清除 + 信封解封 |
| 30172147 fix(user-portal) | 缴费记录口径仅 SUCCESS |
| 56a835a8 fix(user-portal) | 余额语义裁定 + 合成客户充值边界 |
| 741b8a86 fix(ops) | stripe-tunnel-url.sh 引号/换行 bug |
| 92ddfe88 refactor(user-portal) | 拆 billing_handlers.go + openapi 补 $ref |
| e35bcab4 style | gofmt 收尾 |

## 4. 102 实测留证

- 新端点 `POST /api/admin/v1/ops/notify-emit`:102 实测 200 ok;
  白名单拒绝 `refType=importer` 返回 42200 + 字段级 error。
- 隧道脚本:实际拉到 `cf-stripe-boss` 当前 trycloudflare URL,与
  `GET /stripe-config` 返回的 `webhookUrl` 完全一致;
  模拟 DOWN(`STRIPE_TUNNEL_CONTAINER=cf-nonexistent`)触发 URGENT
  告警入库,二次触发幂等(refType+refID 聚合同一条)。
- admin-web 支付配置页(/base/stripeconfig):102 实测登录后渲染
  两卡片(通道配置/回调配置),字段回显与 API 一致,自检按钮
  可达。截图存 `/tmp/payment-followup-evidence/`。
- 巡检门禁 db-patrol-gate:11 类软引用孤儿均 0;--selftest PASS。

## 5. 关联 adopted notes

- 2026-08-26-payment-hardening.md(本次收口的前置)
- 2026-09-03-portal-wallet-balance-semantic.md(P2-1)
- 2026-09-03-synthetic-customer-recharge-boundary.md(P2-3)
- 2026-09-03-import-task-idempotency.md / 2026-09-03-importer-permission-scope.md
  (同批并列,非本批范围)

## 6. 放弃了什么

- 不做余额消费实现(仅裁定);
- 不重做 13 演示页结构;
- 不动 Stripe 真实退款联动、不引入新通道;
- 不增 admin-web 大 UI 扩展,仅扩展一个运维端点;
- 不在 admin 端 `/payments` 加 include 过滤(审计需要全量)。