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

区划列表响应（GET /geo/subdivisions，选择器懒加载下钻）：行字段 `HasChildren`（hasChildren，存在未停用子节点，无 DB 列，下钻入口渲染用）；查询参数 `parentCode`（非空=直接子节点，传空值=顶层节点，缺省=不过滤）、`keyword`（模糊匹配编码+全部译名，须配合 `country`，缺失拒绝 42200）、`limit`（默认 50，最大 200）。
默认国家（零迁移，复用 biz_params）：键 `geo.default_country`（值=alpha-2，如 PH）；读端点 GET /geo/default-country（登录管理员即可读，不设 menu 门槛）；写路径复用 PUT /params/:key（menu:params）；未配置/值非法返回空值对象 `{countryCode:"",configured:false}`，不兜底部署主国家。

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
> 生命周期（P6，T8，迁移 000198）：`lifecycle_status` PLANNED/IN_BUILD/IN_SERVICE/RETIRED，存量默认 IN_SERVICE；转移 `PUT /odn/facilities/{code}/lifecycle`；RETIRED 终态与 status 双列同步；规则 internal/domain/odn/lifecycle.go（site/device 同规，见 §1.5.5）。

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
> 生命周期（P6，T8，迁移 000198）：局点/设备同样带 `lifecycle_status` 四态，转移 `PUT /odn/sites/{siteNo}/lifecycle`、`PUT /odn/devices/{id}/lifecycle`；RETIRED 双列同步，终态不可逆。

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


### 1.5.7 address_coverage（ODN 覆盖关联，迁移 000197，internal/domain/odn）

> 落地 adopted note 2026-09-06-odn-business-linkage（「网络规划是业务基础」P1 覆盖关联）：地址 ↔ 服务设施/核心设备 1:1 关联 + 可装状态，P1 只「可查可判」不做下单硬校验。管理面 `menu:odn`，REST `/odn/coverage*`（契约 admin/odn.yaml）。

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 地址 | `AddressID` | address_id | BIGINT → addresses，UNQ 一址一覆盖 |
| 服务设施 | `FacilityCode` | facility_code | 可空 → odn_facility(code)（ODB/SDB/分纤点） |
| 服务设备 | `DeviceID` | device_id | 可空 → odn_device(id)（核心链路设备） |
| 可装状态 | `Status` | status | SERVED 可装 / PENDING 规划在建 / UNSERVED 未覆盖；默认 UNSERVED |
| 备注 | `Note` | note | 可空 |
| 更新时间 | `UpdatedAt` | updated_at | TIMESTAMPTZ |

> 校验：`address_coverage_target_chk`——SERVED/PENDING 必须至少挂一个目标（设施或设备），UNSERVED 允许全空。
> 下单覆盖门控（P2，T12）：`BOSS_ODN_COVERAGE_GATE=on` 时 Submit 前置校验地址 SERVED，未 SERVED 以 40900 拒单（透传原因）；灰度默认关。

### 1.5.8 construction_projects / construction_items（施工项目与竣工回填，迁移 000199，internal/domain/odn）

> P6 设计-施工闭环（路线图 T9）：施工单状态机 PENDING→BUILDING→ACCEPTED（线性，同态 no-op）；ACCEPTED 时单内设施 `lifecycle_status` 批量 PLANNED/IN_BUILD→IN_SERVICE（as-built 回填），记竣工人/时间/备注。明细 ACCEPTED 后锁定；设施须 PLANNED 态方可入单。管理面 `menu:odn`，REST `/odn/constructions*`（契约 admin/odn.yaml）。

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 施工单号 | `ProjNo` | proj_no | VARCHAR(32) 唯一 |
| 名称 | `Name` | name | 可空 |
| 城市 | `PrvCode`/`CityPrefix` | prv_code/city_prefix | 可空复合 FK odn_city_code |
| 状态 | `Status` | status | PENDING 待开工 / BUILDING 施工中 / ACCEPTED 已竣工（终态） |
| 竣工备注 | `AsbuiltNote` | asbuilt_note | as-built 记录 |
| 竣工人/时间 | `AcceptedBy`/`AcceptedAt` | accepted_by/accepted_at | → accounts；验收时落 |
| 明细数 | `ItemCount` | —（聚合） | construction_items 计数 |
| 承包商 | `ContractorID`/`ContractorName` | contractor_id/contractor_name | 软引用 → procurement_suppliers(id)（跨域不加 FK，adopted 2026-09-07）+ 名称快照；NULL/空=未指定（存量兼容）；有有效结算单后锁定（列经 000208 补齐；000206 漏列事故见打回记录 2026-09-07） |
| 清单金额 | `ItemsAmount` | —（聚合） | SUM(construction_items.amount)，页面展示列，结算应付同口径（000206） |

#### construction_items 增列（工程量清单，迁移 000206）

| 字段名 | DB 列 | 枚举/说明 |
|:-------|:------|:----------|
| `Quantity` | quantity | NUMERIC(14,2) ≥0 默认 0=未定额；可带 |
| `UnitPrice` | unit_price | NUMERIC(14,2) ≥0 默认 0；可带 |
| `Amount` | amount | NUMERIC(14,2) GENERATED ALWAYS AS (round(quantity*unit_price,2)) STORED——金额一律后端计算，DB 层禁止直写；前端仅展示 |

### 1.5.8a construction_settlements（工程结算单，迁移 000206，internal/domain/odn）

> W1 结算模型（adopted 2026-09-07-contractor-settlement-model）：项目 ACCEPTED 且已指定施工类承包商方可发起；结算单汇总清单金额形成应付；状态机 PENDING→SETTLED；PENDING/SETTLED→VOIDED（原因必填）；VOIDED 终态，作废后重开以新结算单表达（原单保留历史）。同项目同时最多一张有效结算单（部分唯一索引 uq_construction_settlements_project_active）。管理面 `menu:odn` 施工页签内，REST `/odn/constructions/{id}/settlements*`、`/odn/settlements/{id}/settle|void`（契约 admin/odn.yaml）。

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 结算单号 | `SettlementNo` | settlement_no | VARCHAR(32) 唯一（ST-YYYYMMDD-NNNNN，后端生成兜底） |
| 施工项目 | `ProjectID`/`ProjectNo` | project_id/project_no | FK → construction_projects + 单号快照 |
| 承包商 | `ContractorID`/`ContractorName` | contractor_id/contractor_name | 发起时自项目快照（软引用，同上） |
| 应付金额 | `TotalAmount` | total_amount | NUMERIC(14,2) = 发起时 SUM(items.amount)；ACCEPTED 后明细锁定不漂移 |
| 明细数 | `ItemCount` | item_count | INTEGER |
| 状态 | `Status` | status | PENDING / SETTLED / VOIDED（terms.md §4） |
| 作废原因 | `VoidReason` | void_reason | 作废必填 ≤255 字 |
| 发起/结算/作废人 | `CreatedBy`/`SettledBy`/`VoidedBy` | created_by/settled_by/voided_by | → accounts |
| 时间 | `CreatedAt`/`SettledAt`/`VoidedAt` | 同 | TIMESTAMPTZ |


#### 1.5.8b construction_progress（施工进度记录，迁移 000213，internal/domain/odn）

> P0-B 资源级现场事实（odn-construction-management-plan）：进度/坐标/照片绑定具体设施；`(project_id, facility_code, client_msg_id)` 唯一实现弱网重传幂等，重复上报不重复计量；上报设施必须已在项目明细范围内。管理面 `menu:odn` 施工页签内，REST `/odn/constructions/{id}/progress`（契约 admin/odn.yaml）。

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 施工项目 | `ProjectID` | project_id | FK → construction_projects |
| 设施 | `FacilityCode` | facility_code | FK → odn_facility(code)，须在 construction_items 范围内 |
| 累计完成量 | `DoneQty` | done_qty | NUMERIC(14,2) ≥0；聚合口径=同设施 SUM(done_qty) 对比清单 quantity |
| 坐标 | `Lat`/`Lng` | lat/lng | DOUBLE PRECISION 可空，现场采集 |
| 备注 | `Note` | note | ≤500 字 |
| 照片 | `PhotoIDs` | photo_ids | BIGINT[] → attachments |
| 幂等键 | `ClientMsgID` | client_msg_id | VARCHAR(64)，8-64 字符，弱网重传同键不重复计量 |
| 上报人/时间 | `ReportedBy`/`ReportedAt` | reported_by/reported_at | → accounts / TIMESTAMPTZ |

#### 1.5.8c construction_tests / construction_defects（质量测试与整改，迁移 000214，internal/domain/odn）

> P0-C 质量闭环（odn-construction-management-plan）：测试记录 append-only（OTDR/光功率/连通性逐段逐设施留证）；整改 OPEN→RECTIFYING→VERIFIED（OPEN 可直达 VERIFIED 即改即验，VERIFIED 终态）；**验收前置：同项目无未 VERIFIED 整改项**（AcceptProject 事务内校验，40900 + openDefects 计数）。管理面 `menu:odn`，REST `/odn/constructions/{id}/tests|defects`、`/odn/defects/{id}/rectify|verify`（契约 admin/odn.yaml）。

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 资源类型 | `ResourceType` | resource_type | FACILITY / SEGMENT / FIBER / PORT |
| 资源引用 | `ResourceRef` | resource_ref | VARCHAR(64) 软引用；FACILITY 类型须在项目明细范围内 |
| 测试类型 | `TestKind` | test_kind | OTDR / OPTICAL_POWER / CONNECTIVITY |
| 结果 | `Result` | result | PASS / FAIL |
| 衰减/光功率 | `AttenuationDB`/`PowerDBM` | attenuation_db/power_dbm | NUMERIC 可空，0 视为未录 |
| 严重度 | `Severity` | severity | MINOR / MAJOR / CRITICAL |
| 整改状态 | `Status` | status | OPEN / RECTIFYING / VERIFIED（terms.md §4） |
| 描述 | `Description` | description | VARCHAR(255) 必填 |
| 复验人/时间 | `VerifiedBy`/`VerifiedAt` | verified_by/verified_at | → accounts / TIMESTAMPTZ |

#### 1.5.8d 工程预算与里程碑（construction_projects 增列 + construction_milestones，迁移 000218，W6/G2）

> P-INFRA-1 W6（G2 剩余；审查 F8 合并设计）。项目可挂预算金额与里程碑清单（名称/计划完成日/状态）；
> 预算与里程碑清单仅项目 PENDING（BUILDING 前）可改，里程碑状态可标记至 ACCEPTED 前，ACCEPTED 后锁定（terms.md §4）。
> 预算执行进度=已结算金额/预算金额，**只读派生**：已结算金额=同项目 SETTLED 结算单合计（VOIDED 不计），禁止直写；
> 预算未登记（NULL）显示「未登记」而非 0（口径同 1.5.11 settledCost）。管理面 `menu:odn` 施工页签内，
> REST `PUT /odn/constructions/{id}/budget`、`/odn/constructions/{id}/milestones*`、`/odn/milestones/{id}/complete|reopen`（契约 admin/odn.yaml）。

construction_projects 增列：

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 预算金额 | `BudgetAmount` | budget_amount | NUMERIC(14,2) 可空 NULL=未登记；>=0；仅 PENDING 可改（000218） |
| 已结算金额 | `SettledAmount` | —（派生） | SUM(SETTLED settlements.total_amount)，页面展示列，只读派生 |

construction_milestones：

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 施工项目 | `ProjectID` | project_id | FK → construction_projects |
| 名称 | `Name` | name | VARCHAR(128) 必填 |
| 计划完成日 | `PlannedDate` | planned_date | DATE 可空（YYYY-MM-DD） |
| 状态 | `Status` | status | PENDING / DONE（terms.md §4）；DONE 落 done_at |
| 完成时间 | `DoneAt` | done_at | TIMESTAMPTZ 可空 |

#### 1.5.8e 工程应付台账（construction_payables + payments/deductions/invoices，迁移 000219，W6/F8）

> P-INFRA-1 W6（审查 F8：SETTLED 即终点→应付台账闭环）。应付由 SETTLED 结算单**同事务自动生成**
> （金额=结算应付快照，settlement_id 唯一来源引用，一结算单一应付，作废后重开以新结算单+新应付表达）；
> 结算单 VOIDED 同事务冲销应付（原因同源，付款流水保留历史）。部分付款与分期=多次登记付款流水至未付余额耗尽，
> 单笔不超余额（40900）；核减=复审减项（原因必填，append-only），净应付/余额只读派生不落列。
> 管理面 `menu:payables`（挂 billing 分组呈现，域归属见 domain-map §2.1），
> REST `/odn/payables*`（契约 admin/odn.yaml）。状态枚举与 VOIDED 联动规则见 terms.md §4。

construction_payables（应付单头）：

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 应付单号 | `PayableNo` | payable_no | VARCHAR(32) 唯一（AP-YYYYMMDD-NNNNN，后端生成兜底） |
| 来源结算单 | `SettlementID`/`SettlementNo` | settlement_id/settlement_no | settlement_id UNIQUE → construction_settlements + 单号快照（单据链回放锚点） |
| 施工项目 | `ProjectID`/`ProjectNo` | project_id/project_no | 快照自结算单 |
| 承包商 | `ContractorID`/`ContractorName` | contractor_id/contractor_name | 快照自结算单（软引用，同 1.5.8a） |
| 应付金额 | `PayableAmount` | payable_amount | NUMERIC(14,2) = 结算应付快照 |
| 状态 | `Status` | status | OPEN / PARTIAL / PAID / VOIDED（terms.md §4；按流水派生维护） |
| 冲销原因 | `VoidReason` | void_reason | 随结算单作废原因同源 |
| 核减合计/已付/未付余额 | `DeductedAmount`/`PaidAmount`/`Balance` | —（派生） | 子查询 SUM；余额=净应付-已付，VOIDED 可为负（超付如实展示） |
| 生成/更新时间 | `CreatedAt`/`UpdatedAt` | 同 | TIMESTAMPTZ |

construction_payable_payments（付款流水登记）：

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 应付 | `PayableID` | payable_id | FK → construction_payables |
| 付款流水号 | `PaymentNo` | payment_no | VARCHAR(32) 唯一（PAY-YYYYMMDD-NNNNN，后端生成兜底） |
| 金额 | `Amount` | amount | NUMERIC(14,2) >0；单笔不超未付余额（40900） |
| 方式 | `Method` | method | TRANSFER / CASH / CHEQUE / OTHER（terms.md §4） |
| 付款时间 | `PaidAt` | paid_at | TIMESTAMPTZ 可空缺省当前 |
| 凭证号/备注 | `Reference`/`Note` | reference/note | ≤64/≤255 可空 |
| 登记人 | `CreatedBy` | created_by | → accounts |

construction_payable_deductions（核减明细，append-only）：

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 应付 | `PayableID` | payable_id | FK → construction_payables |
| 金额 | `Amount` | amount | NUMERIC(14,2) >0；累计不超应付金额；核减后净应付不得低于已付（40900） |
| 原因 | `Reason` | reason | VARCHAR(255) 必填（留痕） |
| 登记人/时间 | `CreatedBy`/`CreatedAt` | 同 | → accounts / TIMESTAMPTZ |

construction_payable_invoices（发票登记，纯登记不联动税局）：

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 应付 | `PayableID` | payable_id | FK → construction_payables |
| 发票号 | `InvoiceNo` | invoice_no | VARCHAR(64)；UNIQUE(payable_id, invoice_no) 同应付内唯一 |
| 金额 | `Amount` | amount | NUMERIC(14,2) >0 |
| 开票日 | `InvoicedAt` | invoiced_at | DATE 可空 |
| 备注/登记人 | `Note`/`CreatedBy` | 同 | ≤255 / → accounts |

### 1.5.9 odn_port（物理端口占用态，迁移 000200，internal/domain/odn）

> P2 端口占用（路线图 T11）：分光器/终端盒端口级资源，订单预占的物理落地。`order_id` 为订单软引用（E10/E11 独立命名空间，不加 FK）。管理面 `menu:odn`，REST `/odn/devices/{id}/ports`、`/odn/ports/allocate-for-address`（覆盖关联兑现）等。

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 设备 | `DeviceID` | device_id | → odn_device，UQ(device_id, port_no) |
| 端口号 | `PortNo` | port_no | 1~99 |
| 状态 | `Status` | status | IDLE 空闲 / RESERVED 预占 / IN_SERVICE 在网；默认 IDLE |
| 预占订单 | `OrderID` | order_id | 订单软引用（无 FK），释放时清空 |
| 更新时间 | `UpdatedAt` | updated_at | TIMESTAMPTZ |

> 状态机：IDLE→RESERVED（订单预占，SKIP LOCKED 防双占）→IN_SERVICE（开通）→IDLE（拆机释放）；RESERVED→IDLE 可释放。`AllocateForAddress`：地址 → 覆盖设备 → 空闲口，无挂接设备/无空闲口明确报错。

### 1.5.10 odn_bindings（逻辑-物理绑定，迁移 000201，internal/domain/odn）

> P3 售后/反查数据源（路线图 T13）：订单（及可选逻辑资源口）↔ ODN 物理口的绑定事实，一口一绑定；端口须 IN_SERVICE 方可绑定。GIS 点位反查在用订单、售后按单查物理口均以此为准（Amended 2026-08-25-odn-gis-coords-link 的「点位点击不查详情」）。管理面 `menu:odn`，REST `/odn/bindings*`（契约 admin/odn.yaml）。

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 物理口 | `PortID` | port_id | → odn_port，UNQ 一口一绑定 |
| 订单 | `OrderID` | order_id | 订单软引用（无 FK） |
| 逻辑口 | `ResourcePortID` | resource_port_id | 逻辑资源 ports.id，可空软引用 |
| 备注 | `Note` | note | 可空 |
| 绑定时间 | `BoundAt` | bound_at | TIMESTAMPTZ |

### 1.5.11 投资测算读模型（GET /odn/grid-investment + /odn/city-investment + /odn/split-capacity，P-INFRA-1 W2/W5，internal/domain/odn）

> 纯只读聚合：零业务写路径（分光比回写建模的唯一写路径见 1.5.15，000221）。
> 页面 `intel/grid-investment`（投资测算，网格/城市/分光容量三视图），菜单权限 `menu:grid-investment`
> （迁移 000207，授予 sysadmin/analyst/ops）；契约 admin/odn.yaml。
> 口径裁定（W2 执行会话 2026-09-07 基础四条；W5 执行会话 2026-09-07 增补 5-8 条，
> 维度归属总裁定见 adopted 2026-09-07-split-capacity-investment-depth）：
> 1. 设施数：`odn_facility.grid_code` 非空行（即 P/MH）按 `lifecycle_status` 分组计数；TW/CLS/TBX 市域设施无网格维度，不进网格行。
> 2. 覆盖地址数：`address_coverage` 经服务设施（`facility_code → odn_facility.grid_code`）归属网格，按 `status` 分组计数；
>    仅挂核心设备（`device_id`）或未挂目标的行（UNSERVED 默认无目标）不进网格行，全网口径见覆盖页 `/odn/coverage*`。
> 3. 已结算工程成本：读 W1 承包商结算数据（`construction_settlements`，迁移 000206；读法 internal/domain/odn/pg_investment.go，
>    `to_regclass` 守卫：结算表未建即 W1 未合并/未部署时全表 `settledCost=null`）；归集口径 = 结算单 `SETTLED`（PENDING 未结算、
>    VOIDED 作废不计）→ 项目明细设施归属网格 → 汇总明细金额；结算表存在而查询失败按错误上抛，禁止静默吞错；
>    该网格无结算数据时 `settledCost=null`，页面显示「未登记」，禁止显示 0。
> 4. 每可装地址成本：`settledCost ÷ coverageServed`；成本未登记或分母为 0 时同样 `null`（未登记）。
> 5. 规划成本（W5 ①）：`construction_projects.budget_amount`（W6 000218，未开工项目也有预算信号）按**项目明细金额占比**
>    分摊到网格——项目级总额无行级网格维度，占比=网格内明细金额÷项目有网格归属明细合计；零归属明细（无网格设施）的
>    项目不摊，显式「未登记」不硬摊；`budget_amount` 列未部署（W6 代差）经 information_schema 探测回 null。
> 6. 材料成本（W5 ②）：CONFIRMED 出库单（1.5.14）逐台资产经采购价链（`assets.batch_id → procurement_receipts →
>    procurement_order_items` 同单同料取 id 最小一行）取采购单价合计，再按口径 5 同比例分摊；OPEN 备出库/CANCELLED 退库
>    不计；无采购价链（手建批次/编码不匹配）的资产不计额；**与已结算人工成本分列（settledCost/materialCost 两列），禁止混算**。
> 7. 城市卷积（W5 ④）：`GET /odn/city-investment` 网格行按 (prv,city) 向上卷积（与网格行同源 Go 层卷积，逐格一致）；
>    成本列为城市内网格行已登记值合计（全未登记→null）。
> 8. 容量户级口径（W5 ③，home-passed/潜在户数）：一条二级分光端口=一户；潜在户数 potentialHomes=Σ二级分光器容量
>    +Σ无二级链的一级分光器容量（直达户）；已接户数 connectedHomes=Σ已用端口占用（链行端口标签去重）；可扩户数
>    expandableHomes=潜在−已接；`costPerPotential`=(规划+材料+已结算合计)÷潜在户数（审查 F1「每户成本分母不是覆盖
>    户数」）；容量住设备维度、设备无网格维度，**只进城市行（城市域设备）与全网/设备视图（/odn/split-capacity）**，
>    不进网格行，不按比例分摊到网格（顺序五 W3 收口接通导入链与网格/设施后另行裁定）。

| 页面列名 | API 字段 | 聚合源 | 枚举/说明 |
|:---------|:---------|:-------|:----------|
| 网格 | `prvCode` / `cityPrefix` / `gridCode` / `gridName` | odn_grid 复合主键全量行 | 无数据计 0；网格标签 `城市-网格码` |
| 规划 / 施工中 / 在网 / 退役 | `facilitiesPlanned` / `facilitiesInBuild` / `facilitiesInService` / `facilitiesRetired` | odn_facility.lifecycle_status 计数 | 口径 1；枚举见 1.5.3（000198） |
| 可装 / 规划在建 / 未覆盖 | `coverageServed` / `coveragePending` / `coverageUnserved` | address_coverage.status 计数 | 口径 2；枚举见 1.5.7（000197） |
| 已结算工程成本 | `settledCost` | W1 结算数据 | null=未登记（禁止显示 0） |
| 规划成本 | `plannedCost` | W6 项目预算按明细金额占比分摊 | 口径 5；null=未登记（禁止显示 0） |
| 材料成本 | `materialCost` | W8 CONFIRMED 出库采购价按同比例分摊 | 口径 6；与人工成本分列不混算 |
| 每可装地址成本 | `costPerServed` | settledCost ÷ coverageServed | null=未登记（分母 0 同） |

城市卷积行（GET /odn/city-investment，W5）：

| 页面列名 | API 字段 | 聚合源 | 枚举/说明 |
|:---------|:---------|:-------|:----------|
| 城市 | `prvCode` / `cityPrefix` / `gridCount` | 网格行卷积 | 口径 7 |
| 设施数 / 覆盖地址数 | 同网格行四+三列 | 网格行合计 | 口径 1/2 |
| 已结算 / 规划 / 材料成本 | `settledCost` / `plannedCost` / `materialCost` | 网格行已登记值合计 | 全未登记→null |
| 每可装地址成本 | `costPerServed` | 城市结算合计 ÷ 城市可装合计 | null=未登记 |
| 潜在 / 已接 / 可扩户数 | `potentialHomes` / `connectedHomes` / `expandableHomes` | 城市域分光容量卷积 | 口径 8；null=城市无容量建模 |
| 每潜在户数成本 | `costPerPotential` | 全口径成本合计 ÷ potentialHomes | 口径 8；分母不是覆盖户数 |

设备分光容量视图（GET /odn/split-capacity，W5，只读）：行=容量模型设备（1.5.15），列
`deviceId/code/kind/splitLevel/ratio/chainRows/usedPorts/expandable/prvCode/cityPrefix/lifecycleStatus/hasSecondary`
（prv/city null=导入域设备）+ `summary`（全网 devices/potentialHomes/connectedHomes/expandableHomes，含导入域设备）。

### 1.5.12 odn_resource_chain（ODN 资源链批量导入，迁移 000210，internal/domain/odn）

> P-INFRA-1 W3:需求源 docs/ODN网络资源表模板.xlsx(说明/ODN资源/枚举 三 sheet),一行=一条完整资源链(23 列),导入展开为系统关系数据:链上箱体编码(OCC/ODB/OBD/SDB/SBD;ODF 须规范格式)逐级取/建 odn_device 导入域设备(无城市,uq_odn_device_box,迁移 000209)。管理面 `menu:odn`,REST `/odn/resource-chains/import|patrol`、`GET /odn/resource-chains`(契约 admin/odn.yaml);导入中心历史 kind=odn-resource-chain;设备类型字典扩展见 1.5.5(000209)。
> 说明页六规则即导入校验:①层级断链拒绝(OCC→ODB→OBD→一级分光→SDB→SBD→二级分光);②说明页约定前 N 数据行为示例行,过滤不导入;③留空=不入库该字段(NULL),枚举未知值拒绝,禁止默认值推断;④资源状态留空/规划态一律 PLANNED,绝不当作已安装/在网;⑤总分光比=一级×二级自动计算,填报不一致拒绝该行并报行号;⑥指纹(23 列归一化 SHA-256)唯一去重并报告。
> 可观测:逐行 imported/failed/skipped_example/duplicate+原因,失败留 `[odn-import] ROW n FAILED` 可 grep 日志;完成后 `GET /odn/resource-chains/patrol` 孤儿/半链巡检(orphan_box_code 箱体孤儿/broken_ancestor 断祖/split_port_gap 分光半链/info_ref_missing 引用缺失,各采样行号 ≤5),链上箱体引用必须存在。
> 机房/OLT/ODF 模板编码为站点前缀引用(如 SITE001/SITE001_OLT001/SITE001_ODF001_A),非规范设备格式,按文本引用承载不展开(不猜填);纤芯编号同理(规范 5.1 "--" 仅设计图纸不入库)。裁定 adopted 2026-09-07-odn-box-types-import-chain。
> 分光比列(split1/split2/total_split)为**暂存事实**:W5 起经 `POST /odn/resource-chains/backfill-split` 回写建模至设备容量模型(1.5.15,000221);导入本身不自动回写(显式重建幂等可观测)。裁定 adopted 2026-09-07-split-capacity-investment-depth。

| 模板列 | DB 列 | 枚举/映射(枚举 sheet 为校验字典) |
|:-------|:------|:--------------------------------|
| 资源状态 | lifecycle_status | 规划→PLANNED/已安装·已测试→IN_BUILD/在用→IN_SERVICE/已报废→RETIRED/留空→PLANNED |
| 机房编码/机房名称 | site_code/site_name | 文本引用(如 SITE001),不猜填 |
| OLT设备编号 | olt_code | 文本引用(如 SITE001_OLT001) |
| ODF编号/ODF端口 | odf_code/odf_port | odf_code 规范格式(ODF+3 位)则展开设备,否则文本引用 |
| OCC编号/ODB编号/OBD编号 | occ_code/odb_code/obd_code | 展开导入域设备;层级 ODB←OCC、OBD←ODB |
| 一级分光比/一级分光端口 | split1_ratio/split1_port | 枚举 1:2~1:128(存分母 SMALLINT);前置 OBD |
| SDB编号/SBD编号 | sdb_code/sbd_code | 展开导入域设备;层级 SDB←ODB(经一级分光端口)、SBD←SDB |
| 二级分光比/二级分光端口 | split2_ratio/split2_port | 枚举同上;前置 SBD |
| 总分光比 | total_split | =一级×二级自动计算;填报不一致拒绝该行 |
| 光缆/纤芯编号 | fiber_code | 文本引用(规范 5.1 不入库段落实体) |
| FR/TO标签 | fr_to | 文本(跳纤两端标签,规范 4.4) |
| 端口状态 | port_status | 可用→IDLE/已使用→USED/已预留→RESERVED/已封锁→DISABLED |
| 敷设方式 | laying_method | 架空→AERIAL/地下→UNDERGROUND/海缆→SUBMARINE/微沟槽→MICROTRENCH/室内→INDOOR |
| ROW状态 | row_status | 未开始→NOT_STARTED/待处理→PENDING/已批准→APPROVED/已过期→EXPIRED/不适用→NA |
| PECE状态 | pece_status | 待签署→PENDING_SIGN/已签署→SIGNED/已盖章→STAMPED/不适用→NA |
| 备注 | remark | 文本 |
| (批次/行号/指纹) | batch_no/line_no/fingerprint | 指纹=23 列归一化 SHA-256,唯一去重 |
| (网格归属,可选,W3 收口 F2) | prv_code/city_prefix/grid_code | 三列一体:省份码(≤6)/城市前缀(≤5)/网格码 1~99,任一非空即三者必须齐全;带归属的链行导入联动备案网格+网格锚点设施(见下注) |

> 网格归属联动（W3 收口 F2，迁移 000222，审查 F2；裁定 adopted 2026-09-07-infra-w3c-import-linkage）：
> 链行可选携带城市/网格归属。带归属的链行导入同事务：① 备案网格 `odn_grid`（(prv,city) 须在
> odn_city_code 已登记，未登记拒绝该行不猜填；网格名=机房名称或「导入网格-<grid>」，既有网格不动）；
> ② 建/复用网格锚点设施 `odn_facility` kind=MH（编码 MH+2 位网格码+3 位序号，规范 4.1 网格分区模式，
> 名称=机房名称或「链锚-<首箱体码>」，lifecycle 随链行）——投资测算网格行设施计数（1.5.11 口径 1）非空，
> 覆盖登记挂锚点设施码即进该网格行覆盖计数与覆盖页（/odn/coverage*）；③ 指纹并入归属
> （归属=资源关系一部分；不携带归属的行指纹与 000210 完全一致，存量去重语义零变化）。
> 既有五类地理空间设施与城市域设备零改动；W5「城市域优先、导入域兜底」容量规则不受影响
> （容量住设备维度，锚点设施不参与容量模型）。机房/OLT/ODF 仍按文本引用承载（站点前缀引用不展开，不猜填）。
### 1.5.13 odn_permits（ROW 路权与 PECE 许可单，迁移 000211，internal/domain/odn）

> P-INFRA-1 W4（审查 F3 提级）：ROW/PECE 从资源链导入列升为独立工作流实体，可关联施工项目/设施/资源链，
> 含证照档案要素（批复号/管辖机构/有效期/附件引用）。状态机登记 terms.md §4；管理面 `menu:odn`
> （许可与路权页 `/oss/permits`，perm 复用 odn；施工项目详情页有许可关联入口）；
> REST `/odn/permits*`、`/odn/constructions/{id}/permits|permits-check`（契约 admin/odn.yaml）。
> 开工门控（F3）：BOSS_ODN_PERMIT_GATE=on 时项目开工前置校验每类许可达标（ROW=APPROVED 且在有效期；PECE=STAMPED；NA=不适用），
> 未满足 40900+缺失明细拒绝；APPROVED 且 valid_until 已过自动回写 EXPIRED；过期复验走 EXPIRED→PENDING→APPROVED。默认关，与覆盖门控同模式。
> 竣工覆盖联动（F6，BOSS_ODN_ACCEPT_COVERAGE_LINK 默认开，off 关闭）：项目 ACCEPTED 事务内关联设施地址覆盖 PENDING→SERVED，联动数随竣工审计透出；裁定 adopted 2026-09-07-odn-permit-workflow 与 2026-09-07-accept-coverage-linkage。

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 许可单号 | `PermitNo` | permit_no | VARCHAR(32) 唯一（PM-YYYYMMDD-NNNNN，后端生成兜底） |
| 类型 | `Kind` | kind | ROW 路权 / PECE；建单初始状态 ROW=NOT_STARTED、PECE=PENDING_SIGN |
| 名称 | `Title` | title | 可空 |
| 批复号 | `ApprovalNo` | approval_no | ROW 批准时必填回填（transition approvalNo） |
| 管辖机构 | `Authority` | authority | 可空 |
| 有效期 | `ValidFrom`/`ValidUntil` | valid_from/valid_until | DATE 可空；ROW 批准必填（起缺省当日）；valid_until 过期由开工门控自动回写 EXPIRED |
| 状态 | `Status` | status | terms.md §4 状态机；CHECK 随 kind 分域 |
| 关联项目 | `ProjectID`/`ProjectNo` | project_id/project_no | FK construction_projects(id) + 单号快照；可空，link/unlink 维护 |
| 关联设施 | `FacilityCode` | facility_code | → odn_facility(code)，可空 |
| 关联资源链 | `ChainID` | chain_id | 软引用 odn_resource_chain(id)，无 FK |
| 证照附件 | `AttachmentIDs` | attachment_ids | BIGINT[]，attachments(id) 引用（档案页可选附件管理器） |
| 备注/驳回原因 | `Note`/`RejectReason` | note/reject_reason | 驳回/退回/作废原因必填场景见状态机 |

### 1.5.14 odn_asset_registrations / odn_material_issues（资产化凭证与材料出库，迁移 000215，internal/domain/odn）

> P-INFRA-1 W8（审查 F7 资产转固阻塞闭环；裁定 adopted 2026-09-07-odn-asset-capitalization）：odn×asset 关联=桥表软引用
（跨域不加 FK，不动 assets/odn_facility/odn_device 既有 schema）。凭证行=转固事实（登记号/价值/来源/项目与批次溯源），
登记事务内资产置 DEPLOYED+asset_lifecycles 轨迹；冲销 REVERSED 留历史。管理面 `menu:odn`（ODN 设施/设备页资产列+登记入口，
施工项目详情页出库入口），REST `/odn/assets/registrations*`、`/odn/material-issues*`（契约 admin/odn.yaml）。

odn_asset_registrations（资产化凭证，桥表）：

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 凭证号 | `RegistrationNo` | registration_no | VARCHAR(32) 唯一（ZG-YYYYMMDD-NNNNN，后端生成兜底） |
| 对象类型 | `EntityKind` | entity_kind | FACILITY 设施（facility_code 必填）/ DEVICE 设备（device_id 必填，含 W3 导入域箱体）；CHECK 二选一 |
| 资产 | `AssetID` | asset_id | BIGINT 软引用 assets；登记前置资产 IN_STOCK/IN_TRANSIT，一资产至多一张 ACTIVE 凭证 |
| 来源 | `SourceKind` | source_kind | PROCUREMENT 采购入库 / CONSTRUCTION 施工建成（project_id 必填且项目 ACCEPTED）/ DIRECT 直购直转 |
| 施工项目 | `ConstructionProjectID` | construction_project_id | 软引用 construction_projects，可空 |
| 入库批次 | `BatchID` | batch_id | 采购溯源快照，登记时自 assets.batch_id 回填 |
| 转固价值 | `ValueAmount` | value_amount | NUMERIC(14,2) ≥0 |
| 状态 | `Status` | status | ACTIVE / REVERSED（冲销原因必填；冲销时资产仍 DEPLOYED 则回 IN_STOCK+轨迹） |
| 登记人/时间 | `RegisteredBy`/`RegisteredAt` | registered_by/registered_at | → accounts / TIMESTAMPTZ |

odn_material_issues / odn_material_issue_items（材料出库，台账连续性载体；成本归集归 W9）：

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 出库单号 | `IssueNo` | issue_no | VARCHAR(32) 唯一（MI-YYYYMMDD-NNNNN，后端生成兜底） |
| 施工项目 | `ProjectID`/`ProjectNo` | project_id/project_no | 软引用 construction_projects + 单号快照 |
| 状态 | `Status` | status | OPEN 备出库（资产不动）/ CONFIRMED 已出库（明细资产 IN_TRANSIT 在途，000216）/ CANCELLED（CONFIRMED 取消=退库回 IN_STOCK） |
| 出库明细 | `AssetIDs` | odn_material_issue_items.asset_id | 逐台资产（issue_id,asset_id 唯一）；建单前置资产 IN_STOCK，确认/取消任一状态漂移即整体回滚 |
| 备注/操作人 | `Remark`/`CreatedBy`/`IssuedBy`/`CancelledBy` | 同名 | remark ≤255；操作人 → accounts |

### 1.5.15 odn_device_split_capacity（设备分光容量模型，迁移 000221，internal/domain/odn）

> P-INFRA-1 W5（审查 F1 分光比建模；裁定 adopted 2026-09-07-split-capacity-investment-depth）：
> W3 资源链暂存分光比（1.5.12）回写建模为**设备级容量事实**，承载「PON 口还能接几户」容量视角。
> 唯一写路径 `POST /odn/resource-chains/backfill-split`（menu:odn，审计 odn.resource-chain.backfill-split，
> 幂等全量重建，失败留 `[odn-split-backfill] FAILED|UNRESOLVED` 可 grep 日志）；读路径
> `GET /odn/split-capacity`+城市卷积（1.5.11）零写。设备解析：城市域同名**唯一**设备优先
> （容量随城市维度进城市行），跨城同名歧义不建模（不猜填），无城市域登记再退导入域影子设备
> （prv_code IS NULL，只进全网/设备视图）。

| 字段名 | DB 列 | 枚举/说明 |
|:-------|:------|:----------|
| 设备 | `DeviceID` | device_id PK → odn_device(id) ON DELETE CASCADE |
| 分光级别 | `SplitLevel` | split_level：1=一级分光器(OBD) / 2=二级分光器(SBD)；按箱体类型裁定，非行内标签 |
| 容量 | `Ratio` | ratio=分光比分母（2~128）=端口容量；同设备多链行不一致时取 MAX（容量是物理上限） |
| 链行数 | `ChainRows` | chain_rows=参与建模的链行数（有分光比的行） |
| 已用端口 | `UsedPorts` | used_ports=链行端口标签去重计数（空标签不计；不猜填，不解析标签格式） |
| 下挂二级 | `HasSecondary` | has_secondary=BOOL_OR(链行带 SBD)；户级口径据此跳过下挂二级链的一级器 |
| 更新时间 | `UpdatedAt` | updated_at TIMESTAMPTZ（重建时间） |

> 户级口径：一条二级分光端口=一户；潜在户数=Σ二级容量+Σ无二级链的一级容量；已接=Σ已用端口；可扩=潜在−已接；
> total_split（一级×二级）为链行暂存事实**不参与汇总**（两级相加会重复计数，裁定见 adopted note）。
> odn_port（1.5.9）仍为订单流物理端口模型（IDLE/RESERVED/IN_SERVICE），与容量模型互补不相混：
> 链行端口状态（可用/已使用/已预留/已封锁）语义不落 odn_port（枚举不同域，不强行映射）。

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
| 下单时间 | `CreatedAt` | created_at | 下单时刻；admin 订单列表列（fmtTime 上海墙钟展示） |
| 区域 | `RegionPath` | region_path | LTREE；下单时由地址推导快照 |
| 归属公司 | `LegalEntityID` | legal_entity_id | BIGINT → legal_entities；由安装地址推导（address→region→最近覆盖祖先，migrations/000076），下单快照不可变；未匹配子公司覆盖时兜底平台总公司（is_platform，migrations/000077）；调用方直传值仅做冲突校验（adopted note 2026-08-20-order-legal-entity-by-address） |
| 成交价 | `PriceSnapshot` | price_snapshot | 下单时生效价快照（账单金额以此为准） |
| 付费方式 | `BillingMode` | billing_mode | PREPAID/POSTPAID（000102）；下单时客户选定快照，环节 6 建 LO 账号时继承到 lo_accounts；默认 POSTPAID（adopted note 2026-08-22-prepaid-postpaid-billing-mode） |
| 预缴月数 | `BuyMonths` | buy_months | 0~60（000104）；0=按月缴（环节 4 收 1 个月月费），N>0=预缴 N 月（环节 4 收 N×月费，区域覆盖口径同出账） |
| 赠送月数 | `GiftMonths` | gift_months | 0~60（000104）；环节 4 收款时按 gift_duration_rules 阶梯命中回填（如 6送1/12送3/24送6，取 ≤预缴月数的最大档），未命中为 0 |

> lo_accounts 同名列 `billing_mode`（000102）：订购关系上的付费模式权威态；PREPAID 客户不进月度出账（GenerateBills 过滤），预付费在环节 4 合同收费当场收款落缴费流水。
> LO 生效套餐对齐（adopted 2026-09-01-provision-correctness-followup）：`lo_accounts.offer_id`/`billing_mode` 是"当前生效套餐"权威态，环节 6 幂等复用已有 LO 时若与订单套餐不一致，自动对齐到订单套餐（TMF change order 语义）并打 `[order] LO OFFER REALIGN` 留痕——保证环节 7 按新套餐下发模板、RADIUS 按新档限速。
> LOID 接入凭据（000194，A1）：`lo_accounts.password_credential` 落库密文（`v1$gcm$…` AES-256-GCM、密钥外置；空=未设密，默认 Reject，`BOSS_AAA_ALLOW_NO_CRED` 为迁移缓冲开关默认关）；管理端 `POST /lo-accounts/{loid}/reset-password`（menu:loaccount）重置，明文一次性返回；防爆破锁定表 `lo_auth_lockouts`（fail_count/locked_until，默认 5 次锁 15 分钟，可配）。
> 合同月数存档（000202，存量开户导入）：`lo_accounts.contract_months` SMALLINT 可空；存量"月数"入档权威列，`orders.buy_months` 只服务订单态、存量不建历史订单（2026-09-07 裁定 notes/adopted/2026-09-07-legacy-vlan-on-ports）。

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

> 完成时间缺失（finished_at 为 NULL：环节进行中或未回填）时，admin 时间轴显示 i18n 占位文案（zh「未完成」），不显示裸 —。
> 写入口径（2026-09-03 T17）：推进成功（result=DONE）即写 finished_at=now()，与 CheckResource 自愈 UPDATE 同源；PENDING/DOING（等待/失败/进行中）保持 NULL。环节1 下单成功即 DONE（下单完成时刻=创建时刻）。历史行仅回填可从权威痕迹表精确推导者（下单→orders.created_at；预下发配置→provision_tasks.created_at），其余保持 NULL，见 scripts/ops/stage-finished-at-backfill.sql。

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
| 类型 | `Type` | type | 白名单 ONU/ROUTER/OLT（P4-T2 写入白名单，白名单外 42200；权威码 ONU，`光猫`方言经 000191 归一；展示冗余，权威=model_id→asset_models.category） |
| 型号 | `ModelID` | model_id | BIGINT → asset_models（可空，000187+P1-T3） |
| 入库批次 | `BatchID` | batch_id | BIGINT → asset_batches |
| 部署地址 | `AddressID` | address_id | BIGINT → addresses（可空，未部署为空） |
| 状态 | `Status` | status | IN_STOCK/IN_TRANSIT/DEPLOYED/MAINTENANCE/SCRAPPED（见 terms.md 第 4 节；IN_TRANSIT=W8 000216 出库在途） |
| 序列号 | `SN` | sn | 可空 text；全网唯一（部分唯一索引 uq_assets_sn，000188；存量不回填不强制） |
| MAC 地址 | `MAC` | mac | 可空 text；冒号/横杠/裸 hex 三形态输入，写入归一为大写冒号规范形 `AA:BB:CC:DD:EE:FF`（000190，与应用层 NormalizeMAC 及表达式唯一索引 upper(regexp_replace(mac,'[:. -]','','g')) 同一归一空间；000188 建列） |
| LOID | `LOID` | loid | 可空 text；电信 LOID 鉴权标识（uq_assets_loid，000188） |

> 页面 asset.html 的「标签编号/EPC 码」经 `tag_id → tags` 反查展示，「位置」= `address_id`，「生命周期」= `status`。
> 状态轨迹（TS 实体）：`asset_lifecycles`，资产每次状态/位置变更一行，含事发时 `address_id` + `address_name` 快照 + `changed_at`，历史不随当前状态漂移。
> 区域/企业锚点（TS 实体）：`region_id`/`region_name`（部署地址所在经营区域，未部署为空）、`legal_entity_id`/`legal_entity_name`（企业），按地区/企业统计资产。
> 区域快照回补（W3 收口，2026-09-07）：存量开户导入 347 台资产快照为空（regionId=0）。
> `POST /assets/region-backfill`（menu:asset，幂等可重跑，审计 asset.region-backfill）按「地址行级节点归属推导」
> 回补：① 部署地址节点自身 `region_id`，为空沿地址链向上取最近非空祖先（000076/000077 建址继承同口径）；
> ② 地址链全空归属时，存量开户导入批（asset_batches.name='存量开户导入'）回退该批既有缺省区域
> root 集团（与该批 customers/lo_accounts 同域，见 adopted 2026-09-07-legacy-vlan-on-ports 决策 4）；
> ③ 其余无归属不猜填（保持 0，计 skipped）。仅回补空快照行，重跑零更新；失败留
> `[asset-region-backfill]` 可 grep 日志。
> 数据质量闸门（000184，adopted 2026-09-06-asset-tag-quality-gate）：`status` 列 DB CHECK 枚举兜底
> （assets 四态 / tags 三态，terms.md §4 为权威）；`assets.type` 000184 仅拦空串，P4-T2（000191+）
> 起写入走应用层白名单 ONU/ROUTER/OLT（`internal/domain/asset/type_whitelist.go`，白名单外
> 42200；巡检层同名 SQL 白名单见 `scripts/ops/db-patrol-gate.sh` ASSET-TYPE-UNKNOWN 查，双闸同步维护）。
> 类型归一（000191，Lead 裁定 2026-09-06）：权威类型码 ONU，存量 49 行 `光猫` UPDATE 归一（down
> 刻意为空，方言归一不可逆）；MI-ONU/SMOKE 系 e2e 残留不入类型体系，由
> `scripts/ops/clean-asset-type-residue.sh` 按「无标签绑定/领用/换新/盘点/四码引用才删」清理。
> 建档即留痕（000184 同批）：CreateAsset 与采购入库确认（ConfirmReceipt）同事务落
> `asset_lifecycles` 首行（初始 status，changed_at=now()）；CreateAsset/CreateTag 写侧事务化，
> 双绑回填冲突（ErrBindingConflict）时整单回滚，不再产生半成品孤儿。
> 装机/拆机资产联动（000185+P1-T1，adopted 2026-09-06-asset-tag-p1-wave）：环节9 扫码 MATCH
> 时资产原子置 DEPLOYED+绑地址+落轨迹（与四码 LINKED/扫码日志同事务，失败回滚阻断扫码）；
> 拆机（UnbindRequireScan）置 IN_STOCK+清地址；重装复用时旧件释放/新件部署；SCRAPPED 终态
> 拒绝自动联动（ErrAssetScrapped 转人工）。存量漂移由 000185 补账 + 巡检「LINKED but asset
> not DEPLOYED」「SCRAPPED but tag still bound」两查兜底（只报不修）。
> 标签绑定事件流（000186+P1-T2）：`tag_events`（tag_id,asset_id,action
> BIND/UNBIND/RECYCLE,actor_account_id,detail,changed JSONB;event_id UUID 唯一,append-only）。
> 历史回填（000189+P3-T4）：零事件资产各补一条 CREATE 事件（tag_id=0 哨兵）、
> 在绑但缺当前绑定对 BIND 事件的标签各补一条 BIND 事件；changed 带 backfill=000189
> 来源标记（down 仅删回填行；NOT EXISTS 守卫幂等重跑零新增）；查询 action 白名单含 CREATE。
> CreateTag/CreateAsset 绑定成功即写 BIND；端点 POST /tags/{tagId}/unbind（预期不符 40900、
> 未绑定 ErrTagUnbound→40000 族）写 UNBIND；POST /assets/{assetId}/scrap（reason 必填,终态
> 幂等；P3-F 起另收 confirmAssetCode/confirmSn/confirmTagNo 三要素,服务端强校验防绕过前端:
> 编码须精确相等;有 SN 时 confirmSn 必填相等、无 SN 须空串;已绑标签时 confirmTagNo 必填
> 等于标签号、未绑须空串;任一不符 42200 且信息只指明要素不回显现值;审计只落 SN 尾 4 位）
> 强制解绑写 RECYCLE——报废软回收禁硬删（adopted 2026-09-06-asset-tag-p1-wave、P3-F 同日）。

> 标签事件消费面（P2-T4，2026-09-06）：查询端点 GET /tags/{tagId}/events 与
> GET /assets/{assetId}/events（id 倒序；limit 缺省 50 上限 100；action 多值白名单
> BIND/UNBIND/RECYCLE 过滤，白名单外 42200；before_id 游标预留只取更小 id，首版 UI 不用；
> 标签/资产不存在 40400，统一信封 items 返回）。前端：标签页操作列（既有 Dropdown 组件
> 体系，禁用原生 select）提供 解绑/报废绑定资产/事件记录，资产页操作列提供 状态轨迹/
> 事件记录/报废；解绑/报废二次确认三要素=影响面清单+不可逆/恢复路径说明+红色确认键默认
> 禁用（原因必填；P3-F 起报废升级三要素确认=资产编码+SN/标签号按有无动态必填,弹窗展示三要素参考值核对实物铭牌且参考值不可复制,提交前本地预校验,服务端同规则强校验）；确认框影响面口径与事后可查回的事件字段
> 对齐（解绑→UNBIND 事件、报废→RECYCLE 事件+SCRAPPED 轨迹行，原因均入事件 detail）；
> 事件时间轴抽屉一行一事件（时间/操作人/动作徽标/对象），changed JSONB 只渲染实际变化键
> （键: 旧值 → 新值，等宽字体），不 dump 全量 JSON、不引 diff 库。明确不做：全局事件
> 大屏、游标分页 UI、全文搜索、SSE/轮询推送、导出。
> admin 台账 CRUD 四端点（P2-W1-T1）：POST /assets（建档：batchId 必填缺失 42200，
> 企业归属快照自批次回填与采购入库同口径；状态固定 IN_STOCK；modelId 可选须存在且在用，
> type 缺省由型号类别派生；tagId 可选绑定写 BIND，已被其他资产占用 40900 整单回滚；
> assetCode 可空，缺省服务端按 A-{批次8位}-{序号5位} 生成）+ GET /assets/{assetId}
> （详情含企业/区域快照，未命中 40400）+ PUT /assets/{assetId}（受限编辑：
> 类型/型号/标签/批次 四键 + sn/mac/loid 身份三要素（000188+P3-T2：建档接收，空串存 NULL，
> SN/LOID 去首尾空格，MAC 冒号/横杠/裸 hex 三形态均可输入、写入归一为大写冒号规范形（000190）；编辑指针语义缺省=保持、空串=清除；
> 表达式唯一索引冲突 40900 且 message 含冲突字段名，MAC 非法 42200），标签换绑同事务 UNBIND+BIND 冲突 40900 整单回滚，批次仅
> IN_STOCK 态可改并同步企业快照，四键无变化幂等成功；审计记变更前后键值）+
> DELETE /assets/{assetId}（守卫删除：仅 IN_STOCK 且无标签绑定/持有台账/换新单/
> 盘点明细/四码关联引用可物理删，命中任一引用 40900 且 message 列全阻断项；
> SCRAPPED 一律拒绝硬删提示走报废端点；审计附资产编码）。状态与部署地址不经编辑
> 端点变更，一律走业务流转（装机扫码/报废/换新）。
> 全表管理端点（P2-W2-T1，2026-09-06）：POST /tags（建标签：编号+EPC+频段必填且唯一
> （tags_tag_no_key / tags_epc_code_key 冲突 40900），状态缺省 UNBOUND，法人必填且须存在；
> EPC 写路径校验（000188+P3-T2）：24 位 hex 大小写不敏感入库统一大写，头部字节须在
> 30/32/33/35 内（SGTIN/GRAI/GIAI/GID-96），违者 42200）+
> POST /tags/{id}/disable（停用：仅 UNBOUND，BOUND 必须先解绑 40900；已停用幂等；
> DISABLED 标签全绑定链路拒绝——建档绑签/编辑换绑前置查状态）+ POST /tags/{id}/enable
> （启用：仅对 DISABLED 生效回 UNBOUND，非停用态幂等成功）+ GET /tags/{id}/events
> （事件流：append-only 只读回放，BIND/UNBIND/RECYCLE 按时间倒序）+ PUT /asset-models/{id}
> （编辑：vendor/model/category/partNumber/spec 五键；四元组冲突 40900；停用型号拒绝编辑
> 40900 提示先启用）+ POST /asset-models/{id}/disable|enable（is_active 直改；停用不物理删，
> 引用由 model_id 承载；幂等）+ POST /asset-batches（建批次：名称+法人必填，批次编码缺省
> RK-YYYYMMDD-NNNNN 自动生成对齐采购入库；法人不存在 42200）+ POST /asset-assignments
> （领用：资产+师傅+事由必填；仅 IN_STOCK 可领用 40900；落持有台账开段 effective_to=NULL；
> 领用【不改】资产状态——装机扫码(000185)才置 DEPLOYED，领用是台账事实）+
> POST /asset-assignments/{id}/return（归还：闭合段 effective_to=now；重复归还 40900）+
> POST /replacements/{id}/cancel（取消：仅 PENDING → CANCELLED 终态，非 PENDING 40900）。
> 以上写操作均写审计（数据变更/状态变更；target=tag/asset_model/asset_batch/
> asset_assignment/replacement）。

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
| 外层 VLAN | `Svlan` | svlan | INTEGER 可空；NULL=未配置（000202 建列 SMALLINT，000203 放宽 INTEGER） |
| 内层 VLAN | `Cvlan` | cvlan | INTEGER 可空；NULL=未配置（000202，000203 放宽） |
| Internet 内层 VLAN | `InternetCvlan` | internet_cvlan | INTEGER 可空；NULL=未配置（000202，000203 放宽：存量源值达 10 万级） |
| TR069 内层 VLAN | `TR069Cvlan` | tr069_cvlan | INTEGER 可空；NULL=未配置（000202，000203 放宽） |
| 存量光缆层级 | `LegacyPath` | legacy_path | TEXT 可空；OCC06/ODB040/OBD01/P05 单列无损承接（000202） |

> 状态变更历史（TS 实体）：`port_change_history`，端口每次状态/占用变化一行（变更后 status + order_id 快照 + changed_at），历史不随当前状态漂移。
>
> 扩容单写侧（P5 补齐，`POST /expansions/{expansionNo}/execute|reject`，permCode menu:transfer）：
> execute=`PENDING→DONE` 单请求内完成——按 `expectedPorts` 在目标设备（resourceId 必填，须归属扩容单同一法人）批量建端口，
> 端口码 `P-<设备码去横杠>-<序号>` 续号（跳过已占码），`quad_code` 初始=端口码（四码关联建立时细化），状态 IDLE 入池，
> 法人/区域名按扩容单 ID 现查落快照；已建满则直接完成（created=0）；单口失败整单留 PENDING（已建端口保留，重试续建）。
> reject=`PENDING→DONE` 终态。台账页只读不变：端口的预占/占用仍由订单状态机管理，扩容执行只产 IDLE 端口。

### 4.2.0 PON 端到端链路反查视图（P5-W2，派生只读，无新表新列）

> GET /api/admin/v1/ports/{portId}/path（portId=端口 ID 纯数字或端口编码；permCode menu:resource）。
> 沿 `ports.resource_id`（→resources 归属）与 `resources.parent_id`（→上级）派生 **端口→分光器→PON口→OLT** 逐跳物理链路；
> 端口直挂 OLT（归属资源 type=OLT）时链路两跳终止；断链只落 missing 跳 + 断点原因码，**禁止猜链补链**。

| 页面列名 | 字段名 | 来源 | 枚举/说明 |
|:---------|:-------|:-----|:----------|
| 跳序 | `seq` | 派生 | 1 起物理顺序 |
| 类型 | `kind` | 派生 | PORT / SPLITTER / PON_PORT / OLT（资源跳取 resources.type 实际值） |
| 编码 | `code` | ports.port_code / resources.code | PON 跳为 PONID `NA-<框>-<槽>-<口>`（组装口径同 §4.4） |
| 名称 | `name` | resources.name | PON/缺失跳为空 |
| 状态 | `status` | ports.status / resources.status | 枚举见 terms.md 第 4 节；PON_PORT 无独立状态恒空 |
| 占用 | `occupiedBy` | ports.order_id LEFT JOIN orders.order_no | 仅 PORT 跳，`{kind:"order", id, orderNo}`；空闲为空 |
| 断点 | `missing` + `reason` | 派生 | missing=true 时 code/status 为空 |
| 链路完整 | `complete` | 派生 | 无断点且末跳为 OLT |

> 断点原因码（reason，只描述库里缺这条边，不做推断）：`PORT_SPLITTER_MISSING`（孤儿端口：resource_id 无对应资源行）/
> `SPLITTER_NO_PARENT`（分光器无上级，无 OLT 归属）/`PON_PORT_UNASSIGNED`（端口未配 PON 框/槽/口，pon_* 为 NULL）/
> `OLT_UNREACHABLE`（归属资源缺失无法上溯）。端口不存在返回 40400（resource: not found），不返回断链视图。
> 区域/企业锚点（TS 实体）：`region_id`/`region_name`（地址所在经营区域）、`legal_entity_id`/`legal_entity_name`（所属设备企业），按地区/企业统计端口；`lo_accounts` 同挂 `region_id`/`region_name`（客户所在经营区域）。

### 4.2.2 资源台账稽核 inventory-audit（P5-W3，GET /inventory-audit + 每日快照 period=oss-audit-daily）

| 类别标识 | 中文 | 检查码 | 口径 |
|:---------|:-----|:-------|:-----|
| `ownership` | 归属断裂 | PORT_SPLITTER_MISSING | 端口引用的分光器不存在 |
| `ownership` | 归属断裂 | SPLITTER_UPSTREAM_MISSING | 分光器引用的上游不存在（parent 缺失/悬空） |
| `state` | 状态机违例 | USED_PORT_NO_QUAD_LINK | USED 态端口无四码 LINKED 关联 |
| `state` | 状态机违例 | RESERVED_PORT_STALE | RESERVED 态超阈值未推进未释放（阈值 biz_params `resource.audit.reservedStaleHours` 默认 48 小时，000193） |
| `coding` | 编码规范违例 | RESOURCE_CODE_BAD | 设备编码不符 `OLT-*`/`SPL-*`（按 type 前缀强校验，中缀横杠合法，如 OLT-MNL-01） |
| `coding` | 编码规范违例 | PORT_CODE_BAD | 端口编码不符 `P-<设备码去横杠>-<序号>`（如 SPL-01 → P-SPL01-01）或 `P-<设备码全码>-<序号>`（如 OLT-MNL-01 → P-OLT-MNL-01-01）任一形态 |

> 只读只报不修，不改状态机语义；`GET /inventory-audit?category=`（menu:resource）返回 counts/items/total，明细每检查采样 ≤50 条、计数为全量；每日 04:00 循环落 `report_snapshots(period=oss-audit-daily)` 同日覆盖，失败留 `[oss-audit] ... FAILED` 可 grep 日志。

### 4.2.1 stocktakes / stocktake_items（盘点任务与差异明细，迁移 000007/000156，S10 流程）

stocktakes（盘点任务，页面 `/ams/stock`「盘点管理」）：

| 页面列名 | 字段名 | DB 列（约定） | 枚举/说明 |
|:---------|:-------|:--------------|:----------|
| 盘点任务 | `ID` | id | BIGSERIAL PK |
| 所属公司 | `LegalEntityID` | legal_entity_id | BIGINT → legal_entities（建单快照范围） |
| 范围 | `Scope` | scope | `全库`/空=主体全部资产;`ODN`=网络资产专项(W8 000215:快照=有 ACTIVE 资产化凭证的在网资产,桥表只读);否则 region_name 精确匹配 |
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

### 4.2.2 资源容量视图与阈值预警（P5-W1，迁移 000192，对标 NRM C6）

> admin 页面 `/oss/capacity`（oss 分组，权限复用 `menu:resource`——任务书裁定容量视图对齐端口台账权限，不新增权限码，check-contract-sync E 项 baseline 豁免登记）。
> API：`GET /api/admin/v1/resources/capacity?dim=OLT|SPLITTER&order=usageDesc|usageAsc`（dim 空=全部类型；order 缺省 usageDesc，即任务书「按使用率倒序」）；
> `POST /api/admin/v1/resources/capacity/alert-scan`（手动跑一轮阈值预警，响应 `{scanned,created,resolved}`）。

容量聚合行（页面列名 ↔ API 字段，来源 resources LEFT JOIN ports）：

| 页面列名 | 字段名 | 口径 |
|:---------|:-------|:-----|
| 对象 | `code` / `name` | resources.code + resources.name |
| 类型 | `type` | OLT / SPLITTER（复用 resources.type 枚举） |
| 总端口 | `totalPorts` | 该对象 ports 全部行数（含 DISABLED） |
| 占用 | `usedPorts` | status='USED' 行数 |
| 使用率 | `usageRate` | USED/(USED+IDLE)×100，百分比两位小数；DISABLED/RESERVED 不计入分母；分母为 0 时=0.00 |

阈值预警语义（复用告警体系，terms.md §4 alarm.level=WARNING 不新增枚举）：

| 项 | 值 | 说明 |
|:---|:---|:-----|
| 告警来源 | `alarms.source='capacity'` | source 枚举新增第四值（device/quadlink/aaa/capacity），常量 device.AlarmSourceCapacity |
| 阈值 | 使用率 >= 80% | resource.CapacityWarnThresholdPct=80.0 |
| 级别/状态 | WARNING / OPEN | 越限且无 OPEN 容量告警才产生 |
| 状态变化 | 越限产生 / 回落自动关闭 | 同一对象同一阈值状态变化才重复告警；周期重跑幂等不重复 |
| 判重反查 | 部分索引 idx_alarms_capacity_open | 迁移 000192：alarms(resource_id) WHERE source='capacity' AND status='OPEN' |
| 触发入口 | 巡检循环（每小时）+ POST alert-scan | 失败路径落 `[resource-capacity] ALERT SCAN FAILED` 可 grep 日志，禁静默 |

### 4.3 replacements（换新单/设备更换单，000007 + 000159 派单三列）

| 页面列名 | 字段名 | DB 列 | 枚举/说明 |
|:---------|:-------|:------|:----------|
| 更换单 | `ReplacementNo` | replacement_no | RPL-YYYYMMDD-序列，后端生成 |
| 设备 | `AssetID` | asset_id | BIGINT 软引用 assets（被更换资产） |
| 原因 | `Reason` | reason | 如 光猫故障 |
| 优先级 | `Priority` | priority | HIGH/MEDIUM/LOW |
| 状态 | `Status` | status | PENDING/DOING/DONE/FAILED/CANCELLED（见 terms.md 第 4 节；PENDING 可取消，CANCELLED 终态，P2-W2-T1） |
| 派单师傅 | `WorkerID`/`WorkerName` | worker_id/worker_name | 000159；worker_id FK→workers，name 快照（0/空=未派） |
| 完成时间 | `FinishedAt` | finished_at | TIMESTAMPTZ 可空；DONE/FAILED 时回填 |

> 状态机：PENDING --assign(派单,POST /admin/replacements/{id}/assign)→ DOING --complete(师傅端 POST /api/worker/v1/replacements/{id}/complete)→ DONE/FAILED；PENDING --cancel(取消,POST /admin/replacements/{id}/cancel,P2-W2-T1)→ CANCELLED；终态不可再流转（adopted note 2026-08-27-replacement-ticket-flow）。
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

### 4.5 OSS 建号写接口（POST /resources，T2 存量导入配套）

| 端点 | 必填 | 缺省 | 错误语义 |
|:-----|:-----|:-----|:---------|
| POST /resources | code/type/addressId/legalEntityId | status=ONLINE（白名单 ONLINE/OFFLINE/FAULT） | type 白名单外/必填缺失 42200；code 撞 resources.code UNIQUE → 40900（resource.ErrDuplicate）；parentId/addressId 不存在 42200（ErrForeignKeyViolation） |

> 挂 `menu:resource`；type 仅收 OLT/SPLITTER。端口批量建口由权威 POST /provision/ports 承载（扩容会话 b806d645），POST /ports 不再单设。T1 迁移（000202）的 VLAN/PON 扩展列本期不接收，payload struct 留扩展位（加字段透传即可）。审计：RecordAudit target=resource。

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
4. **归属口径（事发时）**：每单/每评价按发生那一刻的班组归属；工单跨班组时记「派单时班组」，评价跟随工单。事件级事实表 `worker_materials`/`worker_tools`/`asset_returns` 的 `group_id` 同口径 = 记录创建（领用/退回/发放发生）那一刻师傅所在班组；事后调组不回改历史行（db-design-review D8）。
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
| provision_logs | admin/provlog.html 下发日志 | task_id/resource_id/template_id/result/retries/created_at；详情(000179)：commands/device_response(完整指令与设备应答,GET /provision-logs/{logId} 聚合任务/订单/模板维度)；详情(000180)：driver(驱动来源 log/telnet/tl1,SUCCESS 语义随通道不同,审计可溯源) |
| real_name_verifications | admin/customer.html 实名核验 | customer_id/method/verified_at/result/operator_account_id/operator_name |
| channels | order/dispatch 下单渠道 | code/name/status（REQ-ORD-006 必填不可改） |
| alarms | admin/alarm.html 告警 | alarm_no/level/source/content/status |
| cdrs | admin/aaalog.html 话单 | loid/session_time/input_output_octets/billing_status |
| auth_logs | admin/aaalog.html 认证日志 | loid/result/fail_reason/created_at；fail_reason 枚举：BAD_CREDENTIAL/LOCKED/NOT_FOUND/SUSPENDED/CLOSED（空=SUCCESS 或存量行；000194 A1 认证失败原因码，LOCKED=连续失败锁定窗口内） |
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
>
> 2026-09-04 消息列表口径（客户端链路修复）：用户端 `GET /messages` 的 `messageId` 恒为
> portal_messages 数字主键（单条已读 `PUT /messages/{messageId}/read` 的寻址键），payload
> 快照的 MSG- 展示编号降级为 `messageNo` 展示；`read`/`createdAt` 以表列为单一事实源，
> 快照同名键不覆盖（单条已读/read-all/home 红点三者一致）。

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

> 2026-09-04 兑换码业务码口径（客户端链路修复）：`POST /coupons/redeem` 码不存在 →
> 40400「兑换码不存在」；码已用/停用 → 40900「兑换码已被使用或已停用」；模板停用 →
> 40900「券模板已停用」；限领/超发冲突 → 40900 透传原因（替代修复前一律 50000）。

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
| 承建类型 | `ContractorType` | contractor_type | MATERIAL 材料类 / CONSTRUCTION 施工类（terms.md §4；000205；存量默认 MATERIAL 语义不变；暂无 admin 页面列，经供应商 API 入参/回包承载，ODN 施工页承包商下拉按 CONSTRUCTION 过滤） |
| 资质信息 | `Qualification` | qualification | VARCHAR(255) 可空，施工类专有（等级/编号/有效期自由文本；000205） |

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

### 9.8 采购域端点口径（P2-W2-T2 增补，零 DDL）

| 端点 | 口径 |
|:-----|:-----|
| PUT /procurement/suppliers/{id} | 编辑供应商：名称/联系人/电话/备注可改，编码不可改（入参无 code 字段）；部分更新语义（字段省略=null 保持原值）；禁用态同样可改资料且保持禁用；未命中 40400 |
| POST /procurement/suppliers/{id}/enable | 启用：DISABLED→ENABLED；已 ENABLED 幂等成功；不存在 40400；写审计（状态变更） |
| GET /procurement/orders/{id} | 采购单详情：单头+明细行+状态；未命中沿用 40400 |
| PUT /procurement/orders/{id} | 草稿编辑：仅 DRAFT（否则 40900）；备注/期望日期可改（null=保持）；items 非 null 即整体替换且行 quantity>0（违者 42200）；总金额随明细重算；审计记变更前后键值 |
| POST /procurement/receipts/{id}/reject | 入库驳回：仅 DRAFT→REJECTED（CONFIRMED 等非 DRAFT 40900）；原因 ≤255 字（42200）可空，空则缺省文案「入库驳回」并回写 receipt.remark；写审计（状态变更+原因） |

> 错误码沿用全局：40400 not found / 40900 状态冲突 / 42200 参数非法。
> 明细整体替换仅限 DRAFT：SUBMITTED 之后 received_qty 参与收货对账，整行删除会破坏已收数量口径。

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
## 8H. 月度填报事实域(internal/domain/monthly,迁移 000181,BI 经营分析)

> 权威输入:docs/books/模板_月度填报.xlsx(_RegionList 51 个标准 Barangay)+ 同目录 3 个标准 CSV
(UTF-8 with BOM,CRLF 行尾,表头带单位后缀)。粒度均为 月×区域,UNIQUE(month,region);
派生列=模板灰色列「公式-勿填」,服务端计算(库端 GENERATED ALWAYS 存储列),导入/接口传入一律无效。
admin API 前缀 /api/admin/v1/monthly/*,权限码 menu:monthly;导入留痕复用 import_tasks(kind=monthly-<table>)。

### 8H.1 monthly_regions(区域白名单)

| 页面列名 | 字段名 | DB 列 | 类型/枚举 |
|:---------|:-------|:------|:----------|
| 区域 | `Region` | region | TEXT PK;51 Barangay 种子(xlsx _RegionList 提取) |
| 状态 | `Active` | active | BOOLEAN 停用即拒绝导入 |

### 8H.2 monthly_user_revenue(用户与收入)

| 页面列名 | 字段名 | DB 列 | 类型/枚举 |
|:---------|:-------|:------|:----------|
| 月份 | `Month` | month | TEXT,CHECK `^[0-9]{4}-(0[1-9]|1[0-2])$` |
| 区域 | `Region` | region | TEXT FK monthly_regions |
| 期初在用 (户) | `OpeningActive` | opening_active | BIGINT ≥0 |
| 当月新增 (户) | `NewUsers` | new_users | BIGINT ≥0 |
| 当月离网 (户) | `ChurnedUsers` | churned_users | BIGINT ≥0 |
| 数据调整 (户) | `AdjustedUsers` | adjusted_users | BIGINT ≥0 |
| 期末在用 (户) | `ClosingActive` | closing_active | 派生=期初+新增-离网+调整(GENERATED) |
| 宽带收入 (₱) | `BroadbandRevenue` | broadband_revenue | BIGINT ≥0 |
| 增值收入 (₱) | `ValueAddedRevenue` | value_added_revenue | BIGINT ≥0 |
| 一次性收费 (₱) | `OnetimeCharge` | onetime_charge | BIGINT ≥0 |
| 优惠减免 (₱) | `DiscountAmount` | discount_amount | BIGINT ≥0 |
| 退款冲销 (₱) | `RefundReversal` | refund_reversal | BIGINT ≥0 |
| 主营总收入 (₱) | `TotalRevenue` | total_revenue | 派生=宽带+增值+一次性-优惠-退款(GENERATED) |

### 8H.3 monthly_network_delivery(网络与交付)

| 页面列名 | 字段名 | DB 列 | 类型/枚举 |
|:---------|:-------|:------|:----------|
| 月份 | `Month` | month | 同 8H.2 |
| 区域 | `Region` | region | TEXT FK monthly_regions |
| 装机申请 (件) | `InstallRequests` | install_requests | BIGINT ≥0 |
| 及时完工 (件) | `OntimeCompletions` | ontime_completions | BIGINT ≥0 |
| 部署端口 (个) | `PortsDeployed` | ports_deployed | BIGINT ≥0 |
| 在用端口 (个) | `PortsActive` | ports_active | BIGINT ≥0 |
| 故障申报 (件) | `FaultReports` | fault_reports | BIGINT ≥0 |
| 修复工时 (小时) | `RepairHours` | repair_hours | BIGINT ≥0 |

### 8H.4 monthly_finance_cost(财务与成本)

| 页面列名 | 字段名 | DB 列 | 类型/枚举 |
|:---------|:-------|:------|:----------|
| 月份 | `Month` | month | 同 8H.2 |
| 区域 | `Region` | region | TEXT FK monthly_regions |
| 开票金额 (₱) | `InvoicedAmount` | invoiced_amount | BIGINT ≥0 |
| 实际回款 (₱) | `CollectedAmount` | collected_amount | BIGINT ≥0 |
| 期末应收 (₱) | `ReceivableEnding` | receivable_ending | BIGINT ≥0 |
| 直接成本 (₱) | `DirectCost` | direct_cost | BIGINT ≥0 |
| 固定费用 (₱) | `FixedCost` | fixed_cost | BIGINT ≥0 |
| CAPEX投入 (₱) | `CapexInvest` | capex_invest | BIGINT ≥0 |

### 8H.5 汇总 KPI(GET /monthly/summary?month=)

| 指标 | JSON 字段 | 口径 |
|:-----|:----------|:-----|
| 期末在用合计 | `closingActiveTotal` | SUM(closing_active) |
| 主营总收入合计 | `totalRevenueTotal` | SUM(total_revenue) |
| 及时完工率 | `ontimeRate` | SUM(ontime_completions)÷SUM(install_requests) |
| 端口利用率 | `portUtilization` | SUM(ports_active)÷SUM(ports_deployed) |
| 回款率 | `collectionRate` | SUM(collected_amount)÷SUM(invoiced_amount) |
| ARPU | `arpu` | SUM(total_revenue)÷SUM(closing_active) |

> 四个比率的分母为 0 时该字段返回 null(不报错);month 为空=全部月份合计。

## 8I. AAA 在线会话域(internal/domain/aaa,迁移 000195,AAA-A2)

radacct 模式在线会话:计账 Start 建(重复 Start 幂等去重)/Interim 累加流量/Stop 关闭;
承载并发会话上限(G4)、CoA 强制下线(G5,RFC 5176)与僵尸清理(G6)。admin 页面挂
`oss/loaccount.html` 会话抽屉(权限码沿用 `menu:loaccount`),路由 `GET /aaa/sessions`、
`POST /aaa/sessions/{sessionId}/disconnect`。

### 8I.1 aaa_online_sessions(在线会话)

| 页面列名 | 字段名 | DB 列 | 类型/枚举 |
|:---------|:-------|:------|:----------|
| 会话ID | `SessionID` | session_id | VARCHAR(64);(loid, session_id) 唯一 |
| LOID | `Loid` | loid | VARCHAR(32) → lo_accounts |
| NAS IP | `NasIP` | nas_ip | VARCHAR(64);CoA 下发目标 |
| 开始时间 | `StartedAt` | started_at | TIMESTAMPTZ |
| 最近更新 | `LastUpdate` | last_update | TIMESTAMPTZ;僵尸判定依据(超 2h 可配) |
| 下行流量 | `InputOctets` | input_octets | BIGINT;Interim 累加 |
| 上行流量 | `OutputOctets` | output_octets | BIGINT;Interim 累加 |
| 状态 | `Status` | status | **ONLINE 在线 / PENDING_OFFLINE 下线待确认(重试中) / OFFLINE 已下线(终态) / OFFLINE_FAILED 下线失败(重试耗尽,终态)** |
| 重试次数 | `DisconnectAttempts` | disconnect_attempts | INT;Disconnect 已重试次数(上限默认 3) |
| 关闭原因 | `CloseReason` | close_reason | ACCT_STOP / COA_DISCONNECT / ZOMBIE_REAP / OFFLINE_FAILED;空=在途 |
| 关闭时间 | `ClosedAt` | closed_at | TIMESTAMPTZ 可空 |

### 8I.2 关联增列(迁移 000195 同对)

| 表 | 字段名 | DB 列 | 枚举/说明 |
|:---|:-------|:------|:----------|
| cdrs | `CloseReason` | close_reason | VARCHAR(32) 可空;本地补录话单原因标记(僵尸清理=ZOMBIE_REAP) |
| auth_logs | `FailReason` | fail_reason | 见 §8A(000194,NOT NULL DEFAULT '');A2 追加枚举值 CONCURRENT_LIMIT(并发会话超限 Reject) |

### 8I.3 配置项(全局,env)

| 配置 | env | 默认 | 说明 |
|:-----|:----|:-----|:-----|
| 并发会话上限 | `BOSS_AAA_SESSION_LIMIT` | 1 | 同一 LOID 在线占用上限(ONLINE+PENDING_OFFLINE 计入) |
| CoA/DM 端口 | `BOSS_AAA_COA_PORT` | 3799 | NAS 动态授权端口(RFC 5176) |
| 重试上限 | `BOSS_AAA_OFFLINE_RETRY_MAX` | 3 | Disconnect 不可达重试次数,耗尽转 OFFLINE_FAILED |
| 僵尸阈值 | `BOSS_AAA_ZOMBIE_AFTER` | 2h | last_update 超时判僵尸并补录 Stop 话单 |


## 8J. AAA per-NAS 注册表与厂商 VSA 限速(internal/domain/aaa,迁移 000196,AAA-A5)

按来源 IP 注册 NAS 客户端(FreeRADIUS clients.conf/nas 表惯例):RADIUS 服务端按请求
来源 IP 查表校验共享密钥,未注册/停用一律拒绝并留 `[aaa]` 告警;CoA/强制下线
使用目标 NAS 自己的密钥与端口。admin 路由 `GET/POST /aaa/nas`、`GET/PUT/DELETE /aaa/nas/{id}`
(权限码沿用 `menu:loaccount`)。

### 8J.1 aaa_nas_clients(NAS 客户端)

| 页面列名 | 字段名 | DB 列 | 类型/枚举 |
|:---------|:-------|:------|:----------|
| ID | `ID` | id | BIGSERIAL |
| 名称 | `Name` | name | VARCHAR(64);告警留痕与列表展示 |
| NAS IP | `NasIP` | nas_ip | VARCHAR(64);请求来源 IP,唯一约束 uq_aaa_nas_clients_ip |
| 共享密钥 | `Secret` | secret_enc | TEXT;密文 v1$gcm$(与 A1 凭据同体系),仅写不回显 |
| 厂商 | `Vendor` | vendor | **HUAWEI 华为 / ZTE 中兴 / GENERIC 通用(不下发 VSA)** |
| CoA 端口 | `CoAPort` | coa_port | INT;默认 3799(RFC 5176),可按设备改配 |
| 启用 | `Enabled` | enabled | BOOLEAN;false=认证/计费/CoA 一律拒绝 |
| 创建/更新时间 | `CreatedAt`/`UpdatedAt` | created_at/updated_at | TIMESTAMPTZ |

### 8J.2 厂商限速 VSA 映射(带宽模板 → Access-Accept 属性)

带宽模板串(product_offers.bandwidth,如 100M/500M/1000M)解析为 kbps,按认证请求来源
NAS 的厂商下发整数限速 VSA;厂商不匹配/无映射/带宽不可解析一律回退现状 FramedPool
带宽串(行为不回退)。属性对可配,不写死:

| 厂商 | Vendor-ID | 默认上行/下行属性 | 配置(env,格式=上行属性名,下行属性名) |
|:-----|:----------|:------------------|:--------------------------------------|
| HUAWEI | 2011 | 78 Huawei-Input-Average-Rate / 80 Huawei-Output-Average-Rate(kbps) | `BOSS_AAA_VSA_HUAWEI`(默认 input-average-rate,output-average-rate) |
| ZTE | 3902 | 84 / 86(镜像华为布局;以设备 RADIUS 私有属性规范为准,上线前必须核对) | `BOSS_AAA_VSA_ZTE` |

可选属性名(语义名=类型码):HUAWEI input-average-rate=78 / input-peak-rate=79 /
output-average-rate=80 / output-peak-rate=81;ZTE input-peak-rate=83 /
input-average-rate=84 / output-peak-rate=85 / output-average-rate=86。

### 8J.3 全局密钥兼容开关与迁移路径

| 配置 | env | 默认 | 说明 |
|:-----|:----|:-----|:-----|
| 全局密钥兼容 | `BOSS_AAA_GLOBAL_SECRET_COMPAT` | 关 | 开启后未注册 NAS 的 RADIUS 报文与 CoA 下发回退全局密钥 `BOSS_AAA_SECRET` 与 `BOSS_AAA_COA_PORT`;**停用(enabled=false)NAS 不回退,一律拒绝** |

## 8K. AAA 建号写接口(POST /lo-accounts,internal/domain/aaa,T2 存量导入配套)

| 项 | 契约 |
|:---|:-----|
| 必填 | loid/customerId/offerId/qosTemplateId（缺失 42200） |
| 唯一 | loid 全局唯一:预查 + DB `lo_accounts.loid` UNIQUE 双保险 → 40900（aaa.ErrDuplicate,reason 含 loid） |
| 引用 | offer 须 PUBLISHED（非在售 40900 ErrOfferNotPublished）;qos_template 软引用须存在/法人存在 → 42200（ErrForeignKeyViolation） |
| 缺省 | status 固定 ACTIVE;billing_mode 可选缺省 POSTPAID(白名单 PREPAID/POSTPAID,白名单外 42200);法人缺省平台总公司(is_platform,000077 兜底链),region_id 缺省 0、region_name 空(导入裁定,设计 §3) |
| 门禁 | `menu:loaccount`;审计 target=lo_account |
| 实现 | `aaa.LoAccountAdminService.CreateLoAccountChecked`(PGStore 扩展);环节 6 既有 CreateLoAccount 链路不受影响;billing_mode 继承逻辑(客户最近订单)仅在直调 CreateLoAccount 时生效 |

> 与 §3.1 LO 契约的关系:本节是管理端建号入口,订单环节 6 建号仍走 order.UserProfileCreator(§3.1 offer 对齐语义不变)。

迁移路径(全局密钥退役,不中断业务):

1. 逐台登记 NAS(名称/来源 IP/厂商/共享密钥/CoA 端口),设备侧同步换密钥;
2. 开启兼容开关灰度(存量未登记设备不断),观察 `[aaa] NAS COMPAT GLOBAL SECRET` 与
   `[aaa] NAS REJECT UNREGISTERED` 日志,逐台登记直至无未注册来源;
3. 全部登记后关闭兼容开关;`BOSS_AAA_SECRET` 自此仅为兼容开关的回退项,不再作为正式认证凭据。
