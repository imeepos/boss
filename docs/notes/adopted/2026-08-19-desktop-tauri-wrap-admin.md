# 管理端桌面客户端 = Tauri 2 壳内嵌 web/admin 构建产物(/api 壳内反代)

> **Amended(2026-08-19)**:本文"/api 壳内反代 + dev 走 Vite proxy"方案已作废。
> 后端配置 CORS 后 admin 一律直连绝对地址,壳内反代与 Vite 代理均移除,
> 见 docs/notes/adopted/2026-08-19-admin-api-direct-cors.md。壳内嵌 dist 的部分仍有效。

日期:2026-08-19

## 决策

`web/desktop` 以 Rust + Tauri 2 包装 `web/admin` 为桌面客户端,前端工程零改动:

- dev:窗口 devUrl 指向 admin 的 Vite dev server(5173),`/api` 沿用 Vite proxy;
- prod:`frontendDist = ../admin/dist` 内嵌静态产物;`/api/*` 由 Rust 侧
  `on_web_resource_request` 拦截后 reqwest 反代到 Go 后端(同源免 CORS),
  后端地址 `BOSS_API_TARGET` 覆盖,默认 `http://192.168.0.102:28080`
  (与 web/admin/vite.config.ts 同口径),不可达回 502 envelope。

## why

- admin 请求层是单通道相对基址 `/api/v1`(client.ts 铁律),桌面化若改前端加
  绝对地址/tauri-plugin-http 会破坏"prod 同源反代"口径,产生第二套请求通道。
- 壳内反代让 web 与桌面共用同一份 dist,不复制不 fork,发布产物单一事实源。

## 放弃了什么

- tauri-plugin-http + 前端注入绝对 API 基址:需要改 admin 请求层,双通道。
- 桌面壳直连后端 origin(后端服务 admin 静态页的部署形态):绑定部署拓扑,
  本地离线/多后端切换不灵活,且 dev 热更新链路断掉。
- Electron:体积与内存开销大,且本仓库无 Node 桌面运行时诉求。

## 关联

- web/desktop/README.md(使用与结构)
- web/admin/vite.config.ts(dev proxy 口径来源)
