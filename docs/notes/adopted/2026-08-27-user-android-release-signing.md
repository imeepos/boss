# user Android 发布签名与版号定档（首发内网直装）

日期: 2026-08-27

## 决策

来源：2026-08-27 上线计划会议（D0 发布基建，阿澈/安然）。

1. **独立 release keystore**：`mobile/user/android/release.keystore`（alias `boss-user-release`，RSA 2048）由
   `scripts/user-android-release-sign.sh --init` 生成；`keystore.properties` 存放密码，二者均被
   `.gitignore` 排除、绝不入库；`keystore.properties.example` 入库作模板。
2. **发布门禁脚本**：`user-android-release-sign.sh`（无参）构建 assembleRelease 后 apksigner verify，
   并把 release 包证书指纹与 debug 包强对比——指纹相同即判定"回落 debug 签名"并阻断出包。
3. **版号定档**：首发 `versionCode=2 / versionName=0.1.0`。历史内测 debug 包均为 versionCode=1，
   首发必须 >1 才能覆盖老机（INSTALL_FAILED_VERSION_DOWNGRADE 风险）；此后每次发版单调 +1。
4. **cleartext 收敛**：删 `usesCleartextTraffic=true`，改 `network_security_config.xml` 白名单
   （192.168.0.102 内网首发 + 43.240.223.138 联调 + 10.0.2.2/localhost/127.0.0.1 模拟器），
   其余域名禁明文。https 公网迁移列首发后第一技术债，届时仅需改白名单 + buildConfig URL。
5. **POST_NOTIFICATIONS 立项**：Manifest 声明 + MainActivity 启动时（API 33+）运行时请求。

## why

- 无独立 keystore 时 release 回落 debug 签名：apk 不可升级、不可审计、商店必拒（内网直装可绕过商店拒审，但签名独立是升级通道与回滚通道的底座）。
- 版本号单调递增是自更新通道（UpdateApi）与灰度回滚（versionCode 白名单）的前提；内测包已占 versionCode=1。
- targetSdk 36 下 Android 13+ 通知必须运行时授权，否则通知功能静默不可用（体验事故）。

## 放弃了什么（被否决项）

- 继续用 debug 签名出 release（被否决：无升级/回滚底座、不可审计）。
- keystore.properties 入库或 CI 明文存密码（被否决：密钥方案不落地仓库，沿用 CI repo secret 解码落地约定）。
- 首发即上 https / 商店渠道（被否决：两周内渠道不现实，内网直装 + cleartext 白名单收敛首发；https 列首发后第一技术债，见会议纪要 b 部分）。

## 出包留档（2026-08-27 D0 验收）

- 当前机器 keystore：`mobile/user/android/release.keystore`（gitignored，alias `boss-user-release`），
  对应证书 SHA-256 指纹：`1C:AC:B3:E1:03:CE:87:6E:9E:22:C9:FD:C8:D2:08:7C:ED:BF:45:58:A4:10:92:F8:EC:25:02:16:00:EE:F2:DA`。
- 该 keystore 仅存本机且不入库，**必须备份**（升级/回滚通道依赖同一把 key，丢失即换钥，存量 APK 无法平滑升级）。
- release 包指纹验签门禁：`scripts/user-android-release-sign.sh`（apksigner verify + 与 debug 包指纹强对比阻断回落）。

## 关联

- 会议纪要 `meeting-minutes/2026-08-27-user-android-two-week-launch.md`（二、Battle 与 五、行动建议 D-5/D-1）
- 构建脚本 `scripts/build-install-user-android.sh`（--release 出包入口）
- `docs/contract/fields.md` 8F（UpdateApi 自更新）