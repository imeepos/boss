# 数据建模 ↔ 三端页面 对齐审计台账（contract/alignment-audit）

> 版本 V1.0（2026-08-17）｜权威源：`terms.md` / `fields.md` / `domain-map.md` / migrations / `server-ts` 实体
> 定位：把「数据建模（62 表）与 admin/user/worker 三端页面」的对齐状态固化为可销项清单。
> 规则：字段/状态/术语以 `terms.md` 为准；页面列名 ↔ 字段名 ↔ 枚举以 `fields.md` 为准；已销项标 ✅ 并注明提交。

## 1. 结论

主链路可对齐，已消除「双模型字段漂移 / 字段命名不一致 / 状态枚举不一致」三类问题；剩余为按 `domain-map` 明确标注的「待建域」与派生视图，本期不臆造。

## 2. 已修复（销项清单）

### 2.1 枚举与状态（D 类）

| # | 问题 | 处置 | 状态 |
|:-:|:-----|:-----|:----:|
| D1 | `RoleCode` 混入 `dispatcher`（实为岗位码），与 `fields.md`/migration 种子的 7 角色集冲突 | 统一为 `customer/technician/asset_admin/resource_admin/ops/analyst/sysadmin` | ✅ e9a2932 |
| D2 | `tag.status` 在 `terms.md` 有、`enums.ts` 与 `Tag` 实体均缺 | 新增 `TagStatus=UNBOUND/BOUND/DISABLED` 并落到 `Tag.status` | ✅ e9a2932 |

### 2.2 字段回写（A 类·实体缺字段）

| # | 问题 | 处置 | 状态 |
|:-:|:-----|:-----|:----:|
| A1 | `Account` 缺 `phone`（migrations/fields 有） | 补 `phone` | ✅ e9a2932 |
| A2 | `AuditLog` 缺 `detail`（migrations/fields 有） | 补 `detail(jsonb)` | ✅ e9a2932 |
| A3 | `Tag` 缺 `status` | 补 `status` | ✅ e9a2932 |
| A4 | `Address` 缺 `parent_id`/`geom`（migrations 有） | 补 `parentId`（派生）、`geom(geography)` | ✅ 本台账 |

### 2.3 字段命名统一（B 类）

| # | 问题 | 处置 | 状态 |
|:-:|:-----|:-----|:----:|
| B1 | assets 资产编码 `asset_no` vs 实体 `asset_code` | 统一 `asset_code`（fields.md 改） | ✅ e9a2932 |
| B2 | ports 端口编号 `port_no` vs 实体 `port_code` | 统一 `port_code` | ✅ e9a2932 |
| B3 | orders 产品 `product_id→products` vs 实体 `offer_id→product_offers` | 统一 `offer_id→product_offers` | ✅ e9a2932 |
| B4 | quad_link 地址 `addr_id` vs 实体 `address_id` | 统一 `address_id` | ✅ e9a2932 |
| B5 | orders `channel_id/brand_id` 指向不存在的 `channels/brands` | 从 orders 字段表删除；渠道/品牌改口径（见 §3） | ✅ e9a2932 |

### 2.4 缺失实体补齐（A 类·页面有列实体缺失）

| # | 页面 | 缺实体 | 处置 | 状态 |
|:-:|:-----|:-------|:-----|:----:|
| E1 | settings.html 业务参数 | `biz_params` | 补 `BizParam` | ✅ e9a2932 |
| E2 | provlog.html 下发日志 | `provision_logs` | 补 `ProvisionLog` | ✅ f4c4fb7 |
| E3 | customer.html 实名核验 | `real_name_verifications` | 补 `RealNameVerification` | ✅ f4c4fb7 |

### 2.5 Go 骨架字段漂移（A 类·双模型）

| # | 问题 | 处置 | 状态 |
|:-:|:-----|:-----|:----:|
| G1 | `order.Order.ProductID→products`（表不存在） | `OfferID→product_offers` | ✅ 0048f56 |
| G2 | `order.Order.BrandID` / `customer.Customer.BrandID` → brands（表不存在） | `LegalEntityID`（品牌=运营主体） | ✅ 0048f56 |
| G3 | `order.Order.ChannelID` 渠道必填但无 channels 表 | 保留字段，标注 CH 域待建 | ✅ 0048f56 |

## 3. 待建域 / 派生视图（本期不臆造，依据 domain-map）

| 项 | 页面 | 归属域 | 说明 |
|:---|:-----|:-------|:-----|
| ~~渠道 `channels`~~ | order/dispatch 下单渠道 | CH 渠道经销商 | ✅ 已落地：`Channel` 实体 + `Order.channel` FK + fields.md + Go 注释 + mock channels 表 |
| ~~告警 `alarms`~~ | alarm.html | MON 网络监控告警 | ✅ 已落地：`Alarm` 实体 + AlarmLevel/AlarmStatus 枚举 + terms.md |
| 设备监控指标（光功率/丢包率） | device.html | MON | 设备身份复用 `resources`，监控指标待建 |
| ~~话单 CDR~~ | aaalog.html | AAA 认证计费 | ✅ 已落地：`cdrs`(CallDetailRecord) + `auth_logs`(AuthLog) 实体 |
| 渠道对账 | paycheck.html | BIL/PAY | 派生聚合（对 payments 的汇总），非基表 |

## 4. 遗留轻微不一致

| 项 | 说明 | 建议 |
|:---|:-----|:-----|
| migrations `addresses.parent_id` 物化列 vs TS 实体 `parentId` 派生列 | 均已补 `parentId`，语义一致（派生，应用层不手填） | 保持 |
| migrations `addresses.geom` vs TS `geom` | 已补 `geom(geography)`，阶段8 GIS 预留 | 保持 |

## 5. 跨层命名映射（实体关系模型 ↔ mock 视图 ↔ 页面列）

> `api/mock/db.js` 是「关系型事实库」，三端视图由它**扁平化派生**（外键冗余成快照列），
> 故字段名与 TypeORM 实体（62 表关系模型）存在**视图导向命名差**，属设计使然、非漂移。
> 语义对齐（状态枚举/单号口径/引用完整/未收费不派单/四码与 GIS 时点）由 `node api/mock/selfcheck.js` 固化，全绿即对齐成立。
> 本表只固化「概念 ↔ 实体字段 ↔ mock 字段 ↔ 页面列」的映射，避免把命名差误判为漂移。

| 概念 | DB/实体 | mock db.js | 页面列 | 说明 |
|:-----|:--------|:-----------|:-------|:-----|
| 资产台账编码 | `assets.asset_code` | `assetNo` | asset.html「资产编码」 | 同名异名，值一致 `A-2025xxxx` |
| 四码资产码(EPC) | `tags.epc_code` | `epc`(四码视图用 `assetCode`) | tag/quad「资产码」 | 实体在 tags，mock 冗余到 assets；**注意 `assetCode` 双义：实体=台账编码、四码=EPC** |
| 端口编码 | `ports.port_code` | `portNo`(物理 `P-001-01`) | resource.html「端口编号」 | mock `portNo` 是物理端口号，≠ 实体 `port_code` |
| 四码端口码 | `ports.quad_code` | `quadCode` | 四码「端口码」 | 一致 `P-SPLxx-yy` |
| 产品 | `product_offers.offer_id` | `productId`/`products` | product.html「产品资费」 | API 资源名 `products`，实体表名 `product_offers` |
| 认证账号 | `lo_accounts` | `loids`/`loid` | loaccount.html | API 用业务名 LOID |
| 报障工单 | `complaints` | `repairTickets` | complaint.html | API 用业务名 repairTicket（TKT-\*）；投诉 CP-\* 同挂该域 |
| 账单/缴费 | `bills`/`payments` | `bills`/`payments` | billing/payment | 一致 |

> 裁定：`assetCode` 一词在实体层=台账编码、在四码/EPC 上下文=电子标签码，二者是不同对象。
> 四码对账用 EPC（`tags.epc_code`），资产台账用 `assets.asset_code`，桥接靠 `tags`（见 cross-end-linkage §四）。

## 6. 销项规则

1. 新缺口按「类别字母 + 序号」登记（A=实体缺字段/缺表、B=命名、D=枚举、G=Go 漂移、E=缺实体）。
2. 字段/状态/术语冲突一律先查 `terms.md`，再回写 `enums.ts` + 实体 + `fields.md`。
3. 页面新增列时，同步在 `fields.md` 登记英文字段名与枚举。

## 7. OpenAPI 接口规范 ↔ DB 设计 对齐检查

> 检查 `api/openapi/`（admin/user/worker 三端，38 文件）与 62 表实体 + `terms.md`/`enums.ts` 的对齐。

### 7.1 已对齐 ✅

- 核心状态枚举：订单 `PENDING/RESERVED/INSTALLING/DONE`（admin）、四码 `LINKED/CONFLICT/UNLINKED`、标签 `UNBOUND/BOUND/DISABLED`、实名 `VERIFIED/PENDING`、账单 `UNPAID/PAID/OVERDUE`、环节 `DONE/DOING/PENDING`（user 端）。
- 字段名：`assetCode`/`portCode`/`quadCode`（admin asset/oss/quad）、实名核验 `result: PASS/FAIL`。

### 7.2 已收敛 ✅（本轮一并收敛）

- 环节结果 `WAIT → PENDING`（admin order.yaml + mock 视图 + order.html）。
- 订单状态补 `CANCELLED`（terms.md §3 + enums.ts OrderStatus）。
- 报障补 `PROCESSING`、扫码补 `OFFLINE_CACHED`（enums.ts 新增 `ComplaintStatus`/`ScanResult` + 实体回写）。
- 激活回调 `ActivationResult = SUCCESS/FAILED`（`RETRYING` 重试中为展示态）；激活(环节10) `PENDING/SUCCESS/FAILED` 为师傅端视图，二者已区分。
- 消息级别统一 `INFO/WARN/URGENT`（openapi 2 处 + mock worker/entities + messages.html + style.css）。
- 缴费方式统一 `wechat/alipay/card/cash`（openapi user/billing + `PaymentMethod` 枚举 + 实体）。
- 环节时间轴字段名 `stageNo/stageName/retryCount → stage/name/retries`（openapi + mock 视图 + order.html + callback.html）。

### 7.3 保留（派生态，非冲突）

| # | 位置 | openapi | 实体 | 说明 |
|:-:|------|---------|------|------|
| 7 | worker/schemas.yaml 工单 `status` | `TODO/ACCEPTED/SCAN_PENDING/DOING/DONE` | TicketStatus `PENDING/DOING/DONE/CANCELED` | 师傅端工单派生态（由订单 stage 派生），保留 |

### 7.4 字段名/资源名分歧（API 业务名 vs DB 技术名）→ 已固化映射 ✅

| 概念 | openapi | 实体 | 裁决 |
|------|---------|------|------|
| 产品 | `productId`/`products` | `offerId`/`product_offers` | 保留 API 名，映射固化于 fields.md §0.1 |
| 认证账号 | `loid`/`lo-accounts` | `lo_accounts` | `loid` 即实体字段，保留 |
| 报障工单 | `repairTickets` | `complaints` | `repairTicket` 更贴切；`complaints` 为报障+投诉同表 |
| 实名核验 | `verify-logs` | `real_name_verifications` | 保留 API 名 |
| 调拨类型 | `ASSET/PORT/DEVICE` | 仅 `resource_id` | API 概念更宽，DB 待扩展 |

> **裁定（最佳实践）**：API 字段/资源名是稳定契约（业务友好名），DB 表/列是内部技术名，分属两层，
> **禁止为求同名而改 API 或 DB**（耦合即反模式）。唯一权威映射已固化于 `fields.md §0.1`，
> 页面/Agent 一律以该表消歧义，不再做破坏性重命名。
