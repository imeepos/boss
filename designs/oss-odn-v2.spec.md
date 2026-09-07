# ODN 无源网络页 UI/UX 设计规格（oss-odn v2，对应四张设计稿）

> 配对设计稿：`oss-odn-tree-v1.png`（主布局·多级资源树）、`oss-odn-relation-v1.png`（关联链视图）、`oss-odn-drawer-create-v1.png`（新增抽屉·全选择器）、`oss-odn-drawer-detail-v1.png`（详情抽屉·关联链）。
> 页面归属：后台 `web/admin`，`pages/oss/odn/` 重构（现路由 `/oss/odn` 不变）。
> 设计工具：gpt-image-2（2026-09-08）。本文可直接转发给前端/AI 编码代理。

## 0. 交互硬约束（用户裁定，验收红线）

1. **新增/编辑一律抽屉（Drawer）**，禁止页面内联表单（现状 `ODNForm` 内联展开必须移除）。
2. **详情一律专门抽屉**，禁止行内展开/内联详情。
3. **一切枚举/关联字段一律选择器**：统一用 `components/Dropdown.tsx`（省/市/类型/网格/局点/上级设备/状态），禁止原生 `<select>`，禁止自由文本填编码。
4. 编码类字段（设施编码/局点序号）自动顺延只读可覆盖，沿用 `/odn/facility-next-code`、`/odn/site-next-no`。

## 1. 整体布局（oss-odn-tree-v1）

- 外壳沿用现有 shell：顶栏 `--shell-topbar-bg #0F1E3B`，侧栏「网络资源 → ODN 无源网络」激活态 `--shell-menu-active-bg #FBF5E8` + 3px `#D5A63A` 左条，内容区 `--shell-content-bg #F4F5F7`。
- 内容区改为**左右两栏**：左侧资源树卡片 280px 固定；右侧主区自适应，两栏间距 16px。
- 左树卡片：头部「资源目录」标题 + 搜索框（前端过滤树节点），节点行高 36px，缩进 16px/级，4 级结构见 §2。
- 右侧主区从上到下：面包屑 → 页头+操作 → 关联统计卡 ×4 → Tab 栏 → 数据表 → 分页。

## 2. 多级资源树（核心变更）

```
行政区划(L1 省)              PHL001 菲律宾        ← /odn/regions
└─ 行政区划(L2 市)          MNL 马尼拉 / CEB 宿务 ← /odn/cities?prvCode=
   └─ 资源分组(L3, 计数badge)
      ├─ OLT 设备 8         ← /resources（resource 域，挂局点）
      ├─ ODN 网格 12        ← /odn/grids
      ├─ 局点 24            ← /odn/sites
      └─ 施工项目 3         ← /odn/constructions
         └─ 资源节点(L4)    网格 01 老城区 / OLT-MNL-01 …
```

- L3 分组行 = 折叠开关 + 图标 + 名称 + 计数 badge（灰底 pill）；L4 叶子行 = 类型小图标 + 编码/名称 + 状态点（绿=ACTIVE/IN_USE，琥珀=容量≥80% 或 RESERVED，灰=RETIRED）。
- 点 L2 市节点：主区显示该市汇总（KPI + 各分组 Tab）；点 L4 叶子：主区过滤到该资源并打开其详情抽屉或列表过滤。
- L3/L4 计数缺接口时先显示已加载行数，不阻塞渲染（禁止为计数新增后端契约，属实现期优化项）。
- 树状态（展开/选中）存 `useState`，路由 `?prv=&city=&tab=` 同步，刷新可还原。

## 3. 关联关系的呈现（用户核心诉求）

- **统计卡**：网格总数 / 设施总数 / 局点总数 / **关联 OLT**，第四张卡用蓝色 tinted 图标强调跨域关联。
- **表格关联列**（网格/设施/局点/设备四个 Tab 均有）：`所属网格`（P/MH 类设施）、`所属局点`（设备，如 MNL001）、`关联 OLT`（蓝色圆角 chip，如 OLT-MNL-01，点击跳转 `/oss/device` 过滤该 OLT）。
- **关联链视图**（oss-odn-relation-v1，挂在「设备」Tab 或详情抽屉「关联关系」区）：横向拓扑 `OLT → 局点 → 分光器 → 接头盒/终端盒 → 覆盖`，节点卡 = tinted 图标 + 编码 + 两行说明，节点可点击切换详情。逻辑侧分光器/端口来自 resource 域（P3 绑定 `odn_bindings`），桥接靠关系数据，不做 FK 直穿。
- 行政区划关联：所有实体行/详情均展示 `所属省份 PHL001 / 所属城市 MNL 马尼拉`（现 prvCode/cityPrefix 字段即 000075 字典，命名按 `docs/contract/fields.md` §1.5）。

## 4. 新增/编辑抽屉（oss-odn-drawer-create-v1）

- 右侧 Drawer 440px，遮罩点击关闭；头部标题「新增设施/编辑设施」+ X；底部右对齐 `取消`（outline）+ `保存`（primary `#273F70`）。
- 表单纵向 16px 间距，label 13px `--color-text-secondary`：省份选择器 → 城市选择器（随省联动）→ 类型选择器（P/MH/TW/CLS/TBX 或局点/设备枚举）→ 所属网格选择器（P/MH 必填，随市过滤）→ 所属局点选择器（设备必填）→ 编码（自动顺延只读灰底 + hint「编码自动顺延,可覆盖」）→ 名称 → 经纬度两栏。
- 全部下拉用 `Dropdown.tsx`；类型变化即时调 next-code 预取；归属链只列合法上级（复用 `nextcode.ts` legalParentKind 规则）。
- 批量导入按钮保留在页头（`BatchImportEntry`），不进抽屉。

## 5. 详情抽屉（oss-odn-drawer-detail-v1）

- 右侧 Drawer 480px；头部：编码 tag（mono）+ 名称 + 状态 Badge + X。
- 分区（细分割线）：**基本信息**（两列 key-value：省/市/网格/名称/经纬度/生命周期）→ **关联关系**（纵向链行卡：OLT→局点→设施/设备，每行 tinted 图标 + 编码 + 名称 + chevron，点击换详情对象）→ **资产信息**（有 `assetReg` 时显示凭证号/登记号 badge，无则「未登记」灰字）→ **操作记录**（时间线：巡检/状态变更/备案）。
- 底部操作：`退役`（红字文本钮，danger confirm）/ `编辑`（打开编辑抽屉）/ `保存`（编辑态）。

## 6. 视觉令牌（与主题系统对齐）

- 卡片 `--shell-card-bg` 白底 radius 8 + `--shell-card-shadow`；标题 `--shell-heading #172744`；辅助文字 `#5F6671`。
- 金色强调仅用于：Tab 激活下划线、树选中态、 L4 容量预警点（`#D5A63A`）。
- 状态色面积 ≤5%：绿=在用/正常、琥珀=预警、灰=停用/退役，Tag 用描边浅底样式（`StatusTag`）。
- 关联 OLT chip：蓝 tinted 底（14% alpha）+ mono 12px，非状态色。
- 暗色主题：全部经 `[data-theme]` 变量，禁止硬编码色值（tinted 图标底用 `color-mix` 或主题变量）。

## 7. 契约与 i18n 对齐

- 复用接口（不新增后端）：`/odn/regions` `/odn/cities` `/odn/grids` `/odn/facilities` `/odn/sites` `/odn/devices` `/odn/facility-next-code` `/odn/site-next-no` `/odn/constructions`（见 `api/openapi/admin/odn.yaml`）；OLT 用 `/resources` + `/device/metrics`。
- 信封形状：列表接口返回裸数组（红线 23：不是 `{items}`），`/resources` 返回 `{items}`——对接时逐个 curl 复核。
- i18n：新增 key 挂 `t.pages.odn` 下（tree/kpi/drawer 分组），三份 locale + types.ts 同步，禁止面板内硬编码中文（constructions/assets 面板 W1 先例除外）。
- 单文件 ≤300 行：树组件抽 `ResourceTree.tsx`，抽屉抽 `ResourceDrawer.tsx` + `DetailDrawer.tsx`，现 `forms.tsx` 内联表单改造为抽屉内容源。

## 8. 实现验收标准（供 /devloop 立项引用）

1. `/oss/odn` 呈现左树右表布局，树含 省→市→分组→节点 四级，点击叶子主区联动。
2. 新增/编辑只经抽屉发生，页面无内联表单 DOM（grep 断言无 inline form 容器）。
3. 详情只经抽屉呈现，表格行「详情」点击开抽屉。
4. 页面 DOM 无原生 `<select>`（grep 断言），全部走 `Dropdown.tsx`。
5. 设施/设备表可见 所属网格/所属局点/关联 OLT 关联列，KPI 含「关联 OLT」卡。
6. `pnpm typecheck && pnpm test && pnpm build` 全绿；双主题截图各一张经 cdp-capture 验证。