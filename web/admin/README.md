# boss-admin-web(A0 基座)

Vite + React 18 + TS strict。请求层单通道直连 Go 后端 `/api/v1`(契约见 `docs/plan/admin-a0-plan.md`)。

## 开发

```bash
# 1) 起后端(真 PG):种子账号 admin/admin123
go run ./scripts/devseed
BOSS_HTTP_ADDR=:18080 BOSS_GRPC_ADDR=:19090 go run ./cmd/server

# 2) 起前端(代理 /api → 18080)
cd web/admin && pnpm install
BOSS_API_TARGET=http://127.0.0.1:18080 pnpm dev   # http://localhost:5173

# 3) 冒烟(envelope 形状断言)
API=http://127.0.0.1:18080 scripts/smoke.sh
```

## 门禁

- `pnpm typecheck && pnpm build`
- `pnpm test`(vitest:envelope 解包 + client 单通道)
