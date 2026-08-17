# 字段字典（contract/fields）

> 版本 V1.0（2026-08-17）｜权威源：migrations（阶段1）、全案 4.2 业务对象、admin 页面列名、terms.md 状态枚举
> 定位：锁死「页面中文列名 ↔ 英文字段名 ↔ 状态枚举」三者，消除 AI 并行实现时各猜各的字段名。
> 与他处字段名冲突时，以本文件为准。

## 0. 命名规则（全局强制，所有 Agent 遵守）

| 层 | 规则 | 示例 |
|:---|:-----|:-----|
| DB 列名 | lower_snake_case | `legal_entity_id` |
| Go struct 字段 | PascalCase（外键引用 + ID 后缀） | `LegalEntityID` |
| JSON/API 字段 | lowerCamelCase | `legalEntityId` |
| 状态枚举 | 大写 SNAKE_CASE（字符串常量） | `INSTALLING` |
| 外键冗余展示列 | DB 存 `xxx_id`，另存 `xxx_name` 冗余 | `dept_id` + `dept_name` |
| 时间列 | `created_at` / `updated_at`，TIMESTAMPTZ | — |

> 现状依据：migrations 已用 snake_case（`real_name`/`legal_entity_id`），org.go struct 已用 PascalCase + ID 后缀（`LegalEntityID`），保持一致。

### 0.1 跨层字段命名映射（API ↔ DB，权威口径）

> API 字段/资源名是**稳定契约**，允许与 DB 表/列名不同：API 用业务友好名，DB 用技术名。
> 下表是唯一权威映射，改任何一侧命名前必查；未列入者按 §0 规则（JSON/API 字段 = 实体字段 lowerCamelCase）。

| 概念 | 页面列 | API 字段/资源 | 实体字段 | DB 表/列 |
|:-----|:-------|:--------------|:---------|:---------|
| 产品资费 | 产品资费 | `productId` / `products` | `offer` | `product_offers.offer_id` |
| 认证账号 | LOID | `loid` / `loids`（路径 `/lo-accounts`） | `loid` | `lo_accounts.loid` |
| 报障工单 | 报障 | `repairTickets` | （无独立） | `complaints`（报障+投诉，`type` 区分） |
| 实名核验 | 实名核验 | `verify-logs` | （无独立） | `real_name_verifications` |
| 调拨类型 | 类型 | `type`（`ASSET/PORT/DEVICE`） | （无独立） | `transfers.resource_id`（资产/端口维度待扩展） |

> 规则：禁止为求「同名」而改 API 契约或 DB 表名（两者分属不同层，耦合即反模式）；
> 新增跨层差异概念时在此登记，页面/Agent 一律以本表为准消歧义。

## 1. 阶段1 · 系统管理与组织（internal/domain/user，已定型）

### 1.1 accounts（账号）

| 页面列名 | 字段名 | DB 列 | 类型/枚举 |
|:---------|:-------|:------|:----------|
| — | `Username` | username | VARCHAR(64) UNIQUE |
| — | `PasswordHash` | password_hash | TEXT |
| — | `RealName` | real_name | VARCHAR(64) |
| — | `Phone` | phone | VARCHAR(32) |
| — | `RoleID` | role_id | BIGINT → roles |
| 状态 | `Status` | status | 1启用 / 0停用 |
| — | `LegalEntityID` | legal_entity_id | BIGINT → legal_entities（可空） |
| — | `DeptID` | dept_id | BIGINT → departments（可空） |
| — | `PostID` | post_id | BIGINT → posts（可空） |
| — | `RegionScope` | region_scope | LTREE，空=全集团 |

### 1.2 roles / permissions（角色·权限）

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| — | `Code` | code | 7 角色码：customer/technician/asset_admin/resource_admin/ops/analyst/sysadmin |
| — | `Name` | name | 中文名 |
| 权限码 | `Code`(perm) | code | 如 `asset:create`、`menu:order` |

### 1.3 regions（经营区域）

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| — | `Path` | path | LTREE 物化路径，唯一权威 |
| — | `Level` | level | 1集团 2大区 3省 4城市（=nlevel(path)） |
| — | `Name` | name | 名称 |

### 1.4 legal_entities / departments / posts（子公司/部门/岗位）

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 子公司 | `Code` | code | LEG-A/LEG-B/LEG-C |
| 子公司 | `Name` | name | — |
| 部门 | `LegalEntityID` | legal_entity_id | BIGINT → legal_entities |
| 部门 | `Name` | name | 子公司内唯一 |
| 岗位 | `Code` | code | dispatcher/cashier/agent/field_tech…（部门内唯一） |
| 岗位 | `DeptID` | dept_id | BIGINT → departments |
| 岗位 | `Roles` | （聚合） | post_roles 关联 |

### 1.5 addresses（地址层级）

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| — | `Path` | path | LTREE，唯一权威 |
| — | `Level` | level | 1市 2区 3街道 4小区 5楼栋 |
| — | `Name` | name | — |
| — | `ParentID` | parent_id | 派生（=反查 path 父节点） |

> 区域硬关联（TS 实体）：楼栋级地址挂 `region_id`（→ regions，经营区域）+ `region_name` 快照，固化「地址→经营区域」映射；客户/资产/端口/LO账号经此继承区域，杜绝「有地址无订单则不知属哪个区域」的孤儿。

### 1.6 audit_logs（审计日志）· biz_params（业务参数）

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 操作人 | `AccountID` | account_id | BIGINT |
| 操作 | `Action` | action | 数据变更/状态变更/权限变更 |
| 对象 | `TargetType` / `TargetID` | target_type/target_id | — |
| 详情 | `Detail` | detail | JSONB |
| IP | `IP` | ip | INET |
| 参数 | `Key` / `Value` | key/value | value=JSONB |

> 快照列（TS 实体）：`account_name`/`dept_name`/`legal_entity_name`，操作时冻结，调岗/调部门/改名不改历史日志；`account_id` 为弱引用（日志只读不 FK，账号删除不影响日志）。

## 2. 阶段2 · 客户与资费（internal/domain/customer）

### 2.1 customers（客户档案，源自 customer.html）

| 页面列名 | 字段名 | DB 列（约定） | 枚举/说明 |
|:---------|:-------|:--------------|:----------|
| 客户 | `Name` | name | 个人姓名/企业名 |
| 联系电话 | `Phone` | phone | — |
| 证件类型 | `IdType` | id_type | 身份证/护照/营业执照/无 |
| 证件号码 | `IdNo` | id_no | — |
| 实名状态 | `RealNameStatus` | real_name_status | VERIFIED 已实名 / PENDING 待补登 |
| 服务状态 | `ServiceStatus` | service_status | ACTIVE 在网 / ARREARS 欠费 / SUSPENDED 停机 |
| 地址 | `AddressID` | address_id | BIGINT → addresses（挂接楼栋） |

> 区域锚点（TS 实体）：`region_id`/`region_name`（地址所在经营区域），`legal_entity_id`（归属公司），按地区/企业统计客户；客户搬家/转品牌经 `customer_histories` 台账快照事发区域。

### 2.2 资费三级模型（源自 product.html；V1.1 修正：不同公司/区域产品与价格不同）

| 实体 | 页面列名 | 字段名 | DB 列（约定） | 枚举/说明 |
|:-----|:---------|:-------|:--------------|:----------|
| product_offers(公司产品) | 所属公司 | `LegalEntityID` | legal_entity_id | BIGINT → legal_entities |
| | 产品名称 | `Name` | name | **公司级名称,必填**(各公司叫法不同) |
| | 带宽 | `Bandwidth` | bandwidth | 如 300M/500M/1000M(公司自定) |
| | 基础月费 | `MonthlyFee` | monthly_fee | NUMERIC |
| | 生效时间 | `EffectiveAt` | effective_at | 上架/调价生效 |
| | 状态 | `Status` | status | DRAFT/PUBLISHED/OFFLINE |
| region_offers(区域运营包) | 区域 | `RegionPath` | region_path | LTREE，须落在该公司经营区域 |
| | 区域名称 | `Name` | name | 可空;展示名回退: 区域名→公司名 |
| | 区域月费 | `MonthlyFee` | monthly_fee | 生效价覆盖基础价 |
| orders(订单侧) | 成交价 | `PriceSnapshot` | price_snapshot | 下单时生效价快照 |

> 计价规则：生效价 = 区域价(前缀匹配) ?? 公司基础价；展示名两级回退；订单只存快照。
> 约束（seed 已校验）：未经营区域不得设区域价、不得下单；账单金额 = 快照价。

## 3. 阶段5 · 订单与计费（internal/domain/{order,billing}）

### 3.1 orders（订单，源自 order.html + 全案 4.2 Order）

| 页面列名 | 字段名 | DB 列（约定） | 枚举/说明 |
|:---------|:-------|:--------------|:----------|
| 订单号 | `OrderNo` | order_no | 如 ORD-20250817-001 |
| 客户 | `CustomerID` | customer_id | BIGINT → customers |
| 渠道 | `ChannelID` | channel_id | BIGINT → channels（REQ-ORD-006 必填不可改） |
| 产品 | `OfferID` | offer_id | BIGINT → product_offers |
| 地址 | `AddressID` | address_id | BIGINT → addresses |
| 当前环节 | `Stage` | stage | 1~12（见 terms.md 第 1 节） |
| 状态 | `Status` | status | PENDING/RESERVED/INSTALLING/DONE（见 terms.md 第 3 节） |
| 区域 | `RegionPath` | region_path | LTREE |
| 成交价 | `PriceSnapshot` | price_snapshot | 下单时生效价快照（账单金额以此为准） |

> 快照列（TS 实体）：`customer_name`（客户姓名）、`offer_name`（产品名），下单时冻结，改名/调价不影响历史订单（与 `price_snapshot` 同规则）。

### 3.2 order_stages（订单环节时间轴）

| 页面列名 | 字段名 | DB 列（约定） | 枚举/说明 |
|:---------|:-------|:--------------|:----------|
| 环节 | `Stage` | stage | 1~12 |
| 完成时间 | `FinishedAt` | finished_at | — |
| 耗时 | `Duration` | duration | 可派生 |
| 重试 | `Retries` | retries | INT |
| 结果 | `Result` | result | DONE/DOING/PENDING（见 order.html） |

### 3.3 bills（账单，源自 billing.html）

| 页面列名 | 字段名 | DB 列（约定） | 枚举/说明 |
|:---------|:-------|:--------------|:----------|
| 账单号 | `BillNo` | bill_no | — |
| 客户 | `CustomerID` | customer_id | BIGINT |
| 账期 | `Period` | period | — |
| 金额 | `Amount` | amount | NUMERIC |
| 状态 | `Status` | status | UNPAID/PAID/OVERDUE |

> 快照列（TS 实体）：`customer_name`（客户姓名）、`legal_entity_id`/`legal_entity_name`（企业）、`region_id`/`region_name`（经营区域），账单生成时冻结，客户改名/转品牌/搬家不改历史账单。

## 4. 阶段3/4 · 资产与资源（internal/domain/{asset,resource}）

### 4.1 assets（资产台账，源自 asset.html + 全案 4.2 Asset）

| 页面列名 | 字段名 | DB 列（约定） | 枚举/说明 |
|:---------|:-------|:--------------|:----------|
| 资产编码 | `AssetCode` | asset_code | 如 A-20260001 |
| 绑定标签 | `TagID` | tag_id | BIGINT → tags（可空） |
| 类型 | `Type` | type | 光猫/ONU/路由器等 |
| 入库批次 | `BatchID` | batch_id | BIGINT → asset_batches |
| 部署地址 | `AddressID` | address_id | BIGINT → addresses（可空，未部署为空） |
| 状态 | `Status` | status | IN_STOCK/DEPLOYED/MAINTENANCE/SCRAPPED（见 terms.md 第 4 节） |

> 页面 asset.html 的「标签编号/EPC 码」经 `tag_id → tags` 反查展示，「位置」= `address_id`，「生命周期」= `status`。
> 状态轨迹（TS 实体）：`asset_lifecycles`，资产每次状态/位置变更一行，含事发时 `address_id` + `address_name` 快照 + `changed_at`，历史不随当前状态漂移。
> 区域/企业锚点（TS 实体）：`region_id`/`region_name`（部署地址所在经营区域，未部署为空）、`legal_entity_id`/`legal_entity_name`（企业），按地区/企业统计资产。

### 4.2 ports（端口，源自 resource.html + 全案 4.2 Port）

| 页面列名 | 字段名 | DB 列（约定） | 枚举/说明 |
|:---------|:-------|:--------------|:----------|
| 端口编码 | `PortCode` | port_code | 如 P-SPL01-01 |
| 四码端口码 | `QuadCode` | quad_code | — |
| 所属 OLT/分光器 | `ResourceID` | resource_id | BIGINT → resources |
| 地址 | `AddressID` | address_id | BIGINT |
| 状态 | `Status` | status | IDLE/RESERVED/USED/DISABLED（见 terms.md 第 4 节） |
| 占用订单 | `OrderID` | order_id | BIGINT，RESERVED 时非空 |

> 状态变更历史（TS 实体）：`port_change_history`，端口每次状态/占用变化一行（变更后 status + order_id 快照 + changed_at），历史不随当前状态漂移。
> 区域/企业锚点（TS 实体）：`region_id`/`region_name`（地址所在经营区域）、`legal_entity_id`/`legal_entity_name`（所属设备企业），按地区/企业统计端口；`lo_accounts` 同挂 `region_id`/`region_name`（客户所在经营区域）。

## 5. 阶段6 · 四码合一（internal/domain/quadlink）

### 5.1 quad_link（四码关联，源自全案 4.2 + REQ-AMS-003）

| 字段名 | DB 列 | 枚举/说明 |
|:-------|:------|:----------|
| `AssetID` | asset_id | BIGINT → assets |
| `CustomerID` | customer_id | BIGINT → customers（四码第 2 项=客户，非系统账号 user） |
| `PortID` | port_id | BIGINT → ports |
| `AddressID` | address_id | BIGINT → addresses |
| 状态 | `status` | LINKED/CONFLICT/UNLINKED（见 terms.md 第 4 节） |

> 四列各建索引 + 唯一约束（见技术栈方案 3.3），任一码反查单表索引。
> **口径裁定**：四码=资产-客户-端口-地址（全案 REQ-AMS-003/REQ-CONS-002 权威）。
> 技术栈方案 3.3 原文写 `quad_link(asset_id, user_id, port_id, addr_id)`，`user_id` 系笔误，
> 应为 `customer_id`（`user` 是系统账号域，`customer` 是客户域，二者不同，见 domain-map）。

## 6. 跨域通用列（所有实体强制）

| 列 | 类型 | 说明 |
|:---|:-----|:-----|
| `id` | BIGSERIAL PK | 自增主键 |
| `created_at` | TIMESTAMPTZ | 默认 now() |
| `updated_at` | TIMESTAMPTZ | 有变更流的实体才加 |
| `brand_id` / `region_path` | — | 品牌区域横切维度，主数据实体按需带 |

## 7. 师傅域（cross-domain worker，server-ts/src/entities/worker.ts）

> 本节实体落在 server-ts（TypeORM），字段名用 TS 实体名，DB 列经 SnakeNamingStrategy 转 snake_case。

### 7.1 worker_groups / workers（班组·师傅）

`worker_groups`（班组）：

| 字段名(TS实体) | DB 列 | 枚举/说明 |
|:---------|:------|:----------|
| `code` | code | 班组编码，公司内唯一（稳定标识，name 可改 code 不变） |
| `name` | name | 班组名称（可改名） |
| `legalEntity` | legal_entity_id | BIGINT → legal_entities |
| `leader` | leader_id | BIGINT → workers（组长，可空） |
| `leaderName` | leader_name | 组长姓名快照 |

`workers`（师傅）：

| 字段名(TS实体) | DB 列 | 枚举/说明 |
|:---------|:------|:----------|
| `staffNo` | staff_no | 工号，如 WK-1024（唯一） |
| `name` | name | 师傅姓名 |
| `group` | group_id | BIGINT → worker_groups（当前归属，可变更） |
| `regionId` | region_id | 服务区域，须落班组公司经营区域 |
| `phone` | phone | 联系电话（脱敏） |
| `status` | status | 1在职 / 0离职 |
| `joinedAt` | joined_at | 入职时间 |
| `leftAt` | left_at | 离职时间，null=在职 |

### 7.2 worker_group_memberships（班组归属台账，新增）

换班组只新增行、不覆盖；历史归属与当前 `group` 解耦。

| 字段名(TS实体) | DB 列 | 说明 |
|:---------|:------|:-----|
| `worker` | worker_id | BIGINT → workers |
| `group` | group_id | BIGINT → worker_groups（仅导航） |
| `groupName` | group_name | 班组名快照，改名不影响历史 |
| `legalEntityId` | legal_entity_id | 班组所属公司快照（公司级历史归属） |
| `regionId` | region_id | 当时服务区域快照（区域维度统计） |
| `regionName` | region_name | 当时服务区域名快照 |
| `reason` | reason | 调组原因（可追溯） |
| `operatorAccountId` | operator_account_id | 操作人账号 id（谁执行的调组） |
| `effectiveFrom` | effective_from | 归属生效时间 |
| `effectiveTo` | effective_to | 归属结束时间，null=至今 |

### 7.3 归属快照与月度粒度铁律

1. **班组关系（双向）**：师傅事件级/月度级事实（`dispatch_tickets`/`worker_feedbacks`/`worker_materials`/`worker_tools`/`asset_returns`/`worker_performances`/`worker_commissions`/`worker_schedules`）均以 `@ManyToOne → worker_groups` 挂 `group`（FK 列 `group_id`），`WorkerGroup` 侧对应 `@OneToMany` 反向集合；另存 `group_name` 快照（`xxx_id` + `xxx_name` 冗余，见第 0 节）。冻结语义靠「事实行不可变 + `group_name` 快照」：班组改名不改历史，班组删除被 FK 阻止（应软删）。
2. **月度粒度**：月度级事实（`worker_performances`/`worker_commissions`/`worker_schedules`）粒度必须为「师傅 × 月 × 班组 × 区域」，唯一键 `(worker_id, period, group_id, region_id)`；师傅月中调组/换区拆多行，一行=该月在该班组该区域的一段贡献。评「本月最佳班组」= `GROUP BY group_id`；「按地区统计」= `GROUP BY region_id`。
3. **区域维度**：师傅事件级/月度级事实均冗余 `region_id` + `region_name` 快照（事发时服务区域），与 `group`/`legal_entity` 并列成师傅事实表的第三归属维度，支撑「按地区统计」。
4. **归属口径（事发时）**：每单/每评价按发生那一刻的班组归属；工单跨班组时记「派单时班组」，评价跟随工单。
5. `worker_settings`（当前接单设置）与 `worker_messages`（站内通知）不加快照，跟随当前班组。
6. **姓名快照**：`dispatch_tickets`/`worker_feedbacks` 另存 `worker_name` 快照，师傅改名不改历史工单/评价。

## 8. 归属台账实体（通用深度关联模式，六张）

可变归属/状态 + 派生历史 → 配「台账」四件套：FK 双向 + `xxx_name` 快照 + 时间区间 + `reason`/`operator_account_id` 追溯。

| 台账实体 | 主体 | 归属维度 | 除通用字段外的关键列 |
|:---------|:-----|:---------|:----------|
| `account_org_histories` | accounts | 公司/部门/岗位 | `legal_entity`/`dept`/`post`(FK) + 各自 name 快照 |
| `customer_histories` | customers | 公司/地址 | `legal_entity`/`address`(FK) + name 快照 |
| `product_price_histories` | product_offers | 价格 | `old_monthly_fee`/`new_monthly_fee`/`effective_at` |
| `dispatch_transfers` | dispatch_tickets | 师傅(改派) | `from_worker`/`to_worker`(FK) + name 快照 + `transferred_at` |
| `resource_assignments` | resources | 公司/区域/地址 | `legal_entity`/`address`(FK) + name 快照 + `region_id`/`region_name` |
| `asset_assignments` | assets | 师傅/地址 | `worker`/`address`(FK) + name 快照 |

> 通用字段：`effective_from` / `effective_to`(null=至今) / `reason` / `operator_account_id`；主体 `@ManyToOne` + `@OneToMany` 反向集合；归属维度 `@ManyToOne` + `xxx_name` 快照。

### 8.1 企业锚点铁律（跨企业评估对比）

凡归属到某企业的「业务主单/事实」记录，冗余 `legal_entity_id` + `legal_entity_name` 快照，冻结事发时企业，作为跨企业评估对比的 O(1) 锚点（不依赖易变的归属链：客户转品牌/师傅换班组/设备调拨）。

已覆盖：`orders`/`dispatch_tickets`/`complaints`/`dismantles`/`bills`/`worker_performances`/`worker_commissions`/`worker_schedules`/`worker_materials`/`worker_tools`/`worker_feedbacks`/`asset_returns`/`assets`/`ports`/`lo_accounts`/`quad_links`/`replacements`/`transfers`。

不覆盖：集团共享数据（`roles`/`permissions`/`regions`/`addresses`/`biz_params`，本就跨企业共享）；纯时间轴/日志子记录（`order_stages`/`scan_logs`/`reserve_records`/`port_change_history`/`asset_lifecycles`/`payments`，经父主单继承企业，防冗余爆炸）。

> 追溯补充：`order_stages` 冗余 `operator_account_id` + `operator_name`（环节执行人）；`scan_logs` 冗余 `worker_name`（扫码师傅）。`region_price_histories` 为区域调价台账（与 `product_price_histories` 同构）。

## 8A. 对齐三端页面补齐的实体（server-ts）

> 为消除「页面有列、实体缺失」的缺口补的实体，字段名沿用 TS 实体，DB 列经 SnakeNamingStrategy 转 snake_case。

| 实体 | 承接页 | 关键列 |
|:-----|:-------|:-------|
| provision_logs | admin/provlog.html 下发日志 | task_id/resource_id/template_id/result/retries/created_at |
| real_name_verifications | admin/customer.html 实名核验 | customer_id/method/verified_at/result/operator_account_id/operator_name |
| channels | order/dispatch 下单渠道 | code/name/status（REQ-ORD-006 必填不可改） |
| alarms | admin/alarm.html 告警 | alarm_no/level/source/content/status |
| cdrs | admin/aaalog.html 话单 | loid/session_time/input_output_octets/billing_status |
| auth_logs | admin/aaalog.html 认证日志 | loid/result/created_at |
| device_metrics | admin/device.html OLT 监控 | resource_id/optical_power/packet_loss/status |
| device_maintenances | worker 设备健康 | device_no/health_score/fault_count/priority |

> 仍缺实体、但按 domain-map 属「待建域」或派生视图的页面（本期不臆造）：`settings.html`→`biz_params`（migrations 已建表，TS 实体已补 `BizParam`）、`paycheck.html`→渠道对账（派生聚合，非基表）。

## 9. 字段字典的使用规则（写入 Agent 输入包）

1. 实现实体前，先查本文件是否已定其字段；已定则**照抄字段名与枚举**，不得另起别名。
2. 未定字段（本文件无该实体）时，字段名遵循第 0 节命名规则，并**回写本文件**补一节，避免下个 Agent 再猜。
3. 页面列名与字段名必须一一对应；页面新增列时，同步在此登记英文字段名与枚举。
4. 状态枚举一律引用 `terms.md`，本文件不重复定义枚举值（仅标注引用来源）。
