# Stripe 测试环境接线:webhook endpoint 经 REST API 创建,隧道/凭据固定入 102 部署 compose

日期:2026-08-26

## 决策

1. **webhook endpoint 用 Stripe REST API 创建,不用 stripe CLI listen**:
   凭 `.env` 的 `STRIPE_SK` 调 `POST /v1/webhook_endpoints`,URL=`cloudflared 快速隧道 +
   /api/user/v1/webhooks/stripe`,事件仅 `payment_intent.succeeded` / `payment_intent.payment_failed`;
   签名密钥(whsec)在创建响应中一次性返回,与 sk_test 一起按 2026-08-18"内网私有 gitea 简化
   CI 直接入库"裁定落入 `deployments/docker-compose.102.app.yml` server 段 environment。
2. **公网入口用 cloudflared 快速隧道**(容器 `cf-stripe-boss`,`--url http://server:8080`
   boss-app 网络,`--no-autoupdate`,restart=unless-stopped);快速隧道 URL 重启会变,
   隧道重启后需按下方"重建步骤"重新注册 webhook endpoint 并同步 whsec。
3. **币种 php 已实测可用**:`POST /v1/payment_intents`(amount=10000,currency=php)成功,
   `BOSS_STRIPE_CURRENCY=php` 显式写入 compose。
4. **充值页对齐 Android**:android 用户端充值仅有微信/支付宝,后端无充值用 Stripe intent
   端点(webhook 的"无账单落账"路径已支持,customer_id 归属),故 docs/user/topup.html 移除
   「银行卡」选项,只保留缴费页(pay.html)卡通道走托管收银台。

## why

- 没有 Stripe 账号登录凭据(仅有 API key),stripe CLI 的 OAuth 登录不可行;REST 建 endpoint
  是钥匙可及的闭环,whsec 一次返回且长期有效(与 endpoint 生命周期一致),适合入库固化。
- 102 为内网地址,Stripe 回调不可达;无 Cloudflare 域名/账号可用命名隧道,快速隧道是
  现有钥匙(用户已备 NGROK/隧道意图)内最快闭环。
- 缴费页「银行卡」此前是模拟直落账假象,与 Android 已接托管收银台不一致;本次接线使其一致。

## 放弃了什么

- 放弃 stripe CLI listen/登录式方案(需浏览器 OAuth 登录 Stripe 账号,无账号凭据)。
- 放弃命名隧道/固定公网域名(无 Cloudflare 域名与账号)。
- 放弃为充值新增 Stripe intent 端点(新 API 面 + openapi/契约变更,收益低于 Android 端
  充值本就无卡通道的事实;无账单落账路径已存在,未来要加只加一个端点)。
- 放弃在 .env 层面改名映射:server 只认 `BOSS_STRIPE_*`,材料键名 STRIPE_SK/STRIPE_PK/
  NGROK_KEY 是给人工接线看的,不引入额外映射层。

## 隧道/endpoint 重建步骤(隧道重启后)

```bash
# 1) 找新 URL(102 上)
docker logs cf-stripe-boss 2>&1 | grep -i "trycloudflare" | tail -1
# 2) 用新 URL 重建 endpoint 并取新 whsec(本机,SK 在 .env)
curl -u "$SK:" -X POST https://api.stripe.com/v1/webhook_endpoints \
  --data-urlencode "url=https://<新URL>/api/user/v1/webhooks/stripe" \
  -d "enabled_events[]=payment_intent.succeeded" \
  -d "enabled_events[]=payment_intent.payment_failed"
# 3) 用新 whsec 同步 compose 的 BOSS_STRIPE_WEBHOOK_SECRET 并重启 boss-server
```

## 关联

- 2026-08-23-stripe-card-channel.md(通道形态裁定,本 note 是它的测试环境落地)
- 2026-08-20-sms-payment-channel.md(网关抽象/凭据走环境变量)
- `deployments/docker-compose.102.app.yml`、`internal/httpapi/user/stripe*.go`、
  `internal/pkg/stripe/`、`docs/user/{api.js,pay.html,topup.html}`