# 用户端/师傅端推送对接调研

> 状态:调研结论,已裁定采用 JPush。日期:2026-08-23。
> 后续:后台配置项设计定稿见 `docs/plan/push-config-design.md`(§2-§6)。
> 背景:两端均为原生 Kotlin/Compose Android(`mobile/user/android`、`mobile/worker/android`,iOS 目录为空占位,H5 未启动),后端 Go 单体 + Kafka 事件底座,`internal/domain/notify` 现仅覆盖 admin 侧通知,移动端无任何推送基建。

## 1. 需求场景与触达要求

| 端 | 场景 | 触达要求 | 时机 |
|:----|:-----|:---------|:-----|
| 师傅端 | 新派单(环节 8 dispatchOrder) | **强**,师傅在野外/锁屏,App 大概率被杀 | 事件驱动,实时 |
| 师傅端 | 订单取消/改约、公告 | 中 | 事件驱动 |
| 用户端 | 订单进度(已派单/装维中/已激活) | 中,用户主动关心时看 | 事件驱动 |
| 用户端 | 账单出账、缴费提醒 | 中 | 周期性 |

核心约束:国内 Android 无统一推送,App 进程被杀后只有厂商系统级通道能送达。师傅端"新派单"是收入来源,漏推直接丢工单,**必须走厂商通道**。

## 2. 渠道选型(裁定:极光 JPush 聚合)

| 方案 | 优点 | 缺点 | 结论 |
|:-----|:-----|:-----|:-----|
| 极光 JPush 聚合 | 一套 SDK/REST 覆盖华为/小米/OPPO/vivo/荣耀/魅族厂商通道 + 极光自有通道;官方 Go SDK([jpush-api-go-client](https://github.com/jpush/jpush-api-go-client));iOS(APNs)同一 SDK,后续上 iOS 零额外选型 | 免费版有量级/功能限制;依赖第三方 SaaS | **采用** |
| 个推 | 同类聚合,到达率口碑好 | 无官方 Go SDK,需裸调 REST | 备选 |
| 友盟 U-Push | 免费、带统计 | 推送能力弱于前两者,Go 支持差 | 不采用 |
| 自接 5 家厂商通道 | 无第三方依赖 | 每家独立申请+鉴权+协议;OPPO/vivo/荣耀要求应用上架市场+软著,集成与维护成本 ×5 | 不采用 |
| FCM | Google 官方 | 国内不可达 | 不适用 |
| 仅自建 WebSocket | 无厂商依赖 | App 被杀即失效,师傅端场景不达标 | 仅作在线通道补充 |

选 JPush 的决定性理由:后端是 Go 且官方维护 Go SDK;师傅端/用户端两端 Android 共用同一集成;未来 iOS 直接复用。

## 3. 厂商通道前置条件(不可回避,尽早启动)

聚合 SDK 不豁免各厂商开发者资质,且**审核周期长(1-2 周)**:

1. 各厂商开发者账号(华为 HMS/小米/OPPO/vivo/荣耀),企业资质优于个人。
2. 厂商通道参数(AppID/AppKey/AppSecret 等)申请并配到 JPush 控制台,参见[极光厂商通道参数指南](https://docs.jiguang.cn/jpush/client/Android/android_3rd_param)。
3. OPPO/vivo/荣耀普遍要求应用**上架应用市场**,需提前准备软著/备案。
4. 注意[厂商消息分类与配额](https://docs.jiguang.cn/jpush/practice/vendor_classification_quota):部分厂商对即时通讯/订单类消息需自分类权益申请,否则限量送达。

## 4. 对接架构(三层:在线长连接 / 厂商离线 / 短信兜底)

```
订单环节事件(order workflow)
  → Kafka(复用 internal/pkg/events)
  → push consumer(notify 域扩展)
      ├─ App 在前台:  WebSocket/SSE 实时下发(二期;一期沿用现有轮询)
      ├─ App 在后台:  JPush REST(Go SDK)→ 厂商通道 / 极光通道
      └─ 师傅派单兜底: 短信(复用既有 aliyun 短信通道)
```

### 服务端(BOSS Go)

- `internal/domain/notify` 扩展 push 子域:
  - 设备注册表 `push_devices(id, subject_type user|worker, subject_id, registration_id, vendor, last_active_at)`,App 启动/登录时上报 RegistrationID。
  - 发送服务:按 subject 定向(JPush alias 设为 `user:{id}` / `worker:{id}`),失败降级短信(仅派单)。
  - 留痕:每次推送写 `push_records`,对齐"留痕为权威、推送尽力而为"。
- 配置:JPush masterSecret 走既有 secret file mount 方案(见 adopted 2026-08-21)。

### 客户端(Android,两端同构)

1. 集成 JPush Android SDK(CocoaPods 无关,Gradle 依赖),初始化后取 RegistrationID 上报绑定接口。
2. 通知点击 deep link → 订单详情页(用户端 `/order/{id}`,师傅端工单详情)。
3. 前台时收到推送走 in-app 提示,不打系统通知。

## 5. 分期建议

- 一期:厂商通道资质申请(立即启动,周期最长)+ JPush SDK 集成 + 服务端定向推送 + 派单短信兜底。
- 二期:WebSocket/SSE 在线通道(与 admin notify 设计中预留的实时推送合并评估)、推送记录 admin 页面、iOS APNs。

## 6. 开放问题

- 是否上架应用市场?不上架则 OPPO/vivo/荣耀通道不可用,师傅端只能依赖华为/小米/极光自有通道 + 短信,需评估师傅机型分布。
- JPush 免费版配额是否够用(师傅数 × 派单量 + 用户订单量)。
