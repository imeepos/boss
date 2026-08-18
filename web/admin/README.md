# boss-admin-web(A0 基座)

Vite + React 18 + TS strict。请求层单通道直连 Go 后端 `/api/v1`(契约见 `docs/plan/admin-a0-plan.md`)。

## 开发

```bash
# 1) 起前端(代理 /api → 192.168.0.102:28080,默认对接已部署 API)
cd web/admin && pnpm install
pnpm dev   # http://localhost:5173

# 2) 指定后端(可选,覆盖默认)
BOSS_API_TARGET=http://127.0.0.1:18080 pnpm dev

# 3) 冒烟(envelope 形状断言)
API=http://192.168.0.102:28080 scripts/smoke.sh
```

## URL 一次性偏好(可分享链接)

启动时消费并抹除,权威存储仍是 localStorage:

- `?theme=dark|light` —— 主题覆盖
- `?lang=zh-CN|en-US|ms-MY` —— 语言覆盖
- `?token=<jwt>` —— dev 免登录(仅 dev 构建;prod 抹除但不应用)

## dev 免登录(真实 token,禁止假数据)

```bash
# 真实 /auth/login 换 JWT,打印免登录 URL(默认 admin/admin123 → 102 部署)
node scripts/dev-token.mjs
# 直接拼进截图/脚本
open "$(node scripts/dev-token.mjs)"
```

token 即真实会话,/auth/me 与业务接口全走真实后端,能暴露真实问题;
仅 `pnpm dev` 生效,生产构建不应用 `?token=`。JWT 过期后重跑本脚本即可。

## 门禁

- `pnpm typecheck && pnpm build`
- `pnpm test`(vitest:envelope 解包 + client 单通道)
