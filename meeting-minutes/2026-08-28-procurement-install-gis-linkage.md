# 会议纪要：采购-库存管理-安装施工-GIS 联动

> 日期：2026-08-28｜主持：会议主持人｜工作目录：boss
> 议题：在现有 BOSS 装维全流程系统中加入"采购-库存管理-安装施工-GIS 联动"功能模块
> 性质：方案评审（未进入编码阶段；落地前用户确认）

---

## 与会人

| 姓名 | 角色 | 视角 |
|---|---|---|
| 墨白 | 系统架构师 | 9 阶段路线、域边界、状态机、契约登记、迁移号 |
| 云舟 | 资产·采购域专家 | asset 表结构、批次/盘点/换新单、采购入库与库存 |
| 凛冬 | 订单·施工域专家 | 12 环节状态机、派单、施工回单 |
| 清辉 | GIS 联动域专家 | gis 域、Kafka 拓扑、图层、瓦片 |
| 鹿野 | 前端 admin 体验专家 | admin 菜单分组、组件、设计系统 |

---

## 共识（5 人一致）

1. **12 环节状态机零改动**：terms.md §1 明令禁止增删改序；"到场/在途/施工进度"等只能在工单层派生，**不得**插入环节表（参考 RETRYING 派生展示态先例）。
2. **GIS 联动保持环节 12 updateMap 单次幂等**：不引入"多次 updateMap"；环节 12 自身 selfHeal，可放心让图层靠事件流式刷新。
3. **install_logs（施工回单）独立建表**：承载现场照片/签收/备注，回单齐才允许工单置 DONE。材料消耗复用既有 `worker_materials`，不重复建表。
4. **资产状态机枚举不变**：`IN_STOCK/DEPLOYED/MAINTENANCE/SCRAPPED`（terms §4）保留；新增的"采购/入库/库存事务/施工回单"枚举在 terms §4 同提交登记。
5. **核心约束统一**：单文件 ≤300 行、函数 ≤60 行；字段命名按 fields.md §0；后端对接 102 部署环境 `http://192.168.0.102:28080`；不可逆决策当日落 `docs/notes/adopted/`；worktree 合并协议遵守；失败路径必须有 `[module] ... FAILED|ALERT` 级可 grep 日志。

---

## 分歧与 battle（严重冲突）

### Battle 1：采购-库存是否独立建 `internal/domain/procurement`？

| 立场 | 主张 | 论据 |
|---|---|---|
| **墨白** | 独立建域 `internal/domain/procurement` | domain-map 21 域虽无"采购"域，但 AMS/asset 管单件生命周期，采购管供应商/批量/钱，职责异构；塞 asset 必撞 300 行红线与跨域 import 禁令（domain-map §4 规则 2）；先例 CMS/OPENPLAT/APPREL 均"增量"挂靠 |
| **云舟** | 不新建域，扩展 `internal/domain/asset`（新建 4 表 + 分文件） | 新表与 asset_batches 通过 batch_id 关联，单事务防孤儿；"分文件"即可避开 300 行 |

**现场证据**：
- 主持人实测：`internal/domain/asset/pg.go` 当前 228 行（未到 300 红线），但 `pg_binding.go/pg_replacement.go/pg_stocktake.go/pg_write.go` 共 5 个文件累计 1304 行——**域整体已重**。
- asset 域 000158、000159、000156 等迁移长期迭代，文件持续膨胀；新加 4 张表必然继续推高。
- domain-map.md §4 规则 2："禁止跨 A 域 import 他域 implementation（只经契约/事件）"——本议题的核心目标是给"采购"立独立语义边界，混进 asset 会让"供应商/批量/钱"被当资产生命周期管理，越界。

**主持人裁决**（依据：现场事实 + domain-map §4 规则 2 + 单文件 ≤300 行）：

> **采纳墨白方案**：新建 `internal/domain/procurement` 包。**例外**：云舟坚持"single transaction 防孤儿"是对的——`procurement_receipts` 与 `assets`（通过 batch_id）的双写必须在 procurement 包内的事务里完成，但**不直连 `internal/domain/asset` 的实现**，经 `procurement_orders.CreateReceipt(ctx, …)` 公开方法，asset 包反向调用（即 procurement 域暴露服务接口、asset 域注入依赖），既保事务原子性又保域边界。
>
> 同时按鹿野的菜单建议（前端视角）——这与"包独立"不冲突：菜单挂 ams 组（按业务动线），后端实现归属 procurement 域（按职责）。

### Battle 2：GIS 联动拓扑（直调 vs Kafka）

| 立场 | 主张 |
|---|---|
| **墨白**（事实纠偏） | "代码中不存在 asset.changed 事件，GIS 联动实为环节 12 同步直调" |
| **清辉**（事实纠偏） | cmd/gis 是 Kafka 消费者（topic `boss-order-events`），按 `stage==12` 过滤触发同步 |

**主持人实测**（`internal/domain/order/pg_workflow.go:232` `UpdateMap` 函数 + `cmd/gis/main.go:36` `Handle`）：

> 两者**并存**：order 域的 `UpdateMap` 是同步直调函数（订单侧写入链路），cmd/gis 又是 Kafka 消费者（运维侧冗余触发）。两者目的不同：order 域 UpdateMap 走业务流程；cmd/gis 走异步入库保障。**这是"双链路"事实**，双方都不完整。
>
> **主持人裁定**：当前架构保留双链路。新增 `inventory.changed` 事件（清辉建议）经同一 topic `boss-order-events`，type 区分（清辉建议，避免繁衍 topic），cmd/gis 消费者扩 type 集合；`install.progress` 不新立、复用 stage 事件流；`worker.location` **不进 Kafka**（清辉建议，写放大风险），经 `worker_locations`（迁移 000144）POST 直写。

### Battle 3：底图坐标系（事实风险）

**清辉实测**：
- `web/admin/src/components/business/maps/tile-source.ts` **L12**：`LIGHT_TILE_URL='https://webrd01.is.autonavi.com/appmaptile?lang=zh_cn&size=1&scale=1&style=8&x={x}&y={y}&z={z}'`（高德 webrd，GCJ-02）
- `web/admin/src/pages/intel/gis/index.tsx` **L162** `<PgisMap …/>` 未传 tileUrl → 默认走高德
- `docs/contract/fields.md` **L880** 原文"默认 OSM 公开瓦片 + CartoDB Dark Matter"——**「默认 OSM」与代码不符**（亮色实为高德）。且该行页面路径记 `admin/gis.html`，实际已是 `web/admin/src/pages/intel/gis/index.tsx`，**路径引用同步过时**。

**主持人裁定**（这是事实冲突，先登记待办）：

> 1. **当日必改 `docs/contract/fields.md` L880**：把"默认 OSM"修正为"默认高德栅格（GCJ-02）"；页面路径修正为 `web/admin/src/pages/intel/gis/`。
> 2. **WGS84 vs GCJ-02 偏移决策（待用户拍板）**：两条路 ——
>    - 路 A：点位采集口径改为 GCJ-02（与底图一致），代价是改 worker_locations/odn_device 等所有 WGS84 字段
>    - 路 B：底图换 WGS84（OSM 或自建 PMTiles），代价是 tile-source.ts 默认 URL 改写
>    - 路 C：前端点位图层渲染时做坐标转换（gcoord.js），代价是每帧加一个 lib
>    **建议**：本期走路 B（自建 PMTiles 是 README §阶段8 已规划路径，迟早要做），本期先换默认 OSM/CartoDB 不一致问题，把高德栅格迁到 dark/light 全部统一为 WGS84（OSM），预留 PMTILES_TILE_URL 切换点已就位。**此决策未拍板前不动底层代码，先记入 `docs/notes/adopted/2026-08-28-tile-coordinate-rationale.md`**。
> 3. `cmd/gis` 消费器失败路径补 `[gis] ... FAILED` 级可 grep 日志（清辉指出"现 log.Printf 静默 continue"，违反 AGENTS.md 失败路径可观测信号铁律）。

---

## 主持人综合方案（融合 5 人发言）

### 1. 域划分与新包

| 能力 | 后端归属 | 菜单挂靠 | 页面路径 |
|---|---|---|---|
| 供应商/采购单/采购明细/到货入库 | **`internal/domain/procurement`**（新建） | ams 组 | `/ams/purchase-req`、`/ams/purchase`、`/ams/inventory` |
| 安装施工/施工回单/到场打卡 | `internal/domain/order`（dispatch_tickets 增列）+ 新表 `install_logs` | boss 组 | `/boss/install-board`（含详情/回单/轨迹 3 个 Drawer 下钻） |
| GIS 图层扩展（库存分布/施工实时/完成点位） | `internal/domain/gis` + cmd/gis | intel 组 `/intel/gis` 内图层 Tab | 改造既有页面 |

### 2. 新增表（约 6 张）+ 字段补丁

| 表 | 用途 | 关键字段 |
|---|---|---|
| `procurement.suppliers` | 供应商 | code / name / contact_name / contact_phone / status（ENABLED/DISABLED） |
| `procurement.procurement_orders` | 采购单 | procurement_no（PO- 前缀）/ supplier_id+name 快照 / legal_entity_id+name（§8.1 锚点）/ status=DRAFT/SUBMITTED/PARTIAL/RECEIVED/CANCELLED / total_amount |
| `procurement.procurement_order_items` | 采购明细 | order_id / asset_type / quantity / received_qty / unit_amount |
| `procurement.procurement_receipts` | 到货入库单 | receipt_no / order_id / batch_id→asset_batches / received_by+at / status=DRAFT/CONFIRMED/REJECTED |
| `dispatch_tickets` 增列 | 到场打卡 | arrived_at / arrive_lat / arrive_lng |
| `install_logs`（归属 order 域） | 施工回单 | ticket_id→dispatch_tickets(N:1) / worker_id+name 快照 / photos JSONB / sign_name+sign_image_url+signed_at / note |

**字段命名严守 fields.md §0**（snake_case/PascalCase/lowerCamelCase 三层一致）。
**新枚举全部当日同步登记 terms.md §4**（采购单状态/入库单状态/施工回单状态）。

### 3. 状态机与现有链路

- **采购单**：DRAFT → SUBMITTED → PARTIAL（部分到货）→ RECEIVED（CANCELLED 任意点可取消）
- **入库单**：DRAFT → CONFIRMED（建批次+逐台建 assets IN_STOCK，同事务）/ REJECTED（拒收）
- **库存扣减**：环节 9 `scanBind` MATCH 同事务 `IN_STOCK→DEPLOYED`（凛冬论据：实物归属时点最稳；激活失败/CANCELLED 须有资产回补闭环，由 `asset_returns` 已有通道承接）；环节 10/12 不重复扣减
- **施工回单**：`install_logs` 齐才允许 dispatch_ticket 置 DONE；阶段灰度
- **GIS 联动**：订单环节 12 维持单次 updateMap；新增 `inventory.changed` Kafka 事件驱动库存分布图层（仓库补 lat/lng，循 000143 odn_device 先例）；worker_locations JOIN dispatch_tickets 出施工实时图层（师傅端 30~60s 节流上报）

### 4. 契约登记（三个文件同提交）

| 文档 | 改动 |
|---|---|
| `docs/contract/terms.md` §4 | 增 4 个枚举：供应商 status、采购单 status、入库单 status、施工回单 status |
| `docs/contract/domain-map.md` 表 1+表 2 | 增 1 行（PUR/内部 procurement 包/ams 组/4 页）+ 表 2 反向索引加 ams 4 页 |
| `docs/contract/fields.md` | §4.1 增入库单弱引用；新增 §4.1.1 asset_batches；新增 §9 采购/施工回单字段；§6（intel/gis）L880 底图文字修正（清辉发现的事实不符） |

### 5. 菜单与页面（鹿野定）

- **ams 组**（新增）：
  - `/ams/purchase-req` 采购申请（列表+新建 Drawer）
  - `/ams/purchase` 采购单管理（行内"入库确认"按钮）
  - `/ams/inventory` 库存查询（按物料聚合，跳资产台账）
- **boss 组**（新增）：
  - `/boss/install-board` 施工实时看板（状态泳道+异常列）
  - 3 个 Drawer 下钻：施工详情/施工回单/施工轨迹（**不占菜单位**）
- **intel 组**（改造）：
  - `/intel/gis` 加图层 Tab（设施/局点/设备/**库存分布**/**施工实时**）
- **现有页面行内跳转**：
  - `order.html`：行加"施工详情"按钮，仅 INSTALLING 时显示
  - `dispatch.html`：行加"查看施工"
  - `asset.html`：行加"关联工单"小卡
  - `gis.html`：施工点位气泡→施工详情 Drawer，库存点位→/ams/inventory
  - 采购入库确认完成→跳资产台账新增资产

**菜单注册**（worktree 协议要求）：menu.def.ts / App.tsx / i18n types+locale / fields.md 注册类改动**压成独立小提交**，不埋进大 feature 提交。

### 6. 阶段归属与迁移号

- **阶段**：归"增量"，不开阶段 10（墨白结论 + CMS/OPENPLAT/APPREL 先例）
- **迁移号**：开工前查 `ls migrations | tail` + 逐分支 `git ls-tree <branch> -- migrations/`，主树最新 000162；新表约 6 张，但按"独立 revert"原则拆：
  - **000163** 采购域（suppliers/procurement_orders/procurement_order_items/procurement_receipts + asset_batches 字段补丁）
  - **000164** 施工回单（install_logs + dispatch_tickets 增列）
  - **000165** GIS 图层（仓库 lat/lng 补列，按需）
  - 每个迁移与契约改动同提交；契约门禁 `make check` D 项机械拦截跨分支撞号
- **域边界**：procurement 域暴露 `CreateReceipt` 服务接口，asset 包注入依赖反向调用，保证"建批次→逐台建 assets IN_STOCK→回填 receipt"单事务但不跨域 import 实现（domain-map §4 规则 2）

---

## 待澄清问题（用户拍板）

1. **底图坐标系（WGS84 vs GCJ-02）**：上面 Battle 3 给的 3 条路（A/B/C），本期是否走"路 B 换 OSM"？还是先记入 adopted note 待 PMTiles 自建时再统一切换？
2. **是否一次性落地**：本次交付是 MVP（先做"采购-入库-库存"+"施工回单"，GIS 图层二期）？还是一期完整闭环（同时含 3 类图层）？MVP 更稳，建议二期上 GIS 实时图层。
3. **apprelease 同步**：本议题涉及师傅端"到场打卡"+"施工回单"，与 `apprelease`（迁移 000137）灰度发版策略如何协同（凛冬风险 d）？
4. **采购财务联动**：采购单 `total_amount` 是否走 `billing` 域应付/已付？还是仅作台账不入账？
5. **库存与 GIS 联动粒度**：仓库要否支持树形区域（如"某大区/省/市的某个仓库"）？影响仓库表是否挂 `region_id`。

---

## 行动建议（下一步）

1. **本日落 `docs/notes/adopted/2026-08-28-procurement-install-gis-linkage.md`**，把"独立 procurement 域 + 双向事件 + GIS 双链路保留 + 底图决策推迟 + 6 张新表 + 3 个迁移号"作为不可逆裁定固化。
2. **本日落 `docs/notes/adopted/2026-08-28-tile-coordinate-rationale.md`**，登记底图坐标系事实不符，等待用户拍板决策路 A/B/C。
3. **本日修 `docs/contract/fields.md` L880**（清辉发现的"默认 OSM"事实不符 + 路径过时），随契约修补小提交。
4. **进入开发前**：用户对"待澄清问题"5 项拍板；拍板后开 worktree 分支（feature/procurement + feature/install-logs + feature/gis-layers 三条），按"独立 revert"原则并行推进。

---

## 记录备注

- 辩论主持 subagent（id 5893dbbc-6c5f-4b88-bdd5-5ffaaf5e19e9）作为 depth=2 子代理，主办人无法 send_message 直接读取其结论，**Battle 1 双方最终立场与让步原文未能落盘归档**。本纪要 Battle 1 段为**主持人基于双方初版发言 + 现场证据（pg.go 行数 / domain-map §4 规则 2 / 单文件红线）的当面裁决**，不替代辩论主持的子代理结论；如下次复盘需还原原始辩论过程，需由辩论主持自行落盘 `meeting-minutes/debate-procurement-domain-raw.md`。
- 纪要文件：`meeting-minutes/2026-08-28-procurement-install-gis-linkage.md`
