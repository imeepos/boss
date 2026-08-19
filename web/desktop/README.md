# boss/desktop — 管理端桌面客户端(Tauri 壳)

用 Rust + Tauri 2 将 `web/admin` 包装成 macOS/Windows 桌面客户端。前端零改动:

- dev:窗口加载 `http://localhost:5173`(admin 的 Vite dev server);
- prod:内嵌 `web/admin/dist` 静态资源。

API 访问:admin 前端一律直连所选服务端的绝对接口地址(后端已配置 CORS;
登录页/服务端配置页切换,localStorage 记忆,内置默认 `http://192.168.0.102:28080`)。
壳内不再做 `/api` 反代 —— 请求通道唯一,均为浏览器直连。

## 前置

- Rust stable(`cargo 1.97+`),macOS 需 Xcode CLT;
- Node + pnpm(构建 admin 前端产物)。

## 使用

```bash
# 开发:先起 admin dev server,再起桌面壳
cd web/admin && pnpm dev          # 5173
cd web/desktop && pnpm install && pnpm desktop:dev

# 生产打包
cd web/admin && pnpm build        # 产出 web/admin/dist
cd web/desktop && pnpm desktop:build
```

纯 `cargo run`(不经 tauri-cli)时使用 `frontendDist`(需先构建 admin dist);
`cargo tauri dev` 才会走 `devUrl`。

## 结构

- `src/main.rs` — 建窗口;
- `tauri.conf.json` — devUrl/frontendDist/窗口与打包配置;
- `capabilities/default.json` — 最小 IPC 权限(桌面壳不暴露插件)。
