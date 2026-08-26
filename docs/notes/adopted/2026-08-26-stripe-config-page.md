# Stripe 支付配置页化:沿用 biz_params 热更模式,env 凭据降级为兜底

日期:2026-08-26

## 决策

1. **支付(stripe)通道配置收口到后台配置页**:按短信/实名/推送同一模式接入——存储复用
   `biz_params`(key 前缀 `stripe.`),secret 字段(apiKey/webhookSecret)经 secretbox AES-GCM
   密文落库、GET 掩码、PUT 空串=不修改;运行时由 `stripe.Dynamic` 消费(60s 热生效)。
2. **env 凭据降级为兜底而非主源**:`stripeConfigResolver` 先读 biz_params,DB 无值回退
   `BOSS_STRIPE_*`(与 sms/realid/push resolver 同构);"." 含义:后台保存约 1 分钟热生效,
   端口行为不变——未启用/缺 apiKey → 发起端点 400、webhook 503("密钥未配即降级"裁定延续)。
3. **apiKey 与 webhookSecret 分组存储**:channel 组(enabled/apiKey/publishableKey/currency/
   apiBaseUrl)与 webhook 组(webhookSecret);自检完整性跨两组,通过后以草稿+已存配置真实
   探活 `GET /v1/balance`(零副作用)。
4. **快速隧道保持测试环境专属**:公网入口(cloudflared cf-stripe-boss)仅为 102 测试阶段
   webhook 可达所需,不上公网生产;配置页的 webhookSecret 与 2026-08-26-stripe-test-env-wiring
   的重建流程配套使用。

## why

- 用户明令支持后台配置(同短信/实名);凭据动态化后,改密钥/换币种/开关通道不再依赖发版+env。
- 沿用既有四通道模式,不另起配置存储/加密/热更机制(避免第五套轮子),契约与审计记录自动对齐。

## 放弃了什么

- 放弃"配置页仅存 UI、运行时仍读 env"的假配置形态。
- 放弃为 stripe 另建"配置表/注册中心/长连接"——biz_params + Dynamic 已覆盖(60s 热更新窗口,
  与 sms/realid/push 一致)。
- 放弃把 publishableKey 当 secret:pk 本就公开(服务端不校验),仅存 SK 签发场景。

## 关联

- 2026-08-23-stripe-card-channel.md(通道形态)、2026-08-26-stripe-test-env-wiring.md(测试凭据)
- `internal/pkg/stripe/{config,dynamic}.go`、`internal/app/wiring_stripe.go`、
  `internal/httpapi/admin/stripeconfig*.go`、`migrations/000147_stripeconfig_menu`、
  `docs/contract/fields.md §1.6.9`、web/admin `pages/base/stripeconfig`