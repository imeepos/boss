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

### 1.6 audit_logs（审计日志）· biz_params（业务参数）

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 操作人 | `AccountID` | account_id | BIGINT |
| 操作 | `Action` | action | 数据变更/状态变更/权限变更 |
| 对象 | `TargetType` / `TargetID` | target_type/target_id | — |
| 详情 | `Detail` | detail | JSONB |
| IP | `IP` | ip | INET |
| 参数 | `Key` / `Value` | key/value | value=JSONB |

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

### 2.2 products（产品资费，源自 product.html）

| 页面列名 | 字段名 | DB 列（约定） | 枚举/说明 |
|:---------|:-------|:--------------|:----------|
| 产品 | `Name` | name | — |
| 带宽 | `Bandwidth` | bandwidth | 如 300M/500M/1000M |
| 月费 | `MonthlyFee` | monthly_fee | NUMERIC |
| 生效时间 | `EffectiveAt` | effective_at | 调价生效时间 |
| 状态 | `Status` | status | DRAFT/PUBLISHED/OFFLINE（见全案 4.2 Product/Plan） |

## 3. 阶段5 · 订单与计费（internal/domain/{order,billing}）

### 3.1 orders（订单，源自 order.html + 全案 4.2 Order）

| 页面列名 | 字段名 | DB 列（约定） | 枚举/说明 |
|:---------|:-------|:--------------|:----------|
| 订单号 | `OrderNo` | order_no | 如 ORD-20250817-001 |
| 客户 | `CustomerID` | customer_id | BIGINT → customers |
| 产品 | `ProductID` | product_id | BIGINT → products |
| 地址 | `AddressID` | address_id | BIGINT → addresses |
| 当前环节 | `Stage` | stage | 1~12（见 terms.md 第 1 节） |
| 状态 | `Status` | status | PENDING/RESERVED/INSTALLING/DONE（见 terms.md 第 3 节） |
| 渠道 | `ChannelID` | channel_id | BIGINT → channels |
| 品牌 | `BrandID` | brand_id | — |
| 区域 | `RegionPath` | region_path | LTREE |

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

## 4. 阶段3/4 · 资产与资源（internal/domain/{asset,resource}）

### 4.1 assets（资产台账，源自 asset.html + 全案 4.2 Asset）

| 页面列名 | 字段名 | DB 列（约定） | 枚举/说明 |
|:---------|:-------|:--------------|:----------|
| 资产编码 | `AssetNo` | asset_no | — |
| 标签编号 | `TagNo` | tag_no | 电子标签 |
| EPC 码 | `EpcCode` | epc_code | — |
| 类型 | `Type` | type | — |
| 入库批次 | `BatchNo` | batch_no | — |
| 位置 | `Location` | location | — |
| 生命周期 | `Lifecycle` | lifecycle | — |
| 状态 | `Status` | status | IN_STOCK/DEPLOYED/MAINTENANCE/SCRAPPED（见 terms.md 第 4 节） |

### 4.2 ports（端口，源自 resource.html + 全案 4.2 Port）

| 页面列名 | 字段名 | DB 列（约定） | 枚举/说明 |
|:---------|:-------|:--------------|:----------|
| 端口编号 | `PortNo` | port_no | — |
| 四码端口码 | `QuadCode` | quad_code | — |
| 所属 OLT/分光器 | `ParentID` | parent_id | BIGINT → resources |
| 地址 | `AddressID` | address_id | BIGINT |
| 状态 | `Status` | status | IDLE/RESERVED/USED/DISABLED（见 terms.md 第 4 节） |
| 占用订单 | `OrderID` | order_id | BIGINT，RESERVED 时非空 |

## 5. 阶段6 · 四码合一（internal/domain/quadlink）

### 5.1 quad_link（四码关联，源自全案 4.2 + REQ-AMS-003）

| 字段名 | DB 列 | 枚举/说明 |
|:-------|:------|:----------|
| `AssetID` | asset_id | BIGINT → assets |
| `CustomerID` | customer_id | BIGINT → customers（四码第 2 项=客户，非系统账号 user） |
| `PortID` | port_id | BIGINT → ports |
| `AddrID` | addr_id | BIGINT → addresses |
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

## 7. 字段字典的使用规则（写入 Agent 输入包）

1. 实现实体前，先查本文件是否已定其字段；已定则**照抄字段名与枚举**，不得另起别名。
2. 未定字段（本文件无该实体）时，字段名遵循第 0 节命名规则，并**回写本文件**补一节，避免下个 Agent 再猜。
3. 页面列名与字段名必须一一对应；页面新增列时，同步在此登记英文字段名与枚举。
4. 状态枚举一律引用 `terms.md`，本文件不重复定义枚举值（仅标注引用来源）。
