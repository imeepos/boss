# 消息中心页 实现规格（对应 messages-center-v1.png）

## 给前端的实现提示词（可直接复制）

实现一个 Android Compose 的消息中心页面（`Route.Messages`），非 pinned 骨架页，
使用 `TopBar` + `Column(verticalScroll)` 结构。复用 `ui/Widgets.kt` 已有组件。

### 整体布局（从上到下）

1. **TopBar**："消息中心"，左返回箭头 `nav.pop()`，右"订阅设置"→ `Route.Notify`
2. **分类筛选**：`PillTab(plain=true)` 横排，5 个分类（全部/账单缴费/余额预警/故障公告/优惠活动），
   滚动区水平 padding 14dp，vertical 8dp，spacedBy 8dp。选中态=主色实底白字。
3. **消息统计摘要卡**：`AppCard`，水平排列：
   - 左侧：消息总数 20sp/Bold + "条消息" 12sp muted；如有未读追加 12sp/Bold 主色 "X 条未读"
   - 右侧：仅未读 > 0 时显示"全部已读"文字按钮（13sp 主色，48dp 点击热区）
4. **消息列表**：每条消息为独立 `AppCard`，卡片间距 vertical 6dp：
   - **未读**消息：`Card` 额外加 `border(1.5.dp, Palette.primary.copy(alpha=0.3f), 12.dp)` 左侧蓝色竖线或描边
   - **已读**消息：普通白卡无描边
   - 卡内结构：
     - 顶行：左侧 `IconTile` 40dp 圆角 12dp（按 category 取语义色）+ 标题 14sp/W600（未读）或 14sp/W500（已读）
       + 时间戳 12sp muted 右对齐
     - 内容行：`content` 12sp muted（未读）或 12sp subtle（已读），maxLines=2，ellipsize
     - 底行左侧：`Tag` 组件（tag 文字 + tagLevel 色）
5. **空态**：`EmptyState("暂无消息", Icons.Outlined.Notifications)` — 复用 Widgets.kt
6. **加载态**：`CircularProgressIndicator` 居中，保留骨架布局
7. **错误态**：14sp 红文字 + "点击重试" 蓝链接 48dp 热区

### 色彩规范

- 页面背景：`Palette.bg` #F5F6F8
- 卡片底：`Palette.panel` #FFFFFF
- 未读标题：`Palette.ink` #1C1C1E
- 已读标题：`Palette.muted` #3F3F46
- 分类图标色：按 category — billing=蓝 primary, balance=橙 orange, fault=紫 purple, promo=红 err, 默认=灰 muted
- Tag 色：按 tagLevel — balance/bill=橙 orange, promo=红 err, info=蓝 primary, 默认=灰 muted
- 未读描边：`Palette.primary.copy(alpha=0.3f)`
- 统计摘要：总数 ink, 未读数 primary

### 字体层级

| 元素 | 字号 | 字重 | 颜色 |
|---|---|---|---|
| 摘要总数 | 20sp | Bold | Palette.ink |
| 未读数 | 12sp | Bold | Palette.primary |
| 消息标题(未读) | 14sp | W600 | Palette.ink |
| 消息标题(已读) | 14sp | W500 | Palette.muted |
| 消息内容 | 12sp | Normal | Palette.muted(未读)/Palette.subtle(已读) |
| 时间戳 | 12sp | Normal | Palette.muted |
| Tag | 11sp | Normal | 语义色 |

### 交互状态

- 分类切换：触发 `LaunchedEffect(category)` 重新加载
- "全部已读"：`ProfileApi.readAllMessages()` + `nav.refreshTick` 刷新
- 消息点击：`nav.push(routeOf(category))` 跳转对应详情
- 下拉刷新：复用 `PageRefresh` 机制

### 数据接口

- `GET /messages?category=billing|balance|fault|promo` → `{ items: Message[] }`
- `POST /messages/read-all` → Ok
- Message 字段：`messageId`, `category`, `title`, `content`, `tag`, `tagLevel`, `createdAt`, `read`
- **注意**：无单条已读接口，只有批量已读

## 注意事项

- 本页非 pinned 骨架页（不用 `PinnedGradientPage`），走 `TopBar` + scroll 结构
- 页边距：非 pinned 页用 `AppCard` 默认 `outer = PaddingValues(horizontal=14.dp, vertical=6.dp)`
- 分类图标只在设计稿中使用（语义区分），Compose 实现可省略 IconTile 仅保留 Tag 色彩区分
- 时间戳格式由后端 `createdAt` 提供，前端仅展示不做格式化（保持原始字符串或按需 toLocalDate）
- 消息总数和未读数从 `items.size` 和 `items.count { !it.optBoolean("read") }` 计算
- 破坏性操作（如果有）遵循白底红字规范，本页无破坏性操作
- 设计稿中文字为 AI 生成占位，真实文案以后端数据为准
