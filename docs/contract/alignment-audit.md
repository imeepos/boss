# 数据建模 ↔ 三端页面 对齐审计台账（contract/alignment-audit）

> 版本 V1.0（2026-08-17）｜权威源：`terms.md` / `fields.md` / `domain-map.md` / migrations / `server-ts` 实体
> 定位：把「数据建模（68 表）与 admin/user/worker 三端页面」的对齐状态固化为可销项清单。
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
| ~~设备监控指标（光功率/丢包率）~~ | device.html | MON | ✅ 已落地：`device_metrics` + `device_maintenances` 实体 |
| ~~话单 CDR~~ | aaalog.html | AAA 认证计费 | ✅ 已落地：`cdrs`(CallDetailRecord) + `auth_logs`(AuthLog) 实体 |
| 渠道对账 | paycheck.html | BIL/PAY | 派生聚合（对 payments 的汇总），非基表 |

## 4. 遗留轻微不一致

| 项 | 说明 | 建议 |
|:---|:-----|:-----|
| migrations `addresses.parent_id` 物化列 vs TS 实体 `parentId` 派生列 | 均已补 `parentId`，语义一致（派生，应用层不手填） | 保持 |
| migrations `addresses.geom` vs TS `geom` | 已补 `geom(geography)`，阶段8 GIS 预留 | 保持 |

## 5. 跨层命名映射（实体关系模型 ↔ mock 视图 ↔ 页面列）

> 历史阶段 `api/mock/db.js` 曾承担「关系型事实库」(已于 2026-08-19 随 mock 层移除),
> 故字段名与 TypeORM 实体（68 表关系模型）存在**视图导向命名差**，属设计使然、非漂移。
> 语义对齐（状态枚举/单号口径/引用完整/未收费不派单/四码与 GIS 时点）曾由 `node api/mock/selfcheck.js` 固化。
> Amended 2026-08-19: mock 层已整体移除(正式环境全真实业务支撑),对齐口径以 Go 单测/e2e 为准。
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

> 检查 `api/openapi/`（admin/user/worker 三端，37 文件）与 68 表实体 + `terms.md`/`enums.ts` 的对齐。

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

## 8. 最终裁定（by-design 差异，非缺陷）

> 彻底对齐后，以下差异属**设计使然**，已在权威文档固化口径，后续 Agent 不得再当缺口回改。

| 差异 | 裁定 |
|:-----|:-----|
| Go `internal/domain/*` 骨架 vs TS 实体（如 `Region.Parent string`、`Department.LegalEntity string`） | Go 是「阶段骨架参考实现，落地替换为 DB」，字段用展示/派生形态，与 TS 关系模型粒度不同，保留 |
| Go `aaa.Profile.Status = ACTIVE/SUSPENDED` vs `LoAccountStatus = ACTIVE/SUSPENDED/CLOSED` | AAA 认证档案只关心「在服/停服」认证维度；`CLOSED 注销`是业务维度，二者正交，保留 |
| `complaints` 实体 = 报障+投诉同表（客服工单域） | 报障(TKT-\*)与投诉(CP-\*)经 `type` 区分；API 侧 `repairTickets` 单列报障，映射见 §0.1 |
| `accounts`(系统账号) / `customers`(客户) / `lo_accounts`(认证账号) 三义 | 三个不同对象，`fields.md §5.1` 已裁定四码第 2 项=客户，非系统账号 |
| `paycheck`(渠道对账)、`analytics`/`report`(BI)、`gis` | 派生聚合/视图，无基表，不建实体 |

> 对齐状态：**数据建模(68 表) ↔ 三端页面 ↔ OpenAPI ↔ mock ↔ Go 骨架** 全部对齐；本台账为唯一销项记录。

## 9. REST 信封三方对齐（接口 ↔ 契约 ↔ 客户端，2026-08-19）

> 背景：真实服务端(Go `httpx.Respond`)所有响应为 **HTTP 恒 200 + 统一信封 `{code,msg,data}`**
> （错误码对齐 `pkg/apitypes`，D3 决策）；而 worker 端 OpenAPI 曾把业务 schema 写成响应根、
> H5 `api.js` 透传 `r.json()`、师傅端 Android 直接把信封根当业务对象消费——三方只有路由对齐，载荷形态三方不一致。

### 9.1 裁定（信封为运行时权威，客户端统一解信封）

| 方 | 事实/处置 | 状态 |
|:---|:---|:----:|
| 接口（Go server） | HTTP 恒 200,`{code,msg,data}`,code!=0 即业务错误；实测 `curl /api/worker/v1/home` → `{"code":401,"msg":"missing bearer token"}` | 事实 |
| 契约（OpenAPI） | `worker/schemas.yaml` `responses.Ok` 改为完整信封定义；各业务 schema 语义 = **data 内载荷形态** | ✅ 本节 |
| H5（docs/worker/api.js） | `request()` 增 `unwrap`:code!=0 抛错,成功返回 `data`;页面继续消费平铺字段 | ✅ 本节 |
| 师傅端 Android | `Api.kt` 增解信封(与用户端 Android `Api.unwrap` 同构);LoginScreen 去掉手动 `data.token` 补丁 | ✅ 815b97e |
| 用户端 Android | 既有 `Api.unwrap` 已解信封,无需改动（先例） | ✅ 既有 |

### 9.2 路由三方清点（对账当日事实）

- 服务端 57 路由 = OpenAPI worker.yaml 56 + `POST /worker-registrations`（师傅自助注册,此前漏契约）→ 本节补入 `worker/auth.yaml` + 聚合 ✅
- `docs/worker/api.js` 56 方法与 OpenAPI 56 路径一一对应；师傅端 Android `WorkerApi.kt` 与 api.js 一一对应。
- mock(`api/mock/combined.js`,8091)已移除,docs 中残留引用已更新为真实服务端口 28080。

### 9.3 后续 Agent 注意

- 新增 worker/user 端点时,响应必须走 `respond()`/`httpx.Respond` 信封,OpenAPI 引用 `responses.Ok` 或内联 data 形态,**禁止**裸 return 业务对象。
- 客户端(三端)一律在对接层解信封,页面/Composable 只见平铺业务对象。

## 10. ODN 地理空间编码规范对齐（2026-08-20）

> 依据《Suniway ODN 地理空间编码规范》V1.0（docs/pdfs/，2026-08-20 生效）全量对账。
> 结论：规范要求的**无源物理网络层编码体系**（PRV/NodeCode 映射、网格分区、电杆/人井/铁塔/接头盒/终端盒、光缆段落/纤芯）在系统中尚未落地；现有 PSGC 数据（000041，PSA 口径）可作映射锚点。裁定见 `notes/adopted/2026-08-20-odn-geospatial-encoding-alignment.md`。

| # | 规范条款 | 差距 | 处置 | 状态 |
|:-:|:-----|:-----|:-----|:----:|
| E4 | 第2章 PRV 省级编码（`PHL001`~`PHL083`） | 系统用 PSGC 10 位码，无 PRV 格式 | migrations/000075 `odn_region_code` 82 行全量映射（PHL075 预留不落），FK 锚 geo_subdivision，集成测试 `TestODNRegionCityCodes_Integration` 守护 | ✅ 000075 |
| E5 | 第3章 NodeCode 局点编码（`MNL001`） | 全库无 NodeCode 概念 | 000075 `odn_city_code` 119 个城市前缀全量登记（复合主键 省内唯一，规范 2.3）；局点 3 位序号实体随 E6 odn 域落 | ✅ 前缀登记（序号随 E6） |
| E6 | 第4章 网格分区 + 基础设施编码（P/MH/TW/CLS/TBX） | 无电杆/人井/铁塔/接头盒/终端盒实体 | 新域 `internal/domain/odn`，需求驱动再建 | 待建 |
| E7 | 4.7 网格容量预警（800/999/90 三级） | 无 | 随 E6 网格实体落 | 待建 |
| E8 | 第5章 光缆段落/纤芯编码 + A 端方向优先级 | 无光缆段落/纤芯实体 | 随 E6 落 | 待建 |
| E9 | 第7章 红线3（5 位数字系统校验） | 无编码校验逻辑 | 随 E6 落（入库校验 5 位数字） | 待建 |
| E10 | 前缀冲突：`ports.port_code='P-SPLxx-yy'` vs 电杆 `P01001` | 字母前缀 `P` 双义 | 裁定：ODN 规范码落 odn_* 表独立命名空间（`prv_code`/`city_prefix`/未来基础设施码列），`ports.port_code` 保留为 BOSS 内部资源码，二者不混存不互斥，冲突消解 | ✅ 裁定 |
| E11 | 姊妹规范设备码 `OLT001`（无连字符）vs `Resource.Code` `OLT-01` | 核心链路设备编码格式分歧 | 同 E10：ODN 链路编码是 odn 域编码体系，`resource.code` 是 BOSS 内部资源码，经映射关联、不强制改名；odn 域开工时落 `odn_code` 专列 | ✅ 裁定（映射随 E6） |

> 姊妹文档《Suniway ODN 基础设施资源编码规范》V1.0 已对账：核心链路拓扑（`SNW_PRV_NodeCode_ODF?_OCC?_ODB_SDB_PRT_TBP?`）、连接符规则（系统一律 `_`、扩容后缀 `-N`（禁 `-1`）、`--` 仅图纸）、标签规范均归 odn 域编码体系，系统侧校验正则随 E6 落地。
> 规范文档自身缺陷（83省 vs PSA 82省混排 HUC、城市前缀 3字母 vs 索引表 4-5 字母、塔布克/阿拉贝尔/纳本图兰归属错误、Maguindanao 已拆分未更新）已登记 `ISSUE.md`，映射一律按 PSA PSGC 事实裁定并在 note 列留痕。
