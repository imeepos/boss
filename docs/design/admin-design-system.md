# Admin 设计系统规范(Q3 重点④)

> 对象:web/admin(72 页)。目标:列表/表单/详情/流程四类页面统一骨架,新页面照模板写,旧页面按债务清单收敛。
> 本文为裁定文档:与既有页面冲突时,以本文"裁定"列为准;债务清单记录现状与去向。

## 1. 基座组件(唯一入口 `components/business` + `components/ui`)

| 组件 | 用途 | 规则 |
|---|---|---|
| PageHead | 页标题+描述 | 每页必用(现 60/72);不带面包屑(壳层已有) |
| Card / CardHeader / CardContent | 卡片容器 | 新页面一律 Card,不再手写 rounded+border+shadow 长串 |
| Table(ui/table) | 结构化表格 | **见 §2.1 裁定** |
| Drawer + DetailDrawer | 抽屉/详情 | onClose(关 UI)与 onChanged(刷新数据)分开 |
| FormField / Input / Switch / Badge | 表单与状态 | SecretInput 密码不回显(占位"已配置") |
| Dropdown | 全部下拉 | **禁止原生 `<select>`**(两次用户点名) |
| TabBar | 页内平铺页签 | 本规范新增;禁止再手写内联 style 页签 |
| Pagination + pagerTexts(ns) | 分页 | 筛选变化必须 setPage(1) |
| StatusTag + registry | 状态标签 | 状态→颜色只在 registry 定义一次;禁止页面散落色彩映射 |
| ToolbarButton / ErrorBanner / TableStateRow / EmptyState | 工具栏/错误/空态 | 空态含 loading 与 text 双态 |

## 2. 四类页面模板

### 2.1 列表页(最常见)

骨架:`PageHead → Card[ TabBar(可选) → 工具栏(筛选+spacer+刷新) → 表格 → Pagination ]`

- **表格裁定**:现状 57 页用"原生 table + 统一 tailwind 类串"、8 页用 ui/Table。裁定:**两者并存**——简单只读列表沿用原生 table+标准类串(允许复制既有页),列定义复杂/带排序聚合的用 ui/Table 或 DataTable;禁止第三种写法。
- 分页裁定:数据量可全量(≤千行)客户端分页 `pageSlice`;增长型数据(AAA/CDR/账实核对)服务端分页,envelope `{items,total,page,pageSize}`。
- 筛选与 URL:首次 URL 初始化 + 本地 state + setter 双写 `{replace:true}`(useQueryState 模式)。
- 参考页:aaalog(双页签+服务端分页)、paycheck(双页签+汇总条)、params(行内编辑)。

### 2.2 表单页/抽屉

骨架:`DetailDrawer/Dialog → FormField 网格(grid-cols-2) → 底部 ToolbarButton(取消/保存)`

- 必填校验前端提示,提交按钮 busy 态;保存成功 toast(sonner)+ onChanged 刷新。
- 参考页:org/staff(级联下拉)、provision/template(新建抽屉)。

### 2.3 详情页/抽屉

骨架:`DetailDrawer → dl 网格(label+value) → 关联区块(时间线/轨迹)`

- 只读字段用 `—` 占位空值;轨迹/时间线正序展示(参考 invoice tax-events 消费方式)。
- 参考页:asset TrailDrawer、quad 关联查询。

### 2.4 流程页(状态机驱动)

骨架:`PageHead → 状态卡(StatusTag) → 时间线/操作区(按状态条件渲染动作) → 关联列表`

- 动作按钮必须走 ConfirmDialog(危险动作 danger);操作后刷新整卡。
- 参考页:order(12 环节时间线+动作)、dispatch(派单/改派)。

## 3. 横切规则

1. **颜色**:只用 `var(--*)` 令牌;页面出现 `#1677ff/#666/#f0f0f0` 等字面量即债务(现存 19 处,见 §4)。
2. **图标**:描边 SVG(24/stroke 1.8-2/currentColor);禁文字字形。
3. **i18n**:加 key 同步 4 处(types.ts + 三 locale);页面无裸中文/英文串。
4. **主题**:引用每个 `var(--x)` 前确认令牌已定义;新组件双主题目测。
5. **表格行高/密度**:h-11 px-3 文字 13px,沿用标准类串。
6. **tab 页签**:一律 TabBar(本规范起)。

## 4. 收敛债务清单(grep 证据,2026-08 实测)

| 债务 | 现状 | 去向 |
|---|---|---|
| 内联 style 页签 | 6 页(dispatch/worker-ops/device/asset TrailDrawer/aaalog/paycheck);本轮已收敛 aaalog+paycheck 至 TabBar | 余 4 页后续逐页替换,改一处验一处 |
| 硬编码色 | 19 处 `#1677ff` 等 | 换 `--color-border-focus`/`--shell-group-title` 等令牌 |
| 原生 table 类串复制 | 57 页 | 允许;重大改版时再评估 DataTable 化(不强行) |
| ui/Table 低采用 | 8 页 | 复杂表保留,不扩散也不删除 |

## 5. 门禁建议(纳入 web-admin-check 前置)

- `grep -rn "#1677ff\|#f0f0f0" web/admin/src/pages` → 应为 0
- 新 PR 含 `borderBottom: tab ===` → 拒绝,改用 TabBar
- 新 PR 含 `<select` → 拒绝,改用 Dropdown
