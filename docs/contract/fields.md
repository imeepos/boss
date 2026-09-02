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
| 实名核验 | 实名核验 | `verify-logs` | （无独立） | `verifications`（000059 归一，subject_type='customer'；聚合/列表取数同源，PASS 前主档证件号一致性门禁 2026-08-29） |
| 调拨类型 | 类型 | `type`（`ASSET/PORT/DEVICE`） | （无独立） | `transfers.resource_id`（资产/端口维度待扩展） |

> 规则：禁止为求「同名」而改 API 契约或 DB 表名（两者分属不同层，耦合即反模式）；
> 新增跨层差异概念时在此登记，页面/Agent 一律以本表为准消歧义。

## 1. 阶段1 · 系统管理与组织（internal/domain/user，已定型）

### 1.1 accounts（后台账号）

> 固定用途：仅承载管理后台登录用户。后台员工、运营人员、管理员统一使用本表；客户 App 和师傅端不得使用 `accounts` 登录。

| 页面列名 | 字段名 | DB 列 | 类型/枚举 |
|:---------|:-------|:------|:----------|
| — | `Username` | username | VARCHAR(64) UNIQUE |
| — | `PasswordHash` | password_hash | TEXT |
| — | `RealName` | real_name | VARCHAR(64) |
| — | `Phone` | phone | VARCHAR(32) |
| 工号 | `StaffNo` | staff_no | VARCHAR(32)，可空；非空全局唯一（000173，企业员工登录标识） |
| — | `RoleID` | role_id | BIGINT → roles |
| 状态 | `Status` | status | 1启用 / 0停用 |
| — | `LegalEntityID` | legal_entity_id | BIGINT → legal_entities（可空） |
| — | `DeptID` | dept_id | BIGINT → departments（可空） |
| — | `PostID` | post_id | BIGINT → posts（可空） |
| — | `RegionScope` | region_scope | LTREE，空=全集团 |

> 企业员工后台录入（000173，公司管理页「员工」入口，permCode `menu:company`）：
> `GET /legal-entities/{id}/staff` 列表（partner_admin/partner_staff，含工号）、
> `POST /legal-entities/{id}/staff` 录入（staffNo 工号可空 / username 登录名 / password bcrypt 落库 /
> realName / phone / roleCode 白名单限企业两角色；工号冲突 40900、登录名冲突 40900、入参不合法 42200）、
> `PUT /legal-entities/{id}/staff/{accountId}/password` 重置密码（密码不入审计）、
> `PUT /legal-entities/{id}/staff/{accountId}/status` 启停。
> 与企业工作台自助建号（`POST /partner/staff`）平行：工作台=partner_admin 自助（无工号），后台=受权运营视角（可编工号/建管理员）。

### 1.2 roles / permissions（角色·权限）

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| — | `Code` | code | 内置 7 角色码：customer/technician/asset_admin/resource_admin/ops/analyst/sysadmin；自定义角色 code 后端生成（`custom_*`，迁移 000100） |
| — | `Name` | name | 中文名 |
| 类型 | `IsBuiltin` | is_builtin | true=内置只读 / false=自定义可改删 |
| 权限码 | `Code`(perm) | code | 如 `asset:create`、`menu:order` |

> 自定义角色（菜单权限页·角色管理）：新建时可选内置模板整体复制其权限集，落库前可单独增删；编辑=权限码全集全量替换；被账号/岗位引用拒删（40900），内置拒改删（40300）。`/auth/me` 透出 `permissionCodes`，自定义角色的前端菜单按 `menu:<key>` 逐项动态推导。

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
| 子公司 | `TaxJurisdiction` | tax_jurisdiction | ''/CN/PH，空=未定（000109，开票属地配置源） |
| 子公司 | `TaxChannel` | tax_channel | manual/leqi/bir_eis，空缺省归一化 manual |
| 部门 | `LegalEntityID` | legal_entity_id | BIGINT → legal_entities |
| 部门 | `Name` | name | 子公司内唯一 |
| 岗位 | `Code` | code | dispatcher/cashier/agent/field_tech…（部门内唯一） |
| 岗位 | `DeptID` | dept_id | BIGINT → departments |
| 岗位 | `Roles` | （聚合） | post_roles 关联 |

> 接口（admin，`/api/admin/v1`，迁移 000099 配套）：`DELETE /departments/:id`（仍有岗位或在职账号挂靠 → 40900 拒）、`DELETE /posts/:id`（事务内连同 post_roles；仍有账号挂岗 → 40900 拒）。物理删除，无软删标记；占用校验在前，审计记 op=delete。
> 页面「组织架构与人员」（`/org/staff`，permCode `menu:staff`，仅 sysadmin）：左树 = 子公司→部门→岗位（节点带成员数），右侧成员列表 = 该部门在职账号（列：账号/姓名/角色/岗位/状态）；成员赋岗走账号表单级联下拉（企业→部门→岗位）；人员"删"= 停用（status 0，账号有审计外键不做物理删）。

### 1.5 addresses（地址层级）

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| — | `Path` | path | LTREE，唯一权威 |
| — | `Level` | level | 1市 2区 3街道 4小区 5楼栋 |
| — | `Name` | name | — |
| — | `ParentID` | parent_id | 派生（=反查 path 父节点） |
| — | `Geom` | geom | GEOGRAPHY(POINT) 可空，WGS84；写路径 `PUT /addresses/{id}/geom`（menu:address，000174 同期），逆地理最近邻 `GET /addresses/nearest?lat&lng&radiusM`（KNN + ST_DWithin，缺省 500m 上限 50km）从此取数 |

> 区域硬关联（TS 实体）：楼栋级地址挂 `region_id`（→ regions，经营区域）+ `region_name` 快照，固化「地址→经营区域」映射；客户/资产/端口/LO账号经此继承区域，杜绝「有地址无订单则不知属哪个区域」的孤儿。

> 治理标记读取路径（2026-08-29 对账补齐）：admin `GET /addresses?needsReview=1` 仅返回 `needs_review=TRUE` 节点（列形状同 parentId 树分支，前端地址管理页「待治理」toggle 治理队列入口）；与 `unlinked=1` 互斥，needsReview 优先；`=0`/缺省行为不变。写路径见 §1.5.0b。

#### 1.5.0a 用户端地址树查询（user 端契约 misc.yaml，2026-08-26）

| 页面列名 | 字段名 | 端点/字段 | 说明 |
|:---------|:-------|:------|:----------|
| 所在地区 | `AddressPath` | user_addresses.address_path + AddressInfo.addressPath | ltree 字符串，空=历史自由文本地址（迁移 000152） |
| 子节点列表 | — | GET /address-tree?parentId= | 级联懒加载，返回 AddressNode（id/level/name/hasChildren/锚点） |
| 全树搜索 | — | GET /address-tree/search?q= | 命中+祖先链（均含 hasChildren），与 admin /addresses/search 同语义；hasMore=true=超 20 条截断（客户端提示收紧关键字） |
| 路径批量反查 | — | GET /address-tree/lookup?paths= | 逗号分隔≤50；items=[{path,node,ancestors}]（与 search 同形）+missing；缺失路径不整体 404（address_path 弱引用，节点可删），编辑地址/地址簿反显面包屑用 |
| 家庭地址 phone | `Phone` | user_addresses.phone | PUT /addresses/{id}：空=保持原值（列表仅回 phoneMasked，防自定义手机号被账户号静默覆盖）；POST /addresses：空=沿用账户手机号 |

#### 1.5.0b admin 订单域内联建址（POST /api/admin/v1/orders/address，迁移 000171，2026-08-29）

> 开单零阻塞（meeting-minutes/2026-08-29 §九）：开单流程内新用户无地址时就地建址，门禁 `menu:order`（能开单就能建址）；地址管理页 `menu:address` 仍是治理入口。

| 请求字段 | 字段名 | 说明 |
|:---------|:-------|:------|
| 客户 | `customerId` | 必填，客户须已存在（缺失 40400） |
| 五级地址名 | `city`/`district`/`street`/`compound`/`building` | 自上而下全必填（空缺 42200）；单级 ≤60 字；逐级 lookup-miss-then-create 单事务补建，中间层缺失不阻断 |
| 回填客户档案 | `backfillCustomer` | 显式布尔；true=同事务把 customers.address_id 更新为新楼栋并同步 legal_entity_id/region_id 快照（customers.address_id 现网 NOT NULL，故为覆盖式回填，非"仅空时回填"） |

| 响应字段 | 字段名 | 说明 |
|:---------|:-------|:------|
| 楼栋地址 | `addressId` / `fullPath` / `fullPathNames` | 楼栋级（level=5）节点 id / ltree path / 「 / 」连接的展示名链 |
| 归属预览 | `legalEntityId` / `regionPath` / `fallback` | 与下单归属推导同口径（000076/000077）；fallback=true=祖先链无区域覆盖、兜底平台总公司，服务端同时打 `[order-ownership] REGION UNCOVERED ALERT` 日志并生成归属修正 P1 待办（notify todo，幂等键 order-inline-addr:path，DueHours=4） |
| 治理标记 | `needsReview` | 本次内联新建的 1-3 级节点列表（id/level/name），落库 addresses.needs_review=true；4-5 级新建 false。全部新建节点 source='order-inline' |
| 回填结果 | `backfilled` | 本次是否实际更新了客户档案 |
| 冲突语义 | — | 同父同名命中即复用既有节点；`UNIQUE(path)` 撞库（23505）返回 40900（user.ErrDuplicate），label 自动加 `_N` 后缀重试；模糊去重提示由前端 search 承担 |

> label 规则：name 转小写、空白转 `_` 后满足 `^[a-z0-9_]+$` 则直接用（≤48 字符）；否则 `n_`+sha1 前 8 位稳定后缀（中文 name 无法转写，回显靠 name，客服不感知 label）。region 继承：新建节点取最近挂 region_id 祖先；父链全空为 NULL，由归属推导层兜底。

#### 1.5.0c admin 客户域内联建址（POST /api/admin/v1/customers/address，2026-09-01）

> 代客开户场景（meeting-minutes/2026-08-29 同款零阻塞原则延伸）：开户弹框内装机地址树上没有时就地建址，先建址后开户，门禁 `menu:customer`（能开户就能建址）。与 §1.5.0b 同 handler、同请求/响应契约、同治理语义（needsReview/fallback/待办/审计），仅两处差异：

| 差异点 | orders/address | customers/address |
|:-------|:---------------|:------------------|
| 门禁 | `menu:order` | `menu:customer` |
| `customerId` | 必填（缺失 42200），客户须已存在（缺失 40400） | 可省（0=未建档开户场景，跳过存在性检查）；>0 须已存在；`backfillCustomer=true` 时必填（缺失 42200），负值一律 42200 |

> 开户前端用法：建址回执 `addressId` 直接作为 POST /customers 的 `addressId` 入参；开户场景客户尚不存在，恒不传 `backfillCustomer`（`backfilled` 恒 false）。

### 1.5.1 geo_country / geo_subdivision（国际地理基础数据，迁移 000038）

> 依据 ISO 3166-1/2 + UN M49 + CLDR；国家主键 = alpha-2，区划主键 = 完整 ISO 3166-2 码；停用码软删除保留。

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 国家 | `Alpha2` | alpha2 | CHAR(2) 主键，如 PH/CN/US |
| — | `Alpha3` | alpha3 | CHAR(3) 唯一，如 PHL |
| — | `NumericCode` | numeric_code | CHAR(3) 唯一，=UN M49 |
| 国家名 | `ShortName` | short_name | English short name |
| — | `FullName` | full_name | 全称，可空 |
| 状态 | `Status` | status | INDEPENDENT / DISCONTINUED |
| 大洲 | `ContinentCode` | continent_code | AS/EU/NA/SA/AF/OC/AN |
| — | `IsActive` | is_active | 停用码保留 false，不物理删除 |
| 区划码 | `Code`(subdiv) | code | 'PH-NCR'/'CN-BJ'，自引用 parent_code 成树 |
| 层级 | `Level`(subdiv) | level | 1~4，各国深度不同 |
| 类别 | `Category` | category | state/province/region/municipality… |
| 译名 | `Name`(i18n) | name | geo_country_i18n / geo_subdivision_i18n：`(code, locale, name_type)` |

关联表（均一对多）：`country_time_zone(tz_name, IANA)`、`country_currency(currency, is_primary, minor_unit)`、`country_calling_code(calling_code, E.164)`。

`addresses` 国际化挂接（迁移 000038，path 权威不变）：`CountryID → country_code`（→ geo_country，空=历史数据未挂）、`AdminCode → admin_code`（→ geo_subdivision，一级行政区锚点）。

### 1.5.2 odn_region_code / odn_city_code（ODN 地理空间编码映射，迁移 000075）

> 依据《Suniway ODN 地理空间编码规范》V1.0；派生映射表，PSGC（000041）权威不动。裁定见 adopted note 2026-08-20-odn-geospatial-encoding-alignment；审计 alignment-audit §10（E4/E5/E10/E11）。

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| PRV 编码 | `PrvCode` | prv_code | CHAR(6) 主键 `^[A-Z]{3}\d{3}$`，如 PHL001；82 行全量（PHL075 预留不落） |
| PSGC 锚点 | `PsgcCode` | psgc_code | → geo_subdivision.code，如 PH-1400100000 |
| 规范省名 | `SpecName` | spec_name | 规范索引表中文名 |
| 备注 | `Note` | note | 层级差异/规范勘误留痕（NCR→大区节点、HUC、Maguindanao 拆分等） |
| 城市前缀 | `CityPrefix` | city_prefix | VARCHAR(5) `^[A-Z]{3,5}$`（规范索引含 4-5 字母，见 ISSUE.md），与 prv_code 复合主键（规范 2.3 省内唯一） |
| 城市 PSGC | `PsgcCode` | psgc_code | → geo_subdivision.code（HUC 直辖大区时锚大区直属城市节点），119 行全量 |

> 局点 3 位序号（MNL001 的 001 段）属 odn 域后续实体；网格分区与基础设施见 §1.5.3。

### 1.5.3 odn_grid / odn_facility（ODN 无源物理层，迁移 000078，internal/domain/odn）

> 依据《Suniway ODN 地理空间编码规范》V1.0 第 4 章。E6/E7/E9 落地；光缆段落/纤芯（E8）见本节末。
> 管理面：`menu:odn` 权限（迁移 000080，sysadmin+resource_admin），REST `/odn/grids|facilities|segments`（契约 admin/odn.yaml）。

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| PRV | `PrvCode` | prv_code | CHAR(6) → odn_region_code |
| 城市前缀 | `CityPrefix` | city_prefix | VARCHAR(5) → odn_city_code（复合 FK） |
| 网格码 | `GridCode` | grid_code | SMALLINT 1~99，复合主键（规范 4.1） |
| 网格名称 | `Name` | name | 如 马尼拉老城区 |
| 覆盖区域 | `Coverage` | coverage | 文本描述 |
| 状态 | `Status` | status | ACTIVE / RESERVED（预留扩展区）/ RETIRED |
| 占用数 | `Facilities` | —（聚合） | 网格内 P/MH 计数；≥800 触发预警（规范 4.7） |
| 设施编码 | `Code` | code | VARCHAR(8) 主键 `^(P\|MH\|TW\|CLS\|TBX)[0-9]{5}$`（红线 3） |
| 类型 | `Kind` | kind | P 电杆 / MH 人井 / TW 铁塔 / CLS 接头盒 / TBX 终端盒 |
| 所属网格 | `GridCode` | grid_code | P/MH 必填 01~99（编码网格段须一致）；TW/CLS/TBX NULL |
| 坐标 | `Lat`/`Lng` | lat/lng | 可空 |
| 状态 | `Status` | status | IN_USE / RETIRED（报废永久锁定，禁止复用/删除） |

> 校验：`odn.ValidateFacilityCode`（格式 + 序号 000/00000 预留禁用）；入库校验网格已备案（ErrGridMissing）与容量 999（ErrGridFull）。

### 1.5.4 odn_cable_segment / odn_fiber（光缆段落与纤芯，迁移 000079，规范第 5 章/E8）

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| A 端 | `ACode` | a_code | 优先级较高设备（规范 5.2：SNW>ODF>OCC>CLS>ODB>P>MH；TW/TBX 等规范未列者最低档兜底） |
| B 端 | `BCode` | b_code | 优先级较低设备；与 A 端唯一约束 `(a_code,b_code)`，同对幂等 |
| 段落名 | `Name` | name | 可空描述 |
| G 号 | `GNo` | g_no | 1~99，与 segment_id 复合主键（规范 5.3 `段落-G01~G99`） |
| 光缆型号 | `Kind` | kind | 文本描述 |

> 存储口径：`--` 仅设计图纸不入库（规范 5.1），系统侧存两端编码列；`OrderEndpoints` 定向（同优先级拒绝 ErrSamePriority），端点正则覆盖设施码（5 位）与核心设备码（3 位，见 §1.5.5 设备实体）。

### 1.5.5 odn_site / odn_device（局点与核心链路设备，迁移 000081/000143，资产编码规范第 2/3 章）

> 管理面同 §1.5.3（`menu:odn`，`/odn/sites|devices`）；000143 起 odn_device 补坐标列（用户要求"所有物料有地理位置 与地图关联"），与 GIS `/gis/odn-points` 图层接通。

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| PRV/城市 | `PrvCode`/`CityPrefix` | prv_code/city_prefix | → odn_city_code（复合 FK），与 site_no 复合主键 |
| 局点序号 | `SiteNo` | site_no | 1~999；NodeCode = 城市前缀 + 3 位序号（如 MNL001） |
| 局点名 | `Name` | name | 可空 |
| 坐标 | `Lat`/`Lng` | lat/lng | 可空 |
| 状态 | `Status` | status | ACTIVE / RETIRED（须无在用设备方可退役） |
| 设备码 | `Code` | code | `^(SNW\|OLT\|ODF\|OCC\|ODB\|SDB\|PRT\|TBP)\d{3}(-([2-9]\|[1-9]\d+))?$`（规范 5.1，同址扩容 -N 且 N≥2 禁 -1） |
| 类型 | `Kind` | kind | SNW/OLT/ODF/OCC/ODB/SDB/PRT/TBP |
| 归属局点 | `SiteNo` | site_no | 可空（市域设备） |
| 上级 | `ParentID` | parent_id | ODB→OCC / SDB→ODB / PRT→SDB / TBP→PRT（自引用，顶层 NULL） |
| 坐标 | `Lat`/`Lng` | lat/lng | 可空（000143；无坐标设备不上地图点位） |
| 唯一性 | — | uq_odn_device_city | 市域内唯一；SNW 全网唯一（部分索引） |
| 状态 | `Status` | status | IN_USE / RETIRED（报废永久锁定） |

> 校验：`ValidateDeviceCode`（格式+扩容后缀）+ `RequiredParentKind`（归属链）；子级设备上级须为同城在用设备。
> GIS 点位：`GET /gis/odn-points?entity=facility|site|device&bbox`（menu:gis）——设施/局点/设备自带 lat/lng 直接作地图图层，无坐标行过滤，bbox 数值区间过滤。

### 1.5.6 backup_jobs（数据备份迁移任务，迁移 000095，internal/domain/backup）

> 运维工具（SYS 域）：按表导出 gzip JSONL 归档 + 追加式导入恢复。管理面 `/base/backup`（menu:backup，sysadmin），API `/api/admin/v1/backup/*`。设计裁定见 adopted note 2026-08-21-backup-local-jsonl。

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 任务ID | `ID` | id | BIGSERIAL 主键 |
| 类型 | `Kind` | kind | backup（导出归档）/ restore（导入恢复） |
| 范围 | `Scope` | scope | all（public 全部业务表）/ tables（选表） |
| — | `Tables` | tables | TEXT[]，选表清单（scope=all 时为全表快照） |
| 状态 | `Status` | status | running / succeeded / failed |
| 文件 | `FileName` | file_name | 归档文件名（服务端本地磁盘，BOSS_BACKUP_DIR，默认 data/backups） |
| 大小 | `SizeBytes` | size_bytes | 归档字节数 |
| 表数 | `TableCount` | table_count | 归档内表数 |
| 行数 | `RowCount` | row_count | 导出/导入行数 |
| — | `Error` | error | 失败原因（failed 时非空） |
| 操作人 | `Operator` | operator_id | → accounts（联 real_name 快照展示） |
| 开始时间 | `CreatedAt` | created_at | TIMESTAMPTZ |
| 结束时间 | `FinishedAt` | finished_at | TIMESTAMPTZ，可空（执行中） |

> 恢复语义 = 只补不删：逐行 `INSERT ... ON CONFLICT DO NOTHING`，冲突行跳过；进程内串行执行（同时至多一条任务）。backup_jobs / schema_migrations 不入备份候选集。


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

### 1.6.1 auth.* 认证配置（页面 `/base/authconfig`「认证配置」，auth-config-v1.spec.md）

存储复用 `biz_params`（key 前缀 `auth.`）；secret 字段 AES-256-GCM 密文（`enc:v1:` 前缀）落库，GET 只回 `{value:"",hasValue}` 掩码标记，PUT 空串=不修改。

| 页面字段 | key（API/DB 同名） | 枚举/说明 |
|:--------|:-------------------|:----------|
| 启用开关(中国区) | `auth.cn.enabled` | true/false |
| AppKey | `auth.cn.appKey` | 极光 AppKey |
| AppSecret | `auth.cn.appSecret` | secret,密文落库 |
| Android 包名 | `auth.cn.packageName` | 如 com.ymm.boss.user |
| 预取号超时 | `auth.cn.preloadTimeoutMs` | 默认 5000,2000–10000 |
| 启用开关(海外) | `auth.my.enabled` | true/false |
| 认证渠道 | `auth.my.provider` | opengateway/none |
| 短信兜底渠道 | `auth.my.smsProvider` | engagelab/twilio/vonage |
| API Key | `auth.my.apiKey` | secret,密文落库 |
| 默认国家码 | `auth.my.countryCode` | 默认 +60 |
| 短信签名 | `auth.my.smsSign` | 如 YMMBOSS |
| 失败降级短信 | `auth.fallback.smsOnFail` | 默认 true |
| 计费告警 | `auth.fallback.billingAlert` | 默认 true |
| 自动注册 | `auth.fallback.autoRegister` | 默认 false |
| 隐私协议版本 | `auth.compliance.privacyVersion` | 如 v2026.02 |
| 授权页协议链接 | `auth.compliance.agreementUrl` | URL |

> 接口：`GET /auth-config`、`PUT /auth-config/{cn|my|fallback}`、`POST /auth-config/{group}/test`（permCode `menu:authconfig`，sys.yaml）。

### 1.6.2 realid.* 实名核验配置（页面 `/base/realidconfig`「实名核验配置」，迁移 000069）

存储复用 `biz_params`（key 前缀 `realid.`，与 auth.*/sms.* 同一套加密/掩码约定）。运行时由 realid.Dynamic 消费（60s 热生效）；未启用/未配置 → 实名提交落 PENDING 人工核验（客户档案 `POST /customers/{id}/real-name/verify` 不受影响）。

| 页面字段 | key（API/DB 同名） | 枚举/说明 |
|:--------|:-------------------|:----------|
| 启用开关 | `realid.enabled` | true/false，默认 true |
| 核验服务商 | `realid.provider` | aliyun_cloudauth（固定） |
| AccessKey ID | `realid.accessKeyId` | — |
| AccessKey Secret | `realid.accessKeySecret` | secret,密文落库 |
| 服务地址 | `realid.endpoint` | 空 = cloudauth.aliyuncs.com |

### 1.6.3 minio.* 存储配置（接口 `/storage-config`，permCode `menu:params`）

存储复用 `biz_params`（key 前缀 `minio.`，与 auth.*/realid.* 同一套加密/掩码约定）。运行时每次上传按当前配置解析（`internal/app/wiring_minio.go`，DB 配置非空覆盖 env `BOSS_MINIO_*` 兜底）；未配置时附件/照片上传不可用。

| 页面字段 | key（API/DB 同名） | 枚举/说明 |
|:--------|:-------------------|:----------|
| 服务地址 | `minio.endpoint` | host:port，空 = env 兜底 |
| AccessKey | `minio.accessKey` | — |
| SecretKey | `minio.secretKey` | secret,密文落库；空串=不修改 |
| 存储桶 | `minio.bucket` | 附件/照片对象桶 |
| 启用 TLS | `minio.useSSL` | true/false |

> 接口：`GET/PUT /storage-config`（sys.yaml；PUT values 仅接受上表五键，未知 key 42200）。

### 1.6.4 attachments（附件登记，通用组件 AttachmentManager，迁移 000065/000093）

对象实体存 MinIO（object_key），DB 只存登记。三端上传（admin/user/worker），admin 端查询/软删除。000093 加 `deleted_at` 软删除（被 verifications 等业务引用，禁物理删；MinIO 对象保留可审计）。

| 组件列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 文件名 | `fileName` | file_name | 原始文件名，关键词 ILIKE |
| 类型 | `contentType` | content_type | MIME |
| 大小 | `sizeBytes` | size_bytes | 上传上限 32MB（42200） |
| 上传者 | `uploaderType` + `uploaderId` | uploader_type / uploader_id | account / worker / customer（与 apikey 主体模型对齐） |
| 上传时间 | `createdAt` | created_at | RFC3339 |
| — | — | deleted_at | 软删除标记；列表/查询默认过滤 |

> 接口（admin，`/api/admin/v1`）：`POST /attachments/upload`（multipart `file`）｜`GET /attachments?uploaderType=&uploaderId=&keyword=&limit=&offset=`（类型+id 必须成对，缺省=当前登录身份；返回 `{items,total}`）｜`DELETE /attachments/:id`（软删，40400=不存在或已删，记审计 attachment.delete）｜`POST /attachments/batch-get` `{ids}`（选择回显，≤200，后端去重滤已删）。

### 1.6.5 push.* 推送配置（页面 `/base/pushconfig`「推送配置」，迁移 000094）

存储复用 `biz_params`（key 前缀 `push.`，与 auth.*/realid.*/minio.* 同一套加密/掩码约定）。运行时由 push.Dynamic 消费（60s 热生效）；凭据缺失/未配置 → 降级日志通道（不外呼）。设计契约：docs/plan/push-config-design.md。

| 页面字段 | key（API/DB 同名） | 枚举/说明 |
|:--------|:-------------------|:----------|
| 启用开关 | `push.enabled` | true/false，默认 true |
| 推送服务商 | `push.provider` | jpush（固定，预留多供应商路由） |
| AppKey | `push.jpush.appKey` | 极光控制台"应用设置" |
| Master Secret | `push.jpush.masterSecret` | secret,密文落库；空串=不修改 |
| REST 端点 | `push.jpush.apiUrl` | 默认 https://bjapi.push.jiguang.cn/v3 |
| iOS APNs 环境 | `push.jpush.apnsProduction` | true=生产/false=开发，默认 true；Android 无此区分 |
| 离线保留（秒） | `push.jpush.liveTime` | 默认 86400（1 天） |

> 接口：`GET /push-config`、`PUT /push-config/channel`、`POST /push-config/channel/test`（无 target=完整性校验；带 target+targetKind（registration_id|alias）=真实试发一条测试通知；permCode `menu:pushconfig`，迁移 000094）。env 兜底：`BOSS_JPUSH_APP_KEY`/`BOSS_JPUSH_MASTER_SECRET`。

### 1.6.6 push_devices（推送设备注册，迁移 000095）

App 启动/登录后上报 JPush RegistrationID；发送链路按主体反查定向目标。RegistrationID 全局唯一：设备换账号登录时**改绑**到新主体（UPSERT），原主体不再可见。

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| — | `ID` | id | BIGSERIAL PK |
| 主体类型 | `SubjectType` | subject_type | user / worker（与 apikey 主体词一致） |
| 主体 ID | `SubjectID` | subject_id | BIGINT；customer_id 或 worker_id |
| RegistrationID | `RegistrationID` | registration_id | 10-64 位字母数字；UNIQUE，冲突即改绑 |
| 厂商 | `Vendor` | vendor | jpush（预留多供应商） |
| 最近活跃 | `LastActiveAt` | last_active_at | 每次上报刷新 |
| 注册时间 | `CreatedAt` | created_at | — |

> 接口（两 portal 同形，JWT 鉴权）：`POST /api/user/v1/push/device`、`POST /api/worker/v1/push/device`，body `{registrationId, vendor?}`；形态非法 42200（契约 user/misc.yaml、worker/misc.yaml）。管理端只读（后台页二期随推送留痕一起评估）。

### 1.6.7 push_records（推送留痕，迁移 000096）

定向推送逐设备一行留痕；"留痕为权威、推送尽力而为"——发送失败不影响业务事务。触发点：派单指派（`POST /dispatch/pool/{ticketNo}/assign`）、admin 转派、师傅端转派（目标师傅）；文案统一出口 `push.TicketAssignedAlert`。

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 主体类型 | `SubjectType` | subject_type | user / worker |
| 主体 ID | `SubjectID` | subject_id | BIGINT |
| RegistrationID | `RegistrationID` | registration_id | SKIP_* 时空 |
| 标题/内容 | `Title` / `Alert` | title / alert | — |
| 扩展参数 | `Extras` | extras | JSONB，如 `{"ticketNo":"TK-1"}` |
| 状态 | `Status` | status | SENT / FAILED / LOG（日志通道）/ SKIP_NO_DEVICE / SKIP_DISABLED |
| 服务商消息 | `MsgID` | msg_id | JPush msg_id；日志通道为 `log` |
| 失败原因 | `Error` | error | FAILED 时填 |
| 时间 | `CreatedAt` | created_at | — |

> 无独立 API；后台查看页二期评估（现经 SQL/留痕表审计）。

### 1.6.8 client_crash_logs（客户端崩溃日志，迁移 000136）

App 本地留痕后启动补传；服务端入库即视为成功，App 端成功即删本地文件防重传（尽力端端）。日志文本服务端按 64KB 截断保尾部堆栈。

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| ID | `ID` | id | BIGSERIAL |
| 主体类型 | `SubjectType` | subject_type | 默认 worker（当前仅师傅端上报） |
| 主体 ID | `SubjectID` | subject_id | BIGINT；未登录=0 |
| App 标识 | `App` | app | 例 `boss-worker/0.1.0` |
| 崩溃日志 | `Log` | log | 服务端截断 64KB 保尾部 |
| 时间 | `CreatedAt` | created_at | — |

> 上报接口：`POST /api/worker/v1/client/crash`（worker/misc.yaml，wauth）；管理端查询 `GET /api/admin/v1/crash-logs?limit=N`（admin/sys.yaml，permCode `menu:crashlogs`，迁移 000140）。

### 1.7 api_keys（免登录 API key，internal/domain/apikey，迁移 000042/000043/000045）

> 固定用途：CLI/自动化（bossctl）免登录认证。key 与三类主体绑定（`subject_type` account/worker/customer，000043 三表登录边界 + 000044 主体扩展）；只存 sha256(key) 哈希，明文仅创建时返回一次（安全约定见迁移 000042 头注）。

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| — | `ID` | id | BIGSERIAL PK |
| 名称 | `Name` | name | 用途说明，如 ci-pipeline / bossctl-local |
| — | `KeyHash` | key_hash | CHAR(64) UNIQUE，sha256 十六进制，永不落明文 |
| 状态 | `Status` | status | 1启用 / 0停用（停用即级联禁用该主体全部 key） |
| 主体 | `SubjectType` / `SubjectRef` | subject_type / subject_ref | account→accounts.id / worker→workers.id / customer→customers.id |
| — | `LastUsedAt` | last_used_at | 供审计/巡检 |
| — | `ExpiresAt` | expires_at | 空=永不过期 |
| 权限模板 | `TemplateCode` | template_code | 空=完整继承主体 RBAC；非空=仅允许模板权限子集 |

`api_key_permission_templates` 与 `api_key_template_permissions` 提供受限 API key 权限模板；模板由平台维护，签发时仅引用 code，不保存明文密钥。`Lookup` 将模板码带入认证上下文，RBAC 门禁对受限 key 仅放行模板列出的权限；伙伴订单只读 key 使用 `partner-orders-read` 模板。

### 1.8 partner_commission_ledger（渠道佣金结算台账，000118）

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:---------|:------|:---------|
| 订单 | `OrderID` | order_id | BIGINT → orders |
| 渠道企业 | `LegalEntityID` | legal_entity_id | BIGINT → legal_entities |
| 订单金额 | `OrderAmount` | order_amount | NUMERIC(18,2)，下单金额快照 |
| 佣金比例 | `CommissionRate` | commission_rate | 0~1 |
| 佣金金额 | `CommissionAmount` | commission_amount | 订单金额×比例，保留两位 |
| 状态 | `Status` | status | ACCRUED / SETTLED / VOID |
| 结算时间 | `SettledAt` | settled_at | 已结算时填写 |
| 结算人 | `SettledBy` | settled_by | → accounts |

> 仅覆盖渠道企业订单的佣金台账与结算，不建设跨运营商批发结算或融资。

### 1.9 伙伴区域权限

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:---------|:------|:---------|
| 区域路径 | `RegionPath` | accounts.region_scope | LTREE；空=本企业全区域，非空=该区域子树 |
| 所属企业 | `LegalEntityID` | accounts.legal_entity_id | 伙伴企业租户边界 |

接口 `GET/PUT /partner/region-scope` 仅允许伙伴账号访问；设置值必须属于本企业覆盖区域，订单归属仍以地址推导结果为权威。

伙伴员工和企业订单查询都按当前账号 `region_scope` 使用 LTREE 子树过滤；空范围表示本企业全区域。

### 1.8 开放平台（open_apps / open_webhook_subscriptions / open_usage_day，internal/domain/openplat，迁移 000122）

> 固定用途：Q4 开放平台与互操作（docs/plan/q4-open-platform-plan.md）。外部集成方应用凭证 AppId+Secret，HMAC-SHA256 请求签名验签（区别于 1.7 的内部 bearer key：Secret 需原文落库，决策见 adopted note 2026-08-22-open-platform-secret）；开放面前缀 /api/open/v1，只读或经内部服务校验，禁止直写核心事实表。

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| — | `ID` | open_apps.id | BIGSERIAL PK |
| AppID | `AppID` | app_id | op_<16hex>，公开标识，UNIQUE |
| — | `Secret` | secret | ops_<32hex>，仅创建时返回一次 |
| 名称 | `Name` | name | 集成方用途说明 |
| 状态 | `Status` | status | 1启用 / 0停用（停用即验签失效） |
| 限流 | `RateLimitRPM` | rate_limit_rpm | 每分钟调用上限，缺省 60 |
| 配额 | `DailyQuota` | daily_quota | 日调用配额，缺省 10000（open_usage_day 计数） |
| 沙箱 | `Sandbox` | sandbox | true=沙箱应用（M4 沙箱环境） |
| — | `LastUsedAt` | last_used_at | 供审计/巡检 |
| 订阅事件 | `EventType` / `EventTypes` | open_webhook_subscriptions.event_type | 如 order.stage.done；同 app+事件+端点唯一。新增订阅请求 `eventTypes[]` 批量（一个端点订阅多事件=多行，单语句原子写入、重复幂等跳过，≤32 个/次），单数 `eventType` 保留兼容；目录接口 `GET /openplat/event-types`（登记制：emit 侧落地后在 internal/domain/openplat/events.go 登记，未登记不对外） |
| 回调端点 | `EndpointURL` | endpoint_url | HTTPS 回调地址（M2 投递器消费） |
| 投递状态 | `Status` | open_webhook_deliveries.status | 0待投递 / 1已投递 / 2死信（超过 6 次重试，迁移 000125） |
| 幂等键 | `EventID` | open_webhook_deliveries.event_id | 同订阅+事件唯一（UNIQUE + DO NOTHING），重放不重复执行业务动作 |
| 重试次数 | `Attempts` | attempts | 失败按 30s×2^n 指数退避（封顶 1h），`NextAttemptAt` 排下次 |
| 投递结果 | `HTTPStatus` / `LastError` | http_status / last_error | 2xx 成功；非 2xx/网络错误记错误进重试 |

### 1.6.9 stripe.* 支付配置（页面 `/base/stripeconfig`「支付配置」，迁移 000147）

存储复用 `biz_params`（key 前缀 `stripe.`，与 auth.*/realid.*/minio.* 同一套加密/掩码约定）。运行时由 `stripe.Dynamic` 消费（60s 热生效；`internal/app/wiring_stripe.go`）。**2026-09-05 裁定：env 兜底 `BOSS_STRIPE_*` 移除，凭据仅存 DB**（docker-compose 不再注入；App 端支付方式/师傅端现场收款方式也以本配置为准）；未启用/缺 `stripe.apiKey` → 发起端点 400、webhook 503、支付方式默认线下（"密钥未配即降级"裁定）。

| 页面字段 | key（API/DB 同名） | 枚举/说明 |
|:--------|:-------------------|:----------|
| 启用开关 | `stripe.enabled` | true/false，默认 true |
| Secret Key | `stripe.apiKey` | sk_ 开头,secret,密文落库;空串=不修改 |
| Publishable Key | `stripe.publishableKey` | pk_ 开头,前端 Stripe.js 预留(托管收银台不需要) |
| 记账币种 | `stripe.currency` | ISO 小写三字码,默认 php |
| API 地址覆盖 | `stripe.apiBaseUrl` | 测试/代理用,空=官方 api.stripe.com |
| Webhook 签名密钥 | `stripe.webhookSecret` | whsec_ 开头,secret,密文落库;空串=不修改 |
| 期望回调 URL | `stripe.webhookUrl` | 隧道快速 URL + `/api/user/v1/webhooks/stripe`,自愈循环比对源 |

> 接口：`GET /stripe-config`、`PUT /stripe-config/{channel|webhook}`、`POST /stripe-config/channel/test`（完整性校验跨两组,草稿跨 channel/webhook 组合并——webhookSecret 未保存即可参与;通过后真实探活余额 + 核对后台 webhook endpoint 与期望 URL 一致性,零副作用;permCode `menu:stripeconfig`,迁移 000147,openapi sys.yaml）。env 兜底已于 2026-09-05 移除（同 commit 清理 config.Stripe 结构 + deployments 环境变量；102 测试账号凭据需在配置页重新录入）。配置消费方：用户端 `GET /payments/methods`（stripe 可用才下发银行卡）、师傅端 `GET /tickets/:no/charge` payMethods（可用才含 CARD）。

## 2. 阶段2 · 客户与资费（internal/domain/customer）

### 2.1 customers（普通用户/客户主体，源自 customer.html）

> 固定用途：普通个人/企业客户及其客户 App 登录主体。客户下单、查询订单、缴费、报障等均以本表客户身份为准；不得使用 `accounts` 登录后台。

| 页面列名 | 字段名 | DB 列（约定） | 枚举/说明 |
|:---------|:-------|:--------------|:----------|
| 客户 | `Name` | name | 个人姓名/企业名 |
| 联系电话 | `Phone` | phone | — |
| 证件类型 | `IdType` | id_type | 身份证/护照/营业执照/无 |
| 证件号码 | `IdNo` | id_no | — |
| 实名状态 | `RealNameStatus` | real_name_status | VERIFIED 已实名 / PENDING 待补登 |
| 服务状态 | `ServiceStatus` | service_status | ACTIVE 在网 / ARREARS 欠费 / SUSPENDED 停机 |
| 地址 | `AddressID` | address_id | BIGINT → addresses（挂接楼栋）；000176 起可空=建档时未登记，接口回 0；维护走档案页"地址"动作（POST /orders/address backfillCustomer=true） |
| — | `PasswordHash` | password_hash | TEXT，客户 App 密码哈希；空值不可密码登录 |
| 登录状态 | `AuthStatus` | auth_status | 1允许登录 / 0禁止登录 |
| 用户码 | `CustomerCode` | customer_code | VARCHAR(32) UNIQUE,前缀 `C-` 后 8 位 = `id` 左零;四码 `quad.customerCode` 展示字段,对账/扫码/外键仍以 `CustomerID` 为权威(adopted 2026-08-21) |

> 区域锚点（TS 实体）：`region_id`/`region_name`（地址所在经营区域），`legal_entity_id`（归属公司），按地区/企业统计客户；客户搬家/转品牌经 `customer_histories` 台账快照事发区域。

#### 2.1.1 用户详情聚合口径（admin `GET /users/{customerId}`，bss/user 详情抽屉，2026-09 收口）

> 详情页回答「现在是什么状态」，不展示「发生过哪些记录」；以下口径为权威，前端 detail-view.ts 与契约同步。

| 聚合段 | 页面语义 | 取数口径 | 说明 |
|:-------|:---------|:---------|:-----|
| `plans` | 当前在用套餐 | `user_plans` 仅 `upper(status)='ACTIVE'` | 对齐用户端 `portalHomePlan`（ACTIVE 优先）；无在用套餐则段为空；「当前套餐」芯片同源派生 |
| `faults` | 报障工单 | `complaints` 且 `type NOT LIKE '用户投诉:%'`；`type` 展示侧 strip「用户报障: 」前缀 | 双口径：用户端 `POST /faults`（`no_internet/slow/ont_fault/other`）+ 装维域故障码（SINGLE_OUTAGE 等，见 complaint-type-map.md §2 用户端口径）；strip 同 `portalFaultTypeLabelFromStored` |
| `complaints` | 投诉工单 | `complaints` 且 `type LIKE '用户投诉:%'` | 用户端 `POST /complaints` 落库前缀「用户投诉: 」，见 complaint_handlers.go |

> 变更背景：此前两段都直读全量 `complaints`（同源重复、语义重叠），收口后按 type 前缀区隔；`faults` 与 `complaints` 不再同时出现重复行。

### 2.2 资费三级模型（源自 product.html；V1.1 修正：不同公司/区域产品与价格不同）

| 实体 | 页面列名 | 字段名 | DB 列（约定） | 枚举/说明 |
|:-----|:---------|:-------|:--------------|:----------|
| product_offers(公司产品) | 所属公司 | `LegalEntityID` | legal_entity_id | BIGINT → legal_entities |
| | 产品名称 | `Name` | name | **公司级名称,必填**(各公司叫法不同) |
| | 分类 | `Category` | category | broadband/fusion/addon(用户端契约枚举,迁移 000066;空回退 broadband) |
| | 带宽 | `Bandwidth` | bandwidth | 如 300M/500M/1000M(公司自定) |
| | 基础月费 | `MonthlyFee` | monthly_fee | NUMERIC |
| | 生效时间 | `EffectiveAt` | effective_at | 上架/调价生效 |
| | 状态 | `Status` | status | DRAFT/PUBLISHED/OFFLINE |
| region_offers(区域运营包) | 区域 | `RegionPath` | region_path | LTREE，须落在该公司经营区域 |
| | 区域名称 | `Name` | name | 可空;展示名回退: 区域名→公司名 |
| | 区域月费 | `MonthlyFee` | monthly_fee | 生效价覆盖基础价 |
| orders(订单侧) | 成交价 | `PriceSnapshot` | price_snapshot | 下单时生效价快照 |
| offer_provision_bindings(套餐↔下发模板绑定,000173) | 下发模板 | `TemplateID` | template_id | BIGINT → provision_templates;`offer_id` UNIQUE 一套餐一模板 |
| | 备注 | `Remark` | remark | 可空;冗余 legal_entity_id 企业锚点 |

> 计价规则：生效价 = 区域价(前缀匹配) ?? 公司基础价；展示名两级回退；订单只存快照。
> 约束（seed 已校验）：未经营区域不得设区域价、不得下单；账单金额 = 快照价。
> 开通绑定（方案B，adopted 2026-09-01-offer-provision-binding）：环节7 模板解析顺序
> ①显式绑定（本表,模板须 ENABLED）→ ②带宽兜底（同法人 ENABLED 模板 `content->>'bandwidth'`=套餐带宽,无带宽套餐不参与）
> → ③`TEMPLATE UNRESOLVED` 显性失败（禁止回退法人任意模板）。
> 无带宽套餐（IPTV/云存储等 addon）必须显式绑定模板,否则环节7 失败。

## 3. 阶段5 · 订单与计费（internal/domain/{order,billing}）

### 3.1 orders（订单，源自 order.html + 全案 4.2 Order）

| 页面列名 | 字段名 | DB 列（约定） | 枚举/说明 |
|:---------|:-------|:--------------|:----------|
| 订单号 | `OrderNo` | order_no | 如 ORD-20250817-001 |
| 客户 | `CustomerID` | customer_id | BIGINT → customers |
| 渠道 | `ChannelID` | channel_id | BIGINT → channels（REQ-ORD-006 必填不可改） |
| 产品 | `OfferID` | offer_id | BIGINT → product_offers |
| 地址 | `AddressID` | address_id | BIGINT → addresses；**归属判定源**（见下行） |
| 当前环节 | `Stage` | stage | 1~12（见 terms.md 第 1 节） |
| 状态 | `Status` | status | PENDING/RESERVED/INSTALLING/DONE（见 terms.md 第 3 节） |
| 区域 | `RegionPath` | region_path | LTREE；下单时由地址推导快照 |
| 归属公司 | `LegalEntityID` | legal_entity_id | BIGINT → legal_entities；由安装地址推导（address→region→最近覆盖祖先，migrations/000076），下单快照不可变；未匹配子公司覆盖时兜底平台总公司（is_platform，migrations/000077）；调用方直传值仅做冲突校验（adopted note 2026-08-20-order-legal-entity-by-address） |
| 成交价 | `PriceSnapshot` | price_snapshot | 下单时生效价快照（账单金额以此为准） |
| 付费方式 | `BillingMode` | billing_mode | PREPAID/POSTPAID（000102）；下单时客户选定快照，环节 6 建 LO 账号时继承到 lo_accounts；默认 POSTPAID（adopted note 2026-08-22-prepaid-postpaid-billing-mode） |
| 预缴月数 | `BuyMonths` | buy_months | 0~60（000104）；0=按月缴（环节 4 收 1 个月月费），N>0=预缴 N 月（环节 4 收 N×月费，区域覆盖口径同出账） |
| 赠送月数 | `GiftMonths` | gift_months | 0~60（000104）；环节 4 收款时按 gift_duration_rules 阶梯命中回填（如 6送1/12送3/24送6，取 ≤预缴月数的最大档），未命中为 0 |

> lo_accounts 同名列 `billing_mode`（000102）：订购关系上的付费模式权威态；PREPAID 客户不进月度出账（GenerateBills 过滤），预付费在环节 4 合同收费当场收款落缴费流水。
> LO 生效套餐对齐（adopted 2026-09-01-provision-correctness-followup）：`lo_accounts.offer_id`/`billing_mode` 是"当前生效套餐"权威态，环节 6 幂等复用已有 LO 时若与订单套餐不一致，自动对齐到订单套餐（TMF change order 语义）并打 `[order] LO OFFER REALIGN` 留痕——保证环节 7 按新套餐下发模板、RADIUS 按新档限速。

> 快照列（TS 实体）：`customer_name`（客户姓名）、`offer_name`（产品名），下单时冻结，改名/调价不影响历史订单（与 `price_snapshot` 同规则）。

> 渠道订单接口 `POST /partner/orders` 只接受伙伴账号身份，服务端从伙伴企业档案取得 `legal_entity_id`，再调用同一 `OrderService.Submit`；因此渠道订单复用直营订单的 12 环节、资源核查/端口预占和计费规则，不复制状态机。

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

### 3.4 invoices（发票，源自 billing 域 TAX/AG-04，CT-007 出账→开票）

| 页面列名 | 字段名 | DB 列（约定） | 枚举/说明 |
|:---------|:-------|:--------------|:----------|
| 发票号 | `InvoiceNo` | invoice_no | ARN 连续编号 `INV-00000001`，作废保留不回收（TAX-004） |
| 账单号 | `BillNo` | bill_no / bill_id | 同一账单仅一张在发票（部分唯一索引 `WHERE status='ISSUED'`） |
| 客户 | `CustomerID` | customer_id | `customer_name` 快照同 bills |
| 抬头 | `Title` | title | 默认同客户名 |
| 净额 | `NetAmount` | net_amount | =账单金额（不含税） |
| 税率 | `VatRate` | vat_rate | 默认 0.12（TAX-001） |
| 税额 | `VatAmount` | vat_amount | = ROUND(net×rate, 2)（GEN-006） |
| 合计 | `TotalAmount` | total_amount | = 净额 + 税额（TAX-002） |
| 状态 | `Status` | status | ISSUED 已生成 / VOIDED 已作废（`void_reason`/`voided_at` 留痕） |
| 税务属地 | `TaxJurisdiction` | tax_jurisdiction | CN 中国数电票 / PH 菲律宾 BIR / 空=未定；开票时经 bills.legal_entity_id 从法人快照（000109） |
| 税务通道 | `TaxChannel` | tax_channel | manual 人工回填 / leqi（预留）/ bir_eis（预留） |
| 税务状态 | `TaxStatus` | tax_status | PENDING 待开具 / SUBMITTED 已提交 / ISSUED 已开具 / FAILED 失败 / BLOCKED 外部资质或凭据不可用（`tax_fail_reason` 留痕；BLOCKED 不得伪造成功，可人工回填或资质就绪后重试） |
| 税局票号 | `TaxNo` | tax_no | CN 数电票 20 位 / PH BIR 回执号；回填后方为有效票据 |
| 外部回执标识 | `ExternalID` | invoice_tax_events.external_id | 外部请求/回执标识；同一发票重复回执唯一幂等，乱序回执不得回退已 ISSUED |
| 税务轨迹 | `TaxEvents` | invoice_tax_events | RECEIPT/BACKFILL/VOID/REISSUE；按 created_at、id 正序查询；失败详情、重试、回放经 `/invoices/{id}/tax-failure|tax-retry|tax-replay` |

> ARN 发号：`arn_sequences` 计数表（`doc_type` INVOICE/RECEIPT 各一序列），事务内 `UPDATE..RETURNING` 原子占号、行锁串行、回滚号回退（决策 note：2026-08-18-tax-invoice-arn-numbering）。链路：收款 `POST /payments`（流水+账单**条件**置 PAID 同事务,000167 起）→ 出账+自动开票 `POST /billing-runs`（幂等，失败账单入 `failedIds`）→ 作废/重开 `POST /invoices/:id/{void,reissue}`。

### 3.5 payments（缴费流水，源自 payment.html；000068 双挂改版）

| 页面列名 | 字段名 | DB 列（约定） | 枚举/说明 |
|:---------|:-------|:--------------|:----------|
| 流水号 | `PayNo` | pay_no | — |
| 客户 | `CustomerID` | customer_id | BIGINT → customers（000068 新增硬 FK，冗余直挂） |
| 账单号 | `BillID` | bill_id | BIGINT → bills（000068 起**可空**：充值/预存无账单） |
| 金额 | `Amount` | amount | NUMERIC |
| 方式 | `Method` | method | wechat/alipay/card/cash/offline（见 terms.md 第 4 节；offline=线下收款,师傅现场 CASH/QR/POS 统一记 offline,2026-09-05;柜面归类按资金通道 2026-08-28:现金 cash/扫码 wechat|alipay/POS card） |
| 状态 | `Status` | status | SUCCESS/FAILED/REFUNDED |
| 网点 | `SiteName` | site_name | 000167 柜面凭证要素;可空(线上渠道为空) |
| 柜台/班次 | `CounterCode` | counter_code | 000167 柜面凭证要素;可空 |
| 操作员 | `OperatorName` | operator_name | 000167 柜面凭证要素;服务端取登录态,前端不传;可空 |
| 退款原因 | `RefundReason` | refund_reason | 000112 全额退款留痕,未退为空 |
| 退款时间 | `RefundedAt` | refunded_at | 000112;可空,退款时落 now() |

> **门户缴费记录口径**(2026-09-03 收口):用户端 `GET /api/user/v1/payments` 默认仅返回
> `Status=SUCCESS`,FAILED 仅留痕排查、REFUNDED 由财务冲账——都不向终端用户默认展示。
> 调试或工单排查经 `?include=failed,refunded` 显式带回,`status` 字段原样回传以便前端
> 识别被过滤项。admin `GET /api/admin/v1/payments` 不在此限,继续全量回传(财务审计需要)。

> 000068 起 payments 同时挂 `bill_id`(可空) 与 `customer_id`：账单缴费走 bill，充值类流水仅挂 customer；
> 存量行已回填 customer_id（取 bill.customer_id）。2026-08-30 收口：落账源头
> `RecordPaymentWithCoupon` 强制双挂——账单流水按 bill 回填 customer_id、显式传错客户拒收，
> 迁移 000148 补清后增空行。
> 缴费成功自动复机（Q3）：SUCCESS 落账后若客户 LO 账号 SUSPENDED 即自动 RESUME（迁移+`stop_resume_tasks` RESUME 流水留痕；
> 失败任务留 FAILED 经 `POST /stop-resume-tasks/:id/retry` 重试），入口覆盖 admin 收款/门户缴费/门户续费/Stripe webhook。
> 欠费催收批处理（Q3）：`POST /dunning-runs`（graceDays 宽限/stopAfterDays 停机线，缺省 15/30）→ 宽限外 UNPAID 账单置
> OVERDUE → `arrears` 快照更新（COLLECTING/STOPPED）→ 超停机线 LO 自动 STOP+流水留痕；账龄自 `bills.created_at` 起算。

> **柜面收款与柜台日结**（000167，纪要 2026-08-28-柜面现金收款）：admin `POST /payments`
> 权限码拆为 `menu:payment:cash`（按钮级写操作，ops 不默认授予，不相容岗位分离；看流水
> 仍 `menu:payment`）；method 白名单外 42200 拒收；cash 单笔限额 `biz_params/payment.cash.singleLimit`
> （元,热调,0/缺失不拦）超限拒绝+审计留痕；`payment.record` 审计 detail 补金额/方式/客户/网点。
> 账单置 PAID 改**条件迁移**：累计实收（按分）≥应收才置，未收齐维持原状态（ledger_recon 金额三角
> 自然呈现 PARTIAL）。柜台日结（T+0 只读汇总+实点回填）：`GET /daily-closings/summary|items`
> （perm `menu:daily-close`，收入/退款分列，退款按流水发生日归属）、`POST /daily-closings`
> （perm `menu:payment:cash`，实点回填 UPSERT `payment_daily_closings`，不平输出 `[paycheck] DIFF`
> 可 grep 日志附网点/操作员上下文）。柜面 POS 收单 manual 渠道源登记列入 paycheck 二期配套。

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
| PON 框号 | `PONFrame` | pon_frame | SMALLINT 可空；NULL=未分配（000178） |
| PON 槽号 | `PONSlot` | pon_slot | SMALLINT 可空；NULL=未分配（000178） |
| PON 口号 | `PONPort` | pon_port | SMALLINT 可空；NULL=未分配（000178） |
| ONUNO | `ONUNO` | onu_no | SMALLINT 可空；NULL=未分配（000178） |

> 状态变更历史（TS 实体）：`port_change_history`，端口每次状态/占用变化一行（变更后 status + order_id 快照 + changed_at），历史不随当前状态漂移。
> 区域/企业锚点（TS 实体）：`region_id`/`region_name`（地址所在经营区域）、`legal_entity_id`/`legal_entity_name`（所属设备企业），按地区/企业统计端口；`lo_accounts` 同挂 `region_id`/`region_name`（客户所在经营区域）。

### 4.2.1 stocktakes / stocktake_items（盘点任务与差异明细，迁移 000007/000156，S10 流程）

stocktakes（盘点任务，页面 `/ams/stock`「盘点管理」）：

| 页面列名 | 字段名 | DB 列（约定） | 枚举/说明 |
|:---------|:-------|:--------------|:----------|
| 盘点任务 | `ID` | id | BIGSERIAL PK |
| 所属公司 | `LegalEntityID` | legal_entity_id | BIGINT → legal_entities（建单快照范围） |
| 范围 | `Scope` | scope | `全库`/空=主体全部资产;否则 region_name 精确匹配 |
| 进度 | `Progress` | progress | SMALLINT 0~100,实扫/计划快照行,扫码自动重算 |
| 差异项 | `DiffCount` | diff_count | MISMATCH/MISSING/EXTRA 行数 |
| 状态 | `Status` | status | DOING/DONE（见 terms.md 第 4 节;存在未处置差异禁止 DONE） |

stocktake_items（盘点差异明细，建单冻结快照 + 扫码回填 + 逐条处置）：

| 字段名 | DB 列 | 枚举/说明 |
|:-------|:------|:----------|
| `TaskID` | task_id | BIGINT → stocktakes |
| `AssetID` | asset_id | BIGINT 软引用 assets,(task_id,asset_id) 唯一 |
| `ExpectedStatus` | expected_status | 建单时资产状态快照;NULL=计划外(EXTRA 行) |
| `ScannedStatus` | scanned_status | 实盘所见状态;NULL=未扫 |
| `Kind` | kind | PENDING/OK/MISMATCH/MISSING/EXTRA(关单时未扫置 MISSING) |
| `Resolution` | resolution | OPEN/CONFIRMED/FIXED/ESCALATED(确认/修正/上报) |
| `HandledBy/HandledAt` | handled_by / handled_at | 处置人账号/时间 |
| `Note` | note | 处理说明(FIX/ESCALATE 必填,≤255) |

> 流程口径（S10）：建任务→扫码回填（预期行判 OK/MISMATCH、计划外行记 EXTRA、进度自动重算）→差异逐条处置
> （CONFIRM=MISMATCH 时按实盘修正 assets.status 并落 asset_lifecycles；FIX=台账为准；ESCALATE=上报转人工）→全处置完才可关单
> （`POST /stocktakes/{taskId}/diff-handle`,存在 OPEN 差异返回 40900）。

### 4.3 replacements（换新单/设备更换单，000007 + 000159 派单三列）

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 更换单 | `ReplacementNo` | replacement_no | RPL-YYYYMMDD-序列，后端生成 |
| 设备 | `AssetID` | asset_id | BIGINT 软引用 assets（被更换资产） |
| 原因 | `Reason` | reason | 如 光猫故障 |
| 优先级 | `Priority` | priority | HIGH/MEDIUM/LOW |
| 状态 | `Status` | status | PENDING/DOING/DONE/FAILED（见 terms.md 第 4 节） |
| 派单师傅 | `WorkerID`/`WorkerName` | worker_id/worker_name | 000159；worker_id FK→workers，name 快照（0/空=未派） |
| 完成时间 | `FinishedAt` | finished_at | TIMESTAMPTZ 可空；DONE/FAILED 时回填 |

> 状态机：PENDING --assign(派单,POST /admin/replacements/{id}/assign)→ DOING --complete(师傅端 POST /api/worker/v1/replacements/{id}/complete)→ DONE/FAILED；终态不可再流转（adopted note 2026-08-27-replacement-ticket-flow）。
> 完成时落 `worker_replace_logs`（ticket_no=更换单号展示快照，dispatch_ticket_id=0）+ 资产联动：旧件→MAINTENANCE、新件（newEpc 反解）→DEPLOYED，各留 `asset_lifecycles`。
> 企业锚点（fields.md §8.1）：`legal_entity_id`/`legal_entity_name` 建单时自资产主档回填。

### 4.4 provision（配置下发 · TL1/网管北向，迁移 000013 基表 + 000178 增补）

> 配置下发 PROV 域基表 `provision_templates`/`provision_tasks`/`provision_logs` 由迁移 000013 建立（下发模板/任务/日志）。
> 迁移 000178（TL1 对接，设计 §8）为 NMS 端点与 PON 定位新增以下表/列，供 T5 app resolver 取数（pass_cipher 加解密属 T5）。

`resources` 增列（迁移 000178）：

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| NMS OLTID | `NMSOLTID` | nms_oltid | VARCHAR(128) 可空；OLT 行的 U2000 侧标识（缺=FAIL "OLT missing nms_oltid"） |

`provision_nms`（法人级 NMS 端点，一法人一行）：

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 法人 | `LegalEntityID` | legal_entity_id | BIGINT NOT NULL UNIQUE → legal_entities |
| 主机 | `Host` | host | VARCHAR(128) NOT NULL |
| 端口 | `Port` | port | INTEGER NOT NULL DEFAULT 13027 |
| 协议 | `Protocol` | protocol | VARCHAR(8) NOT NULL DEFAULT 'tcp'（P2 扩 ssl） |
| 用户名 | `Username` | username | VARCHAR(32) NOT NULL |
| 口令密文 | `PassCipher` | pass_cipher | TEXT NOT NULL；复用 config_secrets 加解密 helpers |
| 创建/更新 | `CreatedAt`/`UpdatedAt` | created_at/updated_at | TIMESTAMPTZ DEFAULT now() |

`pon_onu_alloc`（每 (OLT,PON) 的 ONUNO 分配器，幂等自增）：

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| OLT 资源 | `OLTResourceID` | olt_resource_id | BIGINT NOT NULL → resources(id)；PK 四列联合首列 |
| PON 框/槽/口 | `PONFrame`/`PONSlot`/`PONPort` | pon_frame/pon_slot/pon_port | SMALLINT NOT NULL，与 olt_resource_id 联合主键 |
| 下一序号 | `NextNo` | next_no | SMALLINT NOT NULL DEFAULT 0；分配器首 0 其后 1,2,... |

> PONID 组装：`PONID="NA-<pon_frame>-<pon_slot>-<pon_port>"`（设计 §4）；`ports` 的 PON 四列与 `onu_no`（NULL=未分配）承载装维定位（§4.2 已增列）。

## 5. 阶段6 · 四码合一（internal/domain/quadlink）

### 5.1 quad_link（四码关联，源自全案 4.2 + REQ-AMS-003）

| 字段名 | DB 列 | 枚举/说明 |
|:-------|:------|:----------|
| `AssetID` | asset_id | BIGINT → assets（000086 起可空：纯端口链路/资产注销场景） |
| `CustomerID` | customer_id | BIGINT → customers（四码第 2 项=客户，非系统账号 user；000088 起一客户可多链路） |
| `PortID` | port_id | BIGINT → ports |
| `AddressID` | address_id | BIGINT → addresses |
| 状态 | `status` | LINKED/CONFLICT/UNLINKED（见 terms.md 第 4 节） |

> 四列各建索引，任一码反查单表索引。**唯一约束演进**（以 migrations 为权威）：000056 全表唯一改
> `WHERE status IN('LINKED','CONFLICT')` 部分唯一 → 000086 四列各改 `WHERE xxx IS NOT NULL` 部分唯一（asset 可空）
> → 000088 customer 列降为普通索引（一客户多链路）；asset/port/address 各保持至多一条非空活跃链路。
> **口径裁定**：四码=资产-客户-端口-地址（全案 REQ-AMS-003/REQ-CONS-002 权威）。
> 技术栈方案 3.3（docs/archive/技术栈方案-一步到位.md）原文写 `quad_link(asset_id, user_id, port_id, addr_id)`，`user_id` 系笔误，
> 应为 `customer_id`（`user` 是系统账号域，`customer` 是客户域，二者不同，见 domain-map）。

> **展示冗余**：四码对外 API 同时返回 `assetCode/customerCode/portCode/addrCode`，其中 `customerCode` 取 `customers.customer_code`（adopted 2026-08-21），
> 与 `customer_id` 是 ID ↔ code 双向冗余关系；扫码/对账/外键一律以 ID 为准，code 仅供师傅现场人工口头核验与 UI 展示。

## 6. 跨域通用列（所有实体强制）

| 列 | 类型 | 说明 |
|:---|:-----|:-----|
| `id` | BIGSERIAL PK | 自增主键 |
| `created_at` | TIMESTAMPTZ | 默认 now() |
| `updated_at` | TIMESTAMPTZ | 有变更流的实体才加 |
| `brand_id` / `region_path` | — | 品牌区域横切维度，主数据实体按需带 |

## 7. 师傅域（cross-domain worker，Go 实体）

> 本节实体已迁移至 Go 实体，字段名用 Go struct 字段名，DB 列经 snake_case 命名。

### 7.1 worker_groups / workers（班组·师傅）

> UI 术语裁定（000141）：admin「师傅管理」页与师傅端以**装维队**方式展现班组，**组长**称**队长**；
> 实体/DB 列不变（worker_groups / leader_id），仅展示文案层映射。

`worker_groups`（班组，UI=装维队）：

| 字段名(TS实体) | DB 列 | 枚举/说明 |
|:---------|:------|:----------|
| `code` | code | 班组编码，公司内唯一（稳定标识，name 可改 code 不变） |
| `name` | name | 班组名称（可改名） |
| `legalEntity` | legal_entity_id | BIGINT → legal_entities |
| `leader` | leader_id | BIGINT → workers（组长/UI=队长，可空；000141 起仅允许本队在职成员） |
| `leaderName` | leader_name | 组长姓名快照 |
| `memberCount` | —（子查询） | 在职成员数，仅列表视图冗余（000141） |
| — | deleted_at | 软删（000141 增列）：null=在职队；仍有在职成员时禁止软删（40900） |

> 装维队管理端点（000141，admin `/api/admin/v1`，perm `menu:order`）：
> `PUT /worker-groups/{id}`（改名+指定队长）、`DELETE /worker-groups/{id}`（软删）、
> `GET /worker-groups/{id}/performance?period=`（成员月度业绩）、
> `POST /workers/{id}/transfer`（调队，落 §7.2 台账）。
> 师傅端 `GET /api/worker/v1/team/performance?period=`：仅队长可见，非队长 40300。

`workers`（安装师傅/师傅端用户）：

> 固定用途：仅承载上门安装师傅及师傅端登录主体。师傅端登录名 = `phone`（手机号，worker/auth.yaml `required:[phone,mode]`），凭 `passwordHash` 以 `mode=password` 登录（Amended 2026-09-01：旧文"staffNo 作为登录名"与 worker/auth.yaml、师傅端 App 实现不符，按实现现况修正；staffNo 仅为工号标识）；仅 `status=1`（在职）允许登录。不得使用 `accounts` 登录后台。
> 登录密码管理（2026-09-01）：admin `POST /workers` 录入师傅必带 `password`（bcrypt 落库）；`PUT /workers/{workerId}/password` 重置；`passwordHash` 为空不可密码登录（仅验证码模式）。

| 字段名(TS实体) | DB 列 | 枚举/说明 |
|:---------|:------|:----------|
| `staffNo` | staff_no | 工号，如 WK-1024（唯一，页面展示/检索标识） |
| `passwordHash` | password_hash | 师傅端密码哈希，仅存哈希，不存明文，可空（首次设置前不可密码登录） |
| `name` | name | 师傅姓名 |
| `group` | group_id | BIGINT → worker_groups（当前归属，可变更） |
| `regionId` | region_id | 主区域（第一负责区域；000175 起为 regionIds 首位镜像，兼容派单快照/月度统计链路），须落班组公司经营区域 |
| `regionIds` | worker_regions（§7.1a） | 全部负责区域（000175 多区域；列表/详情回填，主区域首位；未配置时回退 [regionId]） |
| `phone` | phone | 联系电话（列表/详情脱敏展示；师傅端登录名） |
| `status` | status | 1在职 / 0离职（terms.md §4 登记；师傅详情/列表同口径） |
| `joinedAt` | joined_at | 入职时间 |
| `leftAt` | left_at | 离职时间，null=在职 |

### 7.1a worker_regions（师傅负责区域，迁移 000175）

> 一个师傅可配置多个负责区域（2026-09-01 后台需求）；`workers.region_id` 保留为主区域（第一负责区域），
> 扩展区域落本表；存量数据由迁移种子（region_id > 0 → 本表一行）无缝衔接。

| 字段名(TS实体) | DB 列 | 枚举/说明 |
|:---------|:------|:----------|
| `workerId` | worker_id | BIGINT → workers（级联删除） |
| `regionId` | region_id | INTEGER → regions；PK(worker_id, region_id) |

> 端点（admin，perm `menu:dispatch`，worker.yaml）：
> `POST /workers` 录入支持 `regionIds` 数组（首位为主区域；与单值 `regionId` 兼容路径二者至少其一）；
> `PUT /workers/{workerId}/regions` 覆盖式配置（regionIds 首位为主区域，回写 workers.region_id；
> 空集合=仅清空扩展区域、主区域保留；区域不存在 404/40900）。
> 匹配口径：工单区域 ∈ 师傅负责区域集合（主区域 ∪ 扩展区域）即放行——师傅端任务池/抢单/转单目标闸门
> 与 admin 派单/转派跨区 40900 闸门统一走 `Worker.MatchesRegion`；任一方区域缺失（0）仍视为不限区域。

> 师傅详情（admin `GET /workers/{workerId}`，worker.yaml）：后端主档单条返回上述全字段；
> admin 前端详情抽屉展示主档 + 关联子集（工单/绩效/消息/评价，各取第一段，超限折叠）。
> 关联子取数口径见 `api/openapi/admin/worker.yaml` 各 `/worker-*` 端点；`worker_settings` 无 GET 只读路由（仅 PUT），详情不展示接单设置。

### 7.2 worker_locations（师傅实时位置，迁移 000144）

位置上报只保存师傅端身份对应的实时轨迹，当前点按 `reportedAt` 倒序取最新；订单详情通过派单工单的 `workerId` 关联读取。

| 页面列名 | API 字段 | DB 列 | 类型/说明 |
|:---------|:---------|:------|:----------|
| 师傅 | `workerId` | worker_id | BIGINT → workers |
| 纬度 | `lat` | lat | WGS84，-90~90 |
| 经度 | `lng` | lng | WGS84，-180~180 |
| 定位精度 | `accuracyM` | accuracy_m | 米，>=0 |
| 速度 | `speedMps` | speed_mps | 米/秒，>=0 |
| 航向 | `bearing` | bearing | 度，0~<360 |
| 上报时间 | `reportedAt` | reported_at | TIMESTAMPTZ |

> Worker API：`POST /api/worker/v1/location/report`；后台查询 `GET /api/admin/v1/workers/{workerId}/location`；订单详情返回 `latestLocation`。位置服务不信任请求体中的 workerId，以 JWT/API key 主体为准。

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
4. **归属口径（事发时）**：每单/每评价按发生那一刻的班组归属；工单跨班组时记「派单时班组」，评价跟随工单。事件级事实表 `worker_materials`/`worker_tools`/`worker_asset_returns` 的 `group_id` 同口径 = 记录创建（领用/退回/发放发生）那一刻师傅所在班组；事后调组不回改历史行（db-design-review D8）。
5. `worker_settings`（当前接单设置）与 `worker_messages`（站内通知）不加快照，跟随当前班组。
6. **姓名快照**：`dispatch_tickets`/`worker_feedbacks` 另存 `worker_name` 快照，师傅改名不改历史工单/评价。

### 7.4 worker_messages / worker_notices（消息中心页 `/boss/message`）

消息级别统一 `INFO/WARN/URGENT`（见 terms.md「消息 level」）。

**worker_messages（师傅消息页签）**

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 级别 | `level` | level | INFO/WARN/URGENT |
| 师傅 | `workerId` | worker_id | BIGINT → workers |
| 标题 | `title` | title | 下发时必填 |
| 内容 | `content` | content | |
| 时间 | `sentAt` | sent_at | |
| 状态 | `read` | read | 已读/未读 |

**worker_notices（公告管理页签）**

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 标题 | `title` | title | 发布时必填 |
| 分类 | `category` | category | |
| 状态 | `active` | active | true=上架(师傅端可见)/false=已下架 |
| 发布时间 | `publishedAt` | published_at | |
| 操作 | — | — | 上下架切换 `PUT /notices/{id}/toggle` |

**admin_notifications（后台提醒页签,迁移 000090）**

广播+读回执模型:主体面向角色广播,已读状态在 `admin_notification_reads` 按账号记录;`resolved` 由来源域驱动(todo 生命周期),与账号视角的 `read` 分离。API:`GET /notifications`、`GET /notifications/unread-count`、`POST /notifications/read`。

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 级别 | `level` | level | INFO/WARN/URGENT |
| 标题 | `title` | title | 未读加粗;读状态 join reads |
| 类型 | `category` | category | todo=待办/task=任务结果 |
| 时间 | `createdAt` | created_at | |
| 状态 | `read`/`resolved` | reads/resolved | resolved 行灰显且不计未读 |
| 操作 | — | — | 「去处理」跳 `link`(menu.def 路由);已读调 `POST /notifications/read` |

`ref_type` 来源域:importer/worker_reg/provision/report/billing/complaint(幂等键 ref_type+ref_id+category)。

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

## 8A. 对齐三端页面补齐的实体（Go 实体）

> 为消除「页面有列、实体缺失」的缺口补的实体，字段名沿用 Go struct 字段名，DB 列经 snake_case 命名。

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
| report_snapshots | admin/report.html 报告中心 | period/window_start/window_end/payload（唯一键 `(period, window_start)`，同窗口幂等覆盖；payload=派生聚合全文） |
| reports_trend（派生） | admin/report.html trend 曲线 | GET /reports/history?period&limit(默认 12 上限 90)→ items[].{windowStart, payload}(数字孪生 commit B1 后端 / ReportService.History;前端 commit B6 LineTrend SVG 折线消费,选周期 + limit 个历史快照的真趋势曲线) |
| gis_points（派生） | web/admin/src/pages/intel/gis/index.tsx PGIS 真地图点位 | level/parentId/bbox → id/level/lng/lat/status/count/parentId（PGIS 数字孪生 commit 1 后端 / GIS 域 Points 服务；前端 OL `gisPoints/items` 消费） |
| gis_map_tile(前端 OL 配置) | web/admin/src/components/business/maps/pgis-map.tsx PgisMap props | theme?: 'light'\|'dark'、tileUrl?: string（数字孪生 commit A3 + A7 落地；**默认高德栅格（GCJ-02，autonavi webrd）+ CartoDB Dark Matter**，实测于 `web/admin/src/components/business/maps/tile-source.ts` L12/L14/L32；tileserver-gl 自建 PMTiles（WGS84）就位后切 PMTILES_TILE_URL 时需统一坐标系） |

> 仍缺实体、但按 domain-map 属「待建域」或派生视图的页面（本期不臆造）：`settings.html`→`biz_params`（migrations 已建表，TS 实体已补 `BizParam`）、`paycheck.html`→渠道对账（派生聚合，非基表）。

### 7.5 worker_registrations / worker_real_name_verifications（师傅注册·审核·实名，新增 onboarding 子域）

> 迁移 000050。注册申请与正式 `workers` 主档解耦：待审核师傅不入 `workers`（登录主体仅限在职，见 §7.1）。

`worker_registrations`（师傅注册申请，对标 customer 自助建档）：

| 字段名 | DB 列 | 枚举/说明 |
|:------|:------|:----------|
| `id` | id | BIGSERIAL PK |
| `name` | name | 申报姓名 |
| `phone` | phone | 联系电话（脱敏） |
| `idCardNo` | id_card_no | 证件号（实名凭据，脱敏） |
| `group` | group_id | BIGINT → worker_groups（注册目标班组） |
| `regionId` | region_id | INTEGER → regions（服务区域） |
| `status` | status | **PENDING 待审核 / APPROVED 已通过 / REJECTED 已驳回**（terms.md 通用枚举延伸） |
| `reviewNote` | review_note | 审核意见（驳回必填） |
| `reviewerAccountId` | reviewer_account_id | BIGINT → accounts（审核人） |
| `workerId` | worker_id | BIGINT → workers（审核通过后建主档回填，空=未建） |
| `submittedAt` | submitted_at | 提交时间 |
| `reviewedAt` | reviewed_at | 审核时间，null=未审核 |

`worker_real_name_verifications`（师傅实名核验，对标客户 `real_name_verifications` §8A）：

| 字段名 | DB 列 | 枚举/说明 |
|:------|:------|:----------|
| `workerId` | worker_id | BIGINT → workers（1:1 当前态） |
| `method` | method | 人脸/证件OCR/人工/第三方 |
| `realName` | real_name | 申报实名（核验基准） |
| `idCardNo` | id_card_no | 证件号（脱敏） |
| `result` | result | **PENDING 待核验 / PASS 通过 / FAIL 不通过**（terms.md `result` 枚举延伸） |
| `verifiedAt` | verified_at | 核验时间 |
| `operatorAccountId` | operator_account_id | BIGINT → accounts（核验人） |
| `operatorName` | operator_name | 核验人姓名快照 |

### 7.6 customer_registrations / customer_real_name_verifications（客户注册·审核·实名，onboarding 子域延伸）

> 迁移 000051。注册申请与正式 `customers` 主档解耦：审核通过才建客户主档（下单需有效客户）；实名核验对标 §7.5 师傅子域。

`customer_registrations`（客户自助注册申请，公开端点提交，后台 menu:customer 审核）：

| 字段名 | DB 列 | 枚举/说明 |
|:------|:------|:----------|
| `id` | id | BIGSERIAL PK |
| `name` | name | 申报姓名 |
| `phone` | phone | 联系电话（脱敏） |
| `idCardNo` | id_card_no | 证件号（实名凭据，脱敏） |
| `legalEntityId` | legal_entity_id | BIGINT → legal_entities（归属运营主体） |
| `addressId` | address_id | BIGINT → addresses（装机地址） |
| `regionId` | region_id | INTEGER → regions（经营区域快照） |
| `status` | status | **PENDING 待审核 / APPROVED 已通过 / REJECTED 已驳回**（terms.md 通用枚举延伸） |
| `reviewNote` | review_note | 审核意见（驳回必填） |
| `reviewerAccountId` | reviewer_account_id | BIGINT → accounts（审核人） |
| `customerId` | customer_id | BIGINT → customers（审核通过后建主档回填，空=未建） |
| `submittedAt` | submitted_at | 提交时间 |
| `reviewedAt` | reviewed_at | 审核时间，null=未审核 |

`customer_real_name_verifications`（客户实名核验，与 `customers` 1:1 当前态；PASS 同步 `customers.real_name_status=VERIFIED`）：

| 字段名 | DB 列 | 枚举/说明 |
|:------|:------|:----------|
| `customerId` | customer_id | BIGINT → customers |
| `method` | method | 人脸/证件OCR/人工/第三方 |
| `realName` | real_name | 申报实名（核验基准） |
| `idCardNo` | id_card_no | 证件号（脱敏） |
| `result` | result | **PENDING 待核验 / PASS 通过 / FAIL 不通过**（terms.md `result` 枚举延伸） |
| `verifiedAt` | verified_at | 核验时间 |
| `operatorAccountId` | operator_account_id | BIGINT → accounts（核验人） |
| `operatorName` | operator_name | 核验人姓名快照 |
| `rejectReason` | reject_reason | 驳回原因（FAIL 时后台/二要素自动核验填写，000070） |
| `idCardFrontId` | id_card_front_id | BIGINT → attachments（证件人像面，0=未传，000070） |
| `idCardBackId` | id_card_back_id | BIGINT → attachments（证件国徽面，0=未传，000070） |

> 000059 起本表并入统一 `verifications`（subject_type='customer'）；000070 起新增上表三列。
> 2026-08-24 起支持阿里云二要素自动核验：通道配置后提交即判定，结论记录 operator_name=「阿里云二要素」、operator_account_id=0；通道未配置/调用失败保持 PENDING 走人工核验（adopted/2026-08-24-realid-channel-aliyun-cloudauth.md）。

### 7.7 实名审核中心（admin 页面 `/base/realname-review`，迁移 000149，菜单码 `menu:realname-review`）

> 横跨 customer/worker 两类主体的待办与历史轨迹聚合列表（既有 `/customers/:id/real-name/*`、`/workers/:workerId/real-name/*` 是单主体核验端点；本页面提供"批量扫一眼"的运营视角）。

`verifications`（`/verifications` 列表聚合，`subjectType`/`result`/`keyword` 过滤，`page`/`pageSize` 分页，最大 100/页）：

| 字段名 | DB 列 / 来源 | 说明 |
|:------|:-------------|:-----|
| `subjectType` | `verifications.subject_type` | `customer` \| `worker` |
| `subjectId` | `verifications.subject_id` | 软引用 customers.id / workers.id |
| `subjectName` | `customers.name` / `workers.name` | LEFT JOIN 快照，孤儿行回退空串 |
| `subjectPhone` | `customers.phone` / `workers.phone` | 同上 |
| `idCardNoMasked` | `verifications.id_card_no` | 后端按"首 4 尾 2 中间 \*\*\*"规则掩码 |
| `realName` | `verifications.real_name` | 申报实名（核验基准） |
| `method` | `verifications.method` | 人脸/证件OCR/人工/第三方/自助提交 |
| `result` | `verifications.result` | PENDING / PASS / FAIL |
| `rejectReason` | `verifications.reject_reason` | FAIL 时填写 |
| `verifiedAt` | `verifications.verified_at` | ISO8601 字符串 |
| `operatorName` | `verifications.operator_name` | 核验人姓名快照（"阿里云二要素" 等自动通道也算） |
| `idCardFrontId` / `idCardBackId` | `verifications.id_card_front_id` / `id_card_back_id` | 软引用 attachments.id（000070），0=未传 |

> 行内核验：`POST /verifications/:subjectType/:subjectId/verify` body `{result: PASS|FAIL, reason?}`。
> 仅作用于 PENDING 行；PASS 复用既有 `customers.real_name_status=VERIFIED` 同步与主档证件号一致性门禁（`internal/domain/customer/pg_onboarding.go:guardRealNameIdentity`）；师傅端不写 `real_name_status`（worker 主档无该字段）。
> 合成客户（负数段隔离空间 ID，无 customers 主档）PASS 放行：门禁无主档可对照即跳过，结论只落 verifications，主档同步为 0 行 no-op，用户端经核验单回退显示结论（adopted/2026-08-30-synthetic-customer-realname-boundary.md）；证件照上传同步仅拒 `uploaderId==0`，负数合成客户为合法上传者。
> 权限 `menu:realname-review`：sysadmin 全权；其余角色不授，与既有 `menu:realidconfig`（基础配置·实名核验配置）同级。
>
> 2026-08-26 流程补齐（通知闭环 + 后台代录）：
> - 待办通知：PENDING 落单（用户端自助 `POST /auth/verify`、后台代录 customer、师傅代录 worker）Emit 消息中心 todo，refType=`realname`、refID=`{subjectType}/{subjectId}`（同一主体多条 PENDING 共享一个待办，Link `/base/realname-review`）；审核终态 Resolve；已办结后同主体再次落单 → Emit 复活该待办（重置未办/刷新标题/清读回执，行数不增）。自动通道即时判定（PASS/FAIL）不落待办。
> - 审核回执：人工审核终态向客户 portal_messages 写 category=`system` 站内消息（PASS「实名认证已通过」/ FAIL「实名认证未通过」含驳回原因），App 端未知分类回退默认图标并展示于"全部"页签；师傅侧同终态写 worker_messages（INFO/WARN，§7.4，无驳回原因字段故用通用指引文案）。
> - 后台代录入口：客户档案页行操作「实名代录」抽屉——回显 `GET /customers/:id/real-name` 最新单（40410=暂无核验单，含证件照预览），以 `{realName,idCardNo,method}` 提交同端点落 PENDING/自动判定；客户核验记录抽屉结果列三态 PASS/FAIL/PENDING（PENDING 不再渲染为不通过）。

## 8B. Q1 客服与应收信用基础（internal/domain/cs + internal/domain/ar，000118）

> CS 扩展既有 `complaints` 工单；AR 扩展既有 `arrears` 快照。跨域只保存稳定 ID，不复制订单、客户、账单、资源或告警事实。

| 页面/概念 | API 字段 | DB 列 | 说明 |
|:---|:---|:---|:---|
| 工单优先级 | `priority` | `priority` | LOW/NORMAL/HIGH/URGENT |
| 升级级别 | `escalationLevel` | `escalation_level` | 0=未升级，正整数递增 |
| 首次响应 | `firstResponseAt` | `first_response_at` | 首次客服响应时间 |
| SLA 截止 | `slaDueAt` | `sla_due_at` | 未关闭工单的绝对截止时间 |
| 账龄快照日 | `snapshotDate` | `snapshot_date` | 客户每日唯一 |
| 账龄桶 | `days1To30` 等 | `days_1_30` 等 | 1-30/31-60/61-90/90+ 金额 |
| 催收任务 | `collectionTasks` | `ar_collection_tasks` | PENDING/DOING/DONE/FAILED 人工可接管 |
| 承诺还款 | `paymentPromises` | `ar_payment_promises` | OPEN/FULFILLED/BROKEN/CANCELED |
| 核销 | `writeoffs` | `ar_writeoffs` | 金额、原因、审批人和审批时间留痕 |

AR closure API: `POST/GET /ar/aging-snapshots` generates and reads customer daily snapshots idempotently by `(customerId,snapshotDate)`; `GET/POST /ar/payment-promises` and `/status` manage `OPEN/FULFILLED/BROKEN/CANCELED`; `GET/POST /ar/writeoffs` records amount, reason, approver and approval time.
| 服务指标 | `metricKey/numerator/denominator/value` | `service_metric_snapshots` | 按日幂等，禁止无样本伪造数据 |

## 8C. 招商入驻域（internal/domain/partner，000098）

`partner_applications`（入驻申请，公开提交；审核前不入 legal_entities/accounts）：

| 字段名 | DB 列 | 枚举/说明 |
|:------|:------|:----------|
| `companyName` | company_name | 企业名称 |
| `creditCode` | credit_code | 统一社会信用码（同码 PENDING 申请唯一） |
| `contactName` | contact_name | 联系人 |
| `contactPhone` | contact_phone | 联系电话 |
| `email` | email | 邮箱（可空） |
| `businessDesc` | business_desc | 合作意向说明 |
| `status` | status | **PENDING 待审核 / APPROVED 已通过 / REJECTED 已驳回**（terms.md 通用枚举延伸） |
| `reviewNote` | review_note | 审核意见（驳回必填） |
| `reviewerAccountId` | reviewer_account_id | BIGINT → accounts（审核人） |
| `legalEntityId` | legal_entity_id | BIGINT → legal_entities（审核通过后建子公司回填，空=未开通） |
| `adminAccountId` | admin_account_id | BIGINT → accounts（审核通过后建 partner_admin 账号回填） |
| `submittedAt` | submitted_at | 提交时间 |
| `reviewedAt` | reviewed_at | 审核时间，null=未审核 |

> 审核通过 = 建 `legal_entities`（code=P-<信用码>）+ `accounts`（username=pt_<信用码后8位>，
> 角色 `partner_admin`，legal_entity_id 绑定做数据隔离）；初始口令随机 12 位仅审核响应返回一次。
> 企业工作台数据口径：员工 = accounts 按 legal_entity_id 归属（角色限 partner_admin/partner_staff）；
> 订单 = orders 按 legal_entity_id 隔离（只读）。角色 `partner_staff` 无员工管理权限。

## 8C. 营销促销域（internal/domain/promotion，000102）

`coupon_templates`（券模板，L1 依赖 legal_entity）：

| 字段名 | DB 列 | 枚举/说明 |
|:------|:------|:----------|
| `templateId` | template_id | BIGSERIAL PK |
| `legalEntityId` | legal_entity_id | BIGINT → legal_entities |
| `name` | name | 模板名 |
| `type` | type | FULL_CUT / DISCOUNT / CASH（terms.md） |
| `faceValue` | face_value | 分；折扣券为万分比（8500=85折） |
| `threshold` | threshold | 使用门槛（分），0=无门槛 |
| `maxDiscount` | max_discount | 折扣封顶（分），可空 |
| `scopeType` / `scopeRef` | scope_type / scope_ref | ALL / PRODUCT / FIRST_ORDER；PRODUCT 时指向 product_offers.id |
| `totalQty` / `issuedQty` | total_qty / issued_qty | 发行总量（0=不限）/ 已发（行锁防超发） |
| `perCustomerLimit` | per_customer_limit | 每人限领 |
| `validDays` / `validFrom` / `validTo` | valid_days / valid_from / valid_to | 领取后 N 天 / 固定窗口（二选一） |
| `status` | status | DRAFT / ENABLED / DISABLED |

`coupons`（券实例，存量表增量改造）：新增 `template_id`、`type`、`face_value`、`threshold`、
`max_discount`、`scope_type`、`scope_ref`（模板快照，发放时冻结）、`source`
（ADMIN_ISSUE/CAMPAIGN/REDEEM/GIFT/INVITE）、`code`（转赠载体，UNIQUE，非空=转赠中）、
`issued_at`、`used_at`、`payment_id`（核销流水）；status 由 active/disabled 迁移为
ISSUED/USED/EXPIRED/DISABLED。

`coupon_codes`（兑换码批次）：`code_id` PK、`code` UNIQUE、`template_id`、`status`
（UNUSED/REDEEMED/DISABLED）、`redeemed_by` → customers、`redeemed_at`。

`coupon_redemptions`（核销记录，缴费同事务写入）：`redemption_id` PK、`coupon_id` → coupons、
`payment_id`（逻辑关联 payments）、`customer_id`、`deducted_amount`（实际抵扣分）、`created_at`。

`gift_rules`（赠送时长阶梯规则）：`rule_id` PK、`legal_entity_id`、`name`、`scope_type`/`scope_ref`
（ALL/PRODUCT）、`buy_months`、`gift_months`、`status`（ENABLED/DISABLED）。

`gift_records`（赠送发放记录）：`record_id` PK、`rule_id` → gift_rules、`customer_id` → customers、
`product_id`、`buy_months`、`gift_months`、`payment_id`（逻辑关联）。

> 金额单位一律为分（int64）；billing 域缴费金额为元（float64），跨域边界处换算（billing.redeemCoupon）。
> 000106/000105 增量：`invite_config` 增 `reward_template_id`（邀请奖励券模板，可空）；
> `coupon_templates` 增 `points_price`（积分兑换价，0=不可）。

## 8D. 忠诚度积分域（internal/domain/loy，000119 完整化）

`loy_point_ledgers`（积分账本，客户唯一）：`customer_id` PK → customers、`balance`
（CHECK >= 0）、`updated_at`。

`loy_point_entries`（积分流水）：`entry_id` BIGSERIAL PK、`customer_id` → customers、
`delta`（正充负扣）、`balance_after`（落库后余额快照）、`reason`
（terms.md：ADMIN_ADJUST/EXCHANGE/EXCHANGE_REVERSAL/PAYMENT_EARN/PAYMENT_REVERSAL/
TASK_EARN/EXPIRED/COMPENSATION）、`ref_id`（EXCHANGE 时为模板 id；
PAYMENT_* 时为 payment id，(reason,ref_id) 部分唯一索引保幂等）、
`expires_at`（获得类流水有效期，空=永久）、`expired`（过期清算标记）、`created_at`。

`loy_levels`（积分等级，000119）：`level_id` PK、`name`、`min_points`（达标门槛；
等级=累计获得积分的正向流水合计匹配最高档）、`status`、`created_at`。

`loy_tasks`（积分任务，000119）：`task_id` PK、`code`（唯一）、`name`、`points`（>0）、
`period`（ONE_TIME 终身一次 / DAILY 每日 / MONTHLY 每月）、`status`、`created_at`。

`loy_task_completions`（任务完成，000119）：UNIQUE(task_id, customer_id, period_key)
保周期幂等；`period_key`：ONE_TIME=''、DAILY=YYYY-MM-DD、MONTHLY=YYYY-MM。

`loy_earn_rules`（缴费自动积分规则，000119）：`points_per_yuan`（每 1 元=100 分送 N 分）、
`min_cents`（起缴门槛）、`expire_days`（获得积分有效期天数，0=永久）、`status`
（仅最新 ENABLED 行生效，SaveEarnRule 旧行自动失效）。

> 兑换经 LOY→PROMO 服务调用（先扣积分后发券，发券失败或超时由补偿回补/回放，见 adopted note）；
> 过期清算按客户汇总到期获得流水一次性扣减（余额不足只扣到 0，已消费部分不重复扣）。
> 退款回滚按 `(PAYMENT_EARN, payment_id, customer_id)` 幂等生成 `PAYMENT_REVERSAL`；缴费自动积分按生效规则计算，重复回调返回原发放额。

### 8D-2. 券积分对账报表（2028 Q2 交付，GET /coupon-recon 与 /loy/points-recon）

券对账行（一模板一行，`diff=drift` 只看差异行）：

| 页面列名 | 字段名 | 来源 | 枚举/说明 |
|:---------|:-------|:-----|:---------|
| 模板 | `TemplateID` / `Name` | coupon_templates | — |
| 发放计数 | `IssuedQty` | 模板计数器 | 与 `ActualIssued` 核对 |
| 实发数 | `ActualIssued` | coupons 实数 | 漂移即 COUNTER_DRIFT |
| 状态分布 | `ByStatus` | coupons 聚合 | ISSUED/USED/EXPIRED/DISABLED |
| 已核销 | `UsedCount` / `RedemptionCnt` | coupons / coupon_redemptions | 不等即 REDEMPTION_LOST |
| 核销金额 | `RedeemedAmount` | coupon_redemptions.deducted_amount 合计 | 分 |
| 面值敞口 | `FaceValueTotal` | ISSUED 券 face_value 合计 | 分 |
| 差异 | `DiffKind` | — | MATCH / COUNTER_DRIFT / REDEMPTION_LOST |

积分对账行（一客户一行）：`Balance`（账本）vs `EntriesSum`（流水合计），
`LifetimeEarn`/`ExpiredTotal`/`ByReason`（reason→delta 合计）；
`DiffKind` = MATCH / BALANCE_DRIFT。账本 loy_point_ledgers 仍为唯一事实源，
对账只读不改。

### 8D-3. Admin 营销与积分规则页 / 券积分对账页（2028 Q2 交付）

`/bss/marketing`（menu key `marketing`，menu:userdata 门禁）五 Tab，列名直映射 JSON 字段：

| Tab | 列名 | 字段名 |
|:----|:-----|:-------|
| 券模板 | 名称/券类型/面值(元)/门槛(元)/有效天数/已发总量/状态 | `name` / `type` / `faceValue`÷100 / `threshold`÷100 / `validDays` / `issuedQty`/`totalQty` / `status` |
| 赠送时长 | 名称/实购月数/赠送月数/状态 | `name` / `buyMonths` / `giftMonths` / `status` |
| 缴费送积分 | 每元送积分/起缴门槛(元)/积分有效天数 | `pointsPerYuan` / `minCents`÷100 / `expireDays` |
| 积分等级 | 名称/达标门槛/状态 | `name` / `minPoints` / `status` |
| 积分任务 | 任务编码/名称/积分/周期/状态 | `code` / `name` / `points` / `period`（ONE_TIME/DAILY/MONTHLY）/ `status` |

`/bss/marketing-recon`（menu key `marketing-recon`）两 Tab 复用 8D-2 行结构，附加汇总行
（`summary`）与 `diff=drift` 过滤；金额列后端为分，页面 ÷100 展示。

### 8D-4. Admin 用户端配置页 /bss/userdata（menu key `userdata`，menu:userdata 门禁）

七 Tab 列表（`internal/domain/customer/userdata/pg_lists.go` 的 SELECT ... AS 别名即字段契约），
行级动作走对应 PUT 路由；金额列后端为分，页面 fmtFee 展示：

| Tab | 列名 | 字段名 | 行动作 |
|:----|:-----|:-------|:-------|
| 通知设置 | 客户/业务通知/营销通知/接收渠道 | `customerId` / `customerName` / `business` / `marketing` / `channel` | — |
| 增值服务 | 名称/价格/状态/订阅数 | `addonId` / `name` / `price` / `status`（on/off）/ `subscriberCount` | toggle |
| 优惠券 | 客户/名称/面额/状态/有效期 | `couponId` / `customerId` / `name` / `amount` / `status`（ISSUED/USED/DISABLED/EXPIRED）/ `expireAt` | disable |
| 充值档位 | 面额/赠送金额/状态 | `denomId` / `amount` / `bonus` / `active` | — |
| 常见问题 | 分类/问题/状态 | `faqId` / `category` / `question` / `active` | toggle |
| 自助指南 | 标题/分类/状态 | `guideId` / `title` / `category` / `active` | toggle |
| 邀请配置 | 邀请链接/奖励金额/状态 | `id` / `inviteLink` / `rewardAmount` / `active` | — |

状态列走 StatusTag `userdata` 域（布尔配置渲染 active/disabled 语义色）。

## 8E. 官网内容发布域（internal/domain/cms，000134；分类字典 000138）

`cms_posts`（官网动态/文章/新闻，单一内容表 + category 区分；沿用 cs_knowledge_articles 的版本自增与软状态机范式）：

| 页面列 | 字段名 | DB 列 | 枚举/说明 |
|:------|:------|:------|:----------|
| 标题 | `title` | title | 非空，≤160 |
| 别名 | `slug` | slug | URL 友好键，小写字母数字连字符；唯一键 `(slug, lang)`（000155 起，同 slug 多语言变体共用 URL） |
| 语言 | `lang` | lang | zh-CN / en-US / ms-MY（000155），公开 URL `/news/:slug?lang=` 切换，缺变体回退 zh-CN |
| 分类 | `category` | category | 引用 `cms_categories.code`（000138 起自定义字典，NEWS/ARTICLE 为种子值） |
| 摘要 | `summary` | summary | 列表展示，≤500 |
| 封面 | `coverAttachmentId` | cover_attachment_id | 引用 attachments(id)，可空 |
| 正文 | `content` | content | Markdown，TEXT 非空；正文图片引用 `](att/N)`，公开读重写为 `/site/posts/:slug/img/N?lang=` |
| 状态 | `status` | status | DRAFT / PUBLISHED / OFFLINE（terms.md 登记） |
| 定时发布 | `publishedAt` | published_at | 置 PUBLISHED 时落 now()；公开读按 `status=PUBLISHED` 过滤 |
| 版本 | `version` | version | 每次更新自增 |
| 作者 | `authorName` | author_name | 展示用冗余名，可空 |
| 创建/更新 | `createdAt`/`updatedAt` | created_at/updated_at | TIMESTAMPTZ |

API：admin `/site-posts`（GET/POST/PUT/DELETE，menu:site 权限，body 带 lang）；
官网匿名只读 `/site/posts`（列表，仅 status=PUBLISHED，按 published_at 倒序，
支持 `category`/`lang` 过滤与 limit）与 `/site/posts/:slug`（详情，仅 PUBLISHED，
`lang` 参数缺变体时落入默认语言兜底返回，官网切语言不因此 404）。
免鉴权公开读沿用 partner 入驻公开提交先例（admin 前缀内 public 子路由）。
封面 `/site/posts/:slug/cover`；正文图片 `/site/posts/:slug/img/:attId`
（匿名，仅"该文 Markdown 确实引用 + image/*"，防附件枚举；二者均接受 `lang`
以锁定同一变体的封面/引用，图片端点带 `?lang=` 前缀由详情重写生成）。

分类字典 `cms_categories`（000138，页面 `/boss/site/cats`，menu key `site-cats`，权限 `menu:site-cats`）：

| 页面列 | 字段名 | DB 列 | 枚举/说明 |
|:------|:------|:------|:----------|
| 标识码 | `code` | code | 大写蛇形 2~32，唯一；被文章引用时禁删/禁改 |
| 名称 | `name` | name | 非空，≤64，作为展示名默认/回退 |
| 多语言名 | `names` | name_i18n | JSONB（000155）：`{zh-CN, en-US, ms-MY}`→名，键限语言集，值 ≤64；缺语言回退 name |
| 排序 | `sortNo` | sort_no | 升序展示 |
| 启用 | `enabled` | enabled | 停用不影响存量文章，仅新文章不可选 |

API：admin `/site-categories`（GET/POST/PUT/DELETE，menu:site 权限）。

前端编辑页：`/boss/site/new`（新建，语言取默认 zh-CN 可切换）、`/boss/site/:postId`（编辑，
语言与正文同表单维护，同 slug 多语言各存一行）；列表页有语言列与语言筛选。
与列表页同用 menu:site 守卫；正文 MarkdownEditor 双栏编辑（工具栏 + react-markdown 预览），
图片上传走附件域（MinIO/S3）后以 `](att/N)` 引用。

## 8F. 客户端版本发布域（internal/domain/apprelease，000137）

`client_releases`（师傅端/用户端 App 发版记录，单一表 + app 区分两端；APK 对象入 MinIO 复用 attachment 存储抽象）：

| 页面列 | 字段名 | DB 列 | 枚举/说明 |
|:------|:------|:------|:----------|
| 端 | `app` | app | user（用户端）/ worker（师傅端） |
| 平台 | `platform` | platform | android（当前仅 Android） |
| 版本号 | `version` | version | 展示用 semver，如 1.2.0 |
| 版本码 | `versionCode` | version_code | 单调递增整数，升级判定唯一依据 |
| 最低兼容码 | `minSupportedCode` | min_supported_code | 客户端 versionCode 低于此值 → 强制更新 |
| 更新说明 | `notes` | notes | 更新日志，TEXT |
| 强制 | `force` | force | bool，true 时弹框不可忽略（默认 false 不强制） |
| 状态 | `status` | status | DRAFT / GRAY / PUBLISHED / ROLLED_BACK |
| 灰度比例 | `rolloutPercent` | rollout_percent | 0~100，GRAY 生效；0=仅白名单 |
| 白名单 | `whitelistIds` | whitelist_ids | int[]，灰度命中豁免（worker/customer id） |
| 安装包 | `apkObjectKey` | apk_object_key | MinIO object key |
| 包大小 | `apkSize` | apk_size | 字节 |
| 校验和 | `sha256` | sha256 | APK SHA-256，客户端校验 |
| 创建/更新 | `createdAt`/`updatedAt` | created_at/updated_at | TIMESTAMPTZ |

升级判定（GET /client/latest，社区通行做法：latest + minSupported 双门槛 + 确定性灰度分桶）：
按 app+platform 取 status∈(GRAY,PUBLISHED) 中 version_code 最大者；GRAY 时以
`hash(deviceId+releaseId)%100 < rollout_percent` 或命中 whitelistIds 决定是否投放，
未投放回落最近 PUBLISHED。响应 `updateAvailable/force/version/versionCode/notes/downloadUrl/sha256/size`；
`force = 客户端 versionCode < minSupportedCode 或 release.force`。

API：admin `/client-releases`（GET 列表 / POST multipart 上传创建 / PATCH 元数据与状态迁移 / GET :id/apk 下载，menu:release 权限）；
客户端匿名检查 `GET /api/{worker,user}/v1/client/latest`（免登录，启动即查）；
官网匿名 `GET /api/admin/v1/client-releases/latest?app=`（仅 PUBLISHED，首页下载入口）。

## 9. 采购域 + 施工回单（internal/domain/procurement + order.install_logs，迁移 000163/000164）

> 决策依据：docs/notes/adopted/2026-08-28-procurement-install-gis-linkage.md
> 阶段：增量挂靠（不开阶段 10）；菜单分组：ams 资产与标签（采购/库存）+ boss 订单与履约（施工看板）

### 9.1 procurement_suppliers（供应商，迁移 000163）

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| — | `ID` | id | BIGSERIAL PK |
| 编码 | `Code` | code | VARCHAR(64) UNIQUE（如 S-001） |
| 名称 | `Name` | name | VARCHAR(128) |
| 联系人 | `ContactName` | contact_name | VARCHAR(64) 可空 |
| 联系电话 | `ContactPhone` | contact_phone | VARCHAR(32) 可空 |
| 所属公司 | `LegalEntityID` | legal_entity_id | BIGINT → legal_entities（企业锚点 §8.1） |
| 状态 | `Status` | status | ENABLED / DISABLED（terms.md §4） |
| 备注 | `Remark` | remark | VARCHAR(255) |

### 9.2 procurement_orders（采购单头，迁移 000163）

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| — | `ID` | id | BIGSERIAL PK |
| 单号 | `ProcurementNo` | procurement_no | VARCHAR(32) UNIQUE（PO-YYYYMMDD-NNNNN，后端生成兜底） |
| 所属公司 | `LegalEntityID`+`LegalEntityName` | legal_entity_id+name | §8.1 企业锚点铁律 |
| 供应商 | `SupplierID`+`SupplierName` | supplier_id+name | FK → procurement_suppliers + 名称快照 |
| 状态 | `Status` | status | DRAFT / SUBMITTED / PARTIAL / RECEIVED / CANCELLED（terms.md §4） |
| 总金额 | `TotalAmount` | total_amount | NUMERIC(14,2) |
| 预计到货 | `ExpectedDate` | expected_date | DATE 可空 |
| — | `CreatedBy` | created_by | BIGINT → accounts |
| — | `SubmittedAt` / `ReceivedAt` / `CancelledAt` | 同 | 状态流转时间戳 |

### 9.3 procurement_order_items（采购单明细，迁移 000163）

| 字段名 | DB 列 | 枚举/说明 |
|:-------|:------|:----------|
| `OrderID` | order_id | BIGINT → procurement_orders（ON DELETE CASCADE） |
| `MaterialCode` | material_code | VARCHAR(64) 对齐 material_items.code（L0 全局主档，data-layers §1） |
| `Spec` | spec | VARCHAR(128) 可空 |
| `Quantity` | quantity | INTEGER > 0 |
| `ReceivedQty` | received_qty | INTEGER ≥ 0，CHECK ≤ quantity |
| `UnitAmount` | unit_amount | NUMERIC(14,2) |

### 9.4 procurement_receipts（到货入库单，迁移 000163）

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| — | `ID` | id | BIGSERIAL PK |
| 单号 | `ReceiptNo` | receipt_no | VARCHAR(32) UNIQUE（RC-YYYYMMDD-NNNNN） |
| 采购单 | `OrderID`+`OrderNo` | order_id+order_no | FK → procurement_orders + 单号快照 |
| 入库批次 | `BatchID` | batch_id | BIGINT → asset_batches（CONFIRMED 同事务建批次后回填） |
| 所属公司 | `LegalEntityID`+`LegalEntityName` | legal_entity_id+name | §8.1 |
| 接收人 | `ReceivedBy` | received_by | BIGINT → accounts |
| 接收时间 | `ReceivedAt` | received_at | TIMESTAMPTZ |
| 状态 | `Status` | status | DRAFT / CONFIRMED / REJECTED（terms.md §4） |
| 备注 | `Remark` | remark | VARCHAR(255) |

### 9.5 install_logs（施工回单，迁移 000164）

| 字段名 | DB 列 | 枚举/说明 |
|:-------|:------|:----------|
| `TicketID` | ticket_id | BIGINT → dispatch_tickets（N:1，同 ticket 同一时刻最多一条 OPEN） |
| `OrderID` | order_id | BIGINT → orders |
| `WorkerID`+`WorkerName` | worker_id+name | FK → workers + 姓名快照 |
| `Photos` | photos | JSONB（attachments.id 数组，MinIO 证据） |
| `SignName` | sign_name | VARCHAR(64) 用户签收姓名 |
| `SignImageURL` | sign_image_url | VARCHAR(255) 签收图片 |
| `SignedAt` | signed_at | TIMESTAMPTZ |
| `Note` | note | VARCHAR(500) |
| `Status` | status | OPEN / COMPLETED / REJECTED（terms.md §4） |

> 唯一约束：`(ticket_id) WHERE status='OPEN'` 部分唯一（uq_install_logs_ticket_open）。

### 9.6 dispatch_tickets 增列（迁移 000164）

| 字段名 | DB 列 | 枚举/说明 |
|:-------|:------|:----------|
| `ArrivedAt` | arrived_at | TIMESTAMPTZ 可空（师傅到场打卡事实） |
| `ArriveLat` | arrive_lat | DOUBLE PRECISION [-90,90] 可空，WGS84 |
| `ArriveLng` | arrive_lng | DOUBLE PRECISION [-180,180] 可空，WGS84 |

> 派生事实不写回订单状态；GIS 施工实时图层读 arrive_lat/lng 出图。
> 旧版本师傅端无打卡动作时 NULL 兜底；标记作业 DOING + 师傅匹配 + arrived_at IS NULL 才允许 UPDATE（幂等首打卡）。

### 9.6a dispatch_tickets 增列（迁移 000174，站点坐标快照）

| 字段名 | DB 列 | 枚举/说明 |
|:-------|:------|:----------|
| `SiteLat` | site_lat | DOUBLE PRECISION [-90,90] 可空，WGS84 |
| `SiteLng` | site_lng | DOUBLE PRECISION [-180,180] 可空，WGS84 |

> 派单时刻自 orders.address_id → addresses.geom 一次性物化（快照口径同 price_snapshot，
> 不随地址树后续变更漂移）；与 000164 arrive_lat/lng（师傅侧到场事实）语义互不混用。
> 同事务解析 region_id/region_name ← orders.region_path 在 regions 树的最近祖先或自身。
> 半径闸门（worker 接单设置 radiusKm）读此快照；空 = 跳过校验，不误拦。

### 9.7 asset_batches 增列（迁移 000163，GIS 库存分布图层用）

| 字段名 | DB 列 | 枚举/说明 |
|:-------|:------|:----------|
| `WarehouseLat` | warehouse_lat | DOUBLE PRECISION 可空，CHECK [-90,90] |
| `WarehouseLng` | warehouse_lng | DOUBLE PRECISION 可空，CHECK [-180,180] |

> 入库确认时由 ConfirmReceipt 入参 warehouseLat/Lng 一并写入；GIS `GET /gis/inventory-points?entity=warehouse&bbox` 读此列做聚合图层。

## 10. 字段字典的使用规则（写入 Agent 输入包）

1. 实现实体前，先查本文件是否已定其字段；已定则**照抄字段名与枚举**，不得另起别名。
2. 未定字段（本文件无该实体）时，字段名遵循第 0 节命名规则，并**回写本文件**补一节，避免下个 Agent 再猜。
3. 页面列名与字段名必须一一对应；页面新增列时，同步在此登记英文字段名与枚举。
4. 状态枚举一律引用 `terms.md`，本文件不重复定义枚举值（仅标注引用来源）。

## 8G. 系统授权域（internal/domain/license，release-platform 离线授权）

无本地表：授权证书为 release-platform 签发的 Ed25519 离线令牌（admin 页可读到
claims 并非本地权威事实，仅展示）。门禁语义见 adopted/2026-09-01-license-gate-release-platform.md。

「系统授权」页（base 组，登录即可达）：

| 页面列 | 字段名 | 来源 | 枚举/说明 |
|:------|:------|:------|:----------|
| 已激活 | `activated` | license/status claims 校验结果 | bool；false 时业务接口 403 LICENSE_REQUIRED |
| 门禁启用 | `enabled` | License 是否注入(未启用返回 false) | bool；svc=nil(开发/演示)时 false |
| 授权编号 | `licenseId` | license_id | release-platform activation code id |
| 产品 | `productId` | product_id | release-platform 产品 ID（如 boss-server） |
| 授权类型 | `licenseType` | license_type | duration / lifetime / trial |
| 绑定设备 | `deviceId` | device_id | 激活时上报的本实例标识 |
| 机器指纹 | `fingerprint` | fingerprint_hash | 防复制绑定维度(与 deviceId 双验) |
| 有效期至 | `expiresAt` | expires_at | RFC3339；宽限期内 still 可用（grace） |
| 宽限期 | `graceEndsAt` / `inGrace` | grace_ends_at | inGrace=true 表示已过期但未出宽限 |
| 核验时间 | `checkedAt` | 服务端核验时刻 | RFC3339；每次授权检查都刷新 |
| 失效原因 | `reason` | 核验失败原因(展示用) | 不暴露签名细节；仅未激活/失效时出现 |

激活：admin `POST /license/activate`（体 `activationCode`）→ release-platform
`POST /v1/activations` 兑码 + `POST /v1/licenses/{id}/offline-token` 取令牌 →
本地落盘 `/var/lib/boss/license.json`。门禁豁免路径：`/license/status`、
`/license/activate`、各端 `/auth/*`（登录/注册/登出）与 `/client/latest`（版检）。
