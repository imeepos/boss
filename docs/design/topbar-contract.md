# TopBar 两端共享契约（用户端/师傅端）

- 目的：两端二级页共用同名 `TopBar` 组件，防"改一端忘一端"；本契约由两端实现反推成文（零视觉变化，方案 B），只登记现状，不改任何代码
- 路径约定同 `mobile-ui-standard.md`：`u/` = `mobile/user/android/app/src/main/java/com/ymm/boss/user`，`w/` = `mobile/worker/android/app/src/main/java/com/ymm/boss/worker`
- 证据：`u/ui/Widgets.kt:84-97`、`w/ui/Widgets.kt:44-66` 实测复核（2026-08-28）

## 1. 共同契约（两端一致，改任一处必须评估另一端）

- 签名：`TopBar(title: String, onBack: (() -> Unit)? = null, action: String? = null, onAction: (() -> Unit)? = null)`
- 行为语义：`onBack == null` 不渲染返回键；`action == null` 不渲染右侧动作；`onAction` 允许为 null（点击为空操作）
- 布局：标题 `weight(1f)` 占满中段、左对齐；横向 padding 16dp；`CenterVertically` 垂直居中
- 文字：标题 16sp 白字加粗（u W600 / w SemiBold，后者属 standard §3.4 收编存量）；右侧动作 13sp 白字

## 2. 端内视觉参数（各自锁定，跨端不互改）

| 参数 | 用户端 u/ | 师傅端 w/ |
|------|-----------|-----------|
| 高度 | 固定 48dp | 无固定高度，vertical 12dp padding 撑高 |
| 背景 | `statusBarSolid()` 纯色 0xFF006AE5（`u/ui/Theme.kt:78`，standard §2.4 登记异常） | `linearGradient(Primary → Primary2)` = 0xFF1677FF → 0xFF69B1FF（`w/ui/theme/Color.kt:6-7`） |
| 返回键 | "‹" 20sp 白字 | "<" 16sp 白字 + 4dp padding |
| 中段间距 | 标题 `padding(start = 8dp)` | `spacedBy(12dp)` |

## 3. 变更规则

- 改任一端的签名、高度、字号、背景之一，必须先同步更新本契约，再编译对端确认可构建、无语义破坏
- 禁止单端新增必选参数或改变既有参数含义；新增可选参数须先在本契约登记默认值与两端语义
- 第 2 节端内视觉参数跨端不互改：背景分叉沿用 standard §2.3"异端异值需裁决"口径，业务迭代中不得顺手拉齐

## 4. 与 mobile-ui-standard 的引用关系

- §1.3（二级页统一共享 TopBar，禁止页面自造顶栏）：本契约是该规则的组件级登记，组件实现以两端 `ui/Widgets.kt` 为准
- §2.3 / §2.4（primary 异端异值、statusBarSolid 第三蓝登记）：本契约第 2 节"背景"行即其体现；色值归属与收编裁决以 standard 为准，本契约只随动登记
