# worker Android 首发 · 范围与排期确认（M0 交付物）

> 对应 `docs/plan/q4-launch-growth-plan.md` 方向二 2.1 / 里程碑 M2（W5–W6）。
> 本文件是 M0 的"范围与排期确认"结论，不是开发计划本身；M2 开工细化为 W5/W6 任务清单。
> 底线：真机 + 102 真实链路验证，禁 mock；发布走发布列车（`docs/ops/release-train.md`）。

## 1. 现状盘点（2026-09 核实）

- 工程：`mobile/worker/android`，Kotlin + Jetpack Compose，与 user Android 同构。
- UI 已有 50+ 屏：抢单大厅（Hall/Orders）、工单详情（TicketDetail）、扫码（Scan/ScanAbnormal）、
  签到/签退（Checkin/Sign）、完工拍照（Photo）、导航（Navi）、自助维修/修理/拆机/换机/转单/
  领料/收费/报修/投诉/评价/改期/排班、绩效、消息、公告、设置、登录注册、协议、引导（Onboard）。
- API 层：Api（BuildConfig.BOSS_BASE_URL 注入，debug 10.0.2.2:28080 / release HTTPS 生产域）、
  WorkerApi、LocationApi、UpdateApi（自更新，UpdateDialog 已存在）。
- 版本：versionCode=1 / versionName=0.1.0（首发前必须 bump，否则上不了升级）。
- 后端侧已具备：抢单池（worker 域）、扫码 MATCH 推进环节 9、完工/现场动作、绩效、
  消息公告推送（push/device 注册 B 轨闭环，参考 user 端）。

## 2. 差距分析（对照 M2 验收）

| 验收项 | 现状 | 差距 | 处置 |
|---|---|---|---|
| 抢单池 + 抢单 | HallScreen/OrdersScreen 已建 | 未真机+真实数据验证 | M2 真机走查 |
| 扫码推进环节 | ScanScreen + 后端 MATCH 已建 | 扫码-推进-反馈闭环未联调 | M2 联调 + E2E |
| 完工证据（拍照/签名） | PhotoScreen 已建 | 附件上传链路未核 | M2 核验（复用 attachment 软删域） |
| 弱网/失败可感知兜底 | 无统一兜底层 | **缺口** | M2 主项，照抄 user 端方案（超时重试 + 失败态文案） |
| 离线缓存（无网现场） | 无本地持久化（Room/DataStore 未见） | **最大工程缺口** | M2 做最小离线集：当天工单列表 + 扫码记录本地暂存、联网回传 |
| 自更新 | UpdateApi + UpdateDialog 已存在 | 双端发布平台（apprelease）已建 | M2 接线双端 APK 上传/灰度 |
| 定位/导航 | LocationApi/NaviScreen 已建 | 权限与隐私合规核验 | M2 顺带 |
| 发布 | 无 worker 版构建脚本 | **缺口** | 复用 `scripts/build-install-user-android.sh` 模式出 worker 版 |

## 3. 复用地图（不早轮子）

- 构建/发布：user 版 `build-install-user-android.sh` → worker 版；签名/keystore 流程照
  `docs/notes/adopted/2026-08-27-user-android-release-signing.md`；HTTPS 合规照
  `docs/ops/user-android-https-migration.md`。
- 版本发布：`apprelease` 域（`/boss/release`，双端 APK 上传/灰度/回滚，客户端匿名查
  `/api/worker/v1/client/latest`）— worker 端直接复用，不另建下载通道。
- 验收模式：user 端"真机 + connected 测试 + cdp 走查量化 + 真实链路冒烟"整套复用
  （见 `docs/ops/user-android-launch-status.md`）。

## 4. 排期（M2 = W5–W6，2026-09 后两周）

| 周 | 任务 | 产出 |
|---|---|---|
| W5 | 弱网/失败兜底层（复用 user 方案）；离线最小集（稿工单+扫码暂存回传）；扫码-推进-反馈联调 | 三个主差距闭环代码 |
| W6 | 自更新接线 applerelease 双端；真机走查（抢单→到场→扫码→完工证据→签退）；E2E + 102 冒烟；版本 bump 0.2.0；出包发布列车 | 首发候选包 + 验收证据 |

风险：离线集若 W5 做不完则降级为"弱网重试 + 失败留痕"先行（现场先保"不丢活"），
离线回传 W7 补；不阻塞首发。

## 5. 验收口径（放量视角）

- 师傅端 24h 在线率、到场-完工时长中位数可度量（计划 §4）。
- 首发候选包：双端发布平台可灰度、可回滚；升级链路（0.1.0 → 0.2.0）实测通过。
- 真实 102 装维主链路：接单 → 到场签到 → 扫码推进 → 完工证据 → 签退，全程留痕可回放。

## 6. 需要外部输入的项

- 真机 2 台（低版本 + 新版本）用于走查与 E2E（照 user 端机型取舍：放弃机型矩阵）。
- 装维现场弱网/无网名录：决定离线集范围（仅扫码暂存 vs 全列表离线）。
- 首发窗口确认：与 M2 里程碑评审（W5 末）对齐。