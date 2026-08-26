# 支付收口后续真实环境证据补录

日期:2026-09-04

## 1. 102 隧道 URL 变化与 Stripe endpoint 自愈

真实操作:

```text
旧 URL: https://book-surprised-intent-carl.trycloudflare.com/api/user/v1/webhooks/stripe
执行: docker restart cf-stripe-boss
新 URL: https://round-mumbai-officer-compression.trycloudflare.com
执行: /home/imeepos/boss/scripts/ops/stripe-tunnel-url.sh
结果: stripe.webhookUrl -> https://round-mumbai-officer-compression.trycloudflare.com/api/user/v1/webhooks/stripe
```

随后真实调用 `/api/admin/v1/stripe-config/channel/test` 初始返回:

```text
后台 webhook endpoint URL 与期望不一致,
回调将静默失效,自愈循环会自动同步
```

Guard 周期后轮询结果:

```text
05:15:48 配置完整,余额探活成功。后台 webhook endpoint 与期望 URL 一致
GUARD_SELF_HEAL_OK
```

102 cron 实际配置:

```text
*/3 * * * * cd /home/imeepos/boss && ./scripts/ops/stripe-tunnel-url.sh >> /tmp/stripe-tunnel-url.log 2>&1 # stripe-tunnel-url-push
```

随后 `/tmp/stripe-tunnel-url.log` 连续多次输出:

```text
already up to date: https://round-mumbai-officer-compression.trycloudflare.com/api/user/v1/webhooks/stripe
```

## 2. 用户门户真实登录 + 非空缴费记录

真实环境:
- 后端:`http://192.168.0.102:28080`
- 用户门户静态页面通过临时 CORS 代理访问真实 API(代理只转发,不 mock 业务数据)
- 账号:测试客户 `13900001234`,验证码由 102 `BOSS_DEBUG_SMS=1` 真实 debug 端点取得

真实 CDP 流程:

```text
POST /api/user/v1/auth/sms-code -> 200
GET  /api/user/v1/debug/sms-code?phone=13900001234&scene=login -> 200
POST /api/user/v1/auth/login -> 200
GET  /api/user/v1/home -> 200
GET  /api/user/v1/bills -> 200
GET  /api/user/v1/payments -> 200
```

页面断言:

```json
{
  "url": "http://127.0.0.1:18093/pay.html",
  "token": true,
  "records": 4,
  "text": "... ¥1 · 余额充值 ... ¥999 · 2026-08 账期 ... ¥500 · 2026-07 账期 ..."
}
```

结论:真实登录后缴费记录不再空列表,真实 API envelope 拆封有效。截图:
`/tmp/user-pay-login-real.png`。

## 3. bossctl 真实 API 冒烟

```text
/tmp/bossctl --server http://192.168.0.102:28080 --api-key <ops-main> \
  call POST /ops/notify-emit --data '{...}'
=> {"ok":true}
```

## 4. 限定与未声称项

- 本证据使用真实 102 后端与真实客户 API 鉴权;CORS 代理仅用于本地静态门户与 102 API 的浏览器 origin 组合,不提供 mock 数据。
- admin-web 支付配置页面已在此前真实 102 浏览器证据中确认 `/auth/me`、`/stripe-config` 200;短生命周期 toast 不作为唯一证据,以 Network/API 返回为准。
- 本 note 不声称已完成生产客户终验或真实资金支付,仅证明演示门户读路径与 webhook 自愈读/配置路径。