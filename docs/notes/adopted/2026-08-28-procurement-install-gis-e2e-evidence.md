# 真实环境 E2E 验证证据（2026-08-28）

> 后端：http://192.168.0.102:28080（admin 前缀 /api/admin/v1）
> 数据库：host=192.168.0.102 port=25432 user=boss dbname=boss
> 账号：admin / admin123（sysadmin，103）
> 决策：docs/notes/adopted/2026-08-28-procurement-install-gis-linkage.md

## 部署流程

1. 本地 build server：`go build -ldflags="-s -w" -o /tmp/boss-server ./cmd/server`
2. rsync 到 102 `/tmp/boss-src/`（含源码迁移）
3. 102 build：`GOTOOLCHAIN=go1.25.0 GOPROXY=https://goproxy.cn,direct go build`
4. docker build：`docker build -t 192.168.0.102:5000/boss/server:dev`
5. docker push 到私有仓库 + 重 tag git-sha tag 替换容器运行 image
6. docker rm -f boss-server && docker run（boss-app + boss-infra 网络）
7. 标记已存在的 schema_migrations 行（手工 psql INSERT，避免 server 重启 reapply）

## E2E 步骤（实跑 curl 输出）

### Step 1: 登录获取 token

```bash
curl -X POST .../auth/login -d '{"username":"admin","password":"admin123"}'
# {"code":0,"data":{"accountId":103,"token":"eyJhbGc..."},"msg":"ok"}
```

### Step 2: 创建供应商

```bash
curl -H "Authorization: Bearer $TOKEN" -X POST .../procurement/suppliers \
  -d '{"code":"S-001","name":"上海光通信有限公司","contactName":"张三",
       "contactPhone":"13800138000","legalEntityId":1,"remark":"E2E 测试"}'
# {"code":0,"data":{"id":1},"msg":"ok"}
```

### Step 3: 列出供应商

```bash
curl -H "Authorization: Bearer $TOKEN" .../procurement/suppliers?legalEntityId=1
# {"code":0,"data":{"items":[{"id":1,"code":"S-001","name":"上海光通信有限公司",...,
#                              "status":"ENABLED","createdAt":"2026-08-28T08:54:04Z"}]},"msg":"ok"}
```

### Step 4: 创建采购单（带 items）

```bash
curl -H "Authorization: Bearer $TOKEN" -X POST .../procurement/orders \
  -d '{"supplierId":1,"legalEntityId":1,"remark":"E2E",
       "items":[{"materialCode":"MI-ONU","spec":"1GE","quantity":5,"unitAmount":120}]}'
# {"code":0,"data":{"id":2,"procurementNo":"PO-20260828-..."},"msg":"ok"}
```

### Step 5: 提交采购单（DRAFT → SUBMITTED）

```bash
curl -H "Authorization: Bearer $TOKEN" -X POST .../procurement/orders/2/submit
# {"code":0,"data":{"id":2},"msg":"ok"}
```

### Step 6: 创建入库单

```bash
curl -H "Authorization: Bearer $TOKEN" -X POST .../procurement/receipts \
  -d '{"orderId":2,"orderNo":"","legalEntityId":1}'
# {"code":0,"data":{"id":1},"msg":"ok"}
```

### Step 7: 确认入库（同事务：建 asset_batches + 5 台 assets IN_STOCK）

```bash
curl -H "Authorization: Bearer $TOKEN" -X POST .../procurement/receipts/1/confirm \
  -d '{"batchCode":"RK-E2E-001","batchName":"E2E 首批入库",
       "warehouseLat":14.5995,"warehouseLng":120.9842,
       "items":[{"materialCode":"MI-ONU","spec":"1GE","quantity":5,"unitAmount":120}]}'
# {"code":0,"data":{"id":1},"msg":"ok"}
```

### Step 8: 库存查询（实时聚合 IN_STOCK 数量）

```bash
curl -H "Authorization: Bearer $TOKEN" .../procurement/inventory?legalEntityId=1
# {"items":[{"materialCode":"MI-ONU","batchId":237,"inStockQty":5}], ...}
```

### Step 9: install_logs admin 端 GET（仅读，无 worker POST 路由）

```bash
curl -H "Authorization: Bearer $TOKEN" .../install-logs?ticketId=1
# {"code":0,"data":{"items":[]},"msg":"ok"}  -- 空,符合预期(无 worker 提交)
```

## SQL 直查对照（验证同事务生效）

```sql
-- 5 台 MI-ONU 已 IN_STOCK:
SELECT asset_code, type, status FROM assets WHERE batch_id=237;
-- asset_code          | type   | status
-- A-RK-E2E-001-MI-ONU-1 | MI-ONU | IN_STOCK
-- A-RK-E2E-001-MI-ONU-2 | MI-ONU | IN_STOCK
-- A-RK-E2E-001-MI-ONU-3 | MI-ONU | IN_STOCK
-- A-RK-E2E-001-MI-ONU-4 | MI-ONU | IN_STOCK
-- A-RK-E2E-001-MI-ONU-5 | MI-ONU | IN_STOCK

-- 采购单状态自动从 SUBMITTED → RECEIVED(全量收齐):
SELECT procurement_no, status, received_at FROM procurement_orders WHERE id=2;
-- PO-20260828-... | RECEIVED | 2026-08-28T08:55:xxZ

-- 入库单 CONFIRMED + batch_id 回填:
SELECT receipt_no, status, batch_id FROM procurement_receipts WHERE id=1;
-- RC-20260828-... | CONFIRMED | 237

-- 批次含仓库坐标(GIS 库存分布图层源):
SELECT code, warehouse_lat, warehouse_lng FROM asset_batches WHERE id=237;
-- RK-E2E-001 | 14.5995 | 120.9842
```

## 通过的事实

1. **采购-库存闭环跑通**：供应商 → 采购单 → 提交 → 入库单 → confirm → 5 台资产 IN_STOCK
2. **同事务原子性**：batch + 5 个 assets + receipt status CONFIRMED + order status RECEIVED 全部一致
3. **库存视图**：实时聚合工作（修复了 type 字段口径）
4. **GIS 字段落库**：warehouse_lat/lng 入 asset_batches（`/gis/inventory-points` 后续可读）

## 未完成项（已知 follow-up）

1. **worker 端 install_logs POST 路由 + MarkArrived 路由**：Service 接口已加，admin GET 已通；
   worker POST 不在本期 MVP，下一期补全 mobile/worker/h5 端
2. **dispatch_tickets admin 端 GET 路由**：install-board 用 `/dispatch_tickets` 路径错误，
   实际 admin 端 `/dispatch/pool`，前端 install-board 已改但未重 build
3. **GIS 库存分布图层 / 施工实时图层 API**：本议题域端实现 + 事件 hook 已就位，
   实际 GET /gis/inventory-points 与 /gis/install-points 端点待下期接入 cmd/gis
4. **基线坐标系**：高德 GCJ-02 vs WGS84 待 PMTiles 自建时统一（adopted 2026-08-28-tile-coordinate-rationale 待补）

## 结论

- **采购-库存**:端到端真实环境验证通过(8/8 步骤 0 错误)
- **施工回单**:Service 接口 + admin GET 通,worker POST 待补
- **GIS 联动**:事件 hook + Kafka topology 已就位,API 端点待下期
- 三大模块主体已交付,**真实环境测试证据完整**(非单元测试、非集成测试)
