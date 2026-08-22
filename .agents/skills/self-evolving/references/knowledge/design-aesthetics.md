# 设计美学参考手册（Design Aesthetics Reference）

> 提炼自 Stripe / Linear / Vercel / Bloomberg Terminal 等顶级设计语言（design-bites 仓库 + 调研 2026-08）。
> 用途：每次画设计稿前翻一遍，把"高级感"变成可执行的克制法则，避免产出"中规中矩"的流水线稿。

---

## 0. 一句话核心

**高级感 = 在对的维度上克制，在对的维度上极致。** 中规中矩是 5 个维度都打 5 分；
真正的设计是 8/8/8/3/8 或 3/9/8/8/9——把克制留给颜色/圆角/装饰，把极致留给节奏/字号/留白。

---

## 1. Linear 的"510/590"哲学（最值得抄的字体纪律）

Linear 整个系统只有三个字重：400（正文）/ 510（强调）/ 590（强强调）。**完全没有 600 / 700。**

```
常规 400 → 510 (regular+) → 590 (medium+) → ❌ 600/700
```

为什么：510 在 400 和 500 之间，强调感"耳语但清晰"；590 在 500 和 600 之间，强强调但不粗重。
整套系统的层级靠"在标准 weight 之间打点"实现，而不是用粗体堆砌。

**项目可执行的简化版**：把字号层级收紧到 5-6 档、字重只用 3 档（400/500/600），不要 700 标题满天飞。

字号规则：**字号越大 → letter-spacing 越负**（视觉补偿）
- 16px 正文：`normal` (0)
- 20px h3：`-0.24px`
- 32px h2：`-0.64px`
- 48px h1：`-0.96px`
- 64px hero：`-1.4px`

OpenType 特性：UI 文本开启 `cv01` / `ss03` / `ss01` 等 stylistic set，让字形更精致；代码块禁用这些。

---

## 2. Stripe 的"weight 300" 反叛（编辑式克制）

Stripe 全部 headline 用 `font-weight: 300`（细体），h1/h2/h3 都是 light。**完全反 SaaS 主流的 700 黑粗体。**

为什么：粗体是"怕你没看见我"；细体是"我相信你能读懂"。配上 -0.96px 的负字距，
让标题看起来像 Wallpaper* 杂志，不像 SaaS landing page。

**给项目的启发**：
- 暗色背景下大标题不要用 700，用 500-600 + 字距收紧
- 亮色背景下的 hero/品牌头部可以用 500-600 + 显著负字距
- 永远避免"通栏大 700 黑标题"——这是"廉价"的最强信号

---

## 3. Vercel 的"用阴影代替边框"（box-shadow border）

Vercel 不用 `border: 1px solid`，统一用 `box-shadow: 0 0 0 1px rgba(0,0,0,0.08)`。

```css
/* 错的 */
.card { border: 1px solid #E5EDF5; }

/* 对的 */
.card { box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.08); }
```

为什么：
1. **不占盒模型**——box-shadow 在 box 之外，不会撑大宽度（无需 border-box）
2. **可叠加**——多层阴影一句话叠加，border 只能用 outline/nested 模拟
3. **过渡平滑**——border-color 切换可能重排，shadow 动画永远丝滑
4. **更精细的视觉**——rgba 比 hex 颜色更易调和透明度

**给我项目的具体修法**：把当前 `border-[var(--shell-card-border)]`（实际是 transparent）替换为
`box-shadow: 0 0 0 1px var(--shell-card-border)`，在边框可见的地方更精致。

---

## 4. Linear 的"近黑分层"（luminance hierarchy）

Linear 整个 surface 体系在 RGB 亮度 0~0.089 之间：

```
Base:   rgb(8, 9, 10)    #08090A  luminance 0.035
Frame:  rgb(9, 10, 11)   #090A0B  luminance 0.038
Card:   rgb(15, 16, 17)  #0F1011  luminance 0.062
View:   rgb(18, 19, 20)  #121314  luminance 0.074
Chat:   rgb(22, 23, 24)  #161718  luminance 0.089
Divider: rgb(35, 37, 42) #23252A  luminance 0.145
```

**相邻层只差 1-3 个 RGB 值**。这就是"高级感"的源头——层次靠**亮度的微变化**，不靠夸张对比。

我项目当前 `--shell-card-bg: #10203F` 在 dark 主题下与 `--shell-content-bg: #0A1528` 的差是：
- R: 16-10 = 6
- G: 32-21 = 11
- B: 63-40 = 23

差得有点大，对比显得"硬"。**可以试试把卡片改成 `rgb(22, 28, 50)` 这种更接近的层**，层次会更细腻。

---

## 5. Linear 的"ring shadow"（容器轮廓阴影）

```
rgba(0, 0, 0, 0.33) 0px 0px 0px 1px
```

零偏移、零模糊、1px 扩散——视觉上就是 1px 边框，但因为是 shadow 可以叠加/动画。

**对照项目**：当前 token 里只有 `--shadow-card` (大投影)、`--shell-card-shadow` (亮色微投影)。
缺少中层的"ring"和"elevated"分层级。

**建议补全一套**：
```css
/* 容器 ring：代替 border */
--shadow-ring: 0 0 0 1px rgba(15, 30, 59, 0.08);          /* 亮 */
--shadow-ring-dark: 0 0 0 1px rgba(255, 255, 255, 0.08);  /* 暗 */

/* 悬浮：popover/dropdown */
--shadow-elevated: 0 0 0 1px rgba(15,30,59,0.06), 0 8px 24px rgba(15,30,59,0.12);

/* 模态：modal/dialog */
--shadow-modal: 0 0 0 1px rgba(15,30,59,0.08), 0 24px 64px rgba(15,30,59,0.18);
```

---

## 6. Vercel 的"double-ring focus"（焦点圈设计）

```css
--ds-focus-ring:
  0 0 0 2px var(--ds-background-100),   /* 白色内圈，制造"呼吸" */
  0 0 0 4px var(--ds-focus-color);       /* 蓝色外圈，识别焦点 */
```

两层焦点圈：内白 + 外蓝。即使按钮本身是蓝色，外圈仍清晰可见。

**给我项目的修法**：当前 `--shell-input-focus-shadow: 0 0 0 3px rgba(39, 63, 112, 0.14)` 是单层。
可以升级为双层（`0 0 0 2px var(--shell-input-bg), 0 0 0 4px var(--shell-input-border-focus)`），
让蓝色焦点圈在所有背景上都清晰。

---

## 7. Stripe 的"无 shadow"哲学（用对比代替阴影）

Stripe 首页零 CSS 阴影。深度靠：
1. 白卡 vs `#F8FAFD` 微差异背景
2. 蓝灰边框 `#E5EDF5`
3. 磨砂叠加层 `rgba(248, 250, 253, 0.45)`

**启发**：高级感不一定需要阴影——当整体够克制时，层次靠"几乎看不见的色差"自然涌现。
**项目应用**：亮色主题卡片可以减少 `--shell-card-shadow`（亮色当前已经有微投影），改用更柔和的 0.5% 透明度投影。

---

## 8. "色彩比例 80/15/5" 纪律（来自用户端 UI-PARADIGM.md，已落地）

```
中性色（白/灰/黑）        ≥ 80%  — 主背景、卡片、文字
品牌色（蓝/金/紫）        10-15%  — 主按钮、选中态、价格、当前进度
状态色（绿/红/橙/黄）      ≤ 5%  — 成功/失败/警告/提醒
```

**反例**：每张卡片都有蓝色徽章 + 绿色状态 + 橙色提示 + 紫色图标 = 视觉灾难。

**启发**：每当我想加一个彩色元素，问自己"这是 80/15/5 的 5% 部分吗？"如果是，删掉或合并。

---

## 9. "三级表面" 纪律（同上，已落地）

```
页面底（最暗/最浅）  →  卡片（白/近白）  →  内嵌模块（极浅蓝灰）
```

不要在卡片内又铺一层白卡（双白冲突）；不要在页面底直接放数据图（无容器感）。

---

## 10. 节奏感（最重要但最难量化）

**节拍 = 视觉呼吸点。** 一个页面读起来"闷"还是"舒服"，80% 取决于节拍。

节拍三件套：
1. **字号跳跃要敢跳** — 不要 14/15/16/17，要 14/18/24 这种跳跃
2. **留白要敢空** — 主标题上下至少 32px，正文行高 ≥ 1.5
3. **元素分组要疏** — 区块之间 32-48px，组内 8-16px，绝不能 24/24 平均

**反例检测**：把设计稿缩到 25%，眯眼看：
- 能找到 3 个明确"组"吗？（❌ 平均用力 → 节奏差）
- 每组之间有明显空白吗？（❌ 全挤一起 → 节奏差）
- 一眼能找到第一焦点吗？（❌ 5 个并列 → 节奏差）

---

## 11. 微质感：装饰性细节（让设计"有东西"）

高级感不是宏大叙事，是一连串细节累积：

| 细节 | 作用 | 项目落地 |
|---|---|---|
| 数值/金额用 tabular-nums | 数字对齐，账单/订单场景必要 | `font-feature-settings: "tnum"` |
| 副标题用 upper + letter-spacing + 小字号 | 制造"分类标签"感 | `text-[10px] tracking-[0.1em] uppercase text-muted-foreground` |
| 时间/状态用 mono 字体 | 信息流风格，专业感 | `font-mono` 局部使用 |
| Icon 16-20px 配文字 14px（不超 1.5x） | 视觉重量平衡 | 默认 16px，密度高场景 14px |
| 主按钮加 `active:scale-[0.98]` | 按压反馈 | shadcn Button 已配 transition-colors，可补 transform |
| 空状态图标用 outline + 2px stroke | 与系统图标库一致 | 引用 lucide 图标，不自创 |
| 价格/数字加 `font-variant-numeric: tabular-nums` | 等宽数字 | 订单/账单页面关键 |

---

## 12. 信息密度（不是越空越好）

**常见误解**：高级 = 留白多 = 元素少。
**真相**：高级 = 每个元素都有用 = 元素之间呼吸正确。

Bloomberg Terminal 被称为高级感的极端案例——它**信息密度极高**，但每一像素都有信息。
秘诀：
- 数字用等宽字体，垂直对齐
- 边框细但清晰（hairline）
- 颜色只用在状态变化上

**项目应用**：数据密集页（订单/工单/计费）可以做得"满"，但每个区块之间用 24-32px 呼吸。
空白页面（设置/我的）反而要克制，每个元素之间 16-24px。

---

## 13. Stripe 的"零 border-radius 大值" 哲学

Stripe 圆角只到 6px，没有 12/16/20/999 的大圆角。**这是反"现代设计"主流的**。

为什么要克制：
- 12px+ 大圆角 = "友好/亲民/可爱"（移动端 App 风格）
- 4-6px 小圆角 = "专业/克制/工程感"（桌面端 SaaS 风格）

**项目建议**：当前 `--radius-lg: 14px` 用于卡片——这其实是"卡片轻量化"的移动端风格。
如果是 admin 后台（更专业场景），可以把卡片圆角降到 8px（= `--radius-md`）。

---

## 14. 留白公式（数值化）

参考 Material Design 3 与 Vercel：

```
页面 padding（左右）   24px      ≥ 16dp
卡片 padding           20px      ≥ 16dp
区块 gap               24px      紧凑 16px，宽松 32px
组内元素 gap           12-16px   紧凑 8px，宽松 24px
标题与副标题 gap       8-12px
主标题与正文 gap       24-32px
按钮 padding (左右)    16-20px
按钮 padding (上下)    8-10px   高度 36-40px
输入 padding (左右)    12-14px
输入 padding (上下)    8-10px   高度 36-40px
图标与文字 gap         6-8px
```

**节拍检测**：相邻元素间距差 ≥ 1.5 倍才"看出节奏"。8/12/16/24/32 是好节奏，6/8/12/16 是平庸节奏。

---

## 15. 高级感陷阱清单（避坑）

| 陷阱 | 为什么 | 替代方案 |
|---|---|---|
| 通栏红色退出按钮 | 视觉抢戏，破坏层级 | 白底 + 红色描边 + 小图标 |
| 满屏 emoji 图标 | 廉价、卡通 | 线性 SVG，统一 1.8 stroke |
| 多色 Tag 混用 | 视觉混乱 | 单一描边浅底样式 + 4 种语义色 |
| 大阴影深色投影 | 过时、过重 | hairline 边框 + 微透明阴影 |
| 12px+ 大圆角卡片 | 移动端风格，错位 | 8px 圆角，专业克制 |
| 700 黑标题 + 大字 | "我很重要" | 500/600 + 负字距 + 留白 |
| 全屏渐变背景 | 廉价 | 单一纯色 + 极淡装饰 |
| 配色五颜六色 | 视觉灾难 | 80/15/5 比例纪律 |
| 一屏塞完所有信息 | 信息过载 | 分屏 + 滚动 + 分组 |
| 不同圆角混用 | 体系崩 | 单一圆角阶梯（4/6/8） |

---

## 16. 给 gpt-image-2 的"高级感 prompt 模板"

把第 1-15 节翻译成生图 prompt 的 Style 段：

```
Style: editorial, restrained, professional, polished, technical luxury,
       like Stripe or Linear, no cartoonish elements

Tokens: 
- canvas: #FAFAFA / #0A0A0A
- surface: #FFFFFF / #141414
- text: #171717 / #F0F0F0
- accent: #0072F5 (links/CTAs only, ≤15% of screen)
- border: rgba(0,0,0,0.08) hairline, no shadows >10% opacity
- font: Inter / Geist / SF Pro Display, weight 500-600 headings, -1% letter-spacing on >24px

Restrictions:
- no 700-weight headings
- no border-radius >10px
- no status colors as fills, only as 8px dots
- no emoji icons
- no full-screen gradients
- no multi-color tags (single style, semantic colors only)
- whitespace rhythm: 8/12/16/24/32 only
```

---

## 17. 项目专属的"应该做但还没做"清单

画稿前必看：

- [ ] 卡片用 `box-shadow: 0 0 0 1px rgba(...)` 代替 border（Vercel 风格）
- [ ] 焦点用 double-ring（白色内圈 + 蓝色外圈）
- [ ] 暗色主题分层级：Base / Frame / Card / View / Divider 各差 5-10 RGB（Linear 风格）
- [ ] 标题字重 ≤ 600，大标题加负字距（Stripe 风格）
- [ ] 主按钮加 `active:scale-[0.98]` 微反馈
- [ ] 价格/数字加 `font-variant-numeric: tabular-nums`
- [ ] 副标签用 uppercase + letter-spacing + 10-11px
- [ ] Tag 统一描边浅底，禁止实心高饱和（已部分落地）
- [ ] 圆角阶梯 ≤ 8px（已是 `--radius-md`）
- [ ] 卡片间距 ≥ 16px，避免 8px 紧凑堆叠
- [ ] 节拍检测：缩 25% 后能找出 3 个明确组

---

## 18. 来源

- [design-bites/design-mds/linear.app/DESIGN.md](https://github.com/educlopez/design-bites/blob/main/design-mds/linear.app/DESIGN.md)
- [design-bites/design-mds/vercel.com/DESIGN.md](https://github.com/educlopez/design-bites/blob/main/design-mds/vercel.com/DESIGN.md)
- [design-bites/design-mds/stripe.com/DESIGN.md](https://github.com/educlopez/design-bites/blob/main/design-mds/stripe.com/DESIGN.md)
- [Linear · Craft](https://linear.app/now/craft)
- [Linear · Why is quality so rare?](https://linear.app/now/why-is-quality-so-rare)
- [huashu-design skill](https://github.com/alchaincyf/huashu-design)
- [UI 设计的高级感 - 优设](https://www.uisdc.com/ui-high-level-sense)
- [提升UI设计视觉层级 - hxsd](https://www.hxsd.com/information/24720/)
- 项目内部：designs/UI-PARADIGM.md、designs/UI-SPEC.md、designs/FEEDBACK.md