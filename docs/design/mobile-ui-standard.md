# 移动端 UI 一致性标准（用户端/师傅端 Android）

- 任务编号：P0-2（UI 一致性治理）。适用范围：`mobile/user`、`mobile/worker` Android（Kotlin + Jetpack Compose）
- 效力分层：第 1/2/5 节为硬规则（可机械判定）；第 3 节为参考终态（收编时对齐，不要求一次性全量回改）；第 4 节为挂起与豁免登记
- 路径约定：`u/` = `mobile/user/android/app/src/main/java/com/ymm/boss/user`，`w/` = `mobile/worker/android/app/src/main/java/com/ymm/boss/worker`
- 证据口径：文件与行号基于 worktree `feat/ui-consistency-p0`（b51e54cd）grep 实测复核，不确定项显式标"待补验"

## 1. 页面骨架双层规则

一级页按内容形态分两类骨架；二级页统一共享顶栏。判定看页面引用的骨架/顶栏组件，不依赖主观审美。

### 1.1 一级页·浏览型 tab：PinnedGradientPage 渐变骨架

| 端 | 适用 tab | 证据（调用点） |
|----|----------|----------------|
| 用户端 | 首页、我的 | `u/page/UserHomePage.kt`、`u/page/ProfilePage.kt` 引用 `u/ui/PinnedGradientPage.kt` |
| 师傅端 | Home、Profile | `w/ui/HomeScreen.kt`、`w/ui/ProfileScreen.kt` 引用 `w/ui/PinnedGradientPage.kt` |

### 1.2 一级页·列表/功能型 tab：普通表面 + 无返回键标题

- 用户端服务/订单/积分、师傅端 Orders：普通 `Bg` 表面打底，标题栏不带返回键
- 理由：tab 页栈深为 1，返回键无意义，保留会制造"点了没反应"的假交互
- 证据：两端 `PinnedGradientPage` 调用分布仅落在浏览型 tab（1.1 表，grep 实测）

### 1.3 二级页：统一共享 TopBar + 内容流

- 二级页一律复用既有共享 TopBar + 滚动内容流，禁止页面自造顶栏（自绘标题行/返回键）
- 现状背书：TopBar 已被用户端 34 个、师傅端 30 个文件复用，统一无需新建组件
- 证据：`grep -rl TopBar` 统计。注：本次宽松子串计数为 user 36 / worker 34（含定义文件与注释引用），登记口径以 34/30 为准

## 2. 色板槽位表

### 2.1 槽位定义

| 槽位 | 语义 |
|------|------|
| primary / primary2 | 端主色 / 主色渐变副位 |
| bg / panel / line | 页面底 / 卡片面 / 分隔线 |
| ink / muted | 主文字 / 次文字 |
| success / warn / err | 成功 / 警示 / 错误 |

### 2.2 两端同值槽位（锁定，不得分叉）

| 槽位 | 值 | 证据 |
|------|-----|------|
| bg | 0xFFF5F6F8 | `u/ui/Theme.kt:20`、`w/ui/theme/Color.kt:12` |
| panel | 白（0xFFFFFFFF） | 两端卡片面通用底 |
| line | 0xFFE8EAED | `u/ui/Theme.kt:22`、`w/ui/theme/Color.kt:16` |

### 2.3 异端异值槽位（唯一允许的跨端差异）

- primary：用户端 `0xFF007AFF`（`u/ui/theme/Color.kt:9` BrandBlue）/ 师傅端 `0xFF1677FF`（`w/ui/theme/Color.kt:6` Primary）
- 裁决：允许异值，端定位差异；硬规则是"同端同槽不允许多值"——同一端内 primary 只能有一个值

### 2.4 已登记异常

| 异常 | 位置 | 处置 |
|------|------|------|
| 第三种蓝 `statusBarSolid()=0xFF006AE5` | `u/ui/Theme.kt:78`（同值还见于 :74 渐变） | 归属待设计确认（待补验）：收编为 primary 系，或登记为合法常量 |
| RN 族私有蓝 `0xFF086CF5` 串入师傅端 | 登记：`w/ui/DetailActions.kt:59`；实测另见 `w/ui/Widgets.kt:191-192`、`w/ui/RepairScreen.kt:95`、`w/ui/PullRefresh.kt:26`（注释自认主蓝） | 跨端污染，待收编：替换为 worker primary 或白名单常量 |

### 2.5 合法颜色定义文件白名单

- 用户端：`u/ui/Theme.kt`、`u/ui/theme/Color.kt`；师傅端：`w/ui/theme/Color.kt`
- 白名单之外的文件一律禁止 `Color(0x` 硬编码（存量如 `u/page/AuthForm.kt:132` 裸写 bg 值，随基线登记消化）
- 门禁：`scripts/check-ui-consistency.mjs` 机械拦截；存量豁免登记 `scripts/ui-consistency-baseline.json`，只减不增
- 注：脚本与基线随 P0-1 交付，本 worktree（b51e54cd）尚未见此二文件，验收以其可执行为准（待补验）

## 3. 数值规范（参考终态，收编时对齐）

新代码必须按本节取值；存量在触碰对应文件时顺带收编，不做一次性全量回改。

### 3.1 边距

- 页面横向主流档：用户端 14dp / 师傅端 12dp（端内统一，不强求跨端同值）
- 卡片内边距：12dp

### 3.2 圆角

| 元素 | 档位 |
|------|------|
| 卡片 | 10~12dp |
| 小元素 | 8dp |
| 胶囊 | 999dp |

### 3.3 字号

- 主流档：12 / 13 / 14 / 15 / 16sp；禁止 11.5、12.5 等半档值
- 页面标题：16sp + W600

### 3.4 字重

- 标准加粗符号：W600。新代码加粗一律 W600，不再引入 Bold/SemiBold/W500 新用法
- 存量混用登记（收编基线，只减不增）：

| 端 | W600 | W500 | Bold | SemiBold |
|----|------|------|------|----------|
| 用户端 | 39 | 34 | 34 | 0 |
| 师傅端 | 8 | 8 | 21 | 18 |

- 证据：`grep -rEo "FontWeight.(W600|W500|Bold|SemiBold)"` 两端源码计数。注：user W500 本次实测 36、与登记 34 差 2（统计口径差，收编以 baseline 快照为准）；worker 四档计数与本次实测完全一致

## 4. 挂起与豁免

### 4.1 挂起：RN 登录族第二色板

- 现状：RN 登录族存在独立第二色板，有 v2 设计稿背书且已真机验收，不能简单抹掉
- 处置：暂按合法化第二主题执行（RnPalette.kt，冻结新增）；设计若改选并入全局，回退为单提交改值

### 4.2 豁免：功能性硬编码

- 先例：`w/ui/SignScreen.kt:71` 签名画布白底（`background(Color.White)`，签名导出需纯白底）；`w/ui/ScanScreen.kt:135` 扫码区底色 `0xFFE6F4FF`
- 规则：功能必需的硬编码允许保留，但必须在代码注释注明"功能必需"，否则按 2.5 门禁拦截

### 4.3 登记：i18n 分叉

- 现状：用户端页面裸中文，师傅端走 `R.string`（如 `w/ui/ScanScreen.kt` 文案全部 `stringResource`）
- 统一方向已按方案 C 裁决成文（见 4.5）；保留本节作为分叉事实登记

### 4.4 近值对齐与令牌立卡裁定（2026-08-28 主持人裁定）

- 近值对齐：色值单通道差 ≤16 且语义相同（页面底色、错误红等功能槽位）视同同色，一律对齐标准槽位值——页面底 → `bg 0xFFF5F6F8`（同 2.2）、错误红 → `err 0xFFFF4D4F`；单通道差超过 16 的不适用本规则
- 令牌立卡：一次性孤值不新建立卡令牌；同一色值端内复用 ≥3 处、或成对语义（状态灯/边框成组）方可追加；追加位置仅限 `theme/Color.kt`（2.5 白名单门禁豁免文件）
- 本轮适用实例登记：
  - 近值对齐：worker `F6F8FA → bg` ×3、`FF2D2F → err` ×2
  - 成对语义立卡：`DotOnline/DotOffline`、`SuccessBorder/ErrBorder`（`w/ui/theme/Color.kt:34-39`）
  - 孤值灰阶 `B8BCC2/D1D7E0/B0B3B8/7D8593/EEF0F3/F7F8FA` 等：立卡待复用，暂保留字面量
- RN 族单源化进度：族内字面量已收敛至 `u/ui/theme/RnPalette.kt` 单源引用（暂按合法化第二主题执行，同 4.1）

### 4.5 i18n 裁决：方案 C（2026-08-28 主持人裁定）

- 新增文案一律进 `strings.xml`，对齐师傅端 `R.string`/`stringResource` 惯例；两端一致执行，禁止再新增裸中文
- 存量裸中文不迁移；出海需求出现时作为独立结构迁移单独立项，不随 UI 收编顺手改
- 本条成文即解除 4.3"统一方向待定"挂起；4.3 降级为分叉事实登记

## 5. 执行机制

- CI 门禁：`make ui-consistency-check`（含于 `make check`，红即不合入）
- 审查判据（可机械判定，不依赖主观审美）：
  1. 页面是否引用同一骨架/顶栏组件（第 1 节双层规则）
  2. 颜色是否来自白名单文件（第 2.5 节）
- 数值档位（第 3 节）不进机械门禁，作为 review 提示与收编对齐目标

## 附：证据文件索引

- 色板：`u/ui/Theme.kt`、`u/ui/theme/Color.kt`、`w/ui/theme/Color.kt`
- 骨架/顶栏：两端 `ui/PinnedGradientPage.kt` 及 1.1 表调用页、两端共享 TopBar
- 豁免/异常：`w/ui/SignScreen.kt`、`w/ui/ScanScreen.kt`、`w/ui/DetailActions.kt`、`w/ui/Widgets.kt`、`w/ui/RepairScreen.kt`、`w/ui/PullRefresh.kt`
- 门禁：`scripts/check-ui-consistency.mjs`、`scripts/ui-consistency-baseline.json`（P0-1 交付物）
