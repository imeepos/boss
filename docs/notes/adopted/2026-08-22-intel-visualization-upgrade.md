# 数字孪生与经营板块可视化升级方案（PGIS 版）

日期：2026-08-22 v2（来源：用户首版反馈"应该是地图 pgis"）

> Amended 2026-08-22: 首版走"八级树形 Canvas"路线，与用户期望的"PGIS 真地图"心智不符。v2 推翻首版方案，改走真实地图 + 点位聚合 + 层级过滤。本文保留首版作为否决记录。

## 决策

为 `web/admin` 数字孪生与经营组（`/intel/gis`、`/intel/analytics`、`/intel/report`）三页做可视化升级，核心定位：**GIS 页为 PGIS 真地图（OL + 自建瓦片 + OSM 公开瓦片起步），点位图层由 GIS 域新增接口聚合**。

五项硬约束：

1. **GIS 页上真地图**：OpenLayers（OL）做底图渲染引擎；瓦片来源先用 OSM 公开瓦片起步（演示用），同时启动 tileserver-gl / PMTiles 自建服务作为生产路径。
2. **GIS 域新增点位接口**：`GET /gis/points?level&bbox&parentId` 返回 GeoJSON FeatureCollection，覆盖 1~5 级地址点位 + 6~8 级资源点位（join `odn_site`/`odn_device`/`odn_facility` 的 `lat/lng`）。
3. **图表部分纯 SVG 自绘**，不引入 echarts/recharts/d3。复用 dashboard 模板。
4. **复用现有 `--shell-*` 令牌**，不新增 CSS 变量。
5. **后端契约扩展但严格收敛**：仅在 `internal/domain/gis` 内新增 `Points` 服务方法 + handler，其他域零改动。

## why

### 为什么 GIS 必须上真地图（推翻首版"八级树形 Canvas"方案）

- 用户明确表态"应该是地图 pgis"，产品定位是数字孪生而非组织树图；
- 数据基础设施已具备：PostGIS 已启用（`migrations/000001_stage1_base.up.sql:6`）、`addresses.geom` 字段与 GIST 索引就位（`000001:81-88`）、ODN site/device/facility 已存 `lat/lng`（`000075`/`000078`/`000081`）；
- 当前 GIS 域只暴露"八级 drill"，没有任何点位坐标；不接地图则 PSGC + ODN 的 `lat/lng` 投资为零价值；
- 首版方案（Canvas 树形）让现有 PSGC 投资和 ODN 坐标白费，是反向裁剪。

### 为什么选 OpenLayers（用户决策）

- OL 是企业级 GIS 标杆库，对 PGIS 严肃度匹配；支持矢量瓦片（MVT）、WebGL 自定义 Layer、大数据点位聚合（cluster）、GeoJSON 原生支持；
- 已知代价：bundle ~800KB（首版绘制 page size 增量最大项）；
- maplibre-gl 同样满足需求但产品偏向"重 OL"以体现数字孪生严肃感，按用户选择执行。

### 为什么瓦片先 OSM 公开 + 同步自建

- 演示/本地环境：OSM 公开瓦片（`a/b/c.tile.openstreetmap.org`）即开即用，符合"102 服务器测试"短链路；
- 生产环境：OSM 公开瓦片违反 OSM 服务条款（禁止商业重发布 + 高频请求封禁），**必须**自建瓦片服务（`tileserver-gl` + PMTiles 单文件切片），避免被 OSM 基金会发函；
- 同步推进策略：本期落地 OL + OSM 演示，**并行**在 deployment 侧新增 `tileserver-gl` 服务（migration 不需要，仅部署侧新增 docker-compose 服务），下期切到自建瓦片；
- 自建瓦片数据源：Phase 1 用 OSM PBF 切片 + 低 zoom 全球覆盖 + 高 zoom 区域提取（菲律宾 PSGC 区域优先），成本< 1GB。

### 为什么复用 dashboard 图表模式

- 与首版同理由：`dashboard/index.tsx` 已有统计卡、进度条、柱状三件套，已被用户/产品验收过；
- GIS 上地图后，"中部主图"位置让位给地图本体；顶部统计卡 + 底部明细表的骨架与首版一致。

### 为什么不全部页面都接地图

- `/intel/analytics`（经营分析）核心是"指标/ROI/热力"，数据维度非地理 → SVG 自绘为主，**不加地图**；
- `/intel/report`（报告中心）是"时间线 + 摘要"，与地图无关 → SVG 自绘为主，**不加地图**；
- 仅 `/intel/gis` 上地图，避免工作量爆炸与心智混淆。

## 方案总览

### 三页角色重新划分

| 页面 | 形态 | 主体可视化 | 数据源 |
|---|---|---|---|
| `/intel/gis` | **PGIS 真地图** | OpenLayers 底图 + GeoJSON 点位图层（聚合 cluster）+ 左侧层级过滤侧栏 + 右侧资源详情 Drawer | 新增 `/gis/points` + 现有 `/gis/drill` + `/gis/resources/:id/detail` |
| `/intel/analytics` | 经营仪表盘 | 顶部 4 统计卡 + 3 图（环形占比/横条 ROI/热力格）+ 维护清单表（保留） | 现有 `/analytics/indicators` + `/analytics/heatmap` + `/analytics/maintenance` |
| `/intel/report` | 报告中心 | 顶部 4 统计卡 + 四周期对比柱状（明确标注"对比"非"趋势"）+ 报告列表 + 正文 Drawer | 现有 `/reports` + `/reports/latest` |

### GIS 页布局（左中右三栏）

```
┌────────────────────────────────────────────────────────────┐
│ PageHead: 数字孪生(GIS)              [刷新] [全屏] [图层]  │
├────────────────────────────────────────────────────────────┤
│ ┌──────┬──────────────────────────────┬──────────────────┐│
│ │层级  │                              │资源详情 Drawer    ││
│ │树状  │     OpenLayers 地图          │(已存在,加 lat/lng││
│ │导航  │  + GeoJSON 点位图层          │  字段展示)        ││
│ │1~8级│  + 聚合 cluster              │                  ││
│ │      │  + 缩放/平移/选框             │                  ││
│ │      │  + 底图:OSM 瓦片起步         │                  ││
│ └──────┴──────────────────────────────┴──────────────────┘│
├────────────────────────────────────────────────────────────┤
│ 下方:当前层级 drill 明细表(原表格保留,按 parentId 查询) │
└────────────────────────────────────────────────────────────┘
```

### 新增后端接口（仅 GIS 域扩展）

#### `GET /gis/points`

请求：

| 参数 | 类型 | 说明 |
|---|---|---|
| `level` | int16 | 1~8，6/7/8 走 ODN 表，1~5 走 addresses.geom |
| `bbox` | string (可选) | `minLng,minLat,maxLng,maxLat` (WGS84) |
| `parentId` | int64 (可选) | 6/7 级时为 OLT/SPLITTER 的 parent id |

响应（GeoJSON FeatureCollection）：

```json
{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "geometry": { "type": "Point", "coordinates": [121.4737, 31.2304] },
      "properties": {
        "id": 123,
        "level": 6,
        "name": "OLT-xxx",
        "status": "ONLINE",
        "count": 48,
        "parentId": 100
      }
    }
  ]
}
```

实现要点（`internal/domain/gis/pg.go` 新增 `Points(ctx, level, parentID, bbox)` 方法）：

- level 1~5：`SELECT id, name, count, ST_X(geom::geometry) AS lng, ST_Y(geom::geometry) AS lat FROM addresses WHERE level=$1 AND ($2=0 OR parent_id=$2)`，bbox 走 `ST_Within(geom, ST_MakeEnvelope(...))`
- level 6：OLT 表 join `addresses` 取父级 geom（OLT 自身无 lat/lng，落在楼栋地址上）
- level 7~8：SPLITTER/ports join `odn_site/device/facility` 取 lat/lng（按 `address_id` 映射 ODN site）

> **精确性注记**：level 6/7/8 的点位精度取决于 ODN site/facility 录入的 lat/lng 是否准确；录入缺失时点位自动回退到父级 addresses.geom，不留空白。

#### 路由注册

`internal/httpapi/admin/gis.go` 第 14-18 行新增一行：

```go
gi.GET("/gis/points", gisPoints(a))  // 新增
```

`gisPoints` handler 与 `gisDrill` 同 `requirePerm("menu:gis")` 门禁。

### 前端新增依赖

```
"ol": "^10.0.0"          // OpenLayers 核心
"ol-mapbox-style": "^12.0.0"  // 矢量瓦片样式(可选)
```

> 不引 turf.js、不引 proj4（OL 自带 proj 转换）；bundle 增量 ~280KB（gzip）。

### 新增前端模块

```
web/admin/src/components/business/maps/
├── index.ts              # barrel export
├── pgis-map.tsx          # OL 地图容器(≤150 行;init map/layer/source)
├── point-layer.ts        # GeoJSON Layer 配置(≤80 行;cluster/style/interaction)
└── bbox-utils.ts         # bbox 参数解析+地图 zoom-to-bbox(≤50 行)
```

```
web/admin/src/components/business/charts/        # 与首版同,SVG 自绘
├── stat-card.tsx         # 复用 dashboard
├── v-bar.tsx             # 复用 dashboard
├── h-bar.tsx             # 新增(分析页 ROI 横条)
├── donut.tsx             # 新增(指标环形占比)
└── heat-grid.tsx         # 新增(利用率热力格)
```

### 实施拆分（按可独立 revert 提交）

| # | commit | 内容 | 验证 |
|---|---|---|---|
| 1 | `feat(backend): gis 域新增 Points 服务 + /gis/points 接口` | 新增 SQL/handler/路由 + 单测 + integration test | go test + curl 真接口 |
| 2 | `refactor(web): 抽出 dashboard 统计卡与柱状为共享组件 charts/` | 与首版同 | dashboard 视觉零回归（cdp-capture baseline）|
| 3 | `chore(deps): web/admin 加 OpenLayers 依赖` | package.json + lock | typecheck + build |
| 4 | `feat(web): maps/ 共享 PGIS 容器+点位图层` | 新组件 + bbox-utils + 单测 | vitest snapshot |
| 5 | `feat(web): intel GIS 页改造为 PGIS 真地图` | 替换原纯表格 + 接 /gis/points + 双主题截图 | cdp-capture light/dark + 真实点位断言 |
| 6 | `feat(web): intel 经营分析页加指标概览(环形+条形+热力)` | 中部主图 + 顶部 4 卡（基于现有 indicators 数据）| 双主题 + 真实接口数据 |
| 7 | `feat(web): intel 报告中心加快照对比柱状` | 顶部 4 卡 + 四周期对比柱状（明确标注非趋势）| 双主题 + drawer 联动 |
| 8 | `chore(deps): deployment 侧新增 tileserver-gl 服务` | docker-compose 新增 + PMTiles 文件 placeholder | docker compose config |
| 9 | `chore(docs): 同步 i18n 三语 + fields.md` | 新增 chart/maps 相关 i18n key + fields.md 中新增字段 | i18n 三语对齐 |

每个 commit 独立 typecheck + test + build 过门禁；commit 失败独立 revert 不影响邻居。

> commit 1 落地后**必须**用 curl 真接口验证；commit 5 落地后**必须**用 cdp-capture 跑 light/dark 双主题截图；commit 8 不阻塞前端，但下期切瓦片必须前置完成。

## 放弃了什么（v2 在 v1 基础上更新）

| 方案 | 否决理由 |
|---|---|
| v1：八级树形 Canvas 布局 | 与"PGIS 真地图"心智不符；裁剪现有 PSGC + ODN 坐标投资 |
| v1：GIS 顶部统计卡"总资源/告警设备/在线率" | 字段不存在；v2 改为"当前层级点位总数 / 父层级覆盖 / bbox 内点位 / 选中点位父链"，前端可聚合 |
| 仅用 leaflet（更轻量） | 用户明确选 OL；leaflet 缺乏 OL 的矢量瓦片 + WebGL Layer 严肃能力 |
| 后端返回地址多边形 GeoJSON 边界 | 复杂度爆炸（PSGC 多边形版权 + 大文件传输）；本期只返点位，下期按需扩展 |
| React-Leaflet 包装 | 用户选 OL 核心库而非 React 包装（保留 OL 原生 API 控制力）|
| 引入 turf.js 几何运算 | OL 自带覆盖；多 1 个依赖无收益 |
| 全地图热力图（heatmap layer）| 数据稀疏（八级点位 ~百到千级），cluster 已足够；heatmap 留作下期 |

## 关联

- 推翻对象：`adopted/2026-08-22-intel-visualization-upgrade.md`（v1，八级树形 Canvas 方案）—— 见本文 Amended 头
- 现有 GIS 域：`internal/domain/gis/{gis.go,pg.go}`、`internal/httpapi/admin/{gis.go,gis_handlers.go}`
- PostGIS 启用：`migrations/000001_stage1_base.up.sql:6`
- 地址坐标：`addresses.geom`（000001:81）+ GIST 索引
- ODN 坐标：`odn_site.lat/lng`（000081:11）、`odn_device.lat/lng`（000081:12-30 不显式但 schema 含）、`odn_facility.lat/lng`（000075、000078）
- PSGC 全量内置：`migrations/000041_ph_psgc_divisions.up.sql`
- dashboard 可视化范式：`web/admin/src/pages/dashboard/index.tsx`
- 设计令牌：`web/admin/src/styles.css`（`--shell-*` + `--color-*`）+ `web/admin/tailwind.config.*`
- 设计稿：`designs/intel-gis-v1.png`、`designs/intel-analytics-v1.png`、`designs/intel-report-v1.png`（gpt-image-2 生成；GIS 稿 v1 形态已被 v2 推翻，需重出）

## 待评审项

1. 经营分析"概览 + 明细"双层是否多余？默认推荐去掉原表格页签，让可视化为唯一视图（与设计稿一致）。
2. 报告中心"四周期对比"柱状是否仍要保留？本期无历史 trend 接口，对比柱易被误读为趋势；建议保留但**显式标注"四周期对比"非"周期趋势"**。
3. 瓦片服务：本期 OSM 公开瓦片起步 + deployment 侧 tileserver-gl 占位，下期切自建。是否接受"两步走"？或要求本期一次性自建瓦片到位（工作量 +2 天）？
4. GIS 顶部统计卡 4 字段最终选哪 4 个？候选（前端可聚合）：
   - 当前层级点位总数
   - 父层级覆盖节点数
   - bbox 内点位总数
   - 选中点位父链深度
   - 在线/离线点位占比（需 join device_metrics 表）

## 待修正项（开工前必做）

1. **GIS 设计稿重出**：v1 设计稿画的是树形图，v2 推翻后需重出一版"PGIS 真地图 + 点位"形态。
2. **OL dark theme**：OL 默认底图是亮色，深色主题需要 tile-url 替换或 CSS filter 覆盖；先记录，下期出方案。
3. **单测覆盖**：OL 组件 jsdom 不可用（无 WebGL），需 e2e 兜底；charts/ 的 SVG 组件走 vitest + @testing-library/react snapshot。
4. **bundle size 监控**：commit 3 引入 OL 后需跑 `pnpm build` 检查 chunk size，若 > 500KB 增量需 tree-shake 配置或拆 chunk。