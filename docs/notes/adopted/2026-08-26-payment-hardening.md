# 支付链路健壮性收口:充值原子落账 / webhook 隧道自愈 / 账单流水双挂强制

日期:2026-08-26

## 1. 充值落账 = 无账单流水 + 余额原子入账(billing 域新增 RecordTopup)

**决策**:新增 `billing.RecordTopup`——同一事务 INSERT 无账单流水(pay_no 唯一,
必须带 customer_id 防孤儿)并 upsert `portal_wallets.balance`(仅 SUCCESS 增余额,
FAILED 只留痕)。webhook 无账单充值成功走它;`POST /topups` 直接充值同步改走它
(原 AdjustBalance+CreatePayment 两调用非原子)。

**why**:webhook 无账单支付成功此前只落 payments 流水、不增余额,存在"扣了款余额没到账"
缺口(验收发现)。余额权威态在 portal_wallets(D1 裁定),充值记账事务内一并更新与
userdata.AdjustUserBalance 直写 portal_wallets 同型(薄适配),跨域写有先例。

**放弃了什么**:放弃 portal 域提供"带事务参数"的余额接口(跨库事务透传侵入大);
放弃 handler 级两调用+补偿(不能保证原子,失败会留半状态)。

## 2. Stripe webhook 隧道自愈:期望 URL 配置 + 周期比对,UPDATE 保密钥 / CREATE 换密钥

**决策**:新增后台自愈循环(stripe_webhook_guard,5 分钟周期,app 装配启动):
期望 URL 来自 `stripe.webhookUrl`(配置页 webhook 组 + env 兜底 `BOSS_STRIPE_WEBHOOK_URL`,
由 102 cron 脚本 `scripts/ops/stripe-tunnel-url.sh` 在隧道变化时拉平);
周期查询 Stripe 后台 endpoint(s):URL 失配 → `POST /v1/webhook_endpoints/:id` 仅改 URL
(**签名密钥随 endpoint 不变**,无需换 whsec);无本系统 endpoint → 重建并一次性返回新
whsec 落 `stripe.webhookSecret`(biz_params,60s 热生效)。自愈/重建/查询失败都进后台
提醒中心告警(RefID=期望 URL 幂等防刷屏)。配置页自检同步追加 endpoint 一致性核对
(P1-2),未配置期望 URL 时明确提示。

**why**:快速隧道 URL 重启必变,旧手工程序靠文档重建,失败静默无从得知。UPDATE 保密钥是
关键洞察(Stripe 文档:signing secret 绑定 endpoint 而非 URL),常见路径(仅 URL 变)
零配置变更即可自愈。窗口收敛到 隧道发现(2~5 分钟 cron)+ 自愈周期(5 分钟)。

**放弃了什么**:放弃服务端自行发现隧道 URL(cloudflared 快速隧道无本地 URL API,
需外部喂入);放弃 bash 直接调 Stripe 自愈(逻辑进 Go 可测、告警/审计齐);放弃
CREATE 替换 UPDATE(每次重建都换 whsec,徒增配置变更面)。

## 3. 账单缴费落账行强制双挂 customer_id

**决策**:`RecordPaymentWithCoupon` 落账前按 bill 回填 customer_id(双挂语义,
fields.md §3.5 / 000068);调用方显式传入与账单不一致的 customer_id 拒收
(防"账单属 A、流水挂 B"的隐形孤儿,按 A 直挂查询会漏读)。迁移 000148 对存量
"bill 有、customer 空"行按 bill 回填(102 实测 5 行,源自 admin /payments 任意
body 与早期 stripe webhook 落账路径)。

**why**:双挂语义要求账单流水 bill_id 与 customer_id 同挂,漏挂行在
`ListPaymentsByCustomer` 按 customer_id 直挂分支查不到(仅 EXISTS 兜底分支可见),
对账/门户缴费记录口径不齐。

**放弃了什么**:放弃 DB CHECK 约束(bill_id NOT NULL ⇒ customer_id NOT NULL)——会
拒绝未来合法的低层 CreatePayment 调用面,代码层门禁+迁移已够。

## 关联

- 2026-08-23-stripe-card-channel.md / 2026-08-26-stripe-config-page.md /
  2026-08-26-stripe-test-env-wiring.md(本 note 是其健壮性收口)
- `internal/domain/billing/pg_topup.go`、`internal/app/stripe_webhook_guard.go`、
  `internal/pkg/stripe/endpoint.go`、`migrations/000148_payment_dualanchor_backfill`