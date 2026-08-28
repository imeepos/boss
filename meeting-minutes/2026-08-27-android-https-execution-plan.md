# Android 首发 https 硬门槛 · 执行时间表（情形 B 生效）

日期：2026-08-27（情形 B 生效日） ｜ 负责人：安然（发布与风控） ｜ 状态：执行中，发车前置于「https 域名就绪」

## 现状核查（实查 102 + 本地）

- 102 已有 nginx（/etc/nginx/conf.d/ 下 dsh.conf、minio.conf 可复用反代写法）；28080 服务存活。
- ✅ ③后端侧已完成（台账 §十三）：独立 nginx 443→28080 反代，https://192.168.0.102 全端点 200（真码 token /orders /points /profile）；30VU 压测 P95=99.7ms 0 失败；发布记录 id=7（user v0.1.0 versionCode=4 PUBLISHED）已注入，灰度字段 status/rollout_percent/whitelist_ids 齐备。
- 剩余后端侧项：①② 域名+DNS + Let's Encrypt 证书；落地后替换 nginx cert/key 两行即可（台账 §十三 部署要点）。
- 客户端改动点已定位：`app/build.gradle.kts`（release bossBaseUrl 默认值）+ `api/Api.kt`（经 BuildConfig 引用，无需另改）。
- release 签名已就绪：`mobile/user/android/keystore.properties`（rootProject 目录）存在，release.keystore 2798B 已落地并记录指纹，后台验证中；若验证回落 debug 再按红线处理。
- ⚠️ 红线：禁止重生成 keystore——换钥会断存量升级通道（sha256 验签/force 升级链路失效）。

## 执行时间表

| # | 项目 | 负责 | 截止 | 验收 |
|---|---|---|---|---|
| ① | 域名选定 + DNS 解析指向 102 公网入口 | 域名负责人/产品 | D-day 12:00 | `dig` 解析生效 |
| ② | TLS 证书（Let's Encrypt，HTTP-01） | 安然（发布） | D-day 18:00 | 证书签发成功 |
| ③ | nginx 反代 443→127.0.0.1:28080（仿 dsh.conf）业务端零改 | 后端 | D-day 18:00 | ✅ 完成（台账 §十三）：https://192.168.0.102 全端点 200；压测 P95=99.7ms；发布记录 id=7 已注入 |
| ④ | 客户端：release bossBaseUrl→https 域名（worktree 内改）+ networkSecurityConfig 清明文内网域 + 真机全链路回归 | 技术 | D+1 12:00 | 登录/下单/实名/更新检查 over https 通过 |
| ⑤ | 验证现有 keystore 签名出包 + `apksigner verify` 指纹比对 + 归档（禁止重生成换钥） | 安然（发布） | D+1 18:00 | sha256 归档；指纹与 adopted note 一致 |

## 灰度发车门控（风控）

- 发车前置：`https://<域名>/api/user/v1` 通 + 证书链验证通过（非自签）。
- 服务端 /client/latest 注入发布记录（versionCode/URL/sha256/notes/force/灰度状态）；GRAY（白名单+百分比）→ PUBLISHED。
- 旧包强制更新兜底：对 versionCode<新版的旧包返回 force=true + https 下载 URL，公网用户绝不落回内网明文。
- 回滚：服务端回置 GRAY/隐藏，客户端仍指旧版；全量回滚时旧版不再带明文路径。

## 风险登记

- 证书时延：域名解析不生效则无法签；兜底=自有证书（102 已有 TLS 用法可验）或确认备选域名当日可签。
- 旧包明文残留：force 升级必须与 https 发车同日生效，禁止窗口期内旧包直连公网明文。