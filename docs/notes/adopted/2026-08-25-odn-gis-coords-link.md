# ODN 资源坐标补齐与 GIS 图层关联

日期：2026-08-25

## 背景

用户要求：**所有物料都要有地理位置，与地图关联**。此前 ODN 无源物理层（000075/78/79/81）与 GIS（阶段8）是"两张皮"：
- `odn_facility`（000078）、`odn_site`（000081）已有 `lat/lng` 列；
- `odn_device`（000081）**无坐标列**，核心链路设备无法上图；
- `internal/domain/gis` 的 `Points` 注释声称 "level 7~8 SPLITTER/ports join odn_* 取 lat/lng"，但 `pg_points.go` 实际实现**从未 join ODN 表**（注释与实现相悖，调研确认）；
- GIS 前端 `/intel/gis` 地图点位只来自地址（addresses.geom）+ 逻辑资源（resources/ports），无 ODN 设施/局点/设备点位图层；
- ODN 管理页 `/oss/odn` 表格不展示坐标、表单不录入坐标（后端 Facility/Site 结构有 Lat/Lng 但前端类型未暴露）。

## 决策

1. **迁移 000142**：`odn_device` 补 `lat`/`lng` 可空列。历史数据无坐标不受影响；无坐标设备不上地图点位（GIS 查询侧过滤）。
2. **GIS 域新增独立 ODN 点位查询**：`GET /gis/odn-points?entity=facility|site|device&bbox`（`menu:gis`）。ODN 实体自带 `lat/lng`（非 PostGIS geom），bbox 用 `lng/lat BETWEEN` 数值区间过滤，不复用 `Points` 的 `ST_Within`（pg_points.go 的 bboxToSQL 硬编码 `a.geom`）。
3. **图层语义**：ODN 点位 level 9=设施 / 10=局点 / 11=设备，与八级 drill（1~8）正交，**不并入 drill 层级**——避免破坏现有八级下钻契约（`/gis/levels` 前端消费 1~8）。
4. **前端**：ODN 管理页表单加 lat/lng 录入（设施/局点/设备），表格加坐标列；GIS 页加 ODN 图层下拉（关/设施/局点/设备），点位叠加进 OpenLayers 地图。ODN 点位（level 9~11）点击**不触发** `/gis/resources/:id/detail`（该接口只对逻辑资源有效）。

## why

- ODN 规范（docs/pdfs）要求设施/局点可上图；用户明确"所有物料都要有地理位置 与地图关联"，设备坐标是唯一缺口。
- 坐标可空：存量数据无坐标（规划部台账未录入），强制必填会阻塞建档；地图侧过滤空坐标行，点位完整度随录入逐步提升。
- 独立图层而非并入 drill：八级 drill 的 parent/level 语义已被大屏、报表、`/gis/levels` 消费，混入 ODN 实体需改 level 枚举与 drill SQL，收益低于独立端点。
- 设备坐标由 ODN 域自行维护（CreateDevice 带 Lat/Lng），不与 `resources`（逻辑资源层）耦合——延续 E10/E11"ODN 码与 BOSS 资源码独立命名空间"裁定，物理层与逻辑层各自上图。

## 放弃了什么

- 把 ODN 点位并入八级 drill（level 9/10/11 塞进 Drill/Points 同一 switch）：破坏既有 level 语义与前端 levels 下拉，否决。
- 设备坐标改为从 `odn_site` 继承：市域设备（site_no 空）无局点可继承，且设备可同址扩容，独立坐标更准确。
- 前端地图点选录入坐标：本期手动输入打通数据链路，点选留待后续（避免 GIS 页与 ODN 页交互耦合）。

## 关联

- 迁移：000142（odn_device 坐标列）
- 契约：`api/openapi/admin/intel.yaml` `/gis/odn-points`；fields.md §1.5.5
- 域：`internal/domain/gis`（ODNPoints）、`internal/domain/odn`（Device 坐标）
- 前端：`/oss/odn`（坐标录入/展示）、`/intel/gis`（ODN 图层）
