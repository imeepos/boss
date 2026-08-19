# admin API 访问 = CORS 直连绝对地址,移除 Vite 代理与壳内反代

> **Amended(2026-08-19)**:本文"恒含内置默认环境 102"已作废——改为**无默认环境**:
> 登录页**不自动弹框**,未配置时选择器提示并拦截提交,由用户点"管理服务端"
> 手动打开 ServerManagerDialog(增删改查+启用)。存储层同文件 `web/admin/src/lib/serverConfig.ts`。

日期:2026-08-19

## 决策

web/admin 对后端的唯一请求通道是 `client.ts → apiBaseUrl()` 直连**绝对接口地址**:

- 后端已配置 CORS,浏览器跨源直连,不再需要任何代理层;
- `web/admin/vite.config.ts` 删除 `/api` dev 代理(含 BOSS_API_TARGET),禁止回加;
- 服务端配置(localStorage)恒含内置默认环境 `http://192.168.0.102:28080`
  (与原 proxy 目标同址),未做任何选择即默认生效;"同源相对 /api/v1"模式整体移除;
- 桌面壳(web/desktop)随之删除 Rust 侧 `/api` 反代(api_proxy.rs),壳只负责装窗口。

## why

- 代理层(仅同源免 CORS 存在)与 CORS 直连并存会形成两条请求通道,排障口径分裂;
- 服务端配置页 + 登录页选择器都以绝对地址为语义,相对基址是无法表达的残缺形态;
- 少一层转发,桌面/浏览器行为一致。

## 放弃了什么

- Vite dev proxy(BOSS_API_TARGET):本决策直接废除。
- 桌面壳内 on_web_resource_request 反代:随之失去存在意义,删除。
- "prod 同源反代"部署形态:若未来网关要求同源,须重开决策,不许静默回加代理。

## 关联

- 修正:docs/notes/adopted/2026-08-19-desktop-tauri-wrap-admin.md(其"/api 壳内反代"方案作废)
- web/admin/src/lib/serverConfig.ts、src/api/client.ts、src/pages/base/servers/、src/pages/login/serverPicker.ts
