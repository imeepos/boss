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

## 门禁

- `pnpm typecheck && pnpm build`
- `pnpm test`(vitest:envelope 解包 + client 单通道)
