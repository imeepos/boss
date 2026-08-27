# 会议纪要：mobile/user/android 两周上线开发计划

- 日期：2026-08-27
- 议题：mobile/user/android 两周内上线，核心要求「用户体验 ×3」
- 主持人：会议主持人
- 与会人（5 名）：阿澈（Android 技术负责人）、木子（产品与体验设计）、明远（后端接口负责人）、佳宁（质量与测试）、安然（发布与风控）

## 零、现状盘点（会议前置调研）

- Kotlin + Jetpack Compose 单模块应用，`mobile/user/android/app/src/main/java/com/ymm/boss/user/` 已有 89 个 Kotlin 文件；ui/（Nav/Routes/Theme/Widgets/PullRefresh/AppRoot）、page/（产品、订单、账单、我的、实名、优惠券、消息、发票、投诉、地址、套餐、登录注册等）、api/（UserApi/ProductApi/OrderApi/BillApi/ProfileApi/ServiceApi 等）基本齐全。
- 设计规格 `mobile/user/design/user-products-orders-profile.spec.md`；主色 #007AFF（ui/theme/Color.kt 为准）；间距 4dp 网格、行高 ≥44dp 热区。
- 构建脚本 `scripts/build-install-user-android.sh` 支持 --release/--device/--base-url/--connected（connectedDebugAndroidTest + tests 数断言）。
- 当前 versionCode=1、versionName=0.1.0、minSdk 26、targetSdk 36、compileSdk 36、release isMinifyEnabled=false；release 无 keystore.properties 时回落 debug 签名。
- 后端固定 192.168.0.102:28080（user 前缀 /api/user/v1）；契约 `docs/contract/terms.md`（订单 12 环节 + 状态枚举）、domain-map.md、fields.md；api/openapi/user/ 已有多域 spec。
- 底栏 tab「首页/服务/账单/我的」（docs/user/nav.js）；多语言 中文/English/Filipino；UpdateApi + UpdateDialog 已存在（自更新入口基础）。

## 一、成员发言要点

### 阿澈（技术负责人）
- **结论：2 周可行，但只保主链路；复用全部页面与 API，新做仅限横切兜底。**
- 最高优先差距：弱网/失败可感知的统一兜底 + 防重复提交；Api.kt 需复核超时重试。
- 重依赖延迟加载（Stripe / play-location）保启动速度。
- 构建发布硬缺口：release 默认 http://192.168.0.102 内网 IP 非 https（正式包拒审风险）；versionCode=1 上不了升级；keystore 依赖 keystore.properties 存在性需核验。
- 体验红线：冷启 <3s、无 ANR、弱网 10s 必有反馈、下单支付不丢状态。
- 砍项：多语言/主题切换/细节打磨。

### 木子（产品与体验）
- **结论：体验×3 是三条主线——快（感知速度）、稳（不崩不白屏、失败可重试）、顺（手不离焦点、返回不迷路）。关键路径为纲，其余让路。**
- 关键路径与红线：
  1. 注册登录→实名：网络超时给引导页而非死等；实名失败可区分「未提交/审核中/驳回」，驳回须给修改入口；登录态丢失不闪退、回跳首页。
  2. 选购→下单：产品卡首屏必须骨架屏；分类切换 >300ms 上 loading；「立即办理」即时按压反馈（44dp 热区为底线）。
  3. 支付：金额主色高亮、倒计时明确；回调跳转成功页后 Navigation 返回栈必须重置，禁止退回仍显示「待支付」。
  4. 进度（4 节点步骤条）：**节点状态映射错误是最高危体验事故**——装维中显示「已完成」等同欺诈；stage 严格按 spec 的 12→4 映射，宁灰不错。
  5. 售后（投诉/报障/发票）：流程结束必须终态文案，禁止静默成功。
- 第一印象必修：四 tab 空态文案统一；订单卡时间空串不渲染；渐变头与状态栏叠加不穿帮；语言切换失败保留本地高亮。
- 该砍：动效过渡、下拉刷新炫技、图片懒加载优化。

### 明远（后端接口）
- **结论：接口侧风险可控；「进度可见性」与「联调真码链路」是最大阻塞点，D-7 前冻结契约。**
- 必核清单：字段对齐（fields.md 三列逐页）、状态枚举（PENDING/RESERVED/INSTALLING/DONE/CANCELLED 与 12 环节正交）、信封结构（{code,message,data}，0=成功）、错误码 4xx/5xx 语义区分、超时/幂等重试、分页 page/pageSize 首屏 1+20。
- 关键路径接口：登录（验证码 5 分钟一次性真码）、下单、支付（幂等查询渲付）、订单进度（环节+状态双轨，需与前端定轮询策略）、实名。
- 性能/安全：登录/首页聚合 P95<1.5s；图片压缩限尺寸；401 统一拦截刷新；禁明文 http 传 token；手机号/证件响应脱敏。
- 后端承诺：D-7 契约冻结 + 全量接口冒烟；D-3 102 环境核心链路压测；上线窗口监控告警盯 5xx/超时率。

### 佳宁（质量与测试）
- **结论：关键路径全绿 + 体验手测优先 + P1 零放行门禁；不追全量覆盖、不搞重构。**
- 回归范围：6 条关键路径每日全量（登录注册真码→产品浏览→下单→订单/账单→我的实名/地址/券/套餐）；消息/发票/投诉冒烟。
- 机型取舍：低版本优先——1 台低版本真机 + 1 台新版本 + 1 模拟器角落用例，放弃矩阵。
- 自动化:手测 = 3:7：connectedDebugAndroidTest 守逻辑断言；间距/溢出/返回栈/空态靠 uiautomator dump + screencap+PIL 手测（Compose 语义树抓不全）。
- 门禁（不达标不发）：debug+release 双构建成功；关键路径 connected 全绿；无复现性崩溃/ANR。遗留分级：P0/P1 零放行，P2 留档可发。
- 体验高频缺陷零放行清单：加载失败无重试；返回栈（缺 BackHandler 直接退 App）；空态错位（offset 死间隙/heightIn 不居中）；文字溢出（weight 大字号无 maxLines=1）。
- **否决权：质量负责人独享停发权，发布人无权 override。**

### 安然（发布与风控）
- **结论：两周内商店渠道不现实，走「内网直装 + 企业分发」；签名、cleartext、通知权限发版前必须解决；灰度回滚做最小版本（服务端开关）。**
- 发布渠道：release 默认明文 http 内网 IP → 商店必拒；首发内网直装/企业分发，与体验×3 匹配；UpdateApi+UpdateDialog 可作自建升级通道基础。
- 签名与版本：**红线**——无 keystore 回落 debug 签名，上线前必须独立 release keystore + apksigner verify；versionCode 单调递增表 + versionName 对齐；UpdateApi 缺服务端版本元数据端点与静默/强制/回滚策略，需后端限期补齐。
- 合规（targetSdk 36）：cleartext 必须收敛（关 usesCleartextTraffic 或 networkSecurityConfig 白名单内网 IP）；Manifest 无 POST_NOTIFICATIONS（Android 13+ 运行时权限）须立项并进测试用例；后台定位需确认是否涉及 ACCESS_BACKGROUND_LOCATION。
- 灰度/回滚：versionCode 白名单 + 服务端开关渐进放量，绿灯全量；回滚靠服务端指向旧 APK。
- 上线 checklist：签名验证、版本号对齐、cleartext 收敛、权限真机实测、UpdateApi 全链路、多语言+实名回归、出包记录归档。

## 二、冲突 Battle：上线渠道与 https 处理（阿澈 × 安然）

**争议**：阿澈主张 release 明文 http + 内网 IP 是「正式包拒审风险」，https 为上线硬缺口；安然主张两周内商店不现实，内网直装即可，cleartext 白名单收敛即可，不承诺商店链接。

- **第一轮**：安然软化——承认 https 必修但列为首发后第一优先级迁移项，不阻塞首发；拒审风险对象是商店通道，首发单走内网直装则该场景不存在；主张发布基建（签名/versionCode/回滚通道）排序优先于弱网兜底。阿澈修正——接受内网直装+网络白名单明文；但坚持「首发用户全内网需产品/业务书面确认并落决策 note，若含公网/异地用户则 https 公网域名是硬门槛」；https 列为两周后第一技术债；升级必须走 UpdateApi 自更新通道。
- **第二轮**：安然被说服，认可「发布基建=硬性前置、弱网兜底=体验底线（可拆最小子集）」并行两轨，修正自己"排序在后"的表述；若产品无法确认全内网，按最坏情况把 https 列为硬门槛进首发。阿澈让位发布基建前置（小时级配置 vs 数天改造的伪冲突），守一条底线：**弱网最小保障版（OkHttp 统一超时/失败可重试/表单防重复提交，三处横切点约一天工期）必须随首发上线**。
- **第三轮**：安然完全和解——接受「弱网最小保障版随首发」，确认与其红线（不得以弱网兜底推迟发布基建）相容；唯一保留点：若工期挤压，砍弱网范围、不砍发布基建，落地为出包门禁检查项。

**Battle 结果（三轮，被说服和解，无需主持裁决）**：最终排序锁定为 ① 发布基建（当日）→ ② 弱网最小保障版（横切，首发前）→ ③ 全页面细粒度兜底 + https 迁移（首发后第一技术债）。

## 三、共识

1. **两周可行，但只保关键路径**：全量体验优化不现实，关键路径之外降 P2 放行。
2. **体验×3 落地为三主线**：快（首屏/关键动作感知速度）、稳（不崩不白屏、失败可重试）、顺（返回栈/焦点/终态明确）。
3. **步骤条状态映射（12→4）是最高危单点**：木子列为体验事故红线，佳宁列为最高优先级用例，双方一致。
4. **首发渠道 = 内网直装/企业分发**：不承诺商店；cleartext 白名单收敛；release keystore 独立签发；versionCode 递增表。
5. **https 为两周后第一技术债**（前提：首发用户全内网需书面确认）。
6. **质量门禁不可降**：P0/P1 零放行，质量负责人独享停发权。
7. **契约 D-7 冻结**：以 docs/contract/terms.md 12 环节与 fields.md 为唯一事实源。

## 四、分歧（均已收敛或消解）

| 分歧点 | 立场 | 收敛结果 |
|---|---|---|
| 首发渠道 | 阿澈：https 硬缺口 / 安然：内网直装 | 内网直装首发 + https 列两周后第一债（battle 和解） |
| 两周排序 | 阿澈：弱网兜底第一 / 安然：发布基建优先 | ①发布基建 ②弱网最小保障版 ③全页面兜底+https（battle 和解） |
| 体验优化范围 | 木子：关键路径精修 / 佳宁：其余降 P2 | 一致：关键路径为纲，P2 留档 |
| 进度刷新 | 明远：轮询兜底优先 / 木子：进度映射准确性 | 步骤条映射正确性优先，轮询方案 UI-后端协商定 |

## 五、行动建议（两周倒排，D 日 = 上线日）

### D-7（契约冻结日）
- [ ] 后端：契约冻结 + 全量 user 接口冒烟；产品/业务书面确认「首发用户全内网」并落决策 note（docs/notes/adopted/）。
- [ ] 接口逐页核对 fields.md 三列对齐；信封/错误码/幂等/分页语义确认。
- [ ] 后端补齐 UpdateApi 服务端版本元数据端点（静默/强制/回滚策略）。

### D-5（发布基建）
- [ ] 生成独立 release keystore，apksigner verify 通过；keystore.properties 不入库。
- [ ] versionCode/versionName 规划递增表（首发版号定档）。
- [ ] cleartext 收敛：release 走 networkSecurityConfig 白名单内网 IP；定位使用场景确认（是否需要背景定位权限）。
- [ ] POST_NOTIFICATIONS 立项：Manifest 声明 + 运行时弹窗 + 测试用例。

### D-3（体验兜底与联调压测）
- [ ] 弱网最小保障版：OkHttp 统一超时 / 失败可重试 / 表单防重复提交（三横切点，约一天）。
- [ ] 重依赖延迟加载（Stripe / play-location）保冷启 <3s。
- [ ] 步骤条 12→4 映射逐状态用例（佳宁最高优先级）。
- [ ] 后端 102 压测：登录/下单 QPS 基线，P95<1.5s；监控告警接 5xx/超时。

### D-1（门禁与收尾）
- [ ] debug+release 双构建成功；关键路径 connected 全绿；无复现崩溃/ANR。
- [ ] 6 条关键路径每日全量回归通过；体验高频缺陷（加载失败/返回栈/空态/溢出）零放行。
- [ ] 发版 checklist 签字：签名验证、版本号对齐、cleartext 收敛、权限实测、UpdateApi 全链路、多语言+实名回归、出包记录归档。
- [ ] 灰度开关与服务端回滚通道就绪；versionCode 白名单渐进放量。

### 首发后（第一技术债批次）
- [ ] https 公网域名迁移（客户端 buildConfig 换 URL + 自更新通道，后端证书/反代）。
- [ ] 全页面细粒度兜底（非横切部分）并入排期。

## 六、主持人注

- 依赖方关键动作：产品/业务在 D-7 前书面确认用户群体（决定 https 是否提前为硬门槛）；后端 UpdateApi 端点两周内必须有，否则回滚仅剩人工 adb。
- 全部成员发言均忠于原意，分歧与和解过程已如实记录。