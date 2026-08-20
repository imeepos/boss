

```
根据：docs/design/design-to-prompt-template-android.md
结合：mobile/user/design/user-home.png
生成提示词
```

```
请根据设计稿用 Jetpack Compose 实现页面，规格如下：

## A. 任务定位
- 页面名称：用户首页（UserHomeScreen）
- 页面类型：底部 Tab 页
- 所属端：mobile/user
- 设计稿：mobile/user/design/user-home.png，画布 1080×2400 px，换算系数 ÷3（360×800 dp 基准）

## B. 技术栈与硬约束
- Kotlin + Jetpack Compose（BOM 2026.06.00）+ material3；minSdk 26
- 颜色一律走 ui/theme 的 MaterialTheme.colorScheme / 项目色板常量，禁止页面内硬编码 Color(0xFF...)
- 新增色值先落 ui/theme/Color.kt 并配深色值，双主题都要可读
- 单文件 ≤300 行；composable 拆分到职责单一；不使用 emoji，图标用 material-icons 或 SVG→ImageVector
- 交互动作一律 onClick = {...} 显式命名传参，禁止裸尾随 lambda

## C. 画布基准与屏幕适配
- 设计稿：1080×2400 px，即 360×800 dp 基准
- 适配策略：宽度用 fillMaxWidth + 权重（weight），禁止按固定宽写死；关键列表用 LazyColumn
- 横屏/平板：简单自适应即可

## D. 整体骨架（Compose 结构树，必填）
Screen
├── Scaffold
    ├── topBar：无（纯内容页）
    ├── bottomBar：NavigationBar（4 个 Tab：首页、服务、账单、我的）
    └── content
        ├── {{ 头部问候 + 服务状态卡片（Column，内边距 16dp） }}
        ├── {{ 快速操作网格（4 宫格图标按钮，行高 80dp） }}
        ├── {{ 进行中订单卡片（LazyColumn，单行） }}
        └── {{ 我的服务卡片（Row + 图标） }}

- edge-to-edge：根布局 windowInsetsPadding(WindowInsets.safeDrawing)，内容不被状态栏/导航栏遮挡

## E. 区域精确规格
| 区块 | 尺寸(dp) | 背景 | 圆角 | 内边距 | 排列 |
|------|----------|------|------|--------|------|
| 头部问候卡 | 高 120 / fillMaxWidth | colorScheme.surface | 16dp | 16dp | Column |
| 快速操作网格 | 高 80 / fillMaxWidth | colorScheme.surface | 12dp | 16dp | Grid(4) |
| 进行中订单卡 | 高 120 / fillMaxWidth | colorScheme.surface | 12dp | 16dp | Column |
| 我的服务卡 | 高 80 / fillMaxWidth | colorScheme.surface | 12dp | 16dp | Row |

## F. 色板（浅色 + 深色成对）
| 用途 | Color.kt 常量名 | 浅色值 | 深色值 |
|------|-----------------|--------|--------|
| 主色 | BrandBlue700 | #273F70 | #9CB8E8 |
| 表面 | colorScheme.surface | #FFFFFF | #1E1E1E |
| 文字主 | colorScheme.onSurface | #1C1C1C | #FFFFFF |
| 成功 | colorScheme.primary | #4CAF50 | #4CAF50 |
| 辅助 | colorScheme.tertiary | #9E9E9E | #9E9E9E |

## G. 字体层级
| 层级 | sp/行高 | 字重 | 用途 |
|------|--------|------|------|
| 大标题 | 20sp/28sp | Bold | 早上好，XXX |
| 正文 | 16sp/24sp | Medium | 订单号/地址 |
| 辅助 | 14sp/20sp | Regular | 状态/时间 |

## H. 间距栅格
- 4dp 栅格：16dp 为主；区块间 16dp，列表行内 12dp

## I. 组件清单与规格
| 组件 | 尺寸(dp) | 圆角 | 文字 | 备注 |
|------|----------|------|------|------|
| 卡片 | 高 80/120 | 16dp | 16sp | 触控目标 ≥48×48dp |
| 图标按钮 | 宽 72 | 圆角 12dp | 20sp | 图标 + 文字 |
| 进度条 | 宽 200 | — | — | 装维进度 |

## J. 交互状态（移动端无 hover，重点是这些）
- 按压：material3 组件自带涟漪
- 选中/未选中：无
- 禁用：无
- 加载：无
- 空态/错误态：无
- 滚动：LazyColumn + nestedScroll；无下拉刷新
- 返回：自维护导航栈必须配 BackHandler(enabled = stack.size > 1) { pop() }
- 键盘：无输入页

## K. 深色模式
- 设计稿只画了浅色：深色按 F 节成对色值推演，MaterialTheme 跟随系统
- 图片/插画：无

## L. 数据契约绑定
- 字段对齐：docs/contract/fields.md（订单号 OrderNo、订单状态 status、地址 address_id、产品 offer_id 等）
- 状态枚举对齐：docs/contract/terms.md（订单状态 PENDING/RESERVED/INSTALLING/DONE，环节 8 派单）
- API 响应：{code, data} 信封结构，解析前先 curl 核对真实响应再写模型
- mock 数据：3 条真实形状的样例（订单 ORD-20260818001、地址 广东省深圳市南山区科技园、进度 8/12 等）

## M. 资产清单
无

## N. 验收清单
1. 头部问候卡文字 “早上好，XXX” + 手机号 + 状态绿点
2. 快速操作网格 4 个图标按钮（办套餐、查订单、缴费用、报故障等），每个按钮 ≥48×48dp
3. 进行中订单卡：订单号、产品、地址、上门安装进度条、状态标签
4. 我的服务卡：家庭宽带 1000M + 在网绿点
5. 所有可点区域 ≥48×48dp
6. 浅色/深色主题下文字对比度均可读
7. 真机返回键按预期 pop 而非退出 App
8. 弱网/空数据/加载三态不塌布局
```

我将提示词让没有视觉能力的模型去实现
发现底部菜单跟设计稿完全不一致
顶部的蓝色头部的特殊设计也没有了