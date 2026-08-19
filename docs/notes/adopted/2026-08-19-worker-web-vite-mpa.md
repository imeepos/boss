# 师傅端前端落地为 web/worker Vite 多页工程(静态页收编,非 React 重写)

日期:2026-08-19

## 决策

师傅端(装维 H5)以 web/worker 独立前端工程发布:Vite 多页应用(MPA),页面/DOM/CSS 从草稿
docs/worker 原样收编(ES5、无框架、无构建期转译),仅重构对接层 api.js:

- API 基址默认同源 /api/worker/v1(契约 api/openapi/worker.yaml);dev/preview 由
  Vite proxy 转发到 mock(BOSS_API_TARGET 覆盖,默认 http://127.0.0.1:8091),
  生产由网关同源转发;跨域部署可加载前设 window.API_BASE_URL 覆盖。
- 增加登录守卫(非 login.html 无 token 跳登录)与 401 统一清 token 回登录页。

## why

- 草稿页面已按契约完成全量对接(api.js 37 页全覆盖),DOM/class 即视觉契约,重写为零收益高风险;
  MPA 收编保留"保持现有 DOM 结构与 class 不变"的草稿铁律(API-INTEGRATION.md)。
- 同源基址消除跨域 CORS/token 泄露面,且与 web/admin"请求层单通道直连后端"口径一致。

## 放弃了什么

- React/Vite SPA 重写(同 web/admin 形态):移动 H5 30+ 页、无复杂状态,SPA 路由/打包收益为负。
- api.js 内置 mock 端口探测(草稿版 http://<host>:8091 默认):跨域直连 mock 属开发期便利,
  不应固化进发布产物。

## 关联

- 草稿与对接约定:docs/worker/API-INTEGRATION.md
- 契约:api/openapi/worker.yaml(拆分 api/openapi/worker/*.yaml)
- mock:api/mock/combined.js(/api/worker/v1 前缀)

> Amended 2026-08-19: web/worker 路径整体迁移为 web/desktop(Tauri 壳收编草稿),
> 见 2026-08-19-desktop-worker-web-vite-mpa.md。API 前缀 /api/worker/v1 不变。
