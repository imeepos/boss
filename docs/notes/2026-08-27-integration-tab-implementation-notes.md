# 积分升 tab 实施要点（2026-08-27）

> 决策出处:会议 BUG-01 裁定（阿澈）——积分升 tab 支持，与功能补齐同批落地。
> 契约:docs/user/nav.js tabs 数组 5 项（home/products/orders/profile/points，points label='积分'）。

## 现状盘点（已核实,2026-08-27）

- 已完成:Route.Points（Routes.kt:41-42）、PointsPage.kt（已接 PointsApi overview/tier/tasks/exchangeOffers/exchange 四端点）、PointsCards.kt、ProfileMenu「我的积分」入口。
- 缺失:底部 tab 注册 —— Nav.TABS / tabRoute / tabIcon 三处无 "points" 项。
- 结论:升 tab 不涉及新页面、不涉新 API,纯导航注册,工作量小时级。

## 改动点

1. Nav.TABS 追加 `"points" to "积分"`（label 严格取 nav.js）。
2. tabRoute 加 `"points" -> Route.Points`。
3. tabIcon 加 points 分支（选中/未选中双态，material-icons-extended 已在依赖内）。

## 入口去留

ProfileMenu「我的积分」入口保留不动:tab 与菜单双入口同指 Route.Points,常规做法,不删。

## 风险与兜底

- **360dp 窄屏 5 tab 溢出**:2026-08-21 曾 5 胶囊在 360dp 溢出。必须加 DeviceConfigurationOverride.ForcedSize(360dp) 渲染测试（PageRenderTest 已有该模式）,断言「积分」label 可见。
- **导航拓扑变更回归**:独立 commit;connected + 真机全 tab 切换回归,不并入其他改动。
- **PointsPage 网络态**:LoadError 文案已存在（积分加载失败，请检查网络）,升 tab 不改其网络行为。

## 时序

D1-4 功能补齐窗口内落地;独立 commit（type: feat(android-user)/积分 tab 升位）;随 BUG-01 分支或独立分支合入均可。