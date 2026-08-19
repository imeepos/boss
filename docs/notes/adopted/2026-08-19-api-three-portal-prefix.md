# 三端 API 前缀分离 + 门户状态落库(2026-08-19)

## 决策

REST 三端按账号体系独立成前缀与鉴权域,路由入口源文件按端分文件夹:

| 端 | 前缀 | JWT | API key 主体 | 入口包 |
|---|---|---|---|---|
| admin | /api/admin/v1 | aud=admin | account(全量 RBAC)/worker/customer(受限) | internal/httpapi/admin |
| user | /api/user/v1 | aud=user | customer(仅) | internal/httpapi/user |
| worker | /api/worker/v1 | issuer=boss-worker(独立 claims) | 未接 API key | internal/httpapi/worker |

- `auth.Manager.Sign` 增加 aud 参数,`middleware.Authn` 按端精确匹配,跨端 token 一律 401
  (此前仅靠 portalCustomerOnly 运行时守卫,admin 与 user 还共享 /api/v1 前缀、/auth/login 同路径冲突)。
- 自助注册归端:worker-registrations → 师傅端、customer-registrations → 用户端,admin 仅保留审核队列。
- 用户端/师傅端门户状态(验证码/门户账号/偏好/站内消息/钱包/单号)从 handler 进程内 map
  落库 migrations/000052(portal_sms_codes/portal_accounts/portal_prefs/portal_messages/portal_wallets/portal_seq),
  新域 internal/domain/portal(PG 实现 + 内存测试替身)。
- 用户端报修/投诉入客服工单域(complaints 表),不再走进程内台账。
- 用户端 /auth/login(账密)补齐:此前与 admin /auth/login 同路径冲突未注册,前缀分离后无冲突。
- mock 层整体移除(api/mock + 守护脚本),三端数据口径以真实服务端为唯一事实源。

## why

- 三端账号体系/用户表/认证方式完全不同(admin 封闭 RBAC 账号、user 手机号账密+短信、
  worker 手机号+验证码),共用前缀靠运行时守卫隔离是"纪律不靠结构",跨端 token 在密码学上可互认。
- 门户状态进内存:重启即失、多副本不一致,不具备正式环境资格。

## 放弃了什么

- 放弃三端共用一个 /api/v1 + 网关按路径分流:网关层无法承担鉴权语义,端隔离应在服务端边界闭环。
- 放弃给 Claims 加 Subject 字段统一三类主体(仍在演进方向):本次以 aud(端) + issuer(worker)
  实现隔离,最小侵入 auth 包;统一 token 服务落地时再合并(auth 包注释已指向)。
- 放弃 mock 保留开发便利:正式环境接口必须真实业务支撑。
- 短信发送通道(对接运营商)未在本次接入:验证码生成/存储/一次性消费已落库,通道为基础设施配置待接。

## 关联

- Amended 2026-08-18-user-portal-customer-jwt.md 的"与 admin 共享 /api/v1 路径树"表述。
- Amended 2026-08-18 门户 DB 商议清单(短信码存储表缺位/密码模式不可用)。
- 域名注释:internal/httpapi/httpapi.go 映射表。
