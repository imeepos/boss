# Android 设计稿 → 页面布局提示词：范式模板（Compose）

> 内化自 boss 项目 docs/design/design-to-prompt-template-android.md（2025-12），跨项目可复用。
> 用途：拿到移动端视觉设计稿后按本模板填写，产出可直接交给编码 AI 生成 Compose 页面的提示词。
> 与 Web 版的差异：px→dp/sp 换算、无 hover 只有按压涟漪、edge-to-edge insets、Material3 组件、真机 adb 验证。

---

## 0. 使用流程

1. **量取**：`read_image` 读设计稿，按第 2 节清单从宏观到微观提取；**先把 px 换算成 dp**。
2. **视觉转写**：**下游编码 AI 没有视觉能力，提示词就是它看到的全部设计稿**。任何一个"看图才懂"的细节没写进提示词，实现时就必然丢失或退回 material3 默认样式。逐块自问：这块的形状/背景/颜色与默认组件有什么不同？不同就必须在 D+/E 节写死。
3. **填模板**：填第 1 节 `{{ }}` 占位符；设计稿只画了浅色的，深色主题节必须自己推演补全。
3. **验证**：`./gradlew assembleDebug` + `adb install` + `uiautomator dump` 断言（见第 4 节），截图与设计稿并排比对。

**换算规则（必做）**：
- 设计稿标注 px ÷ 画布基准宽 × 160 = dp（如 1080px 稿：px ÷ 3 = dp；750px 稿：px ÷ 2 = dp）。
- 文字字号一律用 sp；尺寸/间距/圆角一律用 dp；**提示词中禁止出现裸 px**。

---

## 1. 完整模板（复制即用）

```markdown
请根据设计稿用 Jetpack Compose 实现页面，规格如下：

## A. 任务定位
- 页面名称：{{ 如：订单详情页 }}
- 页面类型：{{ 列表 / 表单 / 详情 / 底部Tab页 / 登录 }}
- 所属端：{{ mobile/user / mobile/worker，禁止跨 app 复用页面 }}
- 设计稿：{{ 附图路径；画布 {{1080×2400}}，换算系数 ÷{{3}} }}

## B. 技术栈与硬约束
- Kotlin + Jetpack Compose（BOM 2026.06.00）+ material3；minSdk 26
- 颜色一律走 ui/theme 的 MaterialTheme.colorScheme / 项目色板常量，禁止页面内硬编码 Color(0xFF...)
- 新增色值先落 ui/theme/Color.kt 并配深色值，双主题都要可读
- 单文件 ≤300 行；composable 拆分到职责单一；不使用 emoji，图标用 material-icons 或 SVG→ImageVector
- 交互动作一律 onClick = {...} 显式命名传参，禁止裸尾随 lambda（插槽绑错位不报错）
- **视觉保真红线：禁止用组件默认外观替代设计稿样式**。凡设计稿与 material3 默认渲染不同之处（底栏形状、头部背景、异形容器、渐变），必须在 D+/E 节显式写死；提示词未描述的地方按 material3 默认处理

## C. 画布基准与屏幕适配
- 设计稿：{{ 1080×2400 px，即 360×800 dp 基准 }}
- 适配策略：宽度用 fillMaxWidth + 权重（weight），禁止按固定宽写死；关键列表用 LazyColumn
- 横屏/平板：{{ 简单自适应即可 / 需双栏 }}}

## D. 整体骨架（Compose 结构树，必填）
Screen
├── Scaffold
    ├── topBar：{{ TopAppBar（标题、返回按钮）/ 无 —— 样式若非默认须在 D+ 节写死 }}
    ├── bottomBar：{{ NavigationBar（4 个 Tab）/ 无 / 自定义底栏 —— 容器色、指示器、选中态样式必须引用 D+ 节，禁止照搬默认 }}
    └── content
        ├── {{ 区块1：如 状态卡片（Column，内边距 16dp）}}
        ├── {{ 区块2：如 LazyColumn 商品行（行高 56dp）}}
        └── {{ 区块3：如 底部操作条（Row，主按钮占 weight(1f)）}}
- edge-to-edge：根布局 windowInsetsPadding(WindowInsets.safeDrawing)，内容不被状态栏/导航栏遮挡

## D+. 特殊视觉元素转写（必填，无视觉模型的唯一信息源）

> 此节描述**设计稿上与 material3 默认渲染不同的所有视觉细节**。逐项回答：形状（矩形/圆角/弧形切角/波浪）、背景（纯色/线性渐变角度与起止色/图片）、边界（描边/阴影/无边框）、装饰（圆点/插画/水印）。没有特殊元素也必须写"无特殊视觉元素，均为标准 Card 外观"，禁止留空跳过。
> **填写规则：所有数值/色值必须从设计稿量取后填入，下方 {{ }} 内只给"要回答哪些问题"的维度提示，不是示例答案——维度提示中的具体数字一律不得照抄。**

- {{ 元素1 顶部蓝色头部：背景是什么（纯色蓝色#273F70 或渐变）？什么形状（底部圆角多大16dp）？是否延伸到状态栏后方？文字什么颜色？与 TopAppBar 默认灰底的差异点 }}
- {{ 元素2 底部导航栏：容器什么颜色？有无描边/阴影？有无胶囊指示器？选中/未选中态各自样式？高度多少？与 NavigationBar 默认样式的差异点 }}
- {{ 元素3+ 其它异形/渐变/悬浮/叠层元素：位置、背景、层次关系（如某卡片上叠 marginTop -{{N}}dp） }}
- {{ 区块间重合/错位（必答，默认"所有区块自上而下平铺相接"是错的）：相邻区块是否上下重合？谁压在谁上面？重合量多少 dp？典型如"渐变头部高 134dp，首卡片顶边在 103dp 处上移压住渐变底部 31dp"。量法：在设计稿上分别取两区块边界像素行号相减再 ÷ 密度系数 }}
- {{ 卡片内子信息项配色（必答）：每列/每行的背景色（透明/浅色底/描边）与前景色（标签色、数值色、字重）逐项写死；只写文字内容不写颜色，实现必然退回默认灰底黑字 }}

## E. 区域精确规格
| 区块 | 尺寸(dp) | 背景 | 圆角 | 内边距 | 排列 |
|------|----------|------|------|--------|------|
| 顶部蓝色头部 | 高 120 / fillMaxWidth | colorScheme.primary | 16dp | 16dp | Column |
| 家庭宽带卡 | 高 80 / fillMaxWidth | colorScheme.surface | 16dp | 16dp | Column |
| 快速操作网格 | 高 80 / fillMaxWidth | colorScheme.surface | 12dp | 16dp | Grid(4) |
| 进行中订单卡 | 高 120 / fillMaxWidth | colorScheme.surface | 16dp | 16dp | Column |
| 我的服务卡 | 高 80 / fillMaxWidth | colorScheme.surface | 16dp | 16dp | Row |

## F. 色板（浅色 + 深色成对）
| 用途 | Color.kt 常量名 | 浅色值 | 深色值 |
|---|---|---:|---:|
| {{ 主色/背景/表面/边框/文字/状态色… }} | {{ BrandBlue700 }} | {{ #273F70 }} | {{ #9CB8E8 }} |

- 大面积色块（品牌头部、底栏背景）也必须在此登记色值，并在 E 节"背景"列引用；渐变写"角度 + 起止色常量名"

## G. 字体层级
| 层级 | sp/行高 | 字重 | 用途 |
|---|---|---|---|
| {{ 大标题/正文/辅助… }} | {{ 20sp/28sp }} | {{ Bold }} | {{ 页面标题 }} |

- 用 MaterialTheme.typography 定制或直接指定；数字金额可用 FontFeature

## H. 间距栅格
- 4dp 栅格：{{ 4/8/12/16/24/32/48 }}；区块间 {{ 16dp }}，列表行内 {{ 12dp }}

## I. 组件清单与规格
| 组件 | 尺寸(dp) | 圆角 | 文字 | 备注 |
|---|---|---|---|---|
| {{ Button/TextField/Card/Tab… }} | {{ 高 48 }} | {{ 8dp }} | {{ 14sp }} | {{ 触控目标 ≥48×48dp }} |

- 优先 material3 组件（Button/OutlinedTextField/Card/FilterChip）；缺口再自定义

## J. 交互状态（移动端无 hover，重点是这些）
- 按压：material3 组件自带涟漪；自定义可点区块手动 clip + clickable 加涟漪
- 选中/未选中：{{ Tab、Chip 的选中色差 }}
- 禁用：{{ 按钮 contentColor 用 colorScheme.onSurface 低透明 }}
- 加载：{{ 顶部 LinearProgressIndicator / 居中 CircularProgressIndicator }}
- 空态/错误态：{{ 图标 + 文案 + 重试按钮 }}
- 滚动：{{ LazyColumn + nestedScroll；下拉刷新是否需要 }}
- 返回：自维护导航栈必须配 BackHandler(enabled = stack.size > 1) { pop() }
- 键盘：输入页 contentPadding + imePadding()，避免输入框被软键盘遮挡

## K. 深色模式
- 设计稿只画了浅色：深色按 F 节成对色值推演，MaterialTheme 跟随系统
- 图片/插画：{{ 是否准备深色变体 }}；深色下对比度不足的辅助文字需提亮

## L. 数据契约绑定
- 字段对齐：{{ docs/contract/fields.md 关键字段 }}
- 状态枚举对齐：{{ docs/contract/terms.md }}
- API 响应：{code, data} 信封结构，解析前先 curl 核对真实响应再写模型
- mock 数据：{{ 2-3 条真实形状的样例 }}

## M. 资产清单
| 资产 | 位置 | 密度 | 用途 |
|---|---|---|---|
| {{ logo_x.xml }} | res/drawable | 矢量优先 | {{ 品牌区 }} |

规则：矢量图放 drawable（XML vector）；位图放 drawable-nodpi + 按宽度 dp 加载；禁止把大图塞进 APK 不压缩。

## N. 验收清单
1. {{ 如：详情卡片宽 fillMaxWidth、圆角 12dp、内边距 16dp }}
2. {{ D+ 节每条特殊视觉元素逐条对应：如 头部为深蓝渐变且延伸到状态栏后、底栏无胶囊指示器 }}
3. 所有可点区域 ≥48×48dp
4. 浅色/深色主题下文字对比度均可读
5. 真机返回键按预期 pop 而非退出 App
6. 弱网/空数据/加载三态不塌布局
7. 真机截图与设计稿并排比对，差异以"应为 X 而非 Y"反馈迭代
```

---

## 2. 量取信息检查清单（Android 特有项加粗）

1. **骨架**：底不底部 Tab？顶栏有无返回/动作按钮？→ 先定 Scaffold 结构
2. **与默认组件逐项对比**：设计稿每个区块与 material3 默认渲染有什么不同（容器色/指示器/圆角/渐变/异形）？每一处不同都写进 D+ 节 —— **这是无视觉模型唯一能感知差异的途径**，漏一条实现就丢一处
2. **分区**：内容区几块？滚动还是固定？
3. **px→dp 换算**：先问设计稿基准宽（1080/750/375），全部换算完再填
4. **组件**：每块的 material3 对应物（Card/Button/Chip/TextField/Tab）
5. **纵轴节奏**：区块间距、行高归一 4dp 栅格
6. **字阶**：sp + 字重；标题/正文/辅助三档起步
7. **色板**：**浅色 + 深色成对提取**；状态色（成功/警告/危险）齐全；卡片内每个子信息项的背景/前景色逐项登记
8. **区块重合**：**逐对检查相邻区块边界像素**，凡有上下重合/压叠（如首卡压住渐变底部 N dp）必须量出重合量写进 D+ 节，禁止默认平铺相接
9. **触控**：**所有可点元素 ≥48×48dp**，设计稿画小了要在提示词里放大
9. **状态**：按压/选中/禁用/加载/空/错误；**软键盘弹出后的布局**
10. **insets**：状态栏、导航栏、输入法三处遮挡都要在 D 节声明处理方式

---

## 3. 本项目 Android 专用约束块（粘贴到 B 节）

```markdown
- 工程：mobile/user/android 或 mobile/worker/android（独立 Gradle 工程，不跨 app 复用页面实现）
- 技术栈：Kotlin 2.1.21 + Compose BOM 2026.06.00 + material3；minSdk 26；JDK 17（/opt/homebrew/opt/openjdk@17）
- 颜色落 ui/theme/Color.kt，主题走 ui/theme/Theme.kt 的 MaterialTheme.colorScheme
- 列表字段/状态枚举先查 docs/contract/fields.md 与 docs/contract/terms.md，禁止自造字段名
- onClick 显式命名传参；BackHandler 配自维护导航栈；根布局处理 WindowInsets.safeDrawing
- 门禁：./gradlew assembleDebug 通过 + 真机 adb 安装验证；完成后 git commit
```

---

## 4. 验证与迭代（区别于 Web 的 cdp-capture）

```bash
# 构建 + 安装
./gradlew assembleDebug && adb install -r app/build/outputs/apk/debug/app-debug.apk

# 截图与设计稿对比
adb exec-out screencap -p > screen.png   # 然后用 read_image 对比

# UI 断言（点前 dump 取坐标 → input tap → 再 dump 断言文本）
adb shell uiautomator dump && adb shell input tap <x> <y>
adb shell input keyevent 4   # 返回键
```

- 差异反馈同样写"应为 X 而非 Y"：`卡片圆角应为 12dp 而非 8dp；主按钮高度应为 48dp`。
- 数据加载稳定后再取坐标（加载中 bounds 会漂移）。
- 收敛后把定稿规格回写 `docs/design/`，范式对齐 `visual-design-prompts.md`。
