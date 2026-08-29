# 师傅端设计源头 vs App 令牌 对账审计

日期 2026-08-30 · 审计人 林师 · 只读审计，不改代码
源头：`docs/worker/style.css`（173 行）；App：`mobile/worker/android/.../ui/theme/Color.kt`（47 行）+ `Theme.kt`

## 结论（先行）

1. **:root 11 个色板变量，10 同值、1 缺失**：`--subtle:#b0b3b8` 无 App 令牌，`TransferScreen.kt:117` 裸写 `Color(0xFFB0B3B8)`。其余 10 个全部同值，且令牌色在屏幕层**零裸写**（屏幕内裸色只出现在令牌未覆盖的语义上）。
2. **卡片圆角漂移**：源头 `--radius:12px` 只锚定 `.card/.stats-card`；App 无 Shape 令牌，卡片圆角实际**主流 10dp（14 处）> 12dp（7 处）双轨**，与源头 12px 不一致。
3. **TagColor 六组 fg+bg 共 12 色 100% 同值**；但每组 border 色源头有、App 无承接字段，且 `SuccessBorder(0xFF6FD18B)≠#b7eb8f`、`ErrBorder(0xFFFF9B9D)≠#ffa39e`，属收编存量后的漂移。
4. App 屏幕内另有 **8 个非令牌裸色值（10 处使用）**源头不覆盖或与源头不同值（见 §4）。

## 一、:root 变量 ↔ App 令牌

| 源头值 | App 令牌 | 判定 |
|---|---|---|
| --primary #1677ff | Primary 0xFF1677FF | 同值 |
| --primary-2 #69b1ff | Primary2 0xFF69B1FF | 同值 |
| --bg #f5f6f8 | Bg 0xFFF5F6F8（另有 BgArgb 平台衍生） | 同值 |
| --panel #fff | Panel 0xFFFFFFFF | 同值 |
| --line #e8eaed | Line 0xFFE8EAED | 同值 |
| --ink #1f2329 | Ink 0xFF1F2329 | 同值 |
| --muted #8c8c8c | Muted 0xFF8C8C8C | 同值 |
| --subtle #b0b3b8 | 无令牌；TransferScreen:117 裸 0xFFB0B3B8 | **源头有 App 无令牌**（线索①坐实） |
| --success #52c41a | Success 0xFF52C41A | 同值 |
| --warn #faad14 | Warn 0xFFFAAD14 | 同值 |
| --err #ff4d4f | Err 0xFFFF4D4F | 同值 |
| --radius 12px | 无 Shape 令牌（Theme.kt 未定义 shapes） | **漂移**（线索②，见 §三） |

## 二、组件级硬编码色 ↔ App 令牌

| 源头（选择器·值） | App 令牌 | 判定 |
|---|---|---|
| .dot #a6e9a0（在线点） | DotOnline 0xFFA6E9A0 | 同值 |
| .dot.off #ffa39e（离线点） | DotOffline 0xFFFFA39E | 同值 |
| .topbar/.home-head 渐变 primary→primary-2 | Primary/Primary2（状态栏另锚 StatusBarSolid=Primary） | 同值（渐变在 App 侧用令牌组装） |
| tag-green #389e0d/#f6ffed | TagGreen fg/bg | 同值（border #b7eb8f 无承接） |
| tag-blue #1677ff/#e6f4ff | TagBlue fg/bg | 同值（border #91caff 无承接） |
| tag-orange #d46b08/#fff7e6 | TagOrange fg/bg | 同值（border #ffd591 无承接） |
| tag-red #cf1322/#fff1f0 | TagRed fg/bg | 同值（border #ffa39e 无承接） |
| tag-gray #595959/#fafafa | TagGray fg/bg | 同值（border #d9d9d9 无承接） |
| tag-cyan #08979c/#e6fffb | TagCyan fg/bg | 同值（border #87e8de 无承接） |
| .quad .qm.ok 边框 #b7eb8f | SuccessBorder 0xFF6FD18B | **漂移** |
| .quad .qm.bad 边框 #ffa39e | ErrBorder 0xFFFF9B9D | **漂移** |
| .seg 背景 #eef0f3 | 无令牌；OrdersScreen:236/248 裸 0xFFEEF0F3 | 同值但**无令牌**（裸值×2） |
| .tl-item .node #d9d9d9（待办节点） | 无 | 源头有 App 无 |
| .scan-frame 渐变 #f0f7ff→#e6f4ff | 无 | 源头有 App 无 |
| .gitem 背景 #fafafa / :active #f0f0f0 | 无 | 源头有 App 无 |
| body #e9ebee（手机壳外底色） | 无 | 源头有 App 无（桌面 H5 专属，合理缺失） |
| .notice/.msg-banner 警示组 | 无独立令牌 | 源头有 App 无（可复用 Tag 组色值） |

## 三、线索验证

- **① --subtle**：确认。Color.kt 全文无 subtle；仅 TransferScreen.kt:117 `Color(0xFFB0B3B8)`，值与源头同，但绕过令牌层。建议补 `val Subtle = Color(0xFFB0B3B8)`。
- **② --radius**：源头 12px 仅 .card/.stats-card 两类卡片；源头自身还有 22/16/10/9/8/7/6/4px 梯度（btn-block、quad、notice、gitem 均 10px）。App 无 Shape 令牌，卡片主流 **10dp（14 处）**、12dp 仅 7 处（Widgets/OrdersCard/AuthForm/HomeCards/ProfileCards），双轨并存 → 判定**漂移**，建议定 Shapes 令牌（卡片一档）统一。
- **③ TagColor**：六组 fg+bg 12/12 同值；缺 border 维度，且 SuccessBorder/ErrBorder 与源头对应边框色不同值（见上表）。

## 四、App 有源头无（非令牌裸值清单）

| App 裸值（位置） | 语义 | 判定 |
|---|---|---|
| 0xFF0872F4/0x0B82F8/0x1698FA（LoginScreen:137 渐变） | 登录页品牌渐变 | App 有源头无，与 primary→primary-2 渐变**不同值**，疑似第二套品牌蓝 |
| 0xFFF7F8FA / 0xFF7D8593 / 0xFFD1D7E0（AuthForm:56/57/171） | 表单底/图标/勾选框边 | App 有源头无（源头表单为 --line 边 + #fff 底） |
| 0xFFB8BCC2 ×2（SafetyScreen:70、SettingsScreen:102） | 开关关闭态 | App 有源头无（语义近 --subtle 但不同值） |

## 五、行动建议（供决策，不在本次执行）

1. Color.kt 增补 `Subtle` 令牌，替换 TransferScreen 裸值。
2. Theme.kt 定 Shapes（卡片档）并裁决 10dp/12dp 归一方向。
3. SuccessBorder/ErrBorder 与源头 #b7eb8f/#ffa39e 二选一对齐；登录渐变与 AuthForm 灰阶建议并入源头色板或登记为 App 专属扩展。
