# 数据从属关系总览（contract/data-relations）

> 版本 V1.2（2026-08-19）｜权威源：migrations/*.up.sql（55 个迁移，117 表，已全部对账）
> 定位：锁死「谁包含谁、谁归属于谁」的完整实体关系清单。详情页设计、数据权限裁剪一律以本表为准。
> 本版变更：⌛ 标记清零（mock 层实体全部落库）；补齐 geo/userdata/portal/税务/入驻/告警复测/对账/报表等 30+ 表；
> ER 图改为脚本生成（`scripts/gen-er-drawio.mjs` → `docs/boss-entities-er.drawio`），不再手维护。

## 0. 四条铁律

1. **后台管理的核心 = 账号(人) 的四种能力**：能管理什么数据、能看到什么界面、能点什么按钮、能访问什么接口。
   四层一致收敛：菜单(menuperm) / 按钮(权限码) / 接口(后端鉴权) / 数据(region_scope)。
2. **数据权限只按区域 LTREE 子树裁剪**：账号挂 `region_scope`（空=全集团），业务数据带 `region_path`，
   可见 = region_path ∈ region_scope 子树。组织链只做归属展示，不参与权限判定。
3. **同一份列表数据，两个进入维度**：平铺视角（主菜单，全量+自由筛选）；下钻视角（详情页进入，
   预置不可改的归属条件：这个市/客户/师傅/部门/子公司/岗位的数据）。
4. **详情页 = 归属链可视化 + 多维度条件数据聚合**；下钻面板非弹框；取不到的数据显示 "—"，禁止臆造。

## 0.5 引用完整性口径（V1.2 新增，重要）

- **硬 FK（DDL REFERENCES）** 只存在于阶段1-4 的组织/主档表之间与同域强绑定子表（order_stages、payments、worker 事实表等）。
- **软引用（无 FK，仅索引）** 是主流：orders/quad_links/lo_accounts/reserve_records/transfers/alarms/cdrs 等跨域主单
  一律 `xxx_id + 快照列` 软引用。原因：跨域写路径解耦 + 快照口径（详见 §设计问题评估 docs/review/db-design-review.md）。
- **读代码时判别**：字段注释 `→ 表名` 或本文 `sFK` = 软引用；`REFERENCES` = 硬 FK。

## 1. 分层总图

```
权限主体层  roles ←→ permissions ←→ accounts(挂 region_scope) ；api_keys/import_tasks 挂 accounts
组织展示层  legal_entities → departments → posts →(post_roles)→ accounts ；account_org_histories 台账
地理层      geo_country → geo_subdivision(树) → addresses(锚 country/admin_code)
地址挂接层  addresses(LTREE 1市/2区/3街道/4小区/5楼栋)
客户业务层  customers → orders(12环节) → dispatch/dismantles/callbacks/scan_logs/ratings
                         → bills → payments → invoices(税局网关) ；arrears/stop-resume/reconciliations
资产资源层  asset_batches → assets ↔ tags；quad_link(资产-客户-端口-地址 四码软引用)
            resources(树) → ports → reserve_records/port_change_history；transfers/expansions/alarms
师傅执行层  worker_groups → workers → tickets/materials/tools/schedules/performances/commissions/...
            worker_registrations(入驻审核) → workers
用户端层    user_*(20 表挂 customers) ；portal_*(7 表，customer_id 软挂隔离空间)
日志流水层  audit_logs(分区)/auth_logs/cdrs/provision_logs/scan_logs（只读，挂操作主体）
```

## 2. 实体关系清单（权威源：migrations，全部✚已建库表）

标记：▲父外键 ■子集合 ◆多对多/关联 ✚已建库表；FK=硬外键 sFK=软引用(无约束)

### 2.1 权限与组织（migrations 000001-3/29/38-39/42/44，全部✚）

| 实体 | 主键 | 关系 | 基数 |
|:-----|:-----|:-----|:-----|
| roles ✚ | id/UQ code(7角色码) | ■permissions(经 role_permissions M:N) | — |
| permissions ✚ | id/UQ code | ▲无父；被角色引用 | — |
| accounts ✚ | id/username | ▲role_id(FK) ▲legal_entity_id?/dept_id?/post_id?(FK) ▲region_scope(LTREE权限) ■api_keys ■import_tasks ■audit_logs(弱引用) | 组织 1:N 账号 |
| api_keys ✚ | id | ▲account_id(FK CASCADE) ▲created_by(sFK)；subject 扩展见 000045 | 账号 1:N 密钥 |
| import_tasks ✚ | taskId | ▲operator_id(FK accounts) | 账号 1:N 导入任务 |
| legal_entities ✚ | id/code | ■departments ■customers ■worker_groups ■product_offers... | 集团 1:N 子公司 |
| departments ✚ | id | ▲legal_entity_id(FK) ■posts | 子公司 1:N 部门 |
| posts ✚ | id | ▲dept_id(FK) ◆roles(post_roles M:N) | 部门 1:N 岗位 |
| account_org_histories ✚ | id | ▲account_id/legal_entity_id?/dept_id?/post_id?(FK) | 账号归属台账 |
| regions ✚ | id/path(LTREE) | 自引用树(1集团→4城市) | 树 |
| audit_logs ✚ | id | ▲account_id(弱引用) ◆事发快照列 ▲target_type/target_id（按月分区） | 账号 1:N 日志 |
| biz_params ✚ | key | 全局键值(AI 网关三键等) | — |
| menu_perms / data_scopes | — | 已并入 permissions+region_scope，无独立表 | — |

### 2.2 地理与地址（000038/40/41，全部✚）

| 实体 | 主键 | 关系 |
|:-----|:-----|:-----|
| geo_country ✚ / geo_country_i18n ✚ | alpha2 | ■subdivisions ■tz/currency/calling-code |
| geo_subdivision ✚ / geo_subdivision_i18n ✚ | code | ▲country_code(FK) ▲parent_code(自引用树，深度因国而异)；000041 预置 PH PSGC |
| country_time_zone / country_currency / country_calling_code ✚ | 复合 | ▲country_code(FK) |
| addresses ✚ | id/path(LTREE) | ▲parent_id(软，派生列反查) ▲country_code/admin_code(FK geo，CHECK 约束锚定)；被 customers/ports/resources/orders 挂接 |

### 2.3 客户与产品（000004-5/23/26/51，全部✚）

| 实体 | 主键 | 关系 | 基数 |
|:-----|:-----|:-----|:-----|
| customers ✚ | id | ▲legal_entity_id ▲address_id(FK) ■orders ■bills ■arrears ■lo_accounts(软) ◆quad_link ■user_* 全家 | 公司 1:N 客户 |
| customer_registrations ✚ | id | ▲legal_entity_id/address_id(FK) ▲customer_id(通过后回填,sFK) ▲reviewer_account_id(sFK) | 注册审核队列 |
| verifications ✚ | id | ▲subject_type('customer'/'worker') + subject_id(软)；000059 起统一实名表，吸收 000026/000050/000051 三张旧表（已迁移并 DROP） | 主体 1:N 实名轨迹 |
| customer_histories ✚ | id | ▲customer_id/legal_entity_id/address_id(FK) | 客户归属台账 |
| product_offers ✚ | id | ▲legal_entity_id(FK) ■price_history ■region_offers | — |
| region_offers ✚ | id | ▲offer_id(FK) ■price_history | — |
| channels ✚ | id/UQ code | 全局渠道目录；orders.channel_id 软引用 | — |

### 2.4 订单与工单（000010/17-18/27/31/53，全部✚）

| 实体 | 主键 | 关系 | 基数 |
|:-----|:-----|:-----|:-----|
| orders ✚ | id/UQ order_no | ▲customer ▲offer ▲address ▲channel(全软引用+legal_entity快照+region_path) ■order_stages ■dispatch(1:1) ■dismantles ■callbacks ■scan-logs ■reserve_records ■ratings | 客户 1:N 订单 |
| order_stages ✚ | id | ▲order_id(FK)；stage 1~12，状态机权威见 ADR-003 | 订单 1:12 环节 |
| dispatch_tickets ✚ | id | ▲order_id(FK UQ 1:1) ▲worker_id ▲group_id(软) | 订单 1:1 工单 |
| dispatch_transfers ✚ | id | ▲ticket_id(FK) ▲from/to_worker_id(软) | 工单 1:N 改派 |
| dismantles ✚ | id | ▲order_id(FK) ▲asset/port(软) | — |
| complaints ✚ | id | ▲customer_id(FK) ▲order_id(软) | 客户 1:N 投诉 |
| activation_callbacks ✚ | id | ▲order_id(FK)（第11环节） | 订单 1:N 回调 |
| scan_logs ✚ | id | ▲order_id(FK) ▲master/scanned_tag(软) | 订单 1:N 扫码 |
| order_ratings ✚ | id | ▲customer_id(FK) ▲order_no(UQ 软) | 订单 1:1 评价(门户提交) |

### 2.5 计费账务与税务（000011/24/35/48-49，全部✚）

| 实体 | 主键 | 关系 | 基数 |
|:-----|:-----|:-----|:-----|
| bills ✚ | id/billNo | ▲customer_id(FK) ■payments ■invoices | 客户×账期 1:1 |
| payments ✚ | id | ▲bill_id(FK) | 账单 1:N 缴费 |
| arrears ✚ | id | ▲customer_id(FK UQ 1:1) | 客户 1:1 欠费态 |
| stop_resume_tasks ✚ | id | ▲customer_id/loid(软) | — |
| reconciliation_batches ✚ | id/UQ batch_no | 渠道对账（channel 字符串，不挂 FK） ■reconciliation_items | — |
| reconciliation_items ✚ | id | ▲batch_id(FK CASCADE) ▲payment_id(软引用：渠道流水可无系统 payment，000060) | 批次 1:N 行级比对明细 |
| arn_sequences ✚ | doc_type | 发票号序列（作废保留不回收） | — |
| invoices ✚ | id/UQ invoice_no | ▲bill_id(FK) ▲customer_id(软+快照)；tax_* 列=税局网关回填（000049） | 账单 1:N 发票 |

### 2.6 资产与四码（000006-8/12，全部✚）

| 实体 | 主键 | 关系 | 基数 |
|:-----|:-----|:-----|:-----|
| asset_batches ✚ | id | ▲legal_entity_id(FK) ■assets | — |
| tags ✚ | id | ▲legal_entity_id(FK) ◆assets(预绑定 1:1) | — |
| assets ✚ | id/UQ asset_code | ▲batch_id(FK) ▲tag_id/address_id(软) ■lifecycle ■assignments ◆quad_link | — |
| asset_lifecycles ✚ | id | ▲asset_id(FK) | 资产 1:N 状态轨迹 |
| asset_assignments ✚ | id | ▲asset_id(FK) ▲worker/address(软) | 资产归属台账 |
| stocktakes ✚ / replacements ✚ | id | ▲legal_entity_id(FK) / ▲asset(软) | — |
| quad_links ✚ | id | ▲asset ▲customer ▲port ▲address（四列软引用；000056 起唯一约束改为 `WHERE status IN('LINKED','CONFLICT')` 部分唯一索引，UNLINKED 行保留为链路历史）+ legal_entity 快照 | 四码 1:1 活跃链路，1:N 历史 |

### 2.7 网络资源·监控·开通·AAA（000009/13-15/22/25/28/30/32-33/37，全部✚）

| 实体 | 主键 | 关系 | 基数 |
|:-----|:-----|:-----|:-----|
| resources ✚ | id | ▲legal_entity_id/address_id(FK) ▲parent_id(自引用树,软) ■ports | 公司 1:N 设备 |
| ports ✚ | id/UQ port_code | ▲resource_id/address_id(FK) ■change_history ■reserve_records ◆quad_link | 父资源 1:N 端口 |
| port_change_history ✚ | id | ▲port_id(FK) ▲order_id(快照) | 端口 1:N 历史 |
| reserve_records ✚ | id | ▲port_id ▲order_id(全软) | 端口 1:N 预占 |
| resource_assignments ✚ | id | ▲resource_id/legal_entity_id(FK) ▲address_id(软) | 资源归属台账 |
| transfers ✚ / expansions ✚ | id | ▲resource(软) / ▲legal_entity_id(FK) | — |
| qos_templates ✚ | id | ▲legal_entity_id(FK)；lo_accounts 软引用 | — |
| device_metrics ✚ / device_maintenances ✚ | id | ▲resource(软) | 设备健康 |
| alarms ✚ | id/UQ alarm_no | ▲resource_id(软)；source=device/quadlink/aaa | — |
| alarm_retest_tasks ✚ | id | ▲alarm(软) | 告警复测 |
| lo_accounts ✚ | id/UQ loid | ▲customer_id(1:1 软) ▲offer_id ▲qos_template_id(软) + region/legal_entity 快照 | 客户 1:1 LO |
| provision_templates ✚ | id | ▲legal_entity_id(FK) ■tasks | — |
| provision_tasks ✚ | id | ▲template_id(FK) ▲loid(软) | — |
| provision_logs ✚ | id | ▲task_id(软)；result 宽列见 000033 | 任务 1:N 日志 |
| cdrs ✚ / auth_logs ✚ | id | ▲loid(软，无 customer 直连) | AAA 流水 |

### 2.8 师傅（000016/19-21/36/50，全部✚）

| 实体 | 主键 | 关系 | 基数 |
|:-----|:-----|:-----|:-----|
| worker_groups ✚ | id | ▲legal_entity_id(FK) ▲leader_id(软) ■workers ■全部事实表 | 公司 1:N 班组 |
| workers ✚ | id | ▲group_id(FK) ■memberships ■settings(1:1) ■messages ■8 张事实表 | 班组 1:N 师傅 |
| worker_group_memberships ✚ | id | ▲worker_id ▲group_id(FK) ◆group_name 等快照 | 师傅 1:N 归属台账 |
| worker_settings ✚ / worker_messages ✚ | id | ▲worker_id(FK, settings UQ 1:1) | — |
| worker_notices ✚ | id | 全局公告 | — |
| worker_registrations ✚ | id | ▲group_id(FK) ▲worker_id(通过后回填,软) ▲reviewer_account_id(软) | 入驻审核队列 |
| 师傅实名 | — | 统一走 verifications(subject_type='worker')，见 §2.3 | — |
| worker_performances ✚ / worker_commissions ✚ / worker_schedules ✚ | id | ▲worker_id ▲group_id(FK)；UQ(worker,period,group) 月度粒度 | 师傅×月×班组 |
| worker_materials ✚ / worker_tools ✚ / worker_feedbacks ✚ / asset_returns ✚ | id | ▲worker_id ▲group_id(FK)；asset_returns 另软挂 asset | 事件级事实 |

> **归属关系与月度粒度铁律**：月度级事实粒度=「师傅×月×班组」，唯一键 `(worker_id, period, group_id)`，月中调组拆多行；
> 归属口径=事发时，工单记派单时班组；`worker_settings`/`worker_messages` 不加快照。

### 2.9 用户端 userdata（000046-47，全部✚，20+ 表）

| 实体 | 主键 | 关系 |
|:-----|:-----|:-----|
| user_accounts ✚ user_addresses ✚ user_plans ✚ user_usages ✚ user_verify_records ✚ user_bill_items ✚ | id | ▲customer_id(FK)，门户用户侧聚合 |
| ~~user_messages ✚~~ / ~~user_invoices ✚~~ / ~~user_complaints ✚~~ | id | ▲customer_id(FK)，**deprecated（裁定 D1，2026-08-20）**：双胞胎停用，读写走权威表 portal_messages / invoices / complaints |
| ~~user_notify_settings ✚~~ / ~~user_balances ✚~~ | customer_id | ▲customers(1:1, FK)，**deprecated（裁定 D1）**：读写走 portal_prefs.notify / portal_wallets；user_accounts.auto_pay 列停用（走 portal_billing_prefs） |
| addons ✚ / addon_subscriptions ✚ | id/addon_id | subscriptions ▲customer_id+addon_id(FK) |
| coupons ✚ | coupon_id | ▲customer_id(可空 FK) |
| user_faqs/invite_config/diy_guides/agreements/topup_denominations/product_specs ✚ | id | 全局目录，无外键 |

> **裁定 D1 落地（2026-08-20）**：portal(portal_*) 为用户侧唯一权威运行态；userdata 域对双胞胎表
> 降级为读权威表的薄适配层（`internal/domain/customer/userdata/`，方法已标 Deprecated），
> user_invoices 写路径停写（发票开票走 billing 域 TaxService）。

### 2.10 门户 portal（000052-55，全部✚）

| 实体 | 主键 | 关系 |
|:-----|:-----|:-----|
| portal_sms_codes ✚ | (phone,scene) | 验证码，自过期 |
| portal_accounts ✚ | phone | ▲customer_id(UQ **软**——隔离空间合成 ID 取**负数段**，000057 CHECK 禁 0；真实 customers.id 恒正) 1:1 |
| portal_prefs / portal_wallets / portal_billing_prefs ✚ | customer_id | ▲customers(软) 1:1；**用户侧唯一权威运行态（裁定 D1）**：偏好/余额/自动缴费分别取代 user_notify_settings / user_balances / user_accounts.auto_pay |
| portal_messages ✚ | id | ▲customer_id(软)，payload JSONB；取代 user_messages（裁定 D1） |
| portal_seq ✚ | kind | 门户单号序列 PAY/CHG/TKT/MSG/CUST |

> portal 域有意不建 FK：门户库可独立于 boss 主库部署（隔离空间），对账靠 customer_id 逻辑对齐。

### 2.11 报表（000034）

| 实体 | 主键 | 关系 |
|:-----|:-----|:-----|
| report_snapshots ✚ | id | UQ(period,window_start)；payload JSONB 全量快照，不挂业务 FK |

### 2.12 归属台账（跨域通用模式）

| 台账实体 | 主体 | 归属维度 |
|:---------|:-----|:---------|
| account_org_histories ✚ | accounts | 公司/部门/岗位 |
| customer_histories ✚ | customers | 公司/地址 |
| product_price_histories ✚ / region_price_histories ✚ | offers | 价格 |
| dispatch_transfers ✚ | dispatch_tickets | 师傅(改派) |
| resource_assignments ✚ | resources | 公司/区域/地址 |
| asset_assignments ✚ | assets | 师傅/地址 |
| worker_group_memberships ✚ | workers | 班组 |

> 通用：`effective_from`/`effective_to`(null=至今) + `reason` + `operator_account_id`；主体硬 FK；归属维度 FK + `xxx_name` 快照。

> **企业锚点铁律**：归属到企业的业务事实/主单冗余 `legal_entity_id` + `legal_entity_name` 快照，跨企业对比 O(1)；
> 集团共享数据与纯时间轴子记录不冗余（见 fields.md 8.1）。

## 3. 归属维度枚举（下钻视角的"锁定条件"）

| 维度 key | 含义 | 命中的实体（extra 条件字段） |
|:---------|:-----|:---------------------------|
| region | 市/区/街道/小区/楼栋 | customers, ports, orders, expansions, transfers, workers |
| customer | 某客户 | orders, bills, payments, arrears, lo-accounts, quad_link, complaints, user_*, portal_* |
| worker | 某师傅 | tickets, materials, tools, schedules, performances, commissions, feedbacks, asset-returns |
| legal_entity | 某子公司 | departments, accounts, customers, worker_groups, product_offers |
| dept / post | 部门/岗位 | posts, accounts |
| order | 某订单 | order_stages, dispatch, callbacks, scan-logs, reserves, dismantles, ratings |
| asset | 某资产 | tags, lifecycle, quad_link, assignments, asset_returns |
| port | 某端口 | change-history, reserves, quad_link |

## 4. UI 经验沉淀（后续 agent 必读）

1. 全局共享样式必须命名空间隔离（`dp-`/`pg-`/`modal-` 前缀）。
2. 弹框只用于表单；详情一律下钻面板。
3. 黄金模板：列表 billing.html、详情 worker-detail.html；共享组件 common.js。
4. XSS 一律 UI.esc；错误 UI.toast(err)；不臆造接口。
5. 单文件 ≤300 行；列名/枚举对齐 fields.md。
6. 恢复文件先分清 工作区/暂存区/HEAD 三层。

## 5. 使用规则

1. 新增实体先查本表与 fields.md，已定外键照抄；未定的按铁律回写本表，并同步 `scripts/gen-er-drawio.mjs` 规格。
2. ER 图不手改：改规格脚本 → `node scripts/gen-er-drawio.mjs` → draw.io.app 导出 png/svg。
3. 列表页过滤函数统一 `matches(row, extra)`：平铺 extra={}；下钻 extra={维度:值} 且 Tab 头显示锁定 tag。
4. 本表管数据关系，domain-map.md 管域边界，fields.md 管字段名——三者冲突时先在此对齐再改码。
5. 设计问题与风险台账：docs/review/db-design-review.md（两轮评估结论）。
