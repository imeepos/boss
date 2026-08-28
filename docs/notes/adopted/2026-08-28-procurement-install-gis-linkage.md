# 采购-库存-施工回单-GIS 联动：方案固化（2026-08-28）

> 状态：已采纳｜主持：会议主持人｜范围：MVP 单一完整闭环
> 改动规模：6 张新表 + 2 列补丁 + 4 枚举 + 3 页前端 + 2 迁移号
> 关注：feature/procurement-install-gis 分支

## Why（动机）

议题：在 9 阶段已交付到阶段 8（GIS）的 BOSS 装维全流程系统上，加入"采购-库存管理-安装施工-GIS 联动"。
现状痛点：

1. 资产域有 `asset_batches`（公司采购行为批次）但无独立"采购单/供应商/采购明细/到货入库"实体。
2. 安装施工（订单 8-10 派单/扫码/激活）已闭环，但缺施工回单（照片/签收/材料消耗）证据；缺"到场打卡"事实。
3. GIS 联动是订单环节 12 的单次 updateMap + cmd/gis Kafka 消费者（topic `boss-order-events`，`stage==12` 过滤）双链路，但**缺少库存维度的 GIS 联动**与**施工维度的图层**。

## 决策（不可逆）

### 1. 采购-库存独立建域 `internal/domain/procurement`

**why**：domain-map 21 域虽无"采购"域，但 AMS/asset 管单件生命周期（IN_STOCK/DEPLOYED…），采购管供应商/批量/钱，职责异构。`internal/domain/asset/pg.go`（228 行）+ 4 个 pg_*.go（pg_binding/pg_replacement/pg_stocktake/pg_write 累计 1306 行）= 1534 行，**域整体已重**；继续扩展必撞单文件 300 行红线与跨域 import 禁令（domain-map §4 规则 2）。

**怎么落地**：

- 新建 `internal/domain/procurement/`，含 `doc.go` / `procurement.go`（领域类型与接口）/ `pg.go`（PG 读写）/ `pg_test.go`（集成测试）/ `httpapi/`（HTTP 路由注册，admin 端）。
- 4 张表：`procurement.suppliers` / `procurement.procurement_orders` / `procurement.procurement_order_items` / `procurement.procurement_receipts`。
- `procurement_receipts` 入库确认 = 同事务（建 `asset_batches` 批次→逐台建 `assets` IN_STOCK→回填 receipt），由 procurement 域暴露 `CreateReceipt(ctx, …)` 服务接口，`internal/domain/asset` 注入依赖反向调用，**不直连 asset 实现**（保 domain-map §4 规则 2）。
- 菜单挂靠：前端 admin `ams` 组（按业务动线），后端归 `procurement` 域（按职责）——**包归属 ≠ 菜单分组**，两者各自独立。

**放弃的方案**：把采购塞进 `internal/domain/asset`（云舟初版建议）——会撞 300 行红线、跨域 import、菜单管理动线割裂。

### 2. 订单 12 环节零改动；施工回单独立建表 `install_logs`

**why**：terms.md §1 明令 12 环节"禁止增删改序"。"到场/在途/施工进度"只能在工单层派生（参考 RETRYING 派生先例），不得插入环节表。

**怎么落地**：

- `install_logs` 新表：1:1（实际 N:1 可扩）`dispatch_tickets`；含 photos JSONB / sign_name / sign_image_url / signed_at / note 表"现场照片 + 用户签收"。
- `dispatch_tickets` 增列：`arrived_at` / `arrive_lat` / `arrive_lng`（到场打卡事实，不写回订单状态）。
- 回单齐才允许 `dispatch_ticket` 置 DONE——但本期**灰度**：旧版师傅端无 install_logs 上报路径时，`install_logs` 缺席不阻塞 DONE（按"缺失=尚未强制"软策略，运维提示补录）；强制策略入 v2。
- 库存扣减时机：维持环节 9 `scanBind` MATCH 同事务 IN_STOCK→DEPLOYED（凛冬论据：实物归属时点最稳）。激活失败 / CANCELLED 由既有 `asset_returns` 通道承接回补。

**放弃的方案**：

- 在环节 8-10 中插入"到场"环节（违反 terms.md §1）。
- 库存扣减在环节 10 或环节 12（10 是账号动作与实物无关，12 过晚造成窗口期账实不符）。
- `worker_locations` 经 Kafka 上报（写放大，000144 既有 POST 直写足够）。

### 3. GIS 联动：保留双链路 + 新增库存分布图层 + 施工实时图层

**why**：`internal/domain/order/pg_workflow.go:232 UpdateMap` 是订单侧同步直调函数，`cmd/gis/main.go:36 Handle` 是 Kafka 消费者（topic=`boss-order-events`，`stage==12` 过滤触发），**两者并存**——主持人实测核实，非"任一为唯一"。订单侧 updateMap 走业务流程，cmd/gis 走异步入库保障。

**怎么落地**：

- 保留双链路。新增 `inventory.changed` 事件，**同 topic `boss-order-events`**（按清辉建议，避免繁衍 topic），`type` 区分；cmd/gis 消费者扩 `type` 集合。
- `install.progress` 不新立——签收/激活等节点 stage 事件已在流上，消费器扩 stage 集合即可。
- `worker.location` **不进 Kafka**——高频写放大；既有 `worker_locations`（迁移 000144）+ `POST /api/worker/v1/location/report` 直写路径足够。
- 图层① `GET /gis/inventory-points?entity=warehouse&bbox`：仓库表补可空 lat/lng（循 000143 odn_device 先例），聚合粒度=仓库点（批次不上图），不并入八级 drill。
- 图层② `GET /gis/install-points?bbox&activeOnly`：`worker_locations JOIN dispatch_tickets` 取在工师傅最新点。
- 图层③ 沿用 `/gis/points`，零新增（订单完成点位）。
- 实时性：本期无 Redis/WS/SSE 基建，端上 30~60s 节流上报 + 前端 bbox 轮询；二期量大再议 Kafka→SSE。

**放弃的方案**：

- 引 Redis/WS/SSE 基建（与本期主线无关，超范围）。
- 引 Cesium（前端是 OpenLayers，技术决策另立）。
- `worker.location` 进 Kafka（写放大 + 与 `worker_locations` 重复轮子）。

### 4. 底图坐标系：先固化事实，本期暂不切

**why**：清辉实测发现 `web/admin/src/components/business/maps/tile-source.ts L12` `LIGHT_TILE_URL='https://webrd01.is.autonavi.com/appmaptile?...'`（高德 GCJ-02），而 `docs/contract/fields.md L880` 记"默认 OSM"——**事实不符成立**，且路径引用同步过时（实际是 `web/admin/src/pages/intel/gis/`，非 `admin/gis.html`）。

**怎么落地**：本期修契约（L880 改成"默认高德栅格（GCJ-02）"）+ 修路径引用；坐标系转换（A/B/C 三路）待 PMTiles 自建时机切。本期新图层点位以 WGS84 写入，靠 worker_locations 既有口径，新增图层渲染时**仍存在与底图 GCJ-02 的偏移**——已知风险，等 PMTiles 切 OSM/WGS84 时一并解决。

**放弃的方案**：本期硬切 OSM 瓦片（涉及第三方瓦片源稳定性 + 主题一致性调研，超范围）。

## 迁移号

- **000163**：procurement 域（suppliers / procurement_orders / procurement_order_items / procurement_receipts），含 asset_batches 字段补丁。
- **000164**：install_logs + dispatch_tickets 增列（arrived_at / arrive_lat / arrive_lng）。
- 开工前已核：main 最新 000162，worktree `fix/asset-tag-cleanup-and-binding` 占用 000157（已进 main 的 000158 同号，存量），`fix/unauthorized-40100-cleanup` 占用 000160-000162（已进 main）。本分支用 000163/000164 与其他未合并分支无撞号。
- 跨未合并分支让号规则（AGENTS.md）：本分支无需让号。

## 契约改动（三个文件同提交）

| 文件 | 改动 |
|---|---|
| `docs/contract/terms.md` §4 | 增 4 个枚举：供应商 status（ENABLED/DISABLED）、采购单 status（DRAFT/SUBMITTED/PARTIAL/RECEIVED/CANCELLED）、入库单 status（DRAFT/CONFIRMED/REJECTED）、施工回单 status（OPEN/COMPLETED/REJECTED） |
| `docs/contract/domain-map.md` 表 1+表 2 | 增 1 行 PUR（internal 包 procurement/阶段 增量/分组 ams/页面 purchase、purchase-req、inventory） |
| `docs/contract/fields.md` §4.1 补丁 + §9 新增 | 入库单弱引用 + asset_batches 补 §4.1.1 + 新增采购域 4 表字段表 + L880 底图事实修正 |

## 验收口径

- **代码门禁**：`make check`（含 contract-sync D 项迁移号唯一性）。
- **后端门禁**：`go test ./...` + `go build ./...`。
- **前端门禁**：`pnpm typecheck && pnpm test && pnpm build`。
- **真实环境 E2E**（必须，非单元测试）：
  1. admin 登录 102 → 创建供应商（`POST /api/admin/v1/procurement/suppliers`）→ 列表查得到
  2. 创建采购单 + 入库确认（`POST /api/admin/v1/procurement/receipts/{id}/confirm`）→ 库存数量 +N，资产状态 IN_STOCK
  3. 报装工单扫码绑定（既有链路）→ 资产 IN_STOCK→DEPLOYED
  4. 师傅端到场打卡（`POST /api/worker/v1/dispatch-tickets/{id}/arrive`，扩口）→ dispatch_tickets.arrived_at 非空
  5. 师傅端施工回单（`POST /api/worker/v1/install-logs`）→ install_logs 有 photos/sign 记录
  6. GIS /gis/inventory-points 与 /gis/install-points 在 admin /intel/gis 渲染出点位
  7. E2E 完成后 `mainchain-acceptance.sh` 收尾自动 acc_ 清理 + 孤儿巡检门禁通过

## 关联决策

- `2026-08-22-worktree-merge-protocol.md` —— ff-only 合并四步收尾
- `2026-08-29-audit-closeout-rulings.md` §7 —— 真实环境测试不只单元测试；磁盘余量 + 行数 + 并行会话基线

## Amended / Reject 历史

（暂无）
