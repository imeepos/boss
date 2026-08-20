请根据设计稿用 Jetpack Compose 实现页面，规格如下：

## A. 任务定位
- 页面名称：用户首页
- 页面类型：底部Tab页
- 所属端：mobile/user
- 设计稿：mobile/user/design/user-home.png，画布 1080×1536 px，换算系数 ÷3

## B. 技术栈与硬约束
- Kotlin + Jetpack Compose（BOM 2026.06.00）+ material3；minSdk 26
- 颜色一律走 ui/theme 的 MaterialTheme.colorScheme / 项目色板常量，禁止页面内硬编码 Color(0xFF...)
- 新增色值先落 ui/theme/Color.kt 并配深色值，双主题都要可读
- 单文件 ≤300 行；composable 拆分到职责单一；不使用 emoji，图标用 material-icons 或 SVG→ImageVector
- 交互动作一律 onClick = {...} 显式命名传参，禁止裸尾随 lambda（插槽绑错位不报错）
- **视觉保真红线：禁止用组件默认外观替代设计稿样式**。凡设计稿与 material3 默认渲染不同之处（底栏形状、头部背景、异形容器、渐变），必须在 D+ 节写死；提示词未描述的地方按 material3 默认处理

## C. 画布基准与屏幕适配
- 设计稿：1080×1536 px，即 360×512 dp 基准
- 适配策略：宽度用 fillMaxWidth + 权重（weight），禁止按固定宽写死；关键列表用 LazyColumn
- 横屏/平板：简单自适应即可

## D. 整体骨架（Compose 结构树，必填）
Screen
├── Scaffold
    ├── topBar：无
    ├── bottomBar：自定义底栏（蓝色背景，无胶囊指示器，4个Tab图标蓝色选中态）—— 容器色、指示器、选中态样式必须引用 D+ 节，禁止照搬默认
    └── content
        ├── 头部渐变卡片（蓝色渐变延伸状态栏后方）
        ├── 家庭宽带卡片
        ├── 8宫格卡片
        ├── 进行中订单卡片
        ├── 我的服务卡片
        └── 底部导航栏
- edge-to-edge：根布局 windowInsetsPadding(WindowInsets.safeDrawing)，内容不被状态栏/导航栏遮挡

## D+. 特殊视觉元素转写（必填，无视觉模型的唯一信息源）
> 此节描述**设计稿上与 material3 默认渲染不同的所有视觉细节**。逐项回答：形状（矩形/圆角/弧形切角/波浪）、背景（纯色/线性渐变角度与起止色/图片）、边界（描边/阴影/无边框）、装饰（圆点/插画/水印）。没有特殊元素也必须写"无特殊视觉元素，均为标准 Card 外观"，禁止留空跳过。
> **填写规则：所有数值/色值必须从设计稿量取后填入，下方 {{ }} 内只给"要回答哪些问题"的维度提示，不是示例答案——维度提示中的具体数字一律不得照抄。**

- 头部：线性渐变蓝色背景（从深蓝#1E3A8A 到浅蓝#3B82F6，垂直起止色），无圆角，延伸到状态栏后方，无文字描边，文字纯白色，大标题"早上好，张先生" 28sp/32sp Bold，副标题"138****8888" 16sp，状态行绿色圆点"服务在线·网络正常" 14sp，无阴影
- 底部导航栏：蓝色背景#0A84FF，无胶囊指示器，4个图标（首页蓝色选中，服务灰色，未选中灰色），高度48dp，选中态图标白色，文字蓝色选中
- 卡片：白色背景，圆角16dp，轻微阴影（0.5dp offset），内边距16dp，无描边
- 在网标签：绿色背景#34C759，白色文字，圆角8dp，高度24dp
- 进度条：蓝色#007AFF，高度8dp，无圆角，线性填充
- 图标：圆角0dp，尺寸48dp，特定颜色（办套餐蓝色，查订单绿色，缴费橙色，报故障紫色，充值蓝色，查用量绿色，消息橙色，客服紫色），无描边
- 无其他特殊元素（所有卡片均为标准 Material Card 外观，无额外装饰）

## E. 区域精确规格
| 区块 | 尺寸(dp) | 背景 | 圆角 | 内边距 | 排列 |
|------|----------|------|------|--------|------|
| 头部 | fillMaxWidth × 180 | linearGradient(角度135°, #1E3A8A to #3B82F6) | 0dp | 0dp | Column (vertical center, padding 24dp) |
| 家庭宽带卡 | fillMaxWidth × 140 | white | 16dp | 16dp | Column + 3 sub Row |
| 8宫格卡 | fillMaxWidth × 160 | white | 16dp | 16dp | Grid 2x4 |
| 进行中订单 | fillMaxWidth × 120 | white | 16dp | 16dp | Row (progress bar) |
| 我的服务 | fillMaxWidth × 100 | white | 16dp | 16dp | Row |
| 底部导航 | fillMaxWidth × 56 | blue #0A84FF | 0dp | 0dp | Row (center) |

## F. 色板（浅色 + 深色成对）
| 用途 | Color.kt 常量名 | 浅色值 | 深色值 |
|------|----------------|--------|--------|
| 主背景/头部 | BrandBlueGradient | linearGradient(#1E3A8A to #3B82F6) | linearGradient(#0F172A to #1E40AF) |
| 在网/成功 | Green500 | #34C759 | #30D158 |
| 文字主 | OnSurface | #1C1C1E | #F2F2F7 |
| 文字副 | OnSurfaceVariant | #3F3F46 | #A1A1AA |
| 卡片背景 | Surface | #FFFFFF | #1C1C1E |
| 底部导航 | NavigationBar | #0A84FF | #0A84FF |
| 进度条 | Primary | #007AFF | #60A5FA |

## G. 字体层级
| 层级 | sp/行高 | 字重 | 用途 |
|------|--------|------|------|
| 大标题 | 32sp/40sp | Bold | "早上好，张先生" |
| 次标题 | 20sp/24sp | Medium | "家庭宽带 1000M" |
| 金额 | 32sp/40sp | Bold | "¥89.00" 等 |
| 正文 | 16sp/20sp | Regular | 地址、状态、时间 |
| 辅助 | 14sp/16sp | Regular | 标签、按钮文案 |

## H. 间距栅格
- 4dp 栅格：16dp 区块间距，列表行内 12dp，金额行间 8dp
- 头部 padding 24dp top
- 宫格间距 12dp
- 卡片内边距 16dp

## I. 组件清单与规格
| 组件 | 尺寸(dp) | 圆角 | 文字 | 备注 |
|------|----------|------|------|------|
| TopAppBar | 48dp | 0dp | 18sp | 无 |
| NavigationBar | 56dp | 0dp | 12sp | 自定义蓝色，无指示器 |
| Card | 任意 | 16dp | 16sp | 白色背景 |
| Button | 48dp | 8dp | 16sp | 蓝色 |
| LinearProgressIndicator | 8dp | 0dp | — | 蓝色 |
| Text | 任意 | — | sp | 按层级 |
| Icon | 48dp | 0dp | — | 按颜色 |
| Gradient | 任意 | — | — | 用于头部 |

## J. 交互状态（移动端无 hover，重点是这些）
- 按压：material3 组件自带涟漪；自定义可点区块手动 clip + clickable 加涟漪
- 选中/未选中：Tab 选中蓝色图标/文字，未选中灰色
- 禁用：按钮 contentColor 用 colorScheme.onSurface 低透明
- 加载：顶部 LinearProgressIndicator / 居中 CircularProgressIndicator
- 空态/错误态：无
- 滚动：LazyColumn + nestedScroll；下拉刷新是否需要（无）
- 返回：自维护导航栈必须配 BackHandler(enabled = stack.size > 1) { pop() }
- 键盘：输入页 contentPadding + imePadding()，避免输入框被软键盘遮挡

## K. 深色模式
- 设计稿只画了浅色：深色按 F 节成对色值推演，MaterialTheme 跟随系统
- 图片/插画：无图片；深色下对比度不足的辅助文字需提亮（OnSurfaceVariant 提亮为 #E2E2E2）
- 头部渐变在深色下调整为深蓝系

## L. 数据契约绑定
- 字段对齐：docs/contract/fields.md 关键字段（customer name, phone, planName, orderNo 等）
- 状态枚举对齐：docs/contract/terms.md （在网=ACTIVE 等）
- API 响应：{code, data} 信封结构，解析前先 curl 核对真实响应再写模型
- mock 数据：2-3 条真实形状的样例（"张先生", "138****8888", "家庭宽带 1000M", "ORD-20260818001", "装维中", "8/12", "2026-12-31" 等）

## M. 资产清单
| 资产 | 位置 | 密度 | 用途 |
|------|------|------|------|
| home-header-gradient.xml | res/drawable | 矢量优先 | 头部渐变背景 |

## N. 验收清单
1. 头部渐变延伸状态栏后方，无 TopAppBar
2. 底部导航无胶囊指示器，4个图标蓝色选中
3. 所有卡片圆角16dp，白色背景，轻微阴影
4. 在网标签绿色圆角8dp
5. 进度条蓝色8dp
6. 所有文字颜色对比度可读（浅色/深色）
7. 真机返回键按预期 pop 而非退出 App
8. 弱网/空数据/加载三态不塌布局
9. 真机截图与设计稿并排比对，差异以"应为 X 而非 Y"反馈迭代
10. 所有尺寸均换算成dp，禁止裸px