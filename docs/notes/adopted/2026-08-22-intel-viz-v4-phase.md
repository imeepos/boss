# 数字孪生板块下一期 v4:GIS 收尾 + report trend(2026-08-22)

日期：2026-08-22 v4(承接 2026-08-22-intel-viz-next-phase v3)

> 用户 2026-08-22 直接指示"制定下一期的开发计划 制定开发方案 然后按计划和方案执行开发任务",
> 范围确认：①GIS 视域内点位实时计算 + ②GIS 主题持久化 + ③报告 trend 曲线(新增 /reports/history 后端接口)。

## 决策

本期落地 v3 待跟进 7 项中的关键 3 项,每项独立可 revert:

1. **B1 后端 /reports/history 接口**:`report.Store` 加 `LatestSnapshots(period, limit)`,
   `report.ReportService.History(ctx, period, limit)` 业务包装,handler 返回解出 payload 后的 items。
2. **B2 PgisMap onViewportChange 回调**:B3 基础,view extent → EPSG:4326 → 业务页消费。
3. **B3 GIS 页接 moveend 实时计视域内点位**:统计卡第 2 张"视域内点位"接真数据(替换 points.length placeholder)。
4. **B4 抽 useLocalStorage<T> hook**:SSR 安全 + 跨 tab 同步 + quota 抛错兜底;为 B5 提供基础设施。
5. **B5 GIS 主题状态 localStorage 持久化**:useState → useLocalStorage,刷新/重开 tab 保留用户选择。
6. **B6 报告中心 trend 曲线**:新增 LineTrend SVG 折线组件;report 页消费 /reports/history。
7. **B7 本文件 + fields.md 同步**:本 note;fields.md 补 trend/history 字段;i18n keys test 锁住。

## why

### 为什么 B1 在 report 域扩展而非新增 analytics 域

- `report_snapshots` 表已存在(`migrations/000021` 起),窗口 upsert 幂等覆盖天然支持 trend;
- `ReportService` 已封装窗口/快照语义,History 只是同表 N 行读取;
- analytics 域关注"当前期聚合",trend 是"历史快照"语义,边界清晰。

### 为什么 B2 用 onViewportChange 回调而非透传 extent

- 业务页不需要知道 OL 存在(本轮已封装在 maps/);
- 回调仅暴露 WGS84 bbox(与 /gis/points 同形),保持 maps/ 是纯渲染层;
- 后续 OL 升级或换 maplibre 不影响业务页。

### 为什么 B4 用 globalThis.localStorage 而非 window.localStorage

- SSR 环境无 window,代码若用 `window.localStorage` 会直接 throw ReferenceError;
- node 测试环境无 window,`globalThis.localStorage` 同样存在;
- vi.stubGlobal('localStorage', ...) 只设 globalThis,设不到 window。
- globalThis 写法同时 SSR 安全 + 测试友好,代价仅一行 typeof 缩为 truthy 检查。

### 为什么 B6 用 LineTrend SVG 自绘而非 echarts

- 报告页其余图表(StackedBars/Donut/VerticalBars)均自绘,引入 echarts 破坏一致性;
- LineTrend 5 条线 × 12 个点 = 60 个数据点,<polyline> 一行绘出,无 canvas 性能问题;
- 与 charts/ 其它组件共用 PALETTE 调色板,视觉一致。

## 方案

### B1 /reports/history 接口

- `internal/domain/report/report.go`:Store 加 `LatestSnapshots(period, limit)`;Service 加 `History(ctx, period, limit)`(limit 默认 12,上限 90,handler 二次校验);
- `internal/domain/report/pg.go`:SQL `SELECT ... FROM report_snapshots WHERE period = $1 ORDER BY window_start DESC LIMIT $2`;
- `internal/httpapi/admin/analytics.go`:注册 `GET /reports/history`;
- `internal/httpapi/admin/analytics_handlers.go`:`reportHistoryHandler` 解 payload 后返 items;
- 4 个单测覆盖:History_LimitDefaults / History_FilterByPeriod / Latest_None / Store 行为;
- `userdata_gap_test.go` 的 fakeReportStore 补 LatestSnapshots 方法(因接口扩展)。

### B2 PgisMap onViewportChange

- `pgis-map.tsx` 加 `onViewportChange?: (b: ViewportBbox) => void` prop;
- 订阅 `map.on('moveend'/'zoomend')` → `calculateExtent` + `transformExtent(EPSG:3857 → EPSG:4326)`;
- mount 后立即触发一次(初始视域);
- OL `on/un` 多重 OnSignature 联合类型用 loose cast 绕过(`as unknown as (k: string, ...) => ...`)。

### B3 GIS 页视域内点位

- `gis/index.tsx` 加 `inBbox` state + `bboxRef` 缓存最新视域;
- `onViewportChange(b)` 同步:`bboxRef.current = b` + 过滤 points(lng/lat 在 bbox) → `setInBbox(n)`;
- points 变化 useEffect 用上次 bbox 重算(避免 points 拉取后用户未动地图 inBbox 不刷新);
- 顶部第 2 张 StatCard value=inBbox(替换原 placeholder points.length)。

### B4 useLocalStorage hook

- `lib/useLocalStorage.ts`:`useState` 包装 + `useCallback` 写回 + `useEffect` 跨 tab 同步;
- `readStorage` 用 `globalThis.localStorage` 兜底 SSR;
- 接受 `(key: string, initial: T) => [T, setter]`,与 React useState 同形;
- 6 个单测:初始/读回/解析兜底/SSR/Quota/签名。

### B5 GIS 主题持久化

- `useState<Theme>('light')` → `useLocalStorage<Theme>('intel.gis.theme', 'light')`;
- key 加 'intel.' 前缀避免与其它 feature 撞;
- 未来 settings 域接入时可平滑迁移(只需替换 hook 实现)。

### B6 LineTrend + report trend 视图

- `charts/line-trend.tsx`:SVG viewBox 720 自适应宽,<polyline> + 圆点;
  每条指标线各自 max 归一;5 段品牌色 PALETTE;legend 下方;
- 5 个单测:空数据/polyline 数/circle 节点/legend 渲染;
- `charts/index.ts` 加桶导出;
- `report/index.tsx` 改造:
 - trendPeriod state + loadTrend(trendPeriod) useEffect;
 - `apiFetch('/reports/history', { query: { period, limit: 12 } })` 拿真历史;
 - trendSeries:5 条指标线(收入/投入/ROI/告警/待维护 MUST_REPLACE);
 - trendLabels:窗口起点时间戳 slice(5,10)=MM-DD;
 - 新 CardShell:LineTrend + 右上 select 切周期;
 - 快照 < 2 显示 empty 占位;
 - 保留原 StackedBars "四周期对比"卡(view 概览未动)。
- i18n 三语 + types.ts 加 3 个 key(trendTitle/Desc/Empty)。

## 放弃了什么

- **B1 走后端 trend 缓存表(同窗口聚合)**:窗口表已做 upsert 覆盖,N 行读取够用;预聚合要新增 migration+定时任务,工作量 +2 天零收益;
- **B2 透传 OL extent 给业务页**:耦合过紧,业务页需要自己解析 EPSG:3857 与转换;回调 bbox(WGS84)是更稳的边界;
- **B4 localStorage → IndexedDB**:hook 用途小(几 KB JSON),IndexedDB 异步 API 增加复杂度零收益;
- **B5 同步到 settings 域**:本期本机能用即可;settings 域接入时再平滑迁移;
- **B6 echarts-for-react**:与 charts/ 其余组件调色板不一致,引入会破坏视觉;
- **B6 跨指标归一(scale per metric)**:v0 各指标量纲差异(收入 vs ROI vs 告警数)实在太大,归一后折线高度还能看出趋势;绝对值未必要在视图呈现,业务点详情可看真值。

## 关联

- 承接对象:`adopted/2026-08-22-intel-viz-next-phase.md`(v3 待跟进 7 项,本期做 3 项关键)
- 后端契约扩展:`internal/domain/report/`(只读扩展,无 migration)
- 前端 maps/:`pgis-map.tsx` 加回调;`lib/useLocalStorage.ts` 新增;
- 前端 charts/:`line-trend.tsx` 新增(七件套);
- 前端 pages/:`gis/index.tsx` 接 B2+B3+B5;`report/index.tsx` 接 B6;
- 反射规则:`self-evolving` 红线 #9(worktree 收尾)+ AGENTS.md worktree 合并协议。

## 待跟进(再下一期 v5)

- v3 待跟进剩余 4 项:③tileserver-gl 切到 PMTiles 实际切换(LIGHT_TILE_URL 改常量);④报告中心接单期快照自动化(定时任务);⑥GIS 中心区暗色瓦片文字对比度 polish;⑦bundle size 监控跑 `pnpm build` 报告;
- v4 新增:LineTrend 跨指标归一(scale per metric)对比模式(本期绝对值/各自归一,后续业务点详情可加);
- v4 新增:useLocalStorage 接入 settings 域(平滑迁移);
- v4 新增:`/reports/history` 拉满 12 个窗口的 e2e 验证(目前 0 快照场景;真正用要定时任务);
- 后端:`/reports` 自动 generate 定时任务(目前仅手动 trigger);
- 前端:e2e 测试覆盖地图/趋势(PgisMap 走真实浏览器 e2e,LineTrend 可 vitest snapshot)。
