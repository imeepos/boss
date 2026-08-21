# 订单详情页 实现规格(对应 order-detail-v1.png)

## 给前端的实现提示词(可直接复制)

实现一个 Android (Jetpack Compose) 移动端"订单详情页",高保真还原附图设计稿。
基于既有用户端架构:路由 `Route.Order(no)`,API `OrderApi.detail(no)` / `OrderApi.urge(no)`
/ `OrderApi.cancel(no)` / `OrderApi.changeAddress(no, addressId)` / `OrderApi.technicianContact(no)`。

### 整体布局

- 顶部 `TopBar("订单详情", onBack = { nav.pop() }, action = "取消", onAction = ...)` —— 48dp,
  背景 `statusBarSolid() #006AE5`(浅色主题)/ `#4E97EC`(深色),白字,左返回箭头(纯字符 ‹) + 居中 16sp/W600 标题 + 右 13sp 白字"取消"(仅 status ∈ {PENDING,RESERVED,INSTALLING} 时显示,DONE/CANCELLED 时不显示)
- 主体 `Column(verticalScroll(rememberScrollState()))`,非 pinned 骨架页 → 卡外边距走 `AppCard.outer` 14dp
- 区块顺序:状态头 → 订单信息卡 → 进度里程碑条 → 预计上门条 → 12 环节时间线 → 底部动作栏
- 页面可垂直滚动,设计稿画了关键 3-4 屏内容;**禁止**尝试把所有区块塞进一屏

### 色彩(代码真值,USER-APP-SPEC §2)

- 页面背景 `Palette.bg #F5F6F8`
- 卡片底色 `Palette.panel` 浅=`#FFFFFF` / 深=`#1C1C1E`
- 主蓝 `Palette.primary #007AFF`(按钮/链接/进行中节点/价格)
- 状态绿 `Palette.success #34C759`(已完成/激活节点)
- 状态橙 `Palette.warn #FF9500`(RESERVED 待预占)
- 文字主色 `Palette.ink #1C1C1E`(浅)/ `#F2F2F7`(深)
- 辅助文字 `Palette.muted #3F3F46`(浅)/ `#A1A1AA`(深)
- 弱图标 `Palette.subtle #B0B3B8`
- 描边 `Palette.line #E8EAED`
- 错误 `Palette.err #FF3B30`
- 预计上门条背景 `Palette.primary.copy(alpha=0.08f)` + 前景 `Palette.primary`
- TopBar 背景 `statusBarSolid()`(同首页/我的固定色)

### 字体层级(USER-APP-SPEC §3)

- TopBar 标题:16sp/W600/白
- 状态头主标题(状态中文,如"装维中"):18sp/Bold/Palette.ink
- 状态头副标题(描述):12.5sp/Normal/Palette.muted
- 卡内标题(CardTitle):15sp/W600/Palette.ink
- 信息行左标签:13sp/Normal/Palette.muted
- 信息行右值:13sp/Normal/Palette.ink
- 进度节点下文字:10.5sp/Normal;当前节点文字 W600/Palette.primary;已完成 W500/Palette.ink
- 时间线条目标题:13.5sp/Normal/Palette.ink(PENDING 时为 Palette.muted,DOING 时 Bold)
- 时间线条目时间戳:12sp/Normal/Palette.muted(右对齐)
- 底部主按钮文字:15sp/W600/白
- 底部描边按钮文字:14sp/W500/Palette.primary

### 数据区块(从上到下,共 5 个)

#### 区块 1: 状态头卡(`StatusHeaderCard`)

- `AppCard { ... }`,12dp 圆角,白底
- 左 48dp `IconTile(Icons.Filled.CheckCircle, Palette.success, corner = 12.dp)`(DONE 时)/ `(Icons.Filled.Build, Palette.primary, ...)`(INSTALLING 时)/ `(Palette.muted, ...)`(其他)
- 中间列:
  - 顶部 `Text(statusLabel, 18sp, Bold, Palette.ink)` —— `装维中` / `已完成` / `待核查` / `已预占` / `已取消`
  - 底部 `Text(statusDesc, 12.5sp, Normal, Palette.muted)` —— 简短描述,如"工程师预计今日 15:30 前到达"
- 右列:订单号(`orderNo` 16sp/Bold/Palette.ink)+ 复制图标 `Icons.Outlined.ContentCopy` 14dp/Palette.subtle
- 卡内上下 Padding 14dp

#### 区块 2: 订单信息卡(`InfoCard`)

- `AppCard { CardTitle("订单信息"); Spacer; InfoRow ... × 4 }`
- 行结构(左右两列对齐):
  - `产品名称`: `productName`(空时显示 "—")
  - `安装地址`: `address`(可点击调起高德/百度地图,本版先不实现跳转,只 `LocationOn` 14dp/Palette.subtle 图标)
  - `下单时间`: `submitedAt`(ISO 8601,渲染 `yyyy-MM-dd HH:mm`)
  - `装维师傅`: `technicianName + technicianPhoneMasked`(空时显示 "尚未分配")
- 行高 ≥44dp,左标签 13sp/Palette.muted,右值 13sp/Palette.ink
- "装维师傅" 行右侧追加"呼叫"图标 `Icons.Filled.Call` 16dp/Palette.primary(可点击拉起 `ACTION_DIAL`)

#### 区块 3: 进度里程碑条(`MilestoneStepper`)

- **复用 `OrderCard.kt::OrderStepper`(已存在)**,不要重写
- 4 节点文案全 App 固定:"提交订单 · 受理成功 · 上门安装 · 完成"
- 当前 milestone 计算:`milestoneOf(stage)` = stage≤1→1 / 2-7→2 / 8-11→3 / 12→4
- 节点下方追加完成时间(已完成节点)/ "进行中"(当前)/ "待完成"(未到),10.5sp
- 高度 52dp,左右 16dp 内边距让节点居中

#### 区块 4: 预计上门条(`EstimateBanner`,条件渲染)

- 仅当 `estimateFinish` 非空且 status ∈ {INSTALLING} 时显示
- `AppCard(outer = PaddingValues(horizontal = 14.dp, vertical = 6.dp)) { Row(...) }`
- 浅蓝条:`background(Palette.primary.copy(alpha = 0.08f), RoundedCornerShape(8.dp))`,padding 12dp
- 左 `Icons.Filled.Schedule` 16dp/Palette.primary
- 中 "预计上门:{estimateFinish}" 12.5sp/Palette.ink,weight=1f
- 右 `Icons.AutoMirrored.Filled.KeyboardArrowRight` 18dp/Palette.subtle
- 整行 44dp 点击热区(可后续跳改预约时间子页,本版只渲染不跳转)

#### 区块 5: 12 环节时间线(`TimelineCard`)

- `AppCard { CardTitle("装维进度(12 环节)", "已完成 {n}/12") }`
- 内部纵向列表:`forEachIndexed { i, t -> TimelineItem(t, isLast = (i == last)) }`
- 每条目结构:
  - 左列(14dp 宽):8dp 圆形节点 + 2dp 垂直连接线(非末项)
    - 节点颜色:`result == "DONE" → Palette.success` / `"DOING" → Palette.primary` / 其他 → `Palette.line` 描边
  - 右列:阶段号 + 标题(13.5sp,DOING 时 Bold)+ 元信息(12sp/muted,空时省略)
- **不显示**已完成节点的图标,只靠颜色区分(对比设计稿加了对勾图标,本规范去繁从简,与 USER-APP-SPEC §6.3 一致)
- 时间线加载中显示 `Text("进度加载中…", 12.5sp/muted)`

#### 区块 6: 底部动作栏(`ActionBar`,固定在 CardList 末尾,不浮层)

- `Row(Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 12.dp), Arrangement.spacedBy(8.dp))`
- status ∈ {PENDING, RESERVED, INSTALLING}:
  - `Button(主)` "催单" 重量 1f,实心 `Palette.primary` 底,15sp/W600/白
  - `OutlinedButton(描边)` "联系师傅" 重量 1f,14sp/W500/Palette.primary 文字
  - `OutlinedButton(描边)` "变更地址" 重量 1f,14sp/W500/Palette.primary 文字
  - 按压态:主按钮 `alpha=0.8f`;描边按钮 `Palette.primary.copy(alpha=0.08f)` 底
- status ∈ {DONE}:
  - 单一 `OutlinedButton` "去评价"(仅 `canRate=true` 时显示)+ "联系师傅" 同上
  - canRate 默认 false,后端字段缺失时显示 `Text("订单已完成,感谢您的选择!", 14sp/W600/Palette.success)`
- status ∈ {CANCELLED}:
  - 单一 `OutlinedButton` "联系客服"
  - 顶部 "下单失败/已取消" `Text(14sp/Palette.muted)`

### 交互状态(USER-APP-SPEC §9 通用规则)

- **加载**:`CircularProgressIndicator(strokeWidth=2.dp, size=24.dp)` 居中,保留骨架(垂直滚动容器已在,不动布局)
- **错误**:`Notice(err, Palette.err)` 12.5sp 红,接 "点击重试" 蓝链(48dp 热区)
- **空**:此页不会真正"空"(`orderNo` 必传);若详情返回 null 则显示 EmptyState("订单不存在或已删除")
- **下拉刷新**:复用 `nav.refreshTick`,与 OrdersPage 一致

### 取消按钮(顶部 action)

- 点击 → 弹出 `AlertDialog("确认取消该订单?", "取消后将释放预占端口并关闭订单。")`
- 确认:调 `OrderApi.cancel(no)` → 成功 `Toast` + `nav.pop()`,失败红字 `err` 留在卡顶部
- 取消:关闭弹窗

### 数据字段建议(API 返回)

> 来源:`OrderApi.detail(no)` 返回 `JSONObject`,已含 `order` / `timeline` 两个子对象。
> 字段为空时一律渲染"—",禁止 NPE 崩溃。

| 字段 | 路径 | 类型 | 空值处理 |
|---|---|---|---|
| 订单号 | `order.orderNo` | string | 必填 |
| 状态码 | `order.status` | enum | PENDING/RESERVED/INSTALLING/DONE/CANCELLED |
| 状态中文 | `order.statusLabel` | string | 空时按 status 查表兜底 |
| 状态描述 | `order.statusDesc` | string | 空时省略此行 |
| 产品名 | `order.productName` | string | "—" |
| 安装地址 | `order.address` | string | "—" |
| 下单时间 | `order.submitedAt` | ISO8601 | "—" |
| 装维师傅 | `order.technicianName` + `order.technicianPhoneMasked` | string | "尚未分配" |
| 当前环节 | `order.stage` | int 1-12 | 默认 1 |
| 预计上门 | `order.estimateFinish` | string | 仅 INSTALLING 渲染 |
| 可评价 | `order.canRate` | bool | DONE 时默认 false |
| 12 环节时间线 | `timeline[]` | array | 空数组显示"进度加载中…" |

## 注意事项

- **设计稿的图标不可照抄**:gpt-image-2 生成的图标(工人头像/卡车)是示意,实现用 Material Icons 库已有图标
  (`CheckCircle` / `Build` / `Schedule` / `LocationOn` / `Call` / `ContentCopy` / `KeyboardArrowRight`)
- **设计稿的订单号 `2025052512345678` 是占位**:真实订单号后端返回
- **里程碑条必须复用 `OrderCard.kt::OrderStepper`**,禁止重写;时间线条目是新增
- **顶部 TopBar 用 `Widgets.kt::TopBar`**,自带 `statusBarSolid()` 底色,不要改成 `primary` 渐变(状态栏接缝)
- **破坏性操作(取消)走白底弹窗 + 红字确认**,不要做实心通栏红按钮(FEEDBACK.md A 区红线)
- **设计师误把"工程师已接单"放在环节 3**:实际接单在环节 8(派单后),以契约 `terms.md` 为准
- **设计师加了"修改预约时间"按钮**:本版不实现子页,只渲染按钮(可后续扩展 Route.OrderReschedule)
- **设计稿时间线节点包含图标**(`✓`):本规范去掉,只靠颜色+数字+文字三线索区分,降低视觉噪声
- **本规范明确不实现的字段**:`originalPrice`/`features`(USER-APP-SPEC §11);若后端返回这些字段,只展示 fallback
- **依赖既有组件**:`AppCard`/`CardTitle`/`Tag`/`TopBar`/`PillTab`/`IconTile`/`EmptyState`/`Notice`
  (`Widgets.kt`),`OrderStepper`(`OrderCard.kt`),禁止重造