# 师傅端 · UI 设计规范

> 单一事实源：`docs/worker/style.css`（H5 草稿） + `mobile/worker/android/app/src/main/java/com/ymm/boss/worker/ui/theme/Color.kt`（Compose 主题） + `Widgets.kt`（共用组件）。两端口径完全对齐，本文件为后续设计稿生成与前端实现的事实参考。
>
> 与代码冲突时以代码为准，并回改本文件。

---

## 1. 平台与画布

| 项 | 值 |
|---|---|
| 目标平台 | Android 装维师傅端 App + H5 草稿同源（同套 token） |
| 主屏画布 | **375 × 762dp**（iOS 设计稿常用尺寸，与 H5 `max-width:420px` 的视觉重心一致） |
| 业务画布 | `.app` `max-width:420px` 居中，两侧阴影模拟"手机壳"，底部留 70dp 给 TabBar |
| 设计稿尺寸 | 单屏 **`1536×1024` 横版**（仪表盘/工作台等多区块），表单详情可用 `1024×1536` 竖版 |
| 主语言 | 中文（兼英语 / Filipino 切换见 `nav.js` 的 `toggleLang`，仅示意未串接文案包） |
| 字体 | PingFang SC / Microsoft YaHei / -apple-system（系统字体优先，不内置） |

---

## 2. 色彩令牌

> **对齐锚点**：与 `designs/UI-SPEC.md`（用户端 user-home）同源，保证师傅端 / 用户端视觉一致。
> 表中 token 改名仅做语义对应（`Primary` ↔ `brand.primary`、`Bg` ↔ `page.background`），**色值完全一致**。

### 2.1 基础色

| Token（语义名） | Hex | 用户端对照 token | 用途 |
|---|---|---|---|
| `Primary` | `#086CF5` | `brand.primary` | 主色 / 选中态 / 主操作（蓝实底满宽按钮） / 价格 / 链接 |
| `Primary2` | `#0872F4` | Hero 渐变起点 | 头部渐变起点（见 §2.4） |
| `Primary3` | `#1698FA` | Hero 渐变终点 | 头部渐变终点 |
| `Bg` | `#F6F8FA` | `page.background` | 页面背景（`.app` 的 `background`） |
| `Panel` | `#FFFFFF` | `card.background` | 卡片 / 底部 TabBar |
| `Line` | `#E7EAF0` | `divider` | 分割线 / 卡内 cell 底分隔 |
| `BrandSoft` | `#F0F6FF` | `brand.soft` | 图标浅底容器 / 强调浅背景 |
| `Ink` | `#171B23` | `text.primary` | 主文字 / 标题 |
| `Muted` | `#5F6671` | `text.secondary` | 副文字 / 说明 |
| `Tertiary` | `#7B828D` | `text.tertiary` | 辅助 / 未选导航 |
| `Placeholder` | `#AEB4BE` | `text.placeholder` | 占位 / 禁用态描边 |
| `Success` | `#0AA847` | `success` 主色 | 成功 / 完成（仅用于状态徽章与文案色） |
| `Warn` | `#F57900` | `warning` 主色 | 警告 / 缴费提醒 |
| `Err` | `#FF2D2F` | `error` 主色 | 错误 / 异常 |
| `Purple` | `#7A42F4` | `purple` | 故障 / 客服图标 |

### 2.2 Tag 色（统一浅底描边样式，对应 H5 `.tag-*` 与 Compose `TagColor`）

> 与用户端 Tag 同套色（`designs/UI-SPEC.md` §2）：`success/info/warning/error` 四组对应，色值直接复用。

| Tag 名 | 前景 | 底色 | 描边 | 业务映射 |
|---|---|---|---|---|
| `tag-success` | `#0AA847` | `#EFFFF4` | `#6FD18B` | `DONE`（已完成） |
| `tag-info` | `#176BF2` | `#F1F6FF` | `#A9C8FF` | `SCAN_PENDING`（待扫码）/ 默认态 |
| `tag-warning` | `#F57900` | `#FFF5E8` | `#FFC270` | `ACCEPTED` / `TODO`（已接/待领取） |
| `tag-error` | `#FF2D2F` | `#FFF0F0` | `#FF9B9D` | `DOING`（进行中 / 紧急） |
| `tag-gray` | `#5F6671` | `#F6F8FA` | `#E7EAF0` | 中性 / 失效 / 默认占位 |

### 2.3 提示块（Notice）色

| 类型 | 前景 | 底色 | 描边 | 触发 |
|---|---|---|---|---|
| 警告 | `#F57900` | `#FFF5E8` | `#FFC270` | 提醒 / 警告（排期冲突 / 待补料） |
| 错误 | `#FF2D2F` | `#FFF0F0` | `#FF9B9D` | 数据加载失败 / 风控拦截 |
| 成功 | `#0AA847` | `#EFFFF4` | `#6FD18B` | 操作成功反馈 |
| 信息 | `#176BF2` | `#F1F6FF` | `#A9C8FF` | 新公告 / 调度消息 |

### 2.4 头部渐变

```
linear-gradient(160deg, #0872F4 0%, #0B82F8 50%, #1698FA 100%)
```

应用到：`.topbar`（次级页顶栏）+ `.home-head`（首页头部，含状态胶囊 + 通知按钮 + 装饰弧线）。
三段渐变与用户端 `user-home.png` 基准完全一致，保证两 App 头部视觉同源。

### 2.5 状态栏色

按用户端约定：**iOS 透明（跟随头部渐变），Android 半透明（保留系统手势区）**。Compose 实现时用 `WindowCompat.setDecorFitsSystemWindows(false)` + `SystemBarStyle.light(...)`。
具体取值：`Primary` 起点 `#0872F4` 的 ARGB 形式供 `SystemBarStyle` 调用，状态栏前景图标置白以保证对比度。

---

## 3. 字体层级

> 与 `designs/UI-SPEC.md` §3 完全对齐：层级 ≤5 档，字重只用 400 / 500 / 600 / 700。

| 层级 | 字号/行高 | 字重 | 颜色 | 用途 |
|---|---|---|---|---|
| Display（Hero 标题） | `20sp / 28` | 700 | `#FFFFFF` | 首页 `.home-head .hi`（"张师傅"） |
| H1（卡片主标题） | `16sp / 24` | 700 | `Ink` `#171B23` | 卡片标题 / 我的页姓名 |
| TopBar | `16sp / 24` | 600 | `#FFFFFF` | `.topbar .t` 次级页顶栏标题 |
| H2（条目主标题） | `15sp / 22` | 600 | `Ink` | 列表行主标题 / 表单字段值 |
| Body M | `14sp / 20` | 400-500 | `Ink` | `.cell .t` 列表行 / 表单输入 |
| Body S | `12sp / 18` | 400 | `Secondary` `#5F6671` | 副文案 / `.cell .d` / 表单 label / 错误重试 |
| Tag | `11sp / 16` | 500 | Tag 前景 | 状态标签 / 胶囊 |
| Nav | `11sp / 16` | 400-500 | `Muted` 未选 / `Primary` 选中 | `#tabbar a` 底部导航文字 |

字重只用 `400 / 500 / 600 / 700` 四档；金额/数字 `tabular-nums` 等宽；最大字重 `700` 不允许大面积使用。
统计卡数字 `20sp / 28 / 700` 仅用于该卡，按业务语义映射色（接单→`Primary`、已完成→`Success`、进行中→`Warn`）。

---

## 4. 圆角与间距

> 与 `designs/UI-SPEC.md` §4 完全对齐：圆角统一 `md=10dp`（卡片），间距 4dp 网格。

### 4.1 圆角 token

| Token | dp | 用途 |
|---|---|---|
| `xs` | 4dp | `.tag` 标签 |
| `sm` | 8dp | `.btn` 次级按钮 / 输入框 |
| `md` | **10dp** | `.card`/`.stats-card`/`.btn-block`（**主卡片统一值**） |
| `lg` | 12dp | 图标容器 / 胶囊 / `.notice` 提示块 |
| `pill` | 999dp | `.dot` 圆点 / Tag 胶囊 |
| `full` | 50% | 头像 / 红点 |

### 4.2 间距（4dp 网格）

| Token | dp | 用途 |
|---|---|---|
| `xs` | 4dp | Tag 内边距、元素间最小间距 |
| `sm` | 8dp | cell 内垂直 padding、`.stats>div` 数字与 label 间距 |
| `md` | 12dp | `.topbar` 元素间距、`.tl-item` 节点半径 |
| `lg` | 16dp | `.card` / `.stats-card` 内 padding、`.btn-block` 内 padding、按钮横纵 padding |
| `xl` | 20dp | 卡片间距 / 节段间距 |
| `xxl` | 24dp | `.home-head` 顶部 padding |

页面左右边距 **`16dp`**（`.card`/`.stats-card` 用 `margin: 12px 16px`，与用户端一致）；
卡片间距 `8dp`；卡内 padding `16dp`（垂直 `12-16dp`）；
TabBar 固定高度 **`~48-52dp + 安全区`**；最小点击热区 `44×44dp`，列表整行热区 `≥56dp`。

### 4.3 悬浮卡手法

首页"今日业绩"卡用负 margin (`margin: -34px 16px 0`) 向上拉，让其顶部覆盖首页头渐变区域底部 **34dp**，形成"白卡悬浮于彩色头部之上"的层级。
与用户端 `user-home.png` 重叠量一致（34dp），保证两 App 首页"前后景层级"同节奏。这是本端唯一的"前后景重叠"处理点，其他页面不允许模仿。

---

## 5. 按钮

> 与 `designs/UI-SPEC.md` §5 动作分级同源：**主操作=蓝实底满宽**，危险=白底红字，跳转=仅 Chevron。
> 不允许所有操作都做成实心按钮。

### 5.1 主按钮（`PrimaryButton`，蓝实底）

```kotlin
PrimaryButton: Primary(#086CF5) 底, 白字 15sp/600, 圆角 10dp, 垂直 padding 12dp
```

适用于师傅端所有"产出性"操作：接单 / 确认签收 / 提交绑定 / 提交上报 / 提交改约 / 确认收款 / 上班打卡 等。
**废除原"绿主按钮"约定**（旧草稿用 `#52C41A` 表达"完成"，与用户端视觉分裂），统一蓝实底。如需强化"完成"语义，用 **Success 状态徽章 + 文案（"已提交 ✓"）** 而非按钮色。

### 5.2 次按钮 / 轮廓按钮

```
.btn-block.ghost: 白底, Primary 蓝边 1dp, Primary 蓝字 15sp/600
.btn: 白底, Line 灰边, Ink 黑字 13sp, padding 8/16, 圆角 8dp
.btn-primary (小): Primary 蓝底, 白字, padding 6/12, 圆角 8dp（用于工单行内"领取"按钮）
```

### 5.3 按钮尺寸档位

| 档位 | 高度 | 圆角 | 用途 |
|---|---|---|---|
| `lg` 主按钮 | 44dp | 10dp | 表单提交、整行确认 |
| `md` 按钮 | 36dp | 8dp | 卡内次操作（"领取"、"刷新"） |
| `sm` 按钮 | 28dp | 6dp | 行内操作（极少用） |

按钮文字必须居中；最小点击热区 `44×44dp`；
破坏性操作（"退出登录"、"撤回工单"、"解除绑定"）用**白底红字描边**（`Err` 前景 + `Err` 浅底），禁止蓝底或绿底。

---

## 6. 卡片与列表

### 6.1 基础卡片（`.card` / Compose `Card`）

- 白底 `Panel`，圆角 **`10dp`**（统一值，与用户端一致）
- 内 padding `16dp`，外 margin `12px 16px`
- 阴影：`0 3dp 12dp rgba(31,48,72,0.10)`（与用户端 `shadow.card` 一致，**新增**）
- 无描边——层次靠"白卡 + 柔阴影 vs 灰底"对比
- Compose 实现：`Card(elevation = CardDefaults.cardElevation(defaultElevation = 0.dp))` + `Modifier.shadow(elevation = 6.dp, shape = RoundedCornerShape(10.dp))`

### 6.2 悬浮统计卡（`.stats-card`）

- 同卡片样式 + 阴影，但**垂直位置向上拉 34dp**（与头部渐变重叠，见 §4.3）
- 标题行（"今日业绩"）+ 日期行（YYYY-MM-DD）
- 下方三列等分，列间 `1dp` Line 灰分割线
- 数字 20sp/700 / 标签 12sp Muted，数字按业务映射色：
  - 接单 → `Primary`（蓝）
  - 已完成 → `Success`（绿）
  - 进行中 → `Warn`（橙）

### 6.3 列表行（`.cell` / Compose `Cell`）

```
[主文案 14sp/500 + 副文案 12sp Muted flex] [右侧 slot: Tag 或 chevron]
```

- 垂直 padding `12dp`，下边 `1dp Line` 灰分隔（最后一行无）
- 主标题单行省略，副标题最多 2 行
- 行高最小 `48dp`（保证点击热区）
- 右侧 Tag 不被压缩，超出时主标题 ellipsis

### 6.4 快捷入口宫格（`.grid`）

- `4 × 2` 网格，`gap: 8dp`（与用户端 8 一致）
- 单格：白底（**不再用浅灰**——与用户端快捷入口同色，靠阴影与圆角建立边界）+ 圆角 `10dp` + padding `12dp 4dp`
- 文字布局：图标（24-28dp，Primary 浅底 `BrandSoft` 容器 28-32dp 圆角在上） + 标题 `12sp/500` 在下
- 按下反馈：`background: #F0F6FF` 浅蓝（与用户端反馈一致）

---

## 7. Tag 状态映射

工单状态（来自 `worker/schemas.yaml` 的 `TicketStatus` 与 `tagColor()` 函数）：

| 工单状态 | Tag 颜色 | 业务含义 |
|---|---|---|
| `DONE` | `tag-success` | 已完成 |
| `ACCEPTED` | `tag-warning` | 已接单（待上门） |
| `TODO` | `tag-warning` | 待领取（任务池） |
| `DOING` | `tag-error` | 进行中（含紧急 SLA 剩余） |
| `SCAN_PENDING` | `tag-info` | 待扫码绑定 |
| 其他 | `tag-info` | 默认 / 信息 |

Tag 形态统一：`11sp/500` 文字 + `4dp` 圆角 + 横向 `6dp` / 纵向 `2dp` padding + 浅底+前景色+`1dp` 描边（**禁止实心高饱和 Tag**）。

---

## 8. 状态行与时间轴

### 8.1 状态胶囊（`.st-line` / `StatusLine`）

- 节点 `6-8dp` 圆点：在线 `Success` 主色 `#0AA847` / 离线 `Err` 主色 `#FF2D2F`
- `13sp` 白字（透明度 92%），段间用 `·` 分隔："今日接单 5 · 进行中 2 · 待处理 3"
- 容器：白底 `16%` 透明 + `12dp` 圆角（与用户端 Hero 状态胶囊同款，**新增**）
- 出现位置：首页头部下方

### 8.2 时间轴（`.timeline`）

- 节点 `12dp` 圆：未完成 `Placeholder` `#AEB4BE`、已完成 `Success` `#0AA847`、进行中 `Primary` `#086CF5`（带 `3dp` 半透明晕圈）、错误 `Err` `#FF2D2F`
- 连接线 `2dp`：未完成 `Line` 灰、已完成 `Success` 绿
- 节点 `margin-top: 3dp`，文字从 `margin-top: 4dp` 起 ——节点与首行文字视觉对齐
- 阶段标题 `14sp/500` + meta `12sp Muted`

### 8.3 四码矩阵（`.quad`）

- `1fr × 1fr` 网格，`gap: 10dp`
- 单格：浅底（`#F6F8FA` 普通 / `#EFFFF4` OK / `#FFF0F0` BAD）+ 圆角 `10dp` + padding `12dp`
- 描边：`1dp` 对应状态色（`#6FD18B` 绿 / `#FF9B9D` 红 / `#E7EAF0` 普通）
- 标签 `12sp Muted` + 值 `16sp/700`（等宽）+ 状态文字 `11sp`

---

## 9. 表单

| 元素 | 样式 |
|---|---|
| 输入框 | `1dp Line` 描边 + `8dp` 圆角 + `10/12dp` padding + 14sp 文字；聚焦时描边转 `Primary`（`#086CF5`） |
| Label | 12sp `Secondary`，距输入框 `6dp` |
| 提交按钮 | `PrimaryButton`（蓝底 `#086CF5`，§5.1）满宽 |
| 错误反馈 | 行内文字 `Err` `#FF2D2F` + 字段描边转 `Err` |
| 必填星号 | `Err` 红，字号同 label |
| 验证码按钮 | 右侧次按钮（蓝边），倒计时禁用态 `Placeholder` |

---

## 10. 底部导航（`#tabbar` / Compose `AppRoot`）

- 固定底部，`Panel` 白底 + `1dp Line` 顶分隔
- 3 栏等分：`工作台 / 工单 / 我的`
- 图标：24-26dp（草稿用 `⌂/≡/◉` Unicode 占位；Compose 实现用 Material Icons 选中态填充、未选线性）
- 文字：11sp `Tertiary` 未选 / 11sp `Primary` 选中
- 未读气泡：`Err` 红底，`12dp` 圆角，最小 `16×16dp`，绝对定位右上角偏移 `-18/-2dp`
- 内容区高度 `48-52dp`，底部安全区 `~14dp`，背景延伸到屏幕底（不留白带）
- 与用户端同节奏：图标 24-26dp + 文字 11sp 居下，选中态 Primary 色

---

## 11. 平台与安全区

> 与 `designs/UI-SPEC.md` §7 对齐：基准平台 **iOS**（9:41 状态栏、Home Indicator），Android 端按同套 token 实现。

- 顶部内容起点 = 安全区 + `8-12dp`；底部导航背景延伸到屏幕底
- 状态栏描述：
  - **iOS**：透明/跟随头部渐变（前景图标置白）
  - **Android**：半透明（保留系统手势区），实现用 `WindowCompat.setDecorFitsSystemWindows(false)` + `SystemBarStyle.light(...)`
- 设计稿统一以 iOS 视角出图（如标注"Android 版本"则用 iOS 同款 token + Material 3 组件替换）

---

## 12. 与用户端对齐口径（已对齐，2025-08-21）

师傅端与用户端 (`designs/UI-SPEC.md`) 现统一为**同套 token**，核对清单：

| 维度 | 取值 | 对齐锚点 |
|---|---|---|
| 主蓝 | `#086CF5` | `brand.primary` |
| 头部渐变 | `160deg #0872F4 → #0B82F8 → #1698FA` | Hero 三段渐变 |
| 页背景 | `#F6F8FA` | `page.background` |
| 卡片圆角 | `10dp` | `md` |
| 卡片阴影 | `0 3dp 12dp rgba(31,48,72,0.10)` | `shadow.card` |
| 主按钮色 | `#086CF5` 蓝实底 | 主操作同源 |
| 分割线 | `#E7EAF0` | `divider` |
| Tag 色组 | success/info/warning/error 四组 | 同用户端 |
| 悬浮卡重叠 | `34dp` | 与用户端一致 |
| 页边距 | `16dp` | 一致 |
| 卡片间距 | `8dp` | 一致 |
| 卡内 padding | `16dp` | 一致 |
| 状态栏 | iOS 透明 / Android 半透明 | 与用户端一致 |
| 平台归属 | iOS 设计稿基准 | 与用户端一致 |

**仍保留的师傅端特征**（不与用户端冲突，仅业务域不同）：

- 3 Tab 名称：`工作台 / 工单 / 我的`（与用户端 4 Tab `首页/服务/账单/我的` 不同，**业务域差异保留**）
- 工单状态色映射：见 §7 Tag 状态映射（用同套 Tag 色，状态名不同）
- 顶部 Hero 高度：`150dp`（与用户端一致），但内部放"工人姓名+班组+工号"而非用户端的"余额+等级"
- 快捷入口 8 项：排期 / 公告 / 任务池 / 手册 / 测速 / 领料 / 安全 / 维护（**业务差异**，不放用户的"充值/我的账单"等）

---

## 13. 生图 Prompt 用法（ui-proto 预设）

生成师傅端新页面时，把 §2-§11 的 token 逐项翻译进 prompt 的 Style/Colors 段，固定风格关键词：

```
modern field-service worker app, iOS visual language (Android version
uses same tokens + Material 3 components), professional field-tech
feel, clean, info-dense, restrained brand blue (#086CF5)
```

并在 Elements 层写"有什么"（如 status pill + stats card + ongoing tickets list + quick actions grid），不写"长什么样"。所有状态色映射严格按 §7，禁止发明新色；按钮统一蓝底，禁止绿主按钮。

生图时**可参考** `designs/user-home.png` 作为风格基准（同套 token，师傅端只换业务内容）。