# Stripe 卡收单接入:修正 2026-08-20"放弃 Stripe"裁定为"Stripe 作为卡通道之一"

日期:2026-08-23

## 决策

1. **Stripe 以独立卡收单通道接入,不做聚合商**:沿用 2026-08-20 的网关形态裁定
   (接口收敛域内、薄 HTTP adapter、不 vendor SDK、凭据走环境变量引用不落库),
   Stripe 仅承担"发起收单 + webhook 回调"两件事,记账事实源仍是 `payments`+账单状态机。
2. **发起不落账,回调才落账**:`POST /payments/stripe/intent` 只建 PaymentIntent
   (幂等键=payNo,metadata 携 bill_no/customer_id 供回调寻址),返回 clientSecret 交前端;
   `POST /webhooks/stripe` 验签(v1 HMAC-SHA256,5min 容差)后按事件类型落账:
   succeeded→RecordPayment(流水+账单 PAID 同事务),failed→FAILED 流水留痕;
   充值/寻址失败落无账单流水(customer_id 归属)。pay_no 唯一约束兜底渠道重投幂等。
3. **密钥未配置即降级**:`BOSS_STRIPE_API_KEY` 空 → 网关不注册,intent 端点 400,
   webhook 端点 503;既有模拟直落账(POST /payments)不受影响。
4. **method 仍记 `card`、对账 channel 记 `stripe`**:不新增 payMethod 枚举,
   渠道区分交给对账批次 channel 字段,契约 fields/terms 无需变更。

## why

- 用户直接指定对接 Stripe;Stripe 直连(非 Stripe Connect 聚合形态)不外移记账事实源,
  与 2026-08-20 §1"系统侧自持状态"不冲突,原裁定否决的是"聚合商形态"而非卡通道本身。
- 发起/回调两段式是卡收单的最小闭环:意图发起不落账可避免"发起即 SUCCESS"的模拟语义
  污染真实渠道对账。

## 放弃了什么

- 放弃在 payments 表新增 PENDING 状态(意图发起不落库,状态机不变)。
- 放弃 vendor stripe-go:薄 adapter 两个 HTTP 调用面,长尾依赖不划算(延续原裁定)。

## 关联

- 2026-08-20-sms-payment-channel.md(被本文 Amended 的部分:§3"放弃 Stripe"限聚合形态)
- `internal/domain/billing/paygateway.go`、`internal/pkg/stripe/`、
  `internal/httpapi/user/stripe.go`、`internal/app/wiring_stripe.go`
