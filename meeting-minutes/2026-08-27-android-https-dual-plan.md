# Android 首发 https 双预案 + 发布侧未闭环 checkpoint

日期：2026-08-27 ｜ 负责人：安然（发布与风控） ｜ 状态：待产品书面确认「首发用户全内网」

## 背景

产品/业务尚未书面确认首发用户是否全内网。https 处置按双预案待命，产品确认一刻即落地。

## 现状复核（2026-08-27 实读文件确认）

- cleartext 收敛已落地：`app/src/main/AndroidManifest.xml` 已引用 `@xml/network_security_config`，`usesCleartextTraffic=true` 已移除；xml 内 `base-config cleartextTrafficPermitted=false` + 白名单（192.168.0.102 / 43.240.223.138 / 10.0.2.2 / 127.0.0.1 / localhost）。
- 客户端 UpdateApi 已就绪：GET /client/latest（query versionCode+deviceId），返回 force/notes/sha256/downloadUrl；deviceId 本地 UUID 持久化参与服务端分桶。

## 预案 A：全内网确认（维持现状）

- release 保持 networkSecurityConfig 白名单内网 IP，https 公网迁移维持「首发后第一技术债」（换白名单即可）。
- 动作：无代码改动；产品书面确认归档到决策 note 即可放行。

## 预案 B：含公网/异地用户（https 提为首发硬门槛）

前置条件清单（两天可行性评估：**可行，风险低**，前提是域名可当天申请/已有）：
1. 域名注册 + DNS 解析指向 102 公网入口 —— D1 上午（域名负责人）
2. TLS 证书（Let's Encrypt 或自有证书）—— D1 上午
3. 后端反代 HTTPS 终止：443 → 内网 28080（nginx/caddy 反代，业务端零改动）—— D1 下午
4. 客户端改动点（D2，约半天）：
   - `app/build.gradle.kts` release 默认 `bossBaseUrl` 由 `http://192.168.0.102:28080` 换为 `https://<域名>/api/user/v1`
   - networkSecurityConfig 白名单移除明文内网域（或按需保留 debug 联调域）
   - 真机回归：登录/下单/实名/更新检查全链路 over https
5. 收口：`scripts/build-install-user-android.sh --release` 出新包 + apksigner 验证 + 归档
风险：证书/域名审批时延、旧包仍指向内网（需强制更新兜底）。

## 发布侧未闭环 checkpoint（UpdateApi / 灰度门控）

- 步骤：服务端 /client/latest 注入发布记录（versionCode/versionName/下载 URL/sha256/notes/force/灰度状态）；102 上以 curl 冒烟验证新旧版本判定。
- 灰度放量：状态机 GRAY（白名单/百分比命中 deviceId）→ PUBLISHED（全量）；灰度期 force=false，全量后按需置 force。
- 回滚：服务端将该 versionCode 回置 GRAY/隐藏，客户端仍指向旧版；不回滚则旧包无法降级（发布侧唯一硬依赖）。
- 待后端确认：/client/latest 是否已有灰度状态字段，无则需加（见上文 GRAY→PUBLISHED）。