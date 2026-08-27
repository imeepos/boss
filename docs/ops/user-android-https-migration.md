# user Android https 公网迁移方案（首发后第一技术债）

> 依据：2026-08-27 上线计划会议上阿澈×安然 battle 和解结论——首发内网直装 + cleartext 白名单，
> https 公网域名列首发后第一技术债；若首发用户群含公网/异地用户，则本方案提前为硬门槛。

## 前置条件（产品/业务书面确认）

- 首发用户是否全内网：全内网→https 为技术债排期；含公网/异地→https 为首发硬门槛（见 adopted note 发布签名章节）。

## 客户端动作（Android，本仓）

1. **Base URL 换域**：`mobile/user/android/app/build.gradle.kts` 中
   `BOSS_BASE_URL`（debug 联调域可保留，release 必换）改为 `https://<公网域名>/api/user/v1`。
2. **cleartext 收敛收口**：`res/xml/network_security_config.xml` 删除内网明文域例外
   （192.168.0.102 / 43.240.223.138 / 10.0.2.2 等），`base-config cleartextTrafficPermitted=false` 恒生效——
   迁移完成后 App 不得再发任何明文请求（release 抓包验收）。
3. **自更新通道同步**：UpdateApi 指向的 `/client/latest` 与 APK 下载域一并换 https
   （`client.yaml` 契约中的下载地址由服务端下发，服务端负责换域）。
4. **验证**：改域后 6 条关键路径回归（登录真码→产品→下单→支付→订单/账单→实名）+ release 抓包确认零明文。

## 后端动作（102 环境，明远）

1. 公网域名 + TLS 证书（反代 Nginx/Caddy 终止 TLS，转发 28080）。
2. `BOSS_CORS_ORIGINS` 补 https 域（若管理端同域）。
3. UpdateApi 版本元数据端点返回 https 下载地址。

## 回滚

- 客户端 base 支持 `-PbossBaseUrl` 覆盖：极端情况可出临时包切回内网白名单；正常路径为服务端指向旧 APK 灰度回滚。