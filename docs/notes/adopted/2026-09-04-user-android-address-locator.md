# 用户端 App 地址簿：放弃三级联动行政区划，改 GPS + 历史小区

> 日期 2026-09-04｜状态 已采纳｜作用域 mobile/user/android

## 决策

用户端 App 新增/编辑地址入口提供两件事：

1. **"使用当前位置"按钮**：调用 `play-services-location` 的 `FusedLocationProviderClient` 拿 WGS84 经纬度，把 `GPS: N.NNNN°N, E.EEEE°E` 预填到门牌号输入框，用户可改写。
2. **小区输入改为历史下拉**：下拉源为本机地址簿里去重 `community`，不依赖网络/SDK/Key。

**未做**：省/市/区/街道三级选择器、按 GPS 反向地理编码出省市区。

## why

后端 `/api/user/v1` 契约（`api/openapi/user/schemas.yaml:386` 与 `misc.yaml:159`）中 `AddressInfo` 字段固定 7 个：`addressId / label / isDefault / contact / phoneMasked / community / building / door`，**无**省市区字段，**无** `/regions` `/areas` `/geocode` `/reverse-geocode` 端点。admin 端 `/api/admin/v1/geo/countries` 是国际国家维度，不覆盖中国行政区划树。`regions` LTREE 与 `legal_entities` 属于组织域内部数据，未对外开放。

按用户原需求做"GPS 拿到位置 → 自动回填行政区划 → 选择器列出小区"必须依赖任一：

- **A. 第三方地图 SDK**（高德/百度/腾讯）—— 引入 ≥ 3MB SDK、申请 Key、维护隐私合规，且 SDK 不在 `gradle/libs.versions.toml` 内、未走供应商纪律评审（dsh-codebase-wisdom 09-vendoring 原则），单地址簿价值不抵引入成本。
- **B. 后端新建 `/regions` `/reverse-geocode` 端点** —— 涉及服务端的行政区划源（民政部数据/聚合服务商）选型，超出前端任务范围，且后端应先与"地址结构是否要支持行政区划"这一契约变更联动，**不能前端单方面模拟**。

## 放弃了什么（被否决项）

| 选项 | 否决理由 |
|---|---|
| 引入高德/百度地图 SDK 做三级联动 + 逆地理 | 引入重量级依赖、需 Key 申请与隐私合规、未走 vendoring 评审 |
| mock 行政区划数据 + GPS 假坐标 | 违反 AGENTS.md "禁止 mock、对接 102 真实环境"硬约束 |
| 等后端补 `/regions` 端点再做完整版 | 本任务单值不足阻塞；保留扩展点：后端契约补齐后只需把 `door` 字段从经纬度改为 `regionCode` 即可 |
| 只用 GPS 不做历史小区下拉 | 历史小区下拉是"零网络成本"就能减少手填的事，性价比最高 |

## 关联

- 契约：`api/openapi/user/schemas.yaml` AddressInfo、`api/openapi/user/misc.yaml` `/addresses`
- 实现：`mobile/user/android/app/src/main/java/com/ymm/boss/user/api/LocationProvider.kt`
- 改动文件：
  - `app/src/main/AndroidManifest.xml` 加 ACCESS_FINE/COARSE_LOCATION
  - `gradle/libs.versions.toml` + `app/build.gradle.kts` 加 `play-services-location` 与 `kotlinx-coroutines-play-services`
  - `app/src/main/java/com/ymm/boss/user/page/AddressEditorSheet.kt` 弹层顶部加定位卡 + 小区下拉
  - `app/src/main/java/com/ymm/boss/user/page/AddressPage.kt` 把已有地址 community 注入弹层
- 测试：`app/src/test/.../LocationProviderFormatTest.kt`（format 纯函数 JVM 单测）