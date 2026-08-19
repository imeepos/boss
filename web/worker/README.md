# boss-worker-web(师傅端)

Vite 多页应用(MPA),页面收编自草稿 docs/worker(DOM/CSS/ES5 原样保留),
对接 /api/worker/v1(契约 api/openapi/worker.yaml,mock 为 api/mock/combined.js)。

## 开发

```bash
cd web/worker && pnpm install
MOCK_PORT=8091 node api/mock/combined.js &   # 仓库根目录起 mock 契约服务
pnpm dev                      # http://localhost:5174,代理 /api/worker/v1 → mock

# 指定后端(可选):
BOSS_API_TARGET=http://127.0.0.1:8091 pnpm dev
```

## 构建/冒烟

```bash
pnpm build        # → dist/,每页独立 html
pnpm preview      # :4174,同样走代理
pnpm smoke        # 断言全部页面与关键 API 可达(需 preview + mock 在跑)
```

## 约定

- API 基址默认同源 /api/worker/v1;跨域部署在加载 api.js 前设 window.API_BASE_URL。
- 登录守卫:非 login.html 无 token 自动跳登录;401 清 token 回登录。
- 页面改动规则沿草稿 docs/worker/API-INTEGRATION.md(DOM/class 不变、单文件<=300行、工单号走 ?no=)。
决策记录:docs/notes/adopted/2026-08-19-worker-web-vite-mpa.md。
