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

### 1.1 accounts（后台账号）

> 固定用途：仅承载管理后台登录用户。后台员工、运营人员、管理员统一使用本表；客户 App 和师傅端不得使用 `accounts` 登录。

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

> 区域硬关联（TS 实体）：楼栋级地址挂 `region_id`（→ regions，经营区域）+ `region_name` 快照，固化「地址→经营区域」映射；客户/资产/端口/LO账号经此继承区域，杜绝「有地址无订单则不知属哪个区域」的孤儿。

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

### 1.5.5 odn_site / odn_device（局点与核心链路设备，迁移 000081，资产编码规范第 2/3 章）

> 管理面同 §1.5.3（`menu:odn`，`/odn/sites|devices`）。

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
| 唯一性 | — | uq_odn_device_city | 市域内唯一；SNW 全网唯一（部分索引） |
| 状态 | `Status` | status | IN_USE / RETIRED（报废永久锁定） |

> 校验：`ValidateDeviceCode`（格式+扩容后缀）+ `RequiredParentKind`（归属链）；子级设备上级须为同城在用设备。

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

`api_key_permission_templates` 与 `api_key_template_permissions` 提供受限 API key 权限模板；模板由平台维护，签发时仅引用 code，不保存明文密钥。

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
| 订阅事件 | `EventType` | open_webhook_subscriptions.event_type | 如 order.activated；同 app+事件+端点唯一 |
| 回调端点 | `EndpointURL` | endpoint_url | HTTPS 回调地址（M2 投递器消费） |
| 投递状态 | `Status` | open_webhook_deliveries.status | 0待投递 / 1已投递 / 2死信（超过 6 次重试，迁移 000125） |
| 幂等键 | `EventID` | open_webhook_deliveries.event_id | 同订阅+事件唯一（UNIQUE + DO NOTHING），重放不重复执行业务动作 |
| 重试次数 | `Attempts` | attempts | 失败按 30s×2^n 指数退避（封顶 1h），`NextAttemptAt` 排下次 |
| 投递结果 | `HTTPStatus` / `LastError` | http_status / last_error | 2xx 成功；非 2xx/网络错误记错误进重试 |
