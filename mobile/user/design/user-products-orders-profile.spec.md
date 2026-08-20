# user-products-orders-profile 实现规格

> 配对设计稿:`user-products-orders-profile.png`(服务/账单/我的 三个 tab 页)。
> 实现位置:`mobile/user/android/app/src/main/java/com/ymm/boss/user/page/`
> (`ProductsPage.kt` / `OrdersPage.kt` / `ProfilePage.kt` + `ProfileMenu.kt`)。
> 视觉 token 以 `designs/UI-SPEC.md` 为准(色值与代码冲突时以 `ui/theme/Color.kt` 为准,如主色 #007AFF);字段以 `docs/contract/` 与 `api/openapi/user/` 为准。
> 底栏 tab 文案以 `docs/user/nav.js` 为准:首页/服务/账单/我的
> (设计稿截图中的"产品/订单"即"服务/账单"两个 tab)。

## 屏 1 · 服务 tab(产品套餐,Route.Products)

| 区块 | 实现要点 | 数据来源 |
|---|---|---|
| 页头 | 白底居中标题"产品套餐",48dp,16sp/W600 —— 共享组件 `TabHeader` | — |
| 搜索栏 | 胶囊形白底描边(#E8EAED),搜索图标 16dp + 占位"搜索产品或服务";本地过滤 name/bandwidth/description | 本地过滤 |
| 分类胶囊 | 宽带(Wifi 图标)/5G(SignalCellularAlt)/增值服务(CardGiftcard);选中实心主色(Palette.primary,代码令牌 #007AFF 为准)白字,未选中白底描边 —— 共享组件 `PillTab` | category 枚举 broadband/fusion/addon(契约固定,不改 key) |
| 产品卡 | 左 56dp 浅蓝圆角图标容器(`IconTile`);名称 15sp/W600;featured=true 显示绿色"热门" Tag;价格行 ¥(13sp)+金额(20sp Bold 主色)+"/月";特性行:bandwidth(Speed 图标)、合约 N 个月(DateRange)、description(CheckCircle),图标 14dp + 12sp 辅助色;底部通栏"立即办理"实心蓝按钮 44dp/圆角 8;整卡可点进详情 | GET /products?category= |
| 增值服务卡 | 标题"增值服务"+ "进入管理 >";每行 36dp 紫色 IconTile(CardGiftcard)+名称/描述+右侧价格 | 同接口 addons 数组 |

注意:契约 Product 无 originalPrice/特性清单字段,设计稿的划线价与"Wi-Fi 6"类卖点
不落库,实现以 bandwidth/contractMonths/description 三行替代,禁止前端编造。

## 屏 2 · 账单 tab(我的订单,Route.Orders)

| 区块 | 实现要点 | 数据来源 |
|---|---|---|
| 页头 | `TabHeader("我的订单")` | — |
| 状态胶囊 | 全部/进行中/已完成/已取消;选中实心蓝胶囊,未选中纯文字(plain 模式);映射 all/in_progress/done/cancelled | GET /orders?status= |
| 订单卡头 | orderNo 14sp/W600 + 右侧 createdAt 12sp(契约列表无此字段,返回空串时不渲染) | OrderSummary |
| 订单卡体 | 48dp 浅蓝 IconTile(Home)+ 产品名 15sp/W600 + 右侧状态文字 13sp(装维中蓝/已完成绿/已预占橙/其他灰);地址行 LocationOn 14dp + 12sp 单行省略 | OrderSummary |
| 进度步骤条 | 进行中订单(PENDING/RESERVED/INSTALLING)显示 4 节点步骤条:提交订单/受理成功/上门安装/完成。12 环节 → 里程碑映射:stage≤1→1,2-7→2,8-11→3,12→4;已完成节点实心蓝+白对勾,当前节点实心蓝+白数字加粗高亮,未到节点白底描边灰数字;连接线 2dp,已过段主色 | OrderSummary.stage(12 环节见 contract/terms.md §1) |
| 预计上门条 | estimateFinish 非空时:浅蓝 #F0F6FF 圆角条,Person 图标 + "预计上门:{estimateFinish}" + 右箭头 | OrderSummary.estimateFinish |
| 已完成尾区 | DONE:绿色 CheckCircle 20dp + "订单已完成" 14sp/W600 + 副文案 + 右箭头;canRate=true 追加"去评价"蓝链 | OrderSummary.status/canRate |
| 列表尾 | 有数据时底部居中"没有更多订单了" 12sp;空态"暂无订单" | — |

## 屏 3 · 我的 tab(Route.Profile)

| 区块 | 实现要点 | 数据来源 |
|---|---|---|
| 渐变头 | 主色渐变(160° 系)通栏,延伸到状态栏;56dp 圆形头像(白 25% 底 + 2dp 白描边,姓氏首字 22sp);姓名 18sp Bold 白;脱敏手机号 12.5sp 白 85%;realName.status=VERIFIED 时白 20% 胶囊"已实名"(GppGood 12dp);右上角 Settings 齿轮(44dp 热区)→ 账号安全 | GET /profile |
| 三入口卡 | 白卡上浮与渐变头重叠 32dp(offset -32dp);三等分:实名信息(GppGood 蓝,副标题 已实名/待补登)→Verify;家庭地址(Home 橙,副标题 N个地址)→Address;我的套餐(CardMembership 紫,副标题套餐名)→MyPlan | GET /profile |
| 我的服务 | 7 行图标菜单:我的订单(List 蓝)/我的账单(ReceiptLong 蓝)/缴费记录(CurrencyYen 橙)/报障记录(Build 紫)/消息中心(Chat 紫,有未读时标题右侧 8dp 红点)/优惠券与活动(LocalOffer 橙)/电子发票(Receipt 蓝);行结构 = 36dp IconTile + 12 + 标题 14sp/W500 + 右箭头 20dp | 各业务路由;未读数来自 GET /messages(read=false 计数) |
| 账号与设置 | 账号安全(Lock 蓝)/通知订阅设置(Notifications 橙)/投诉与建议(Chat 紫)/帮助中心(Help 蓝)/用户协议与隐私(Description 灰) | 各业务路由 |
| 语言卡 | 中文/English/Filipino 三按钮,选中实心蓝;PUT /profile/language,失败保留本地高亮 | ProfileApi.setLanguage |
| 退出登录 | 白卡 + 居中红字 15sp/W500 + Logout 图标 16dp;先调 logout 端点,失败也清本地 token 并 resetTo(Login)。禁止实心通栏红按钮(见 designs/FEEDBACK.md A 区规则) | UserApi.auth.logout |

## 全局约束

- 共享组件:`TabHeader` / `PillTab` / `IconTile` 已入 `ui/Widgets.kt`,其他页面复用勿重造。
- 间距 4dp 网格:页边距 14(项目既有 AppCard 约定)、卡内 16、行高 ≥44dp 热区。
- 所有列表数据直连 192.168.0.102:28080,接口失败保留骨架 + 错误文案,禁止 mock。
- 状态文案以 `docs/contract/terms.md` §3 枚举为准;环节数固定 12,步骤条只做视觉聚合不改环节定义。
