# 用户端首发 https 提级为硬门槛：含公网/异地用户（决策记录）

> 日期 2026-08-27｜状态 已采纳｜作用域 mobile/user/android（首发窗口）

## 决策

用户拍板：**首发用户包含公网或异地用户**。因此 https 公网域名从「首发后第一技术债」**提级为首发硬门槛**：release 包不得再以明文 http 内网 IP 作为对外入口。

执行：情形 B 双预案（meeting-minutes/2026-08-27-android-https-dual-plan.md）立即可执行，两天窗口：

1. 域名 + DNS（D1 上午）
2. TLS 证书（D1 上午）
3. 反代 443→28080，业务端零改（D1 下午）
4. 客户端 release `bossBaseUrl` 改 https 域名 + network_security_config 白名单清理 + 全链路回归（D2 半天）
5. 新包 apksigner 验证归档

## why

- 首发含公网/异地用户 → 明文 http + 内网 IP 入口不可接受（拦截/篡改/拒审风险，阿澈与安然 battle 中双方都承认此情形下 https 为硬门槛）。
- 现状利好：cleartext 收敛已落地（Manifest 已引用 network_security_config 且移除 usesCleartextTraffic，xml 禁明文+白名单内网域）；UpdateApi 客户端已就绪（/client/latest force/sha256/downloadUrl + deviceId 分桶）。
- 反代方案实现成本最低：业务端零改，仅客户端 baseUrl 与自更新链路配合。

## 放弃了什么（被否决项）

| 选项 | 否决理由 |
|---|---|
| 维持内网直装（情形 A） | 用户确认含公网/异地用户，直接违反明文底线 |
| 首发暂不迁移、公网用户用内网 IP 访问 | 公网无法访问 192.168.0.102，体验归零 |
| 商店渠道首发 | 两周审核+资质+HTTPS 硬性要求不达标（安然，沿用首次会议结论） |

## 风险与兜底

- 证书/域名时延 → 域名当天可办则两天可行，不可则首发顺延（安然门禁：https 就绪是发车前置）。
- 旧包指向内网 → 强制更新兜底（UpdateApi force=true + 服务端回滚通道）。
- 灰度：GRAY→PUBLISHED 门控，灰度期 force=false；回滚=服务端回置。

## 关联

- 执行预案：meeting-minutes/2026-08-27-android-https-dual-plan.md
- 会议纪要：meeting-minutes/2026-08-27-user-android-two-week-launch.md（九节 D13-D14 安全收口前置化）
- 执行台账：docs/ops/user-android-launch-status.md