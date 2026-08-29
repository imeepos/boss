# 用户端设计源头 ↔ App 令牌对账审计

日期：2026-08-28 ｜ 审计人：陈端 ｜ 性质：只读对账，未改任何代码

**结论：漂移 6 处 / 缺失 1 处（另有页面级杂色 17 处未建令牌，登记不计数）/ 无出处 20 处（第三种蓝 1 + RN 族 17 + 渐变两端 2）；字号与圆角全部同值对齐；Color.kt 另有 9 个暗色系死值（App 内 0 引用）源头也无出处，单列不计数。**

- 源头：`docs/user/style.css`（43 页共享样式，`:root` 12 变量 + 页内 `<style>`；登录/注册/忘记密码三页另有内联渐变）
- App 令牌：`ui/theme/Color.kt`（24 行）+ `ui/Theme.kt` `Palette` 对象（15 令牌）

## 一、:root 变量级对账（源头 12 变量）

| 源头值 | App 令牌值 | 判定 |
|---|---|---|
| `--primary:#1677ff` | `BrandBlue 0xFF007AFF` | **漂移**（Ant Design 蓝 vs iOS 系统蓝） |
| `--primary-2:#69b1ff` | 无对应令牌 | **缺失**（源头渐变端色 App 未建令牌） |
| `--bg:#f5f6f8` | `0xFFF5F6F8` | 同值 |
| `--panel:#fff` | `SurfaceLight 0xFFFFFFFF` | 同值 |
| `--line:#e8eaed` | `0xFFE8EAED` | 同值 |
| `--ink:#1f2329` | `OnSurfaceLight 0xFF1C1C1E` | **漂移** |
| `--muted:#8c8c8c` | `OnSurfaceVariantLight 0xFF3F3F46` | **漂移**（源为浅灰，App 为深灰） |
| `--subtle:#b0b3b8` | `0xFFB0B3B8` | 同值 |
| `--success:#52c41a` | `Green500 0xFF34C759` | **漂移** |
| `--warn:#faad14` | `ActionOrange 0xFFFF9500` | **漂移** |
| `--err:#ff4d4f` | `0xFFFF3B30` | **漂移** |
| `--radius:12px` | 卡片默认 `RoundedCornerShape(12.dp)`（Widgets.kt:38） | 同值 |

字号对账：源头 11/12/12.5/13/14/15/16px ↔ App 11/12/12.5/13/14/15/16.sp 全同值；tag 圆角 4px↔4.dp 同值。

## 二、渐变对账（漂移重灾区）

| 位置 | 源头值 | App 令牌值 | 判定 |
|---|---|---|---|
| 品牌渐变（topbar/home-head） | `linear-gradient(135deg,#1677ff,#69b1ff)` | `profileHeaderGradient = BrandBlue(0xFF007AFF)→0xFF3B82F6` | **漂移**（两端全不同） |
| 首页渐变起点 | 无（源头即 135deg 同一渐变） | `homeHeaderGradient 起点 0xFF006AE5` | **无出处**（App 自衍生，见 §四） |
| 渐变基准两端 | — | `0xFF1E3A8A→0xFF3B82F6`（Color.kt:5-6） | **无出处**（Tailwind indigo-900/blue-500，源头 0 命中；且 `BrandBlueGradientStart` App 内 0 引用） |
| 登录页 Hero 渐变 | `linear-gradient(160deg,#0f1f3d,#1a2b4a 55%,#1677ff)`（login/register/forgot） | RN 族 `#0872F4→#0B82F8→#1698FA`（LoginPage.kt:125） | **无出处**（对位位置存在但两端值全不同，见 §四） |

## 三、页面级杂色（源头有、App 无令牌，登记不计数）

tag 系 13 色（`#389e0d/#f6ffed/#b7eb8f/#e6f4ff/#91caff/#d46b08/#fff7e6/#ffd591/#cf1322/#fff1f0/#ffa39e/#595959/#fafafa`）、宫格 `#722ed1`（App 另建 `ActionPurple 0xFFAF52DE` 对位，亦漂移）、状态点 `#a6e9a0/#ffa39e`（App 用 success/err 复用）、外壳底 `#e9ebee`、分段槽 `#eef0f3`、节点灰 `#d9d9d9`、`#f0f4f8`。

## 四、专项调查：第三种蓝与 RN 族出身

对 `docs/user/` 全部 46 个文件做大小写不敏感 grep，结果：

| App 色值 | docs/user 命中 | 出身结论 |
|---|---|---|
| `statusBarSolid()=0xFF006AE5`（Theme.kt:78，同值见于 :74 首页渐变起点） | **0 命中** | **无源头出处**。Theme.kt:71 注释自述"基准上仅起点略加深"——由 `BrandBlue(007AFF)` 手动调深的 App 内部衍生值，非设计稿取值。`docs/design/mobile-ui-standard.md:59` 已登记"归属待设计确认" |
| RN 族 17 色（`RealNamePage.kt:47-66` object RN：`0xFF086CF5/0872F4/0B82F8/1698FA/0AA847/EFFFF4/F57900/FFF5E8/171B23/5F6671/AEB4BE/E7EAF0/FF2D2F/F6F8FA/F7F8FA/7D8593/D1D7E0`） | **0 命中** | **无源头出处**（对 user 端而言）。真正出处是 worker 端 `docs/worker/UI-SPEC.md:31-33`（Primary #086CF5 / Hero 渐变 #0872F4→#0B82F8→#1698FA）与 `mobile/worker/design/worker-login-states-v1.spec.md`；user 端 RealNamePage.kt 定义 `object RN`、LoginPage.kt 直接复用，即实名页与登录页整套装用了 worker 登录 spec 色板 |

RN 族与主 Palette 并行使用的另一后果：`RN.primary(#086CF5)`、`RN.line(#E7EAF0)`、`RN.ink(#171B23)`、`RN.muted(#5F6671)`、`RN.pageBg(#F6F8FA)` 与 `Palette.primary(#007AFF)`、`Palette.line(#E8EAED)`、`Palette.ink(#1C1C1E)`、`Palette.muted(#3F3F46)`、`Palette.bg(#F5F6F8)` 均为同语义不同值的重复令牌。

## 五、Color.kt 死值（App 内 0 引用，源头亦无出处，不计数）

`BrandBlueGradientStartDark 0xFF0F172A`、`BrandBlueGradientEndDark 0xFF1E40AF`、`BrandBlueDark 0xFF60A5FA`、`Green500Dark 0xFF30D158`、`OnSurfaceDark 0xFFF2F2F7`、`OnSurfaceVariantDark 0xFFA1A1AA`、`SurfaceDark 0xFF1C1C1E`、`ActionOrangeDark 0xFFFFB340`、`ActionPurpleDark 0xFFCF8BF0`、`NavigationBlue 0xFF0A84FF` —— 与 Theme.kt:35 注释"暗色配色完全移除"一致，属遗留死代码。

## 六、行动建议

1. 三蓝并存（`#1677ff` 源头 / `#007AFF` Palette / `#086CF5` RN / `#006AE5` 状态栏，实际四系）应收敛为一个 primary，方向二选一：回归源头 `#1677ff`，或修订源头采纳 `#007AFF` 并同步 docs/user。
2. 六处漂移（ink/muted/success/warn/err/渐变）逐一定裁：源头错还是 App 错，裁定后单侧改齐。
3. RN 族 17 色建议收编进 Palette 或标注"借用 worker spec"的正式豁免，消除同语义重复令牌。
4. `0xFF006AE5` 若保留，在 Color.kt 命名化（如 `StatusBarBlue`）并登记出处为"BrandBlue 衍生"；否则并入 primary。
5. 清理 Color.kt 十个死值（含 NavigationBlue），减少误用面。
