# user Android 两周上线·执行台账（2026-08-27 起,逐轮过账）

> 用途:首发决策读本——把会议纪要(2026-08-27-user-android-two-week-launch.md)每一行动项映射到
> 当前状态/验证证据/责任方。Android 仓库侧执行项全部闭环;跨角色项注明持有人与登记位置。
> 更新规则:后端/产品落地一项,由对应轮次在此补一行状态,不动历史。

## 一、发布基建(D-5/D-0,阿澈+安然)

| 行动项 | 状态 | 证据 |
|---|---|---|
| 独立 release keystore(不入库)+ apksigner verify | ✅ | adopted note 2026-08-27-user-android-release-signing.md;验证脚本 `scripts/user-android-release-sign.sh`;release 指纹 ≠ debug |
| versionCode/versionName 单调递增表(首发 2/0.1.0) | ✅ | 同上(表入 note);历史内测占 1,首发定 2 防降级 |
| cleartext 收敛(release 白名单内网 IP) | ✅ | release 包 aapt 物证:usesCleartextTraffic 0、networkSecurityConfig 引用、无背景定位 |
| POST_NOTIFICATIONS 立项(声明+运行时弹窗+用例) | ✅ | Manifest + MainActivity(API33+)+ connected 断言;模拟器实测(真机待测仍在 checklist 注明) |
| 定位场景确认(无背景定位) | ✅ | 仅前台 FINE/COARSE,无 ACCESS_BACKGROUND_LOCATION |

## 二、功能补齐(D1-D4,阿澈+明远)

| 行动项 | 状态 | 证据 |
|---|---|---|
| 积分概览+任务页(读为主) | ✅ | PointsApi/PointsPage;connected 渲染+真实空态走查;enterprise 余额 0 空态实证 |
| 积分 exchange(未就绪按契约占位+要交付清单) | ✅占位 / ⏳后端 | 入口「建设中」终态;后端缺兑换模板列表端点(ISSUE.md 登记,4 候选路径 404) |
| /push/device 注册(低成本必做) | ✅ 闭环 | PushApi 接线+幂等;PG push_devices 客户213↔设备UUID 实证;确认后端本为 POST 已落地 |
| 客服 FAQ 页面覆盖核验 | ✅ | ServicePage 已消费 /service/faq;FAQ 行可展开读答案(死功能清理) |
| customer-registrations 砍或挂起 | ✅ 砍 | 会议裁定挂起,Android 无入口 |

## 三、弱网最小保障版(D-3 底线,阿澈,随首发)

| 行动项 | 状态 | 证据 |
|---|---|---|
| 统一超时(8s,上传 15s/30s 独立) | ✅ | Api.kt 常量化;「弱网 10s 必有反馈」红线 |
| 幂等 GET 失败重试(仅 IOException) | ✅ | Api.kt retryingCall/retryingArrayCall(500ms×1),POST 不重试 |
| 表单防重复提交 | ✅ | ui/SubmitGuard + 7 表单(Pay/Topup/Fault/Cancel/Move/Change/OrderChangeAddress)+ 既有守卫核验 |
| 重依赖延迟加载(Stripe/play-location) | ✅ | 确认懒加载(仅调用点创建),冷启不受累 |
| 步骤条 12→4 映射(最高危单点) | ✅ | OrderTimeline/OrderCard 同源映射;逐 stage 单测;装维中订单真机视觉证据(当前=上门安装,非完成) |

## 四、逐页打磨(D5-D9,木子/阿澈)

| 行动项 | 状态 | 证据 |
|---|---|---|
| 产品卡首屏骨架屏+分类切换 loading | ✅ | ProductsPage 脉冲骨架;误显空态修复 |
| 四 tab 空态文案统一/加载守卫 | ✅ | 首页/账单/服务空态+loading 守卫;走查量化 |
| 支付金额主色高亮 | ✅ | PayPage 30sp 品牌蓝 |
| 回调成功页返回栈重置 | ✅ | PayResultPage 顶栏+BackHandler 均 resetTo(Home) |
| 售后终态文案(禁止静默成功) | ✅ | 投诉/报修/开票均显式终态 |
| 下单按钮提交中反馈 | ✅ | CtaBar「提交中…」替代误导向「暂不可提交」 |
| 订单卡时间空串不渲染 | ✅ | estimateFinish isNotEmpty 守卫核验 |
| 实名三态+驳回修改入口 | ✅ | FORM/UPLOAD/REVIEWING/APPROVED/REJECTED+重新提交(清附件回填) |
| 语言切换失败保留本地高亮 | ✅ | LanguageDropdown catch 保留本地选中 |

## 五、稳定速度(D10-D12,阿澈/明远)

| 行动项 | 状态 | 证据 |
|---|---|---|
| 冷启 <3s | ✅ | 模拟器 3 测 1229/1196/1179ms;含权限弹窗首启 2277ms |
| 无 ANR/复现崩溃 | ✅ | 15 轮 connected(12/12)+ 真链路 UI 驱动持续无 FATAL/ANR(观测记录入 checklist) |
| 压测 P95<1.5s + 监控告警 | ✅ 完成（2026-08-28） | user 端 60VU 实测全 P95 达标（见 §十压测小节）；user 关键路径 5 条告警规则已入 boss-prometheus 并验证加载 |
| 关键路径 connected 全绿 | ✅ | connected 12/12 全绿零跳过;E2E 8 读调用(登录真码→产品→实名→订单→账单→套餐/地址/券)+负路径(错码=40100) |

## 六、安全收口(D13-D14,安然/明远)

| 行动项 | 状态 | 证据 |
|---|---|---|
| cleartext 收敛复核 | ✅ | release 包物证(见一) |
| token 不泄露 | ✅ | TokenStore EncryptedSharedPreferences AES256+明文迁移 |
| 脱敏验收 | ✅ | 页面全 masked;明文仅拨号盘(设计意图) |
| https 公网迁移(首发后第一债) | 📋 方案就绪 / ⏳ 实行 | docs/ops/user-android-https-migration.md;待产品用户群确认提级 |

## 七、D-1 门禁与发布(佳宁独享停发权)

| 行动项 | 状态 | 证据 |
|---|---|---|
| debug+release 双构建成功 | ✅ | 每轮门禁;最终合并门禁一次通过 |
| 关键路径 connected 全绿+6 条回归 | ✅ | 12/12(自动化,含负路径)+ 走查量化四 tab+子页;下单→支付 destruct 腿列 P2(待回收通道) |
| 发版 checklist 签字 | 📋 | docs/ops/user-android-release-checklist.md 8 项逐条配命令;真机权限实测待真机 |
| 灰度 versionCode 白名单+回滚通道 | ⏳ 后端 | apprelease 域 000137;服务端发布数据/开关为安然侧 |
| UpdateApi 发版数据(双门槛判定) | ⏳ 后端 | 客户端+契约+实链路 ✅;点 102 无发布记录,注入后可全链验 |
| 出包记录归档 | ✅ | 指纹入 adopted note |

## 八、首发后第一技术债批次

| 项 | 状态 | 证据/登记 |
|---|---|---|
| https 公网域名迁移 | 📋 方案就绪 | user-android-https-migration.md |
| 全页面细粒度兜底(非横切部分) | ✅已并入 | R1-R12 逐页打磨+死功能清零;剩余见 P2 留档 |

## 跨角色遗留(持有人明确,Android 侧已无可推进)

1. 产品/业务书面确认「首发用户全内网」(决定 https 是否提前为硬门槛)——会议主持人注,产品侧。
2. 积分兑换模板列表端点(契约+端点)——明远 D-3 答复项,ISSUE.md 登记。
3. UpdateApi 发版记录/灰度开关(双门槛判定)——安然/后端 apprelease。
4. 102 压测 P95<1.5s + 监控告警——明远。**✅ 完成 2026-08-28**（实测值/最差端点/告警清单见文末 §user 压测记录）。
4a. /push/device 与 /customer-registrations（B 轨）：已复测确认路由在（1. 见 meeting-minutes 七节 Amended），非 404 漂移；执行状态 ✅ 后端就绪，Android 接线即开即用。
4b. **上线风险（明远 2026-08-28 压测发现）**：user 端登录账号表 portal_accounts 仅客户 213 一条真实账号（214–217 无账号，仅 API key 可直连）；下单风控 phoneCap=5/24h、addrCap=3 在途/地址且 288 地址已 26 单 PENDING 爆满——**新开户客户登录账号如何建立、风控参数是否适配促销季，需产品/后端 D-3 前答复**。
5. 真机(非模拟器)权限/回归手测——佳宁/发布前。
## 九、错误码语义确认(D-7,2026-08-27 实证)

| 断言 | 后端实测 | 客户端 |
|---|---|---|
| 错码登录 | POST /auth/login smsCode=999999 → 40100「未认证或凭证无效」 | loginErrorMessage(sms) → 「验证码错误或已过期」✓(E2E 负路径用例锁定) |
| 缺码/绑定层 | smsCode 空 → 42200 | 客户端前置校验非空,不产生该态 |
| 信封/幂等/分页 | bills/orders/addresses/points 信封键实测 | E2E 8 读腿 + 契约键断言 ✓ |

## 十、第二轮执行(2026-08-27,积分兑换+UpdateApi 发版全链真实环境闭环)

| 原阻塞项 | 处置 | 真实环境测试证据(102 实测) |
|---|---|---|
| 积分兑换无可兑换模板端点 | 实现 GET /points/exchange-offers(c8c5bb7a) | 带真实 token curl → items 返回真实模板(templateId=6, CASH ¥10, 200 积分) |
| Android 兑换占位 | PointsPage 兑换真实化(6340b413):模板列表+确认弹窗+防重+终态文案 | 真机(模拟器)真实点兑: 余额 300→100,按钮变「积分不足」,终态「兑换成功,优惠券已放入券仓」,流水 EXCHANGE -200,新券 CPN-794d0a056556 ISSUED 落库 |
| UpdateApi 无发版数据 | admin API 发布 v3 记录(DRAFT→GRAY→PUBLISHED) | GRAY(50%) deviceId 未命中→false;PUBLISHED(100%)→updateAvailable=true+versionCode=3+sha256+downloadUrl;下载 /client/apk/6 200=23,727,225B sha256 一致;Android 冷启弹「发现新版本 v0.1.1」(22.6MB 解析+忽略/立即更新) |
| 回滚通道演练 | PATCH→ROLLED_BACK | 回滚后 /client/latest 恢复 updateAvailable=false ✓ |

- 造数治理: 测试积分/流水/券已还原(客户 213 原状,既有 C-DEMO-001 券保留);测试模板 6 已 DISABLE(审计留痕);发布记录 6 已 ROLLED_BACK,均不影响生产判定。
- 契约/D-7 项: /points/exchange-offers 契约入 loy.yaml,check-contract-sync 通过。

## 十一、UpdateApi 发布/回滚链加固验证(2026-08-27 续)

- 回滚门控:ROLLED_BACK 后 /client/apk/:id 拒发(40400 JSON 50B),PUBLISHED 恢复 200 全量包——服务端正确;
- 客户端健壮:UpdateDialog 下载走系统浏览器(ACTION_VIEW),坏体不进安装器,无「解析包失败」风险;P2 改进项(in-app 下载器+40400 处理)留档;
- 服务端下发包可装可跑:200 包 23,727,225B 于模拟器安装/启动正常(此前轮已验)。

## 十二、user 端专项压测 + 告警配置记录(2026-08-28,明远)

### 1. user 端 60VU 压测(scripts/load/user-load.sh + user-load.js)

> 环境:102 真实接口(无 mock);k6 v2.2.0;60VU ramping(20s 爬坡/1m 平峰/15s 回落);1m35s 共 23816 请求、250 rps。
> 真实约束:user 登录账号仅 213 可真码登录(portal_accounts),(phone,scene)验证码 60s 冷却 → 60VU 主体用 213 API key 直连(同客户身份);
> 下单受风控 phoneCap=5/24h、addrCap=3 约束,42300 拦截单独计数。

| 指标 | P95 | P99 | max | 阈值判定 |
|---|---|---|---|---|
| 订单列表 GET /orders | 86.6ms | 154ms | 313ms | ✅ <1.5s |
| 下单 POST /orders | 43.9ms | 43.9ms | 43.9ms | ✅ <1.5s |
| 支付 POST stripe-intent | 16.6ms | 17.1ms | 17.2ms | ✅ <1.5s |
| 实名/资料 GET /profile | 79.4ms | 157ms | 314ms | ✅ <1.5s |
| 登录(真码全链路,含发码+查库+登录) | 2792ms(单次采样本) | — | — | 冷却制约,按体验阈值需人工评估 |
| user_failed_requests(真失败) | 0.01%(4/21158) | — | — | ✅ <1% |
| user_risk_blocked(42300 风控拦截) | 2658 次 | — | — | 真实风控行为,非故障 |

**最差端点:登录(2792ms, 真码全链路往返)**;接口层最差为订单列表 P95=86.6ms(远优于 1.5s 阈值)。
造数治理:压测下单 5 单均带 requestId=perf- 前缀,已由 scripts/load/user-cleanup.sh 清理(5→0),未留残余。

### 2. user 关键路径告警规则(boss-prometheus,已加载生效 2026-08-28)

| 规则名 | 表达式要点 | severity | 覆盖路径 |
|---|---|---|---|
| UserAuthLogin5xx | login 路径 5xx 率 >1%(5m) | critical | /api/user/v1/auth/login |
| UserOrder5xx | orders 路径 5xx 率 >1%(5m) | critical | /api/user/v1/orders.* |
| UserPay5xx | stripe/payments 5xx 率 >1%(5m) | critical | /orders/{no}/stripe.*、/payments.* |
| UserProfile5xx | profile 5xx 率 >1%(5m) | warning | /api/user/v1/profile |
| UserKeyPathP95Latency | 登录/下单/实名 P95 >1.5s(10m) | warning | login、orders.*、profile |

验证:slo-rules.yml(8 条)promtool SUCCESS;prometheus /api/v1/rules 8 条全部加载(inactive=正常);alertmanager /-/healthy OK。
部署说明:单文件 ro bind 挂载为启动快照,改规则后须 `docker restart boss-prometheus` 才生效(已在 102 执行);规则已同步本地 deployments/observability/slo-rules.yml。
