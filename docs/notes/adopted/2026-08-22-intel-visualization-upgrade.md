# 数字孪生与经营板块可视化升级方案

日期：2026-08-22（来源：用户直接请求 intel 板块做可视化）

## 决策

为 `web/admin` 数字孪生与经营组（`/intel/gis`、`/intel/analytics`、`/intel/report`）三页做可视化升级，遵循以下四项约束：

1. **图表全部纯 SVG/Canvas 自绘**，不引入 echarts / recharts / d3 等第三方图表库。
2. **GIS 地图本期不上底图**，改用"八级 drill 树形 Canvas/SVG 布局"承载节点空间关系，与现有 `/gis/drill` 接口契约 100% 对齐。
3. **复用现有 `--shell-*` 令牌**，不新增 CSS 变量；配色全部从 `tokens.css`/`styles.css` 已定义的品牌色 + 状态色派生。
4. **复用 dashboard 已有可视化模板**：统计卡、横向进度条、纵向柱状图，作为组件下沉到 `components/business/charts/` 共享。

后端契约零改动，全部消费现有 `internal/httpapi/admin/gis_handlers.go`、`analytics_handlers.go`、`report.go` 暴露的接口。

## why

### 为什么不上第三方图表库

- 项目其他页面（dashboard）已用纯 CSS 柱状 + Tailwind 进度条自绘，整套可视化体积 0KB、零运行时依赖、自定义 token 一致性高。
- 引入 echarts（~900KB gzip）会拖慢 admin 首屏 30%+，且其默认主题需改写适配 `--shell-*` 令牌，工作量大于收益。
- 数字孪生三页需要的图表形态有限（柱状/进度条/占比环/树形），全部 SVG 自绘 ≤ 200 行/图，复杂度可控。
- 若后续出现复杂图表需求（地理热力、流向图），可按需引入；但本期所有图必须自绘，避免锁定。

### 为什么 GIS 不上底图

- 项目内 PSGC 坐标数据完整（`migrations/geo_psgc_*`），但缺少前端可视化需要的多边形边界 GeoJSON；自维护成本高。
- 测试环境 102 服务器可能无外网，拉 OSM 瓦片不稳；自建瓦片服务不在本季度目标内。
- 现有 `/gis/drill` 是八级 ltree 下钻，纯台账式查询，本身就是"层级而非地理"语义；Canvas 树形布局可还原其使用心智，且与 `docs/contract/fields.md` 中 `level` 字段语义一致。
- 当未来 GIS 真正接入坐标时（pending `internal/domain/gis` 阶段 8 收尾），可平滑替换为 leaflet/maplibre，本方案不构成阻塞。

### 为什么复用 dashboard 模式

- `web/admin/src/pages/dashboard/index.tsx` 已实现统计卡、进度条、7 日柱状三件套，UI 风格、间距、令牌引用已被用户/产品验收过。
- 把"统计卡 + 柱状 + 进度条"抽到 `components/business/charts/` 后，dashboard 与 intel 共享同一份视觉规范，未来替换主题色只需改一处。
- 维护：单文件 ≤ 200 行（红线），三个图表 ≤ 60 行/个（红线），命名与 props 直白，符合"能复用就不要早轮子"。

### 为什么不改后端

- 现有接口已返回足够的可视化所需字段：`IndicatorRow.value`、`RegionRoiRow.roi`、`HeatCellRow.utilization`、`AnalyticsMaintRow.healthScore`、`GisNode.count`、`ReportPayload.indicators[]` 等。
- 趋势/对比类需要历史序列，目前未提供。本期先用"聚合指标 + 单期热力"展示数字孪生价值，"周期对比"留作下期与 report 域对齐。
- 改动后端会影响 `internal/app/wiring*.go` 装配链与 e2e 集成测试，本期目标（前端可视化）不应顺带扩大范围。

## 方案总览

### 视觉风格

- **页面布局**：顶部 4 个统计卡（grid-cols-1 md:2 xl:4，与 dashboard 同款）；中部主图表（柱/热力/树）；下方明细表/侧栏列表。
- **配色**：主色 `--color-text-link`（#C69835，金色品牌）；状态色 `--color-success` / `--color-danger` / `--color-warning`（如未定义则用 `#D5A63A`/`#D94B4B`/`#2F8F63`）。
- **容器**：`rounded-md border bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]`（与全站卡片一致）。
- **空/加载**：`EmptyState`/`Spinner`（`components/business/feedback.tsx`）。
- **双主题**：所有 SVG `fill`/`stroke` 必须用 CSS 变量，不能写死色值；通过 `[data-theme="dark"]` 覆盖即可。

### 三个页面的可视化点

| 页面 | 顶部统计卡 | 中部主图 | 底部/侧栏 |
|---|---|---|---|
| GIS 地图 | 总节点数 / 总资源数 / 告警设备 / 在线率（取自 drill 聚合） | **八级树形 Canvas** + 点击下钻 + 节点大小/颜色映射 `count` | 右侧资源详情 Drawer（已有） |
| 经营分析 | 总收入 / 总投入 / 综合 ROI / 待维护设备数 | **三页签**：五大指标环形占比 / ROI 横向条形 / 利用率热力格 / 维护健康度分布散点 | 维护清单表（已有） |

> 经营分析原三页签（indicator/heatmap/maint）保留作为"明细视图"，新增中部主图作为"概览视图"，形成"概览 + 明细"双层。

| 报告中心 | 报告总数 / 本季已推 / 上次生成距今 / 告警覆盖度 | **报告快照时间线**（按 period 渲染日报/周报/月报/季报的指标曲线堆叠柱） | 报告正文 Drawer（已有）+ 结论摘要卡片 |

### 新增共享组件（`web/admin/src/components/business/charts/`）

```
charts/
├── index.ts           # barrel export
├── stat-card.tsx      # 顶部统计卡(沿用 dashboard 风格,≤40 行)
├── h-bar.tsx          # 横向条形(≤50 行,SVG 实现)
├── v-bar.tsx          # 纵向柱状(沿用 dashboard 风格,≤50 行)
├── donut.tsx          # 占比环形(≤80 行,SVG <circle> stroke-dasharray)
├── heat-grid.tsx      # 热力格子(≤100 行,Grid + bg 透明度渐变)
└── hierarchy-tree.tsx # Canvas 树形布局(≤150 行,递归 + Reingold-Tilford 简化版)
```

- 所有组件 props 都接收 `theme?: 'light'|'dark'`，内部用 CSS 变量而非硬编码。
- 单一职责：拿到数据 → 渲染图形，不做交互（点击跳转由父页面用 `<a>` 包裹或 onClick 接管）。
- 文件大小严格守 60 行内（donut/heat-grid ≤ 100 例外，因为 SVG 元素多）。

## 实施拆分（按可独立 revert 提交）

按契约 docs `docs/notes/README.md` 第 1 条"不可逆决策当天过账"，本方案未涉及密钥/ID/分合等不可逆决策，仅作为产品级方案记入 `adopted/`。

代码变更按以下顺序提交（每步可独立 revert）：

1. **commit 1: `refactor(web): 抽出 dashboard 统计卡与柱状为共享组件 charts/`**
   - 把 dashboard/index.tsx 内的统计卡/柱状函数下沉到 `components/business/charts/`
   - dashboard 改用共享组件，行为零变化
   - 验证：dashboard 视觉与功能不变；typecheck + test 通过

2. **commit 2: `feat(web): intel 经营分析页加指标概览(环形+条形+热力)`**
   - 仅在 `/intel/analytics` 顶部加可视区块，原三页签表格保留
   - 验证：双主题截图 + 真实接口数据断言

3. **commit 3: `feat(web): intel GIS 地图加八级树形布局`**
   - `/intel/gis` 上方加 Canvas 树（懒加载），原 drill 表格保留作明细
   - 验证：点击节点下钻交互、节点大小/颜色与 `count` 联动

4. **commit 4: `feat(web): intel 报告中心加快照时间线柱状`**
   - `/intel/report` 顶部加四类周期(日报/周报/月报/季报)指标对比堆叠柱
   - 验证：报告 Drawer 正文与新时间线联动

5. **commit 5: `chore(docs): 同步 i18n 三语 + fields.md`**
   - 新增 chart 相关 i18n key(zh-CN/en-US/ms-MY)
   - 同步 `docs/contract/fields.md` 中新增的 chart 字段命名

每个 commit 独立 typecheck + test + build 过门禁；commit 失败独立 revert 不影响邻居。

## 放弃了什么

- **引入 echarts-for-react**：900KB bundle + 默认主题与本项目 `--shell-*` 令牌体系不兼容；本期所有图 SVG 自绘。
- **GIS 上 leaflet + OSM 底图**：缺 GeoJSON + 外网瓦片不稳；本期用 Canvas 树形，下期接 gis 域坐标数据后再切。
- **后端新增 trend/历史时序接口**：超出本期目标，且改动 wiring+e2e 风险大；下期与 report 域时间窗口对齐时再做。
- **新增 CSS 变量**：所有色值复用现有 `--shell-*` + `--color-*`；新增变量会拉长令牌治理链路。
- **React Native Mobile 端联动**：mobile 端无 intel 模块，本方案不涉及。
- **新增 shadcn 图表组件**：shadcn 自身无 chart 组件，且 chart 库（recharts/echarts）需要 vendor 选择，不在本期决策内。

## 关联

- 现有 dashboard 可视化范式：`web/admin/src/pages/dashboard/index.tsx`
- 后端接口契约：`internal/httpapi/admin/gis_handlers.go`、`analytics_handlers.go`、`report.go`、对应 `internal/domain/{gis,analytics,report}/*`
- 字段对齐：`docs/contract/fields.md`（intel 三页表格列名 + 状态枚举）
- 设计令牌：`web/admin/src/styles.css`（`--shell-*` + `--color-*`）+ `web/admin/tailwind.config.*`
- 设计稿：`designs/intel-gis-v1.png`、`designs/intel-analytics-v1.png`、`designs/intel-report-v1.png`（gpt-image-2 生成，供评审）

## 待评审项

1. GIS 树形布局是否接受替代底图？若无替代需求，后续接入 leaflet 时需 amend 本 note。
2. 经营分析"概览 + 明细"双层是否多余？若产品希望直接以可视化为唯一视图，第 2 步可去掉原表格。
3. 报告中心时间线数据：本期无历史 trend 接口，时间线只能基于 `/reports/latest` 的单期快照演示，柱状显示"当前指标 vs 阈值"。若需要趋势，需后端补接口，下期单独立 note。