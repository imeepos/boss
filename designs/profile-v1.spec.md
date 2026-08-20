# 用户端「我的」（个人中心）实现规格（对应 profile-v1.png）

> 平台：mobile/user/android（Jetpack Compose，Material 3）。项目已有实现：
> `mobile/user/android/.../page/ProfilePage.kt`，本稿与其结构一一对应，可直接对照开发/比对。

## 给前端的实现提示词（可直接复制）

实现一个 Android Compose 的个人中心页面，垂直滚动 Column，结构自上而下：

1. **渐变头部**：`fillMaxWidth`，背景 `Brush.linearGradient(#1E3A8A → #3B82F6)`
   （暗色主题用 `#0F172A → #1E40AF`），padding 16/24。内容为 Row：
   48dp 白色 25% 透明圆形头像（显示姓名首字）+ 右侧两行文字（姓名 18sp Bold 白色、
   `phoneMasked` 12.5sp 白色 85%）。数据来自 `GET /profile` 的 `name` / `phoneMasked`。
2. **实名信息卡**：CardTitle「实名信息」+ 可点「账号安全」；CellRow×3：姓名（`realName.nameMasked`）、
   证件（`realName.idType + idNoMasked`）、核验记录 + Tag（`status == "VERIFIED"` → 绿色
   #34C759「已实名」，否则灰色「待补登」）。
3. **家庭地址卡**：CardTitle「家庭地址」+「管理」→ Route.Address；空态「暂无地址 / 点击管理新增」；
   有数据取前 2 条：label + desc（默认→「默认安装地址」）+ Tag（`isDefault` → 绿「在用」/ 灰「未用」）。
4. **我的套餐卡**：CardTitle「我的套餐」+「详情」→ Route.MyPlan；一行：`plan.name`、
   `¥{monthlyFee}/月 · 合约至 {contractEnd}`、绿色 Tag「在网」。
5. **我的服务卡**：CardRow 列表 7 项（我的订单/我的账单/缴费记录/报障记录/消息中心/优惠券与活动/电子发票），
   每项右侧 `›` chevron，分别跳 Route.Orders/Bills/Pay/Fault/Messages/Coupon/Invoice。
6. **账号与设置卡**：5 项（账号安全/通知订阅设置/投诉与建议/帮助中心/用户协议与隐私）→ 对应 Route。
7. **语言卡**：标题「语言 / Language」；三个 Button pill：中文（选中：底 #007AFF 字白）、
   English、Filipino（未选中：底白字灰）；调用 `ProfileApi.setLanguage(key)`，失败保留本地高亮；
   下方 12sp 灰色说明文案。
8. **退出登录**：`fillMaxWidth` 44dp 高 Button，容器色 `#FF3B30`（暗色 `#FF6961`），白字居中；
   点击调 `UserApi.auth.logout()`（端点失败也继续）→ 清 token → `nav.resetTo(Route.Login)`。

### 设计 token（与项目 Palette 一致，勿另造）

| 项 | 值 |
|---|---|
| 主色 primary | `#007AFF`（暗色 `#60A5FA`） |
| 头部渐变 | 亮 `#1E3A8A→#3B82F6` / 暗 `#0F172A→#1E40AF` |
| 页面背景 | `#F5F6F8`（暗色 `#101114`） |
| 卡片 panel | `#FFFFFF`（暗色 `#1C1C1E`），圆角 12dp |
| Tag | 圆角 4dp，色块 10% alpha，文字 11sp |
| 主文字 | `#1C1C1E` 15sp W600（标题）/ 14sp W500（列表项） |
| 辅助文字 | `#3F3F46` 12sp；分割线 `#E8EAED` |
| success / warn / err | `#34C759` / `#FF9500` / `#FF3B30` |
| 间距 | 卡片外边距 14dp、内 padding 16dp、行垂直 12dp（4 的倍数） |

## 注意事项

- 本稿由 gpt-image-2 生成，**未经目检**（当前模型不支持读图）；图中文字为 AI 渲染，
  可能存在变形/错字，实现一律以本 spec 文案为准，图为布局与配色参考。
- 头像在实现中是「姓名首字 + 25% 白透明圆」，设计稿可能画成人像照片，以代码实现为准。
- 列表图标：设计稿中的图标仅为示意，项目内已有 CellRow 无图标变体，按现状实现可不加图标。
- 语言切换按钮选中态文字必须水平垂直居中，上下留合适边距。
- 状态覆盖：数据加载中姓名显示「加载中…」；接口失败各卡字段显示「—」；地址空态见上。
- 数据契约：全部来自 `GET /profile`（name/phoneMasked/realName/addresses/plan），
  语言写入 `POST setLanguage`，登出 `POST logout`，字段已存在于项目 API，无需后端新增。
