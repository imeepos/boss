# BOSS 系统授权门禁对接 release-platform(license-gate)设计裁定

日期:2026-09-01(按仓库时区)。需求:release-platform 管理 boss 的授权可用能力——
boss 系统拿到授权证书才能正常使用全部功能,授权采用**激活码离线授权 + 本地校验**。

## 事实基础(调研结论)

- release-platform(102:38080)是完备的"软件版本/发布/分发/更新控制平面",含
  激活码子系统(`/v1/activation-codes`、`/v1/activations`)、离线令牌
  (`/v1/licenses/{id}/offline-token`,Ed25519 签名)与 Plan/Entitlement 商业层。
- boss 是"装维全流程服务端",无现成 license 概念;已有 `apprelease`(App 发版)
  与 release-platform 能力重叠但不冲突——本次只接授权门禁,不改发版链路。
- 102 实测:建产品→发激活码(offline_allowed)→激活→X(签发阻塞于
  `LICENSE_SIGNER_UNAVAILABLE`,因 `ARTIFACT_SIGNING_*` 未配置)。

## 裁定(Why)

1. **离线授权 + 本地验签**:boss 内嵌 release-platform 的 ed25519 公钥(build 期),
   证书文件落 `/var/lib/boss/license.json`,每次业务请求本地验签(签名/绑定/时间窗/撤销),
   不依赖网络。续费 = release-platform 重签令牌,替换本地文件。
2. **系统级门禁 + 激活豁免**:业务接口(admin/user/worker/open 四端 authed 组)
   统一套 `LicenseGate` 中间件,未授权返回 403 LICENSE_REQUIRED;
   **豁免 `/license/status`、`/license/activate` 与登录/注册/登出/版检**——
   否则出现"没证书进不去、进不去没法激活"的死锁。
3. **绑定机器指纹**:激活时上报指纹(release-platform 绑定),校验时按 `/etc/machine-id`
   或 hostname 计算比对——证书复制到别的机器即失效(防拷贝)。
4. **门禁开关**:`BOSS_LICENSE_ENABLED=true` 才启用;缺公钥视为配置错误启动即失败
   (显式失败优于静默放行)。未启用 = 完全放行(开发/演示零影响)。
5. **admin 授权页**:base 组新增"系统授权"页,粘贴激活码完成兑码→取令牌→本地落盘→验签展示;
   该页与状态接口不要求菜单权限(登录即可达),保证未授权也能进。

## 放弃了什么

- 不做 release-platform 的 Plan/Stripe 商业层接入:本次只消费"证书可用性";
  授权售卖(计划/定价/支付)仍在 release-platform 侧运营,不在 boss 内复制。
- 不做在线实时校验(Entitlement Check):要求 boss 每次请求联网回源,脆弱且违背离线语义。
- 不做证书热更新循环:续期靠 reload/replace 文件,服务重启后生效;中间件每次请求
  重新读文件+验签,替换文件立即生效(无进程内缓存)。

## 前置配置(102 release-platform)

离线令牌签发必需平台侧签名密钥,已配置:
`ARTIFACT_SIGNING_KEY_ID=boss-integration-v1` + `ARTIFACT_SIGNING_PRIVATE_KEY_HEX=<hex>`
(写入 `/home/imeepos/release-platform-deploy/.env.runtime`,api 容器重启生效)。
签名密钥只存 102,公钥 hex 已随本变更内嵌 boss 配置示例。

## 迁移/接线

- 代码零迁移(无新表)。新增 `internal/domain/license`(验签内核/service/store/client),
  `internal/pkg/middleware/license.go`(门禁),admin 授权页(web/admin/src/pages/boss/license),i18n/menu/路由注册。
- 契约同步:docs/contract/fields.md 追加"系统授权"页列。
---

## Amended(B 档):去掉开关,公钥编译期内嵌强制门禁

原裁定第 4 条"门禁开关 `BOSS_LICENSE_ENABLED`"被推翻(2026-09-01 用户质疑:
开关能被部署方直接关闭,授权码失去意义)。新裁定:

1. **删除 `BOSS_LICENSE_ENABLED` 环境变量开关**——运行时没有任何配置可以关闭门禁。
2. **公钥编译期内嵌**:`internal/pkg/buildinfo.LicensePublicKeyHex` 经
   `-ldflags -X` 注入(见 Makefile `build-server`);wiring 从 buildinfo 读公钥,
   不再读环境变量。内嵌的意义不是藏公钥(公钥本可公开),而是**防止公钥被替换**:
   公钥若在 env/配置文件里,攻击者可把自己的公钥写进去再用自己的私钥签假证书绕过;
   内嵌后换公钥必须重新编译,绕过门槛从"改一行配置"变成"破解二进制"。
3. **门禁语义**:注入公钥 → 强制门禁(无证书/失效 → 403 LICENSE_REQUIRED,激活页豁免可达);
   未注入 → 开发构建,打 ALERT 日志(`[license] ALERT: 未内嵌授权公钥...`)后放行,
   便于无证书环境开发调试。内嵌公钥非法 → ALERT 日志,门禁不装配(不静默)。
4. **证书下发不受影响**:证书仍是运行时文件(`/var/lib/boss/license.json`),
   激活页(在线兑码)与预签文件(离线)两条通道都保留。内嵌的是验签基准,
   下发的是每台机器的证书——两回事,互不冲突。
5. **放弃**:环境变量开关(可被关闭);`BOSS_LICENSE_PUBLIC_KEY_HEX` 运行时配置(可被替换)。
