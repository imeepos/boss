# boss 用户端 App UI 设计规范(4 tab 全量蒸馏)

> 范围:`mobile/user` Android 端 Compose 实现 + 已生成设计稿(`user-home.png` / `profile-v9.png` /
> `user-products-orders-profile.png`)的视觉汇总。供**新页面**设计与**实现 review**共用。
>
> 数据源优先级(避免历史 token 漂移):
> 1. **代码真值**(最高):`mobile/user/android/app/src/main/java/com/ymm/boss/user/ui/theme/Color.kt`、
>    `ui/Theme.kt::Palette`、`ui/Widgets.kt`
> 2. **本规范**(USER-APP-SPEC):代码之外的几何/字号/间距/状态约定
> 3. **旧 UI-SPEC.md**:仅作历史参考,主色与本规范冲突时**以代码为准**
>
> 设计稿验收遵循 ui-prototype skill 的 Loop A(视觉)/Loop B(spec)双自检。

---

## 1. 设计语言总则

| 维度 | 约定 | 出处 |
|---|---|---|
| 平台语言 | **iOS 风**(左滑/右滑手势条、安全区、状态栏文字白) | Nav.kt + iOS 风设计稿 |
| 设计范式 | 浅灰页底 + 纯白卡片 + 三级表面;主色克制 ≤15% | UI-PARADIGM.md |
| 字体 | 系统默认(PingFang/Roboto);**不超过 3 档字重**(W500/W600/Bold);等宽数字靠系统默认 | Widgets.kt |
| 圆角体系 | **统一 12dp**(卡片/胶囊/小元素);4dp(Tag 内层);999dp(胶囊/筛选) | AppCard.shape |
| 间距节奏 | 4dp 网格;页边距 **14dp**(非 pinned 页 AppCard.outer)/ **16dp**(首页·我的滚动区 sideMargin,详见 §4);卡间距 **6dp**;卡内 16dp | AppCard.outer/inner + PinnedHeaderSpec |
| 行高下限 | 列表行 ≥44dp;点击热区 ≥48dp;菜单行 vertical=12dp + 40dp IconTile | MenuRow |
| 阴影 | 卡片 `0.5.dp` elevation 极淡;**禁止重黑投影** | MaterialTheme 默认 |
| 状态色占比 | 中性+白 ≥80%,品牌蓝 10-15%,状态色合计 ≤5% | UI-PARADIGM.md |

> **强约束**(FEEDBACK.md A 区):破坏性操作(退出登录/取消) = 白底卡 + 居中红字 + 小图标,
> **禁止实心通栏红按钮**。

---

## 2. 色彩令牌(代码真值)

| Token | Hex | 用途 | 出处 |
|---|---|---|---|
| `Palette.primary` | `#007AFF` | 主色/选中态/价格/链接/已发生步骤 | BrandBlue |
| `Palette.primary2` | `#3B82F6` | 渐变末端(浅)/`#1E40AF`(深) | BrandBlueGradientEnd |
| `Palette.bg` | `#F5F6F8` | **页面背景**(滚动区底) | Theme.kt |
| `Palette.panel` | `#FFFFFF`(浅)/`#1C1C1E`(深) | 卡片/弹框底 | Surface |
| `Palette.line` | `#E8EAED` | 描边/分割线 | Theme.kt |
| `Palette.subtle` | `#B0B3B8` | 弱图标/箭头 | Theme.kt |
| `Palette.ink` | `#1C1C1E`(浅)/`#F2F2F7`(深) | 主文字 | OnSurface |
| `Palette.muted` | `#3F3F46`(浅)/`#A1A1AA`(深) | 辅助文字 | OnSurfaceVariant |
| `Palette.success` | `#34C759` | 在网/正常/已完成 | Green500 |
| `Palette.warn` | `#FF9500` | 提醒/推荐价/未缴 | ActionOrange |
| `Palette.err` | `#FF3B30` | 异常/未读红点/退出 | Theme.kt |
| `Palette.orange` | `#FF9500` | 快捷图标橙(独立 IconTile 用) | ActionOrange |
| `Palette.purple` | `#AF52DE` | 快捷图标紫/客服/报障 | ActionPurple |
| `OnGradient` | `#FFFFFF` | 渐变头上的文字/图标 | Color.kt |

### 2.1 渐变(渐变头 + 状态栏固定色带)

| 用途 | 浅色 | 深色 |
|---|---|---|
| **首页渐变** | `#006AE5 → #3B82F6` | `#4E97EC → #1E40AF` |
| **我的页渐变** | `#007AFF → #3B82F6` | `#60A5FA → #1E40AF` |
| **状态栏固定色** | `#006AE5` | `#4E97EC` |
| **TabBar 头背景** | `#006AE5`(与状态栏同源,消除接缝) | `#4E97EC` |
| **登录页 Hero 渐变** | `#0872F4 → #0B82F8 → #1698FA` (160°) | (登录页无深色适配,沿用浅色) |
| **实名页 Hero 渐变** | `#0872F4 → #1698FA` | (同上) |

> ⚠️ `homeHeaderGradient` 起点比 `profileHeaderGradient` 略深(`#006AE5` vs `#007AFF`),仅为
> 视觉区分同族不同页面,不允许设计师自行换色。
>
> ⚠️ **登录页/实名页渐变与 4 tab 同族但非同一组**(多了中间色 `#0B82F8`、三段 vs 两段),
> 是有意设计决策——品牌门面页与日常操作页拉开气质,设计师不要试图"统一"成首页渐变。

### 2.2 IconTile 语义色规则(共享快捷/菜单)

- **品牌业务(订单/账单/地址/套餐)**:Palette.primary(蓝)
- **金额/缴费/优惠**:Palette.orange(橙)
- **报障/消息/协议**:Palette.purple(紫) 或 Palette.muted(灰,仅协议)
- 浅底 = `tint.copy(alpha = 0.1f)`,圆角 12dp,图标尺寸 = 容器 × 0.55

### 2.3 登录 / 注册 / 实名 私有 token(RN + LoginHero)

> 来源:`page/RealNamePage.kt::object RN`(46-56)、`page/LoginPage.kt:46/118`、`page/AuthForm.kt:51-52`。
> 登录/注册/实名/找回共用此套私有 token,与全局 `Palette` 同族但色相略偏(主色更"政务蓝"、
> 成功色更深一档、文字色冷调),用于在品牌门面页与日常 4 tab 操作页之间拉开气质。
> 新页面**禁止**继续发明第三套本地 token,如需扩展请合并到 RN 或 Palette。

| Token | Hex | 用途 | 出处 |
|---|---|---|---|
| `RN.primary` | `#086CF5` | 登录/实名主色、按钮、链接 | RealNamePage.kt:47 |
| `RN.heroEnd` | `#1698FA` | 实名页渐变末端 | RealNamePage.kt:48 |
| `RN.success` | `#0AA847` | "已通过/认证成功" 文字色 | RealNamePage.kt:49 |
| `RN.successBg` | `#EFFFF4` | 成功浅底(状态卡背景) | RealNamePage.kt:50 |
| `RN.warn` | `#F57900` | "审核中/未通过" 文字色 | RealNamePage.kt:51 |
| `RN.warnBg` | `#FFF5E8` | 警告浅底 | RealNamePage.kt:52 |
| `RN.ink` | `#171B23` | 主文字 | RealNamePage.kt:53 |
| `RN.muted` | `#5F6671` | 辅助文字(协议行/副文案) | RealNamePage.kt:54 |
| `RN.placeholder` | `#AEB4BE` | 输入框占位 | RealNamePage.kt:55 |
| `RN.line` | `#E7EAF0` | 输入框未聚焦描边 | RealNamePage.kt:56 |
| `RN.error` | `#FF2D2F` | 登录/实名错误提示(独立于 `Palette.err #FF3B30`) | Login/RealNameForm/UploadStep 多处 |
| `pageBg`(登录页) | `#F6F8FA` | 登录页全屏底(比 `Palette.bg #F5F6F8` 亮 1 位) | LoginPage.kt:46 |
| `fieldBg`(AuthForm) | `#F7F8FA` | 输入框底 | AuthForm.kt:51 |
| `iconTint`(AuthForm) | `#7D8593` | 输入框左线性图标 | AuthForm.kt:52 |

**未读红点例外**:`PinnedGradientPage.kt::HeaderBell`(102) 用 `#FF5252` 而非 `Palette.err #FF3B30`,
是真实代码差异——更"亮"的红用于突出未读提醒。新稿对应位置请用 `#FF5252`,不要"统一"成 `#FF3B30`。

**OnlineTag 内联色例外**:`UserHomeCards.kt:OnlineTag` 直接用 `Green500 #34C759` 实色
(而非 `Palette.success.copy(alpha=0.1f)` 的浅底描边样式)——这是"在网"徽章的特例外貌,
**仅限**此一处使用,其余"绿/橙/红"语义色继续走 Tag(浅底描边)规范。

---

## 3. 字体层级(全 App 统一)

| 层级 | 字号/行高 | 字重 | 颜色 | 用途 |
|---|---|---|---|---|
| Hero 标题 | 22sp/28sp | Bold | OnGradient | 首页问候、我的姓名 |
| 大数字 | 30sp/30sp | Bold | Palette.ink | 账单应缴金额 |
| 价格/进度数值 | 20sp/24sp | Bold | Palette.primary | 宽带费用、SubInfo 值 |
| TabHeader | 16sp | W600 | 白 | 顶部白底/渐变头居中标题 |
| 卡片标题 | 16sp/20sp | Bold | Palette.primary | "进行中订单"/"已生效增值服务" |
| 列表主标题 | 15sp/20sp | W600 | Palette.ink | 产品名、订单产品名 |
| 列表主标题(2) | 14sp/20sp | W500 | Palette.ink | 菜单行、服务名 |
| 副标题 | 12sp/16sp | Normal | Palette.muted | 元数据、地址 |
| Tag | 11sp | Normal | 语义色 | "热门"/"装维中"/"在网" |
| 元信息 | 12.5sp | Normal | Palette.muted | Notice/小字 |

> **价格小符号**:¥ 11sp Bold + 金额 16sp Bold + /月 10sp;价格胶囊 8dp 圆角,
> 普通档 `primary.copy(alpha=0.12f)` 底,推荐档 `warn` 橙底白字 + 拇指角标。

### 3.1 代码实测字号分布(grep 全 38 个页面)

| 字号 | 行高 | 字重 | 实测用途 | 出处 |
|---|---|---|---|---|
| 10sp | — | Normal | 步骤条数字、/月单位 | OrderCard.kt:180, ProductsPage.kt:193 |
| 11sp | — | Normal | Tag、价格 ¥ 符号、错误提示 | Widgets.kt:78, RealNameUploadStep.kt:146 |
| 12sp | 12/16 | Normal | 副标题、副文案、tab 文字 | Nav.kt:102, MenuRow |
| 12.5sp | — | Normal | Notice、CardTitle more、手机号 | Widgets.kt:106, ProfilePage.kt:115 |
| 13sp | 16 | Normal | PillTab、菜单行、CellRow 副 | Widgets.kt:150, MenuRow.kt:77 |
| 14sp | 16/20 | W500 / Medium | CellRow 标题、菜单行、状态文字 | Widgets.kt:68, ProfileMenu.kt:77 |
| 15sp | 20 | W500/W600 | 产品名、订单产品名、AuthInputRow | OrderCard.kt:72, ProductsPage.kt:151 |
| 16sp | 20 | W600/Bold | 订单号、TabHeader、卡片标题 | Widgets.kt:129, OrderCard.kt:63 |
| 17sp | — | Bold | 实名状态页大标题 | RealNameStatusPages.kt:73 |
| 18sp | — | Bold | 头像姓氏、支付成功标题 | ProfilePage.kt:109, PayResultPage.kt:86 |
| 20sp | 24 | Medium/Bold | SubInfo 数值、首页宽带名、注册卡大标题 | UserHomeCards.kt:131/158, RegisterPage.kt:119 |
| 22sp | 28 | Bold | 首页问候、我的头像字 | UserHomeCards.kt:67, ProfilePage.kt:109 |
| 24sp | — | Bold | 评价页数值 | RatePage.kt:86 |
| 30sp | — | Bold | 应缴金额(Bills/Pay/Usage 都用 30sp) | BillsPage.kt:81, PayPage.kt:84, UsagePage.kt:87 |
| 32sp | — | — | (未在页面字号中出现,保留备用) | — |

> **字重纪律**(实测):全 App 仅出现 4 档 `FontWeight.Normal / W500 / W600 / Bold`;
> 没有用 `W400`/`W700`。新稿继续守 4 档上限,不要新增 W700。

---

## 4. 布局骨架(4 tab 通用)

```
┌─────────────────────────┐
│ StatusBarBand(高度=系统statusBar inset,固定色)            │  ← 整个 App 一致
├─────────────────────────┤
│                         │
│  Hero 渐变头(176dp,首页/我的)            │
│  或 TabHeader 白底(48dp,服务/账单,服务=搜索栏嵌入)            │
│                         │
├─────────────────────────┤
│  圆角白卡滚动区(顶 16dp 圆角,左右 16dp 边距,底 Palette.bg)            │
│  ┌───────────────┐  ← 首卡与渐变头重叠 32dp("悬浮快捷卡"手法)            │
│  │  卡片1         │            │
│  └───────────────┘            │
│  ─ 6dp ─            │
│  ┌───────────────┐            │
│  │  卡片2         │            │
│  └───────────────┘            │
├─────────────────────────┤
│ BottomTabBar(56dp,无胶囊指示器,选中主色蓝)            │
└─────────────────────────┘
```

**通用几何参数**(PinnedHeaderSpec / AppCard):

- 渐变头高度 = **176dp**(`PinnedHeaderSpec.headerHeight`)
- 渐变尾巴 = **16dp**(`gradientTail`,透出圆角缺口)
- 滚动区上提量 = **46dp**(`regionLift`,即与渐变头重叠量)
- 滚动区顶部圆角 = **16dp**(`corner`)
- 滚动区左右边距 = **16dp**(`PinnedHeaderSpec.sideMargin`)
- 卡片 outer padding = `PaddingValues(horizontal = 14.dp, vertical = 6.dp)`(`AppCard` 默认)
- 卡片 inner padding = `PaddingValues(16.dp)`
- 卡片 shape = `RoundedCornerShape(12.dp)`
- 卡片 elevation = `0.5.dp`(极淡)

> ⚠️ **页边距是"两套上下文"而非一个值**(代码真值,2025-08-21 第三轮核对修正):
>
> | 上下文 | 左右边距来源 | 实测值 |
> |---|---|---|
> | 首页 / 我的(`PinnedGradientPage` 内) | 滚动区 `sideMargin` 提供,卡片 **override `outer` 去掉水平内边距** | **16dp** |
> | 服务 / 账单及各详情页(无 pinned 骨架) | `AppCard` 默认 `outer.horizontal` | **14dp** |
>
> 出处:`ui/PinnedGradientPage.kt:43`(`sideMargin = 16.dp`)、`ui/Widgets.kt:39`(`outer = horizontal 14.dp`)、
> `page/ProfilePage.kt:160` 与 `page/UserHomePage.kt`(pinned 页卡片传 `outer = PaddingValues(top=0, bottom=6)`)。
> 全仓实测 `horizontal = 14.dp` 24 处、`horizontal = 16.dp` 11 处,两者**并存且各有归属**。
>
> 规则:pinned 骨架页新增卡片 → 用 `outer = PaddingValues(vertical = 6.dp)` 让滚动区给边距(勿再叠 14dp);
> 非 pinned 页新增卡片 → 直接用 `AppCard` 默认值。**不要试图把两者"统一"成同一个数**——
> 视觉上两类页面的卡片外沿本就对齐各自容器,统一反而会错位。

---

## 5. 共享组件(Widgets.kt 必须复用)

| 组件 | 关键参数 | 用途 |
|---|---|---|
| `AppCard` | shape=12, outer=(14,6), inner=16 | 所有白卡容器 |
| `CardTitle(title, more?, onMore?)` | 15sp/W600 + 12.5sp 链接 | 卡内标题行 |
| `CellRow(title, desc?, onClick?, right?)` | vertical=12, 14sp/W500 + 12sp 副 | 账单行 |
| `Tag(text, color)` | 11sp + 6/2 padding + 4dp 圆角 + α=0.1 底 | 语义标签 |
| `TopBar(title, onBack?, action?, onAction?)` | 48dp, statusBarSolid 底 | 详情页头 |
| `TabHeader(title)` | 48dp, statusBarSolid 底 | tab 页居中标题 |
| `PillTab(label, active, onClick, icon?, plain?)` | 999dp 圆角,active=主色实底白字 | 分类/筛选胶囊 |
| `IconTile(icon, tint, size=40, corner=10)` | α=0.1 底 + tint 图标 | 列表/快捷图标容器 |
| `PricePill(fee, recommended)` | ¥11/16/10sp,8dp 圆角 | 价格显示 |
| `EmptyState(text)` | 40dp 图标 + 10dp 间距 + 13sp 文字 | 空态 |
| `StatusDot(on)` | 8dp 圆点 | 在线指示 |
| `Notice(text, color)` | 12.5sp | 错误/提示 |
| `BottomTabBar` | 56dp, indicator=透明 | 4 tab 导航 |
| `PinnedGradientPage` | 骨架(176dp 渐变 + 16dp sideMargin + 16dp 圆角 + 46dp 上提) | 首页/我的专用 |
| `HeaderBell(hasUnread)` | 40dp 铃铛 + 8dp 红点 | 首页通知入口 |

**复用铁律**(FEEDBACK.md):TabHeader / PillTab / IconTile / AppCard / EmptyState 已入
`ui/Widgets.kt`,其他页面必须复用,**禁止重造**。

### 5.1 认证流组件(AuthForm.kt + RN 私有 token)

> 来源:`page/AuthForm.kt` + `RealNamePage.kt::RN`。
> **只用于登录/注册/实名/找回四条路由**;新业务页禁止复用(色系与 4 tab 隔离)。

| 组件 | 关键参数 | 用途 |
|---|---|---|
| `AuthCard(title, sub, onBack?)` | `TopBar(48dp)` + 居中白卡(10dp 圆角 + 16dp 内边距) | 注册/找回卡骨架 |
| `AuthInputRow(value, hint, icon, keyboardType, trailing?)` | 52dp 高,`fieldBg #F7F8FA` 底,12dp 圆角,左 20dp 线性图标,focus 时 `RN.primary` 描边 | 单行输入(手机/验证码/密码) |
| `AuthSegment(items, current, onSelect)` | 灰槽(`#F5F6F8` 10dp 圆角)+ 蓝块(`#007AFF` 8dp 圆角,selected) | "验证码/密码" 切换 |
| `AuthAgreeRow(agreed, onToggle, onAgreement?)` | 圆形勾选框(直径与字体比例同步),协议行 12sp | 协议确认行 |
| `RNPrimaryButton(text, enabled, loading, onClick)` | 通栏主按钮,渐变蓝/单色蓝 | 登录/注册/实名/找回提交 |
| `RNFootnote(text, color)` | 12sp 错误/提示小字 | 表单脚错误提示 |
| `LoginHero`(私有) | 212dp 高,三色渐变 `#0872F4→#0B82F8→#1698FA`,两枚半透明白圆 + 圆角白方标(64dp Router 图标) | 登录页 Hero |

> `AuthInputRow` 的 `fieldBg #F7F8FA` 比 `Palette.bg #F5F6F8` 亮 1 位、圆角 12dp 比 `AppCard` 12dp 一致
> 但**不要把 AppCard 改成 12dp**——AppCard 是给 4 tab 用的,AuthInputRow 是认证流专用,两者不通用。

---

## 6. 4 Tab 页面区块清单

### 6.0 认证流(登录 / 注册 / 找回 / 实名)— 不走 4 tab 骨架

> 这四页**不**用 `PinnedGradientPage`,走独立的渐变 Hero + 圆角白卡结构;
> Hero 渐变与 4 tab 同族但色相不同(详见 §2.1 警告),不在 Prompt 里写"首页渐变"就行。

#### 6.0.1 登录页(`Route.Login`)

骨架:`Column + verticalScroll` → `LoginHero(212dp)` → 悬浮白卡(top=176dp,10dp 圆角,16dp 内边距) → 卡外协议行 + 主按钮 + 注册入口。

| # | 区块 | 关键字段 | 设计要点 |
|---|---|---|---|
| 1 | Hero | — | 212dp 高;三色渐变 `#0872F4→#0B82F8→#1698FA`(160°);两枚半透明白圆(160dp α=0.06 + 110dp α=0.08);底部居中 64dp 白 18% 圆角白方标(16dp 圆角 + 35% 描边 + 32dp Router 图标) + 20sp Bold 白 "装维全流程 · 用户端" + 12sp letterSpacing=1 副标题 |
| 2 | 悬浮白卡 | — | top=176dp,水平 16dp 边距,白底 10dp 圆角,16dp 内边距;**仅含分段+表单行** |
| 3 | 模式分段 | `sms/password` | `AuthSegment` 两段("验证码登录" / "密码登录"),selected=8dp 圆角实心蓝(`Palette.primary`,不是 RN.primary) |
| 4 | 手机号行 | `phone` | `AuthPhoneRow`,左 PhoneAndroid 20dp 线性图标 + 15sp 输入 + placeholder |
| 5 | 验证码/密码行(条件) | `code`/`password` + `pwdVisible` | sms 模式 = `AuthCodeRow` + 倒计时按钮;password 模式 = `AuthPwdRow` + 12sp 右对齐 "忘记密码 >" 链接 |
| 6 | 错误提示(条件) | `err` | `RNFootnote(err, #FF2D2F)` 12sp 红 |
| 7 | 协议行(卡外) | `agreed` | 圆形勾选 + 12sp "我已阅读并同意" + RN.primary "《用户协议》" |
| 8 | 主按钮(卡外) | `busy` | `RNPrimaryButton("登录")`,loading 时显示进度 |
| 9 | 注册入口(卡外) | — | 12sp "还没有账号? 立即注册"(后者 RN.primary/W500) |

> **频率分层红线**(FEEDBACK.md C 区):注册是低频路径,只占"立即注册"小字链接,**不与登录同级**。
> 新稿若扩展登录页(例如忘记密码子页),继续保持注册=文字链接、不出主按钮。

#### 6.0.2 注册页(`Route.Register`)

骨架:`AuthCard("自助注册", "手机号注册 · 注册即享宽带自助服务")`。
组件顺序:手机号 → 验证码(倒计时) → 密码(两次,可见性切换) → 错误提示(条件) → 协议行 →
`RNPrimaryButton("注册并登录")` → 12sp "已有账号,去登录" 链接(pop 回登录)。

> 契约:`POST /auth/register` 提交后端审核,成功后 `nav.replace(Login)`。

#### 6.0.3 找回密码页(`Route.Forgot`)

骨架:同 `AuthCard("找回密码")`;组件顺序:手机号 → 验证码 → 新密码 → 协议行 →
`RNPrimaryButton("重置并登录")` → 12sp "返回登录" 链接。
错误:`ForgotPage.kt:50` 用 `Palette.err #FF3B30` 而**非** RN 错误色 `#FF2D2F`——这是局部差异,
**沿用代码原值**(`Palette.err`),不要"统一"成 `#FF2D2F`。

#### 6.0.4 实名认证流(`Route.Verify` + RNPhase)

> 来源:`page/RealNamePage.kt` + `RealNameFormStep` + `RealNameUploadStep` + `RealNameStatusPages`。
> 5 个阶段枚举:`FORM → UPLOAD → REVIEWING → APPROVED / REJECTED`(互斥状态页)。

| # | 区块 | 关键字段 | 设计要点 |
|---|---|---|---|
| 1 | Hero(表单/上传阶段) | — | 212dp,渐变 `#0872F4 → #1698FA`;居中 20sp Bold "信息安全保障" + 14sp 副;返回箭头 + 进度胶囊(如适用) |
| 2 | 进度胶囊(顶部) | `currentStep`/`totalStep` | `RoundedCornerShape(10.dp)` 圆角胶囊,active 实色 RN.primary、白字 15sp/W600,inactive 描边 RN.warn、白字 15sp/W600;步骤标签全 App 统一("填写信息 · 上传证件 · 等待审核") |
| 3 | 表单卡 | name / idNo / phone / code | 10dp 圆角白卡 + 16dp 内边距;行高 52dp;submitErr 走 12sp 红 |
| 4 | 上传区 | `frontUrl/backUrl/handheldUrl` | 100×100dp 1dp RN.placeholder 描边(空)或 RN.line 30% 底(已上传);ErrorOutline `#FF2D2F` 反馈 |
| 5 | 状态卡(审核中/通过/拒绝) | `verifyStatus` / `latestResult` | `RoundedCornerShape(10.dp)` 圆角卡;标题 17sp Bold tint=success/warn/err;bg 默认 `#EFFFF4`(成功浅底),warn 状态用 `#FFF5E8`;副标题 12sp muted |
| 6 | 时间戳条 | `submittedAt/approvedAt/rejectedAt` | 14sp muted 标签 + 14sp/W500 ink 值 |
| 7 | 错误横幅 | `err` | `RoundedCornerShape(10.dp)` 圆角,RN.warn 红 + 白字 15sp/W600 |

> **命名统一红线**(FEEDBACK.md C 区):实名步骤标签全 App 必须复用"填写信息 · 上传证件 · 等待审核"
> 三段命名,分屏稿 prompt 必须逐字写死。

### 6.1 首页(`Route.Home` / nav.key: "home")

骨架:`PinnedGradientPage(homeHeaderGradient)` → `LazyColumn`。

| # | 区块 | 关键字段 | 设计要点 |
|---|---|---|---|
| 1 | Hero 头 | `customerName` / `phoneMasked` / `onlineStatus` / `hasUnread` | 22sp Bold 问候 + 16sp 手机 + 8dp 绿点 + 14sp 状态;右 48dp 铃铛(8dp 红点条件) |
| 2 | 宽带主卡(悬浮) | `plan.name` / `currentBill` / `balance` / `contractEnd` | 顶 Home 图标 + "家庭宽带 {planName}" + 绿色"在网"Tag;三列 SubInfo(本月账单/套餐余额/合约到期),数值 20sp Bold 主色 |
| 3 | 8 宫格快捷 | `办套餐/查订单/缴费用/报故障/充值/查用量/消息/客服` | 2×4,图标 24dp + 13sp;水平 4 列均分 |
| 4 | 进行中订单 | `ongoingOrders` (≤2) | 16sp Bold 标题 + "全部 >";加载/错误/空三态收敛;订单行 = orderNo 16sp/W500 + LocationOn 地址 14sp 灰 + 14sp 状态主色 + "N/12" + 8dp LinearProgress + › |
| 5 | 已生效增值服务 | `services`(filter ACTIVE) | 16sp Bold 标题 + Router 圆形 IconTile + 36dp + 15sp/W500 + 13sp 灰 + "在网"绿胶囊 24dp/8dp |
| 6 | Bottom Tab | — | 4 tab(首页选中) |

### 6.2 服务 tab(`Route.Products` / nav.key: "products")

骨架:TabHeader(48dp statusBarSolid)**变体** = `ServiceSearchHeader`(搜索栏嵌入)
→ `CategorySeg`(三 PillTab)→ `LazyColumn`。

| # | 区块 | 关键字段 | 设计要点 |
|---|---|---|---|
| 1 | 搜索栏 | 本地过滤 name/bandwidth/description | 半透明白胶囊(`Color.White.copy(alpha=0.18f)`,999dp),14/8 padding,16dp 搜索图标 + 13sp 占位 |
| 2 | 分类胶囊 | `broadband/fusion/addon` | 宽带(Wifi)/5G(SignalCellularAlt)/增值服务(CardGiftcard),PillTab 选中主色实底白字 |
| 3 | 产品卡 | `productId` / `name` / `featured` / `monthlyFee` / `bandwidth` / `contractMonths` / `description` | 上行:名称 15sp/W600 + 绿色"热门"Tag(条件)+ PricePill(普通/推荐);下行:44dp 浅蓝 IconTile + MetaLine(带宽·合约月)+ FeatureLine + ›;**底部"立即办理"44dp/8dp 通栏实心蓝按钮**(本 tab 唯一允许的大按钮) |
| 4 | 增值服务卡(条件) | `addons` | 标题"增值服务" + "进入管理 >";行 = 36dp 紫 IconTile + 14sp/W500 + 12sp 描述 + ¥X/月 |
| 5 | Bottom Tab | — | 服务选中 |

> 顺序:**搜索 → 分类 → 产品列表 → 增值服务**(addon tab 才出现末项)。
> 契约 Product 无 `originalPrice`/"Wi-Fi 6"等卖点字段,设计稿的划线价**禁止**前端编造。

### 6.3 账单 tab(`Route.Orders` / nav.key: "orders")

骨架:`TopBar("我的订单", action="开发票")` → `Column+verticalScroll`。

| # | 区块 | 关键字段 | 设计要点 |
|---|---|---|---|
| 1 | TopBar | action → `Route.Invoice` | 48dp statusBarSolid 底,左返回箭头 + 居中标题 + 右"开发票" |
| 2 | 应缴主卡 | `currentDue` / `currentPeriod` | 居中:12.5sp 灰"当前应缴({period} 账期)" + 30sp/Bold 黑色 ¥XX.XX + Row 双按钮(实心"立即缴费" + 描边"余额充值") |
| 3 | 筛选胶囊 | `all/in_progress/done/cancelled` | `PillTab(plain=true)`,全部(选中蓝)/进行中/已完成/已取消;plain 模式未选 = 透明无底 |
| 4 | 订单卡 | `ongoingOrders` | 同首页订单行结构(头+体+步骤条) |
| 5 | 步骤条(进行中) | `stage`(1-12) | 4 节点:**提交订单 → 受理成功 → 上门安装 → 完成**;里程碑映射 `stage≤1→1, 2-7→2, 8-11→3, 12→4`;已过实心蓝白对勾,当前实心蓝白数字加粗,未到白底描边灰数字;连接线 2dp 已过段主色 |
| 6 | 预计上门条(条件) | `estimateFinish` | `Palette.primary.copy(alpha=0.08f)` 圆角条 + Person 图标 + "预计上门:{date}" + › |
| 7 | 已完成尾区(DONE) | `status=PAID` + `canRate` | 绿色 CheckCircle 20dp + "订单已完成" 14sp/W600 + 副文案 + ›;canRate=true 追加"去评价"蓝链 |
| 8 | 列表尾 | — | "没有更多订单了" 12sp 居中 / 空态"暂无订单" |
| 9 | Bottom Tab | — | 账单选中 |

### 6.4 我的 tab(`Route.Profile` / nav.key: "profile")

骨架:`PinnedGradientPage(profileHeaderGradient)` → `Column+verticalScroll`。

| # | 区块 | 关键字段 | 设计要点 |
|---|---|---|---|
| 1 | 渐变头 | `name` / `phoneMasked` / `realName.status` | 56dp 头像(白 25% 底 + 2dp 白描边 + 姓氏首字 22sp Bold 白)+ 18sp Bold 姓名 + 12.5sp 白 85% 手机 + "已实名"白 20% 胶囊(GppGood 12dp);右上角 LanguageDropdown + Settings 40dp 齿轮 |
| 2 | 三入口卡 | `realName.status` / `addresses[].length` / `plan.name` | **首卡与渐变头重叠 32dp**(顶 offset 0);3 列:实名(GppGood 蓝/待补登) / 地址(Home 橙/N 个地址) / 套餐(CardMembership 紫/套餐名);40dp/12dp IconTile + 13sp/W500 + 11sp 副 |
| 3 | 我的服务(7 行) | 7 个 `Route` | 标题行;行 = 40dp IconTile + 12 + 14sp/W500 + (消息有未读时 8dp 红点)+ ›;图标:订单/List 蓝 / 账单/ReceiptLong 蓝 / 缴费/CurrencyYen 橙 / 报障/Build 紫 / 消息/Chat 紫 / 优惠/LocalOffer 橙 / 发票/Receipt 蓝 |
| 4 | 账号与设置(5 行) | 5 个 `Route` | 同上行结构;图标:安全/Lock 蓝 / 通知/Notifications 橙 / 投诉/Chat 紫 / 帮助/Help 蓝 / 协议/Description 灰 |
| 5 | 语言卡(隐于头右上角) | `zh/en/fil` | 头部 `LanguageDropdown` 半透明胶囊,选中后落 `PUT /profile/language`;失败保留本地高亮 |
| 6 | 退出登录 | — | **白卡 + 居中红字 15sp/W500 + Logout 图标 16dp**(FEEDBACK.md 红线) |
| 7 | Bottom Tab | — | 我的选中 |

---

## 7. 订单 12 环节(契约固定,设计稿不可改)

订单阶段严格对齐 `docs/contract/terms.md` §1 的 12 环节枚举;账单 tab 步骤条
**只是视觉聚合(4 节点)**,不删不改环节定义:

- 里程碑 1(stage ≤ 1):**提交订单** = 受理
- 里程碑 2(stage 2-7):**受理成功** = 资源配置/装配/出库等
- 里程碑 3(stage 8-11):**上门安装** = 上门/调试/激活/验收
- 里程碑 4(stage = 12):**完成**

步骤条 4 节点命名 = "提交订单 · 受理成功 · 上门安装 · 完成",**全 App 统一**,
分屏稿必须复用相同文字。

---

## 8. 状态色映射(订单状态 → UI 表现)

| 状态 | 标签 | 颜色 | 适用 |
|---|---|---|---|
| `PENDING` | 待核查 | `#3F3F46`(灰) | 订单行状态文字 |
| `RESERVED` | 已预占 | `#FF9500`(橙) | 订单行状态文字 |
| `INSTALLING` | 装维中 | `#007AFF`(主色蓝) | 订单行状态文字 + 步骤条当前 |
| `DONE` | 已完成 | `#34C759`(绿) | 订单行状态文字 + 步骤条已过 + 绿色 ✓ 区域 |
| `CANCELLED` | 已取消 | `#3F3F46`(灰) | 订单行状态文字 |

> ⚠️ 状态色合计 ≤5% 页面面积;状态至少用 **颜色 + 文字**(必要时加图标)双线索,
> 禁止仅靠颜色传达。

---

## 9. 交互与状态机(全 App 通用)

| 状态 | 触发 | 视觉 |
|---|---|---|
| 加载 | 接口 pending | 居中 `CircularProgressIndicator(strokeWidth=2.dp, size=24.dp)`;保留骨架(不动布局) |
| 错误 | 接口 fail | 居中 14sp 红色 `error.message` + "点击重试"蓝链(48dp 热区) |
| 空 | 数据空 | `EmptyState(text, icon)`,40dp 图标 + 10dp + 13sp 灰 |
| 下拉刷新 | 顶部下拉 | `PullRefresh` 自定义(项目内) |
| 未读 | `messages.read=false count>0` | 铃铛 8dp 红点 + 消息菜单行 8dp 红点 |
| 在线 | `onlineStatus` 含"在线"且无"异常/故障" | 绿点 + "服务在线·网络正常" |

---

## 10. 字体/字重/价格统一纪律(范式)

- 字号 ≤5 档:12 / 14 / 15-16 / 20 / 22-30
- 字重只 3 档:`FontWeight.Normal / W500 / W600 / Bold`
- 价格:¥ 小符号 + 金额加粗 + 单位更小;推荐档橙底白字 + 拇指角标
- 金额一律两位小数(`%.2f`),无值显示 `--`
- 脱敏手机号:`138****8888` 形态,等宽靠系统默认
- 步骤条 4 节点文字 + 完成态时间戳,严格等宽对齐

---

## 11. 已知局限 / 不允许前端编造

- **Product 无 originalPrice** → 设计稿的划线价(¥129/月 → ¥89/月)不落实现
- **Product 无 features 数组** → "Wi-Fi 6"/"稳定加速 全屋覆盖"用 `description` 单字段表达
- **订单列表无 createdAt** → 设计稿的右侧时间戳在 `createdAt=""` 时不渲染
- **Address 无类型字段** → 实名/家庭地址不区分,设计稿的"默认地址"标记不实现
- **Profile 无头像 URL** → 始终显示姓氏首字头像(56dp 白 25% 底 + 2dp 白描边)

---

## 11.5 代码 vs 规范对齐状态(2025-08-21 第二轮审查)

> 对齐原则:**代码真值为准**(grep 全 38 个 page + 6 个 ui 源文件)。
> 本轮新增/修正的 token 与组件:

| 维度 | 修正前 | **修正后(代码真值)** | 出处 |
|---|---|---|---|
| **登录/注册/实名/找回 主色** | 未列 | `#086CF5` (RN.primary) | RealNamePage.kt:47 |
| **登录页背景** | 未列 | `#F6F8FA` (≠ Palette.bg #F5F6F8) | LoginPage.kt:46 |
| **登录页 Hero 渐变** | 未列 | `#0872F4 → #0B82F8 → #1698FA` 三段 | LoginPage.kt:118 |
| **实名页 Hero 渐变** | 未列 | `#0872F4 → #1698FA` 两段 | RealNamePage.kt:137 |
| **认证错误红** | 未列 | `#FF2D2F` (≠ Palette.err #FF3B30) | LoginPage.kt:95, RealNameUploadStep.kt:74 |
| **认证状态色** | 未列 | success `#0AA847` / warn `#F57900`(比全局更深) | RealNamePage.kt:49-52 |
| **认证 ink/muted/line** | 未列 | `#171B23` / `#5F6671` / `#E7EAF0` | RealNamePage.kt:53-56 |
| **认证 input 底/icon 色** | 未列 | fieldBg `#F7F8FA` / iconTint `#7D8593` | AuthForm.kt:51-52 |
| **首页未读红点** | `#FF3B30`(错) | **`#FF5252`** | PinnedGradientPage.kt:102 |
| **"在网"徽章样式** | 误为 Tag 规范 | 实测用 Green500 `#34C759` 实色 8dp 圆角 | UserHomeCards.kt:OnlineTag |
| **找回密码错误色** | 与认证流"统一"(误) | 实测用 `Palette.err #FF3B30` 而非 `#FF2D2F` | ForgotPage.kt:50 |
| **登录页 AuthSegment selected 色** | 未列 | `Palette.primary #007AFF` 而非 RN.primary | LoginPage.kt + AuthForm.kt:142 |
| **字号体系** | 仅 5 档(12/14/16/20/22) | 实测 11/12/12.5/13/14/15/16/17/18/20/22/24/30sp 13 档 | §3.1 |
| **字重体系** | 3 档(W500/W600/Bold) | 实测 4 档(Normal/W500/W600/Bold) | §3.1 |

> **未在本规范出现的私有 token**(未审或未使用):`NavigationBlue #0A84FF`(Color.kt:11)、
> `BrandBlueDark #60A5FA`、`BrandBlueGradientStart #1E3A8A` 等 —— 它们都在
> `ui/theme/Color.kt` 中定义但当前无页面直接 import。如未来扩展深色主题/二级品牌,优先复用这些已定义 token。

---

## 12. 必读伴随文件(验收/扩展用)

- `designs/UI-SPEC.md` —— 历史 token 文档,主色与本规范冲突以本规范为准
- `designs/UI-PARADIGM.md` —— "法"层:色彩比例、信息优先级、悬浮卡、动作分级
- `designs/FEEDBACK.md` —— 历史教训(M5 红线:移动端不能塞一屏、退出不用实心红)
- `mobile/user/design/user-products-orders-profile.spec.md` —— 三屏原始 spec
- `mobile/user/android/app/src/main/java/com/ymm/boss/user/ui/theme/Color.kt` —— 全局色 token
- `mobile/user/android/app/src/main/java/com/ymm/boss/user/page/RealNamePage.kt::object RN` —— 认证流私有 token
- `docs/contract/terms.md` —— 订单 12 环节 / 状态枚举(事实源)
- `docs/contract/fields.md` —— 页面列名 ↔ 字段名 ↔ 状态枚举 对齐