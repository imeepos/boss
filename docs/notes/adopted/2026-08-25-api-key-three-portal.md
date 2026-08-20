# 2026-08-25 三端全量支持 API key 鉴权(对接 AI 操作系统)

## 裁定

三端(admin/user/worker)全部接口均支持 `X-API-Key` 免登录鉴权,为 AI 操作系统/CLI
自动化对接提供统一入口;JWT 通道保持不变,两条链路并行。

## 主体边界(每端只认自己的主体,其余 401)

| 端 | API key 主体 | 说明 |
|---|---|---|
| admin `/api/admin/v1` | account(全量 RBAC)/ worker / customer(受限) | 既有行为不变 |
| user `/api/user/v1` | customer(仅) | 既有行为不变 |
| worker `/api/worker/v1` | **worker(仅)** | 本次新增;此前仅 workerJWT |

worker 端实现:`middleware.APIKeyAuth` + 受限 resolver(非 worker 主体在 API key
层即 401)+ `workerAuth` 回退 workerJWT;主体身份注入 ctxPortalWorkerID/Name,
与 JWT 路径同一取值口径。

## 为什么

- AI 操作系统对接需要稳定免登录凭证,不应模拟短信/JWT 登录流程。
- 密钥可吊销、可按主体授权(000042/000044 体系),比长生命周期 JWT 更可控。

## 放弃了什么

- 不做"万能 account 密钥跨三端通用":三端账号体系本就独立,跨端通用会重新引入
  横向越权面(workerJWT 独立 issuer 的既有裁定见 worker/portal.go 文件头)。
- 不给 worker/customer 主体加 RBAC:菜单门禁对其恒 403 的既有隔离不变。
