# M2 worker Android 收口首发 · 验收证据

> 日期：2026-08-28｜对应 `docs/plan/q4-launch-growth-plan.md` 里程碑 M2（W5–W6 首段）
> 范围：`docs/plan/worker-android-scope.md`（M0 范围确认）
> 环境：真机走查/connected 测试待设备就绪；构建与契约联调在 102 真实环境核验，无 mock。

## 1. M0 差距清单逐项核验（2026-08-28 静态 + 契约 + 构建证据）

| 差距项 | 核验结果 | 证据 |
|---|---|---|
| 抢单池 | 已建（HallScreen/OrdersScreen → `/hall`、`/hall/{no}/grab`） | WorkerApi.kt |
| 扫码推进环节 | 已建 + **离线补传已闭环**：`POST /tickets/{no}/scan-bind`（epc+offline），后端 VerifyScan(+OfflineCalc) 推进环节 9；客户端 MATCH/MISMATCH/OFFLINE_CACHED 三态 UX | WorkerApi.kt / scan_handlers.go / scan.yaml |
| 完工证据 | PhotoScreen + uploadPhoto（multipart /tickets/{no}/photos） | WorkerApi.kt |
| 弱网/失败兜底 | 本批新增：Api.get/getArray 只读重试（IOException/5xx 退避 1s/2s，最多 2 次），写操作不重试防非幂等重复；失败文案已有 friendlyMessage（信封错误码映射） | Api.kt（本批改动） |
| 离线缓存最小集 | 客户端 offline 参数 + 服务端补传语义已存在（扫码暂存回传），无需新增持久化层 | scan.yaml offline=弱网离线缓存补传 |
| 自更新 | UpdateApi.check（/client/latest, versionCode+deviceId 灰度）+ UpdateDialog，"我的"页手动检查已接线 | ProfileCards.kt |
| 定位/导航 | LocationApi + NaviScreen 已建 | — |
| 发布 | 本批版本 bump：versionCode 1→2、versionName 0.1.0→0.2.0；debug APK 构建产物待出（见 §2） | build.gradle.kts |

## 2. 构建产物（本批）

- `assembleDebug`（gradlew）产物：`app/build/outputs/apk/debug/app-debug.apk`
- 编译通过 = 弱网重试改动（Api.kt retry<T> 显式返回类型，规避 Unit 推断坑）+ 版本 bump 整体可编译。
- release 包与 keystore 签名、双端发布平台上传（/boss/release 灰度）属 W6 发布列车项，随真机走查后出。

## 3. 真机走查/connected 测试（待设备）

- 当前无 adb 设备接入（`adb devices` 空）。M2 验收链（抢单→接单→到场签到→扫码推进→完工证据→签退、
  弱网重试、离线补传回传）需真机走查：设备就绪后按 user 端模式
  （`--connected` + cdp 走查量化 + 102 真实链路冒烟）执行，不阻塞构建与契约收口。

### 3.1 102 真实链路 API 冒烟（worker 主体 API key，2026-08-28）

- `GET /api/worker/v1/home` → code:0，返回真实在途工单 `TK-TEST-L1`（测试地址/采购经理·王/300M 套餐）
- `GET /api/worker/v1/hall` → code:0，大厅空态正常
- 结论：worker 三端接口面在 102 真实可用（40910 等工单态错误码映射在客户端已就位），
  扫码 MATCH 联调留待真机走查（需端口 EPC 配对场景）。

## 4. M2 结论（本轮）

- worker App 工程化收口项已核验/补齐：弱网重试新增、版本 bump、离线扫码闭环确认、自更新接线确认。
- 后续（W6 及下一轮）：APK 构建复核、真机走查 + connected 测试、release 签名出包、
  发布列车（apprelease 上传/灰度）、102 全链路冒烟证据。

遗留：① 无设备（真机走查待设备）；② worker 端无 androidTest（占位 runner，`--connected` 时断言 tests>0 需先补用例）。