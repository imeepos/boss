# 2026-09-05 Stripe 配置后端化 + 师傅端现场收款完整落账

## 决策

1. **Stripe 凭据去环境变量**：`BOSS_STRIPE_*`（API_KEY/WEBHOOK_SECRET/CURRENCY/API_BASE/WEBHOOK_URL）自本日起不再是配置源，
   凭据**仅存 biz_params**（admin「支付配置」页 `/stripe-config`，secret 密文落库）。同 commit 清理：
   - `internal/pkg/config/config.go` 删除 `Stripe` struct + 5 行 getenv（含测试断言）
   - `internal/app/wiring_stripe.go` resolver 只读 biz_params；DB 不可读时按未配置形态返回（不再回退 env）
   - `deployments/docker-compose.102.app.yml` 移除三行 sk_test/whsec/currency + 注释
   - 新增 `Application.StripeReady(ctx)` 统一暴露"Stripe 通道可收款"（= 配置页已配密钥且启用,60s 热生效）

2. **未配置即默认线下收款（App 端配置驱动，不再硬编码）**：
   - 用户端新增 `GET /payments/methods`：stripe 可用 → items 含 `card`(银行卡) 且 default=card；
     未配置 → 不含 card、default=cash(线下收款)。App 缴费/充值页改读此接口渲染，失败回落兜底数组（不含 card）。
   - 师傅端 `GET /tickets/:no/charge`：payMethods 依 `StripeReady` 动态下发——可用含 `CARD`，未配置仅 `CASH/QR/POS`。

3. **师傅端现场收款完整落账**（原半成品：amountDue=0、不落账、仅审计留痕）：
   - 应收=预付费订单同环节4 口径（`order.PrepaidAmount`: buy_months×月费,区域覆盖优先）；后付费 amountDue=0 实收自填
   - POST 落账：`pay_no` 派单 + `payments` 流水（bill_id 空 + customer_id 归属,付款成功即 SUCCESS）
   - payMethod 白名单 CARD/CASH/QR/POS → 落账 method：`card` / `offline`（**payments.method 新增枚举 `offline`**，
     VARCHAR(16) 无 CHECK 约束,无需迁移；terms.md §4 + fields.md §3.5 已同步）
   - 响应回 payNo + receiptUrl；落账失败 `[worker-charge] RECORD FAILED` 可 grep 日志留痕
   - Android ChargeScreen：成功展示凭证号、按钮置灰防重复提交（原自动跳 Sign 易致误连重复收）

4. **放弃**：保留 env 兜底的"双源配置"（DB 优先 env 兜底）——一源一处,运维心智与故障排查成本更低；
   不新增订单-支付关联字段（现场收款与环节4 同口径,无 bill,以 pay_no 凭证追溯）。

## 生效/运维影响

- 102 测试环境 sk_test/whsec 需在 admin「支付配置」页重新录入（自检页真实探活）；录入前 Stripe 通道停用,收款默认线下。
- webhook endpoint 自愈循环（stripe_webhook_guard）不受影响：期望 URL 已存 biz_params `stripe.webhookUrl`。

## 契约同步（同 commit）

- `api/openapi/user/billing.yaml` `/payments/methods` + createPayment 枚举加 offline
- `api/openapi/worker/scan.yaml` charge GET/POST 契约（CARD 枚举、落账响应）
- `docs/contract/terms.md` §4 缴费 method 枚举、`docs/contract/fields.md` §1.6.9/§3.5
- bossctl 路由目录手工增量（`/payments/methods`、charge 描述；main 存量 openapi↔routes 漂移非本任务引入,未顺手代偿）