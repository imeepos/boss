# 数字孪生板块下一期:tileserver-gl 自建 + OL dark theme(2026-08-22)

日期：2026-08-22 v3(承接 2026-08-22-intel-visualization-upgrade v2)

> 用户 2026-08-22 直接指示"按方案执行下一期",范围确认：
> ① tileserver-gl 自建 PMTiles 上线(必做)
> ② OL dark theme 适配(必做)
> ③ 经营分析去掉原表格页签(默认推荐)
> ④ 报告中心接单期快照(本期留作下期,见"待跟进")
> ⑤ GIS 顶部 4 卡候选定型(本期做"层级点位/视域点位/在线点位/平均子级数")

## 决策

为数字孪生板块下一期落地三项核心改进：

1. **tileserver-gl 自建 PMTiles 上线**:`docker compose -f docker-compose.tiles.yml up -d`,
   PMTiles 文件由 ops 部署时拉取,前端保留 OSM 兜底直到 PMTiles 就位后切到 PMTILES_TILE_URL。
2. **OL dark theme 双瓦片源**:亮=OSM 公开瓦片;暗=CartoDB Dark Matter 免费层;
   自建 PMTiles 就位后切到 `/tiles/data/world-lowzoom/{z}/{x}/{y}.pbf`。
3. **经营分析页单视图化**:三页签(指标/热力/维护)收敛为单视图,保留维护清单(运维需看见每条设备)
   详情,指标/热力明细表格删除(信息已并入概览 4 卡 + 3 图)。

## why

### 为什么 tileserver-gl 选 PMTiles 单文件切片而非目录切片

- 部署物简单:2 个 `.pmtiles` 单文件 + config.json,无 mbtiles 目录结构;
- Protomaps 公开样例可直接 wget(无 OSM PBF 处理管线);
- HTTP Range Request 直接拉切片,无需 server-side 切片;
- tileserver-gl 5.x 已支持 PMTiles 原生(无需 plugin)。

### 为什么 dark theme 选 CartoDB Dark Matter 而不是 CSS filter

- CSS filter:invert(1) hue-rotate(180deg) 一行加 .ol-tile-layer,看似简洁但颜色严重失真
  (蓝色变橙色、文字不可读、shaded relief 失真);
- CartoDB Dark Matter 是公开免费层,与亮色 OSM 同一访问成本;
- 双瓦片源切换与 PMTiles 上线后切到自建底图是同一改造点(改 URL 即可),工作量零额外。

### 为什么经营分析去掉原表格页签

- v1 三页签(指标/热力/维护)信息密度爆炸,与"数字孪生"心智不符;
- 概览层 4 卡 + 3 图已覆盖指标+ROI+热力信息;维护清单表格保留(运维需要 StatusTag 优先级);
- 对齐设计稿"可视化为唯一视图";
- i18n 字段不动(append-only 规则,indicatorColumns/heatColumns/tabIndicator/tabHeatmap
  等保留以便 tab 模式后续复活)。

## 方案

### A1 依赖:pmtiles@^4.5.0

仅 npm install;tree-shaking 不进 bundle 直到 import。

### A2 maps/ 抽 TileSource 工厂 + PointLayer 主题感知

- `tile-source.ts`:`makeTileLayer(theme, url?)` 返回 OL TileLayer;
  `readDocumentTheme()` 读 `[data-theme]`;
  URL 常量 LIGHT_TILE_URL/DARK_TILE_URL/PMTILES_TILE_URL;
- `point-layer.ts`:`pointColor(status, theme)` 接受 theme 参数,暗主题提亮颜色。

### A3 PgisMap 接受 theme + tileUrl props

- mount 一次,主题变化仅替换 tileLayer;
- points/theme 变化重贴 vectorLayer;
- 业务页传 `theme={theme}` 即可。

### A4 i18n 三语 + types.ts dark theme + tileserver URL

- gisPage 加 themeLight/themeDark/themeSwitchHint + 4 个统计卡字段(statLevelNodes/statInBbox/statOnline/statAvgCount);
- 键集一致性由 `i18n/locales/keys.test.ts` 自动锁住(190 测试)。

### A5 GIS 页加 dark theme 切换 + 顶部 4 张统计卡

- 工具栏右上加亮/暗切换按钮(单页状态);
- 顶部 4 张 StatCard:当前层级点位 / 视域内点位(本期=当前层级,后续接 map.on('moveend'))/ 在线点位 / 平均子级数。

### A6 经营分析去掉三页签

- 删除 indicator/heatmap 表格;
- setTab 状态机改 Promise.all 并发拉三接口;
- 保留 maint 表格(StatusTag 优先级);
- i18n 字段保留不动。

### A7 deployments/tiles/ config.json + 健康检查文档

- `config.json`:tileserver-gl 标准配置,声明 2 个 PMTiles 数据源 + basic 样式 + /health 端点;
- `docker-compose.tiles.yml`:挂载 config.json/fonts,设 `TILE_SERVER_CONFIG`,healthcheck 加 `start_period: 30s`;
- `tiles/README.md`:完整部署手册(PMTiles 拉取命令 + 端到端 curl + 端点约定 + healthcheck 时序约束)。

### A8 决策 note + fields.md 同步

- 本文件;
- `docs/contract/fields.md` 补 PgisMap theme/tileUrl 字段映射。

## 放弃了什么

- **marts.server 端矢量切片(自建 tippecanoe 管线)**:工作量 +3 天,产出相同 PMTiles,选 Protomaps 公开样例;
- **dark theme 用 CSS filter**:颜色失真严重;
- **tileserver-gl 升级到 6.x**:docker 镜像仍是 5.14.0(已支持 PMTiles),无升级必要;
- **GIS 顶部统计卡做视域内点位实时计算**:接 map.on('moveend') + bbox 过滤,本期先展示"当前层级点位"作 placeholder,
  下期再补 moveend 监听(本期 KPI 卡片文案 statInBbox 留作提示);
- **tileserver-gl 启动时强校验 PMTiles 存在**:`/health` 503 触发 healthcheck 失败导致容器反复重启是预期行为,
  ops 拉 PMTiles 后自动通过;不在脚本层做空文件校验(增加复杂度零收益)。

## 关联

- 推翻对象:`adopted/2026-08-22-intel-visualization-upgrade.md`(v1 八级树形 Canvas 已合并,v2 PGIS 真地图已合并,本期是 v3);
- 后端契约:零改动,GIS 域 `/gis/points` 接口在 v2 commit 1 已落地;
- 前端 maps/:`tile-source.ts`、`point-layer.ts`、`pgis-map.tsx` 在 commit A2/A3 改造;
- deployment:`deployments/docker-compose.tiles.yml` 在 v2 commit 8 占位,本期 A7 实配;
- 设计稿:`designs/intel-gis-v1.png` 已在 v2 重出,本期未变;
- 反射规则:`docs/notes/adopted/2026-08-22-worktree-merge-protocol.md`(合并协议)+ self-evolving 红线 #9(ff-merge 失败严禁删 worktree)。

## 待跟进(下期)

1. **视域内点位实时计算**:GIS 页 commit A5 用 `points.length` 作 placeholder,本期补 `map.on('moveend')` + bbox 过滤 + 实时 statInBbox 更新;
2. **GIS 主题状态持久化**:本期 theme 是单页 useState,刷新即丢;后续接 settings 域或 localStorage;
3. **tileserver-gl 切到 PMTiles**:ops 拉 PMTiles 后改 `tile-source.ts` 的 `LIGHT_TILE_URL` 常量(`LIGHT_TILE_URL = '/tiles/data/world-lowzoom/{z}/{x}/{y}.pbf'`);
4. **报告中心接单期快照**:commit 7 已加 StackedBars,但内部数据来自 `view` payload;本期 view 是用户点击触发的单期快照,加载自动期切换需 `/reports` 返回 period index 关联,下期补;
5. **报告中心 trend 曲线**:方案 note 已记录"本期无历史 trend 接口",下期与 report 域对齐时间窗口时补;
6. **GIS 中心区暗色瓦片 + 文字对比度 polish**:目前暗色点位 #E5C985/#9CAAC3 是基础调,实景截图后可能需调透明度/字号;
7. **bundle size 监控**:commit A3 引入 pmtiles 后跑 `pnpm build` 检查 chunk size,若 > 500KB 增量需拆 chunk。