# 数据从属关系总览（contract/data-relations）

> 版本 V1.1（2026-08-17）｜权威源：migrations(阶段1)、api/openapi/admin.yaml、fields.md(mock 层已于 2026-08-19 移除)
> 定位：锁死「谁包含谁、谁归属于谁」的完整实体关系清单。详情页设计、数据权限裁剪一律以本表为准。

## 0. 四条铁律

1. **后台管理的核心 = 账号(人) 的四种能力**：能管理什么数据、能看到什么界面、能点什么按钮、能访问什么接口。
   四层一致收敛：菜单(menuperm) / 按钮(权限码) / 接口(后端鉴权) / 数据(region_scope)。
2. **数据权限只按区域 LTREE 子树裁剪**：账号挂 `region_scope`（空=全集团），业务数据带 `region_path`，
   可见 = region_path ∈ region_scope 子树。组织链只做归属展示，不参与权限判定。
3. **同一份列表数据，两个进入维度**：平铺视角（主菜单，全量+自由筛选）；下钻视角（详情页进入，
   预置不可改的归属条件：这个市/客户/师傅/部门/子公司/岗位的数据）。
4. **详情页 = 归属链可视化 + 多维度条件数据聚合**；下钻面板非弹框；取不到的数据显示 "—"，禁止臆造。

## 1. 分层总图

```
权限主体层  roles ←→ permissions(权限码) ←→ accounts(账号, 挂 region_scope)
组织展示层  legal_entities → departments → posts →(post_roles)→ accounts
区域权限层  regions(LTREE 1集团/2大区/3省/4城市)
地址挂接层  addresses(LTREE 1市/2区/3街道/4小区/5楼栋)
客户业务层  customers → orders(12环节) → dispatch/dismantles/callbacks
                        → bills → payments；arrears/stop-resume/reconciliations
资产资源层  assets ↔ tags(标签)；quad_link(资产-客户-端口-地址)
            resources(OLT/分光器) → ports → reserves；transfers/expansions/lo-accounts
师傅执行层  workers → tickets/materials/tools/schedules/performances/commissions/feedbacks/returns
日志流水层  audit_logs / aaa-logs / scan-logs / provision-logs（只读，挂操作主体）
```

## 2. 实体关系清单（字段以 mock 实测 + fields.md 为准）

标记：▲父外键 ■子集合 ◆关联 ✚已建库表(migrations) ⌛mock先行库待建

### 2.1 权限与组织（internal/domain/user，阶段1，全部✚）

| 实体 | 主键 | 关系 | 基数 |
|:-----|:-----|:-----|:-----|
| roles ✚ | id/code(7角色码) | ■permissions(经 role_permissions 多对多) | — |
| permissions ✚ | code | ▲无父；被角色引用 | — |
| accounts ✚ | id/username | ▲role_id ▲legal_entity_id? ▲dept_id? ▲post_id? ▲region_scope(LTREE权限) ■audit_logs | 组织 1:N 账号 |
| legal_entities ✚ | id/code(LEG-A/B/C) | ■departments ◆cross_regions(跨区经营) | 集团 1:N 子公司 |
| departments ✚ | id/name | ▲legal_entity_id ■posts | 子公司 1:N 部门 |
| posts ✚ | id/code | ▲dept_id ◆roles(post_roles 多对多) ■accounts(在岗员工) | 部门 1:N 岗位 |
| regions ✚ | id/path(LTREE) | ▲path 父节点(1集团→2大区→3省→4城市) | 树 |
| addresses ✚ | id/path(LTREE) | ▲path 父节点(1市→5楼栋)；被 customers/ports 挂接 | 树 |
| menu_perms ⌛ | — | 账号界面层：哪些菜单可见 | — |
| data_scopes ⌛ | — | region_scope 管理入口 | — |
| audit_logs ✚ | id | ▲account_id(弱引用,不FK) ◆account_name/dept_name/legal_entity_name(事发快照) ▲target_type/target_id（分区表按月） | 账号 1:N 日志 |
| biz_params ✚ | key | 全局键值 | — |
| import_tasks ⌛ | taskId | ▲account_id（谁建的导入任务） | — |

### 2.2 客户与产品（internal/domain/customer，阶段2）

| 实体 | 主键 | 关系 | 基数 |
|:-----|:-----|:-----|:-----|
| customers ⌛ | customerId | ▲legal_entity_id(归属公司) ▲address_id(挂楼栋) ■orders ■bills ■arrears ■lo_accounts ◆quad_link | 公司 1:N 客户 |
| products ⌛ | productId | ■price_history(调价记录) ■orders | — |
| 用户端档案(用户侧聚合) | customerId | ◆plans/balances/usages/invoices/messages/addresses/coupons/addon-subscriptions | 客户 1:N 各档案 |

### 2.3 订单与工单（internal/domain/order，阶段5）

| 实体 | 主键 | 关系 | 基数 |
|:-----|:-----|:-----|:-----|
| orders ⌛ | orderNo | ▲customer ▲product ▲address ▲region_path ■order_stages(12环节时间轴) ■dispatch ■dismantles ■callbacks ■reserves(端口占用) ◆scan-logs | 客户 1:N 订单 |
| dispatch pool/my-tickets ⌛ | ticketNo | ▲orderNo ▲worker(master) ◆candidates(师傅候选) | 订单 1:1 工单 |
| dispatch transfers ⌛ | orderNo | ▲orderNo ▲from_master/to_target | — |
| dismantles ⌛ | dismantleNo | ▲customer ▲assetCode ▲portCode | — |
| complaints ⌛ | ticketNo | ▲customer（客服域，跨域挂靠 boss 分组） | 客户 1:N 投诉 |
| activation-callbacks ⌛ | callbackId | ▲orderNo（订单第11环节） | 订单 1:N 回调 |

### 2.4 计费账务（internal/domain/billing，阶段5）

| 实体 | 主键 | 关系 | 基数 |
|:-----|:-----|:-----|:-----|
| bills ⌛ | billNo | ▲customerId ■payments | 客户×账期 1:1 |
| payments ⌛ | payNo | ▲customerId ◆billId | 账单 1:N 缴费 |
| arrears ⌛ | customerId | ▲customerId（应收信用域） | 客户 1:1 欠费态 |
| stop-resume-tasks ⌛ | taskId | ▲customerId ▲loid | — |
| reconciliations ⌛ | batchNo | ▲channel（渠道对账） | — |

### 2.5 资产与四码（internal/domain/{asset,quadlink}，阶段3/6）

| 实体 | 主键 | 关系 | 基数 |
|:-----|:-----|:-----|:-----|
| assets ⌛ | assetCode | ▲batchNo(入库批次) ◆tags(经 tagNo/EPC 预绑定) ◆quad_link ■lifecycle(状态轨迹) | — |
| tags ⌛ | tagNo/epcCode | ◆bound_asset(预绑定资产) | 资产 1:1 标签 |
| quad_link ⌛ | — | ▲asset ▲customer ▲port ▲addr（四码，四列各索引+唯一） | 四码 1:1 链路 |
| quad-conflicts ⌛ | conflictNo | ◆四码冲突单 | — |
| scan-logs ⌛ | — | ▲orderNo ▲master(扫码师傅) ▲scanned_tag | 订单 1:N 扫码 |
| stocktakes ⌛ | taskId | ▲scope(盘点范围) | — |
| replacements ⌛ | replacementNo | ▲device(故障设备→换新) | — |

### 2.6 网络资源（internal/domain/{resource,device}，阶段4/7）

| 实体 | 主键 | 关系 | 基数 |
|:-----|:-----|:-----|:-----|
| resources(OLT/分光器) ⌛ | deviceName | ▲legal_entity ▲address ■ports | 公司 1:N 设备 |
| ports ⌛ | portCode | ▲parent(OLT/分光器) ▲address ▲order_id(RESERVED时非空) ◆quad_code ■change-history ■reserves | 父资源 1:N 端口 |
| reserves ⌛ | reserveId | ▲portCode ▲orderId | 端口 1:N 预占记录 |
| transfers ⌛ | transferNo | ▲resourceCode ▲from_region/to_region | — |
| expansions ⌛ | expansionNo | ▲region(扩容目标区域) | — |
| lo-accounts ⌛ | loid | ▲customerName ▲productBandwidth ◆qos_template | 客户 1:1 LO 账号 |
| olt-devices ⌛ | deviceName | ▲address ■alarms(设备健康) | — |

### 2.7 师傅（跨 internal/worker，页面 worker*.html）

| 实体 | 主键 | 关系 | 基数 |
|:-----|:-----|:-----|:-----|
| worker_groups ⌛ | id/code | ▲legal_entity(公司) ◆leader(组长,1个师傅) ■workers(当前成员) ■memberships(台账) ■tickets/performances/commissions/schedules/materials/tools/feedbacks/asset-returns(班组数据) | 公司 1:N 班组 |
| workers ⌛ | workerId | ▲worker_group(班组→公司) ▲region(服务区域) ◆leader_of_groups(任组长的班组) ■memberships(归属台账) ■settings(1:1) ■tickets ■materials ■tools ■schedules ■performances ■commissions ■feedbacks ■messages ■asset-returns | 班组 1:N 师傅 |
| worker_group_memberships ⌛ | id | ▲worker_id ▲group_id(仅导航) ◆group_name/legal_entity_id/region_id/region_name(事发快照) reason/operator_account_id(追溯) effective_from/effective_to(null=至今) | 师傅 1:N 归属台账(换班组只新增) |
| worker_settings ⌛ | id | ▲worker(1:1) accepting/radiusKm/acceptTypes | 师傅 1:1 设置 |
| worker_messages ⌛ | id | ▲worker level/title/read | 师傅 1:N 消息 |
| worker-feedbacks ⌛ | feedbackId | ▲workerId ▲ticketNo ▲customerName | — |
| asset-returns ⌛ | returnId | ▲workerId ▲epc/assetNo | — |
| notices/faqs ⌛ | id | 全局（师傅端公告/FAQ） | — |

> **归属关系与月度粒度铁律**：事件级/月度级事实均以 `@ManyToOne → worker_groups` 挂 `group`（FK `group_id`）+ `group_name` 快照，`WorkerGroup` 侧 `@OneToMany` 反向；
> 月度级事实（performances/commissions/schedules）粒度=「师傅×月×班组」，唯一键 `(worker_id, period, group_id)`，月中调组拆多行；
> 「本月最佳班组」= 按 `group_id` 聚合月度行。归属口径=事发时，工单记派单时班组；`worker_settings`/`worker_messages` 不加快照。台账见 fields.md 7.2。

### 2.8 日志流水（只读，挂操作主体）

| 实体 | 主键 | 关系 |
|:-----|:-----|:-----|
| audit_logs ✚ | id | ▲account_id ▲target_type/target_id |
| aaa-logs ⌛ | logId | ▲认证账号(loid) |
| provision tasks/templates/logs ⌛ | — | tasks ▲loid ◆template；logs ▲task |

### 2.9 归属台账（跨域通用深度关联）

| 台账实体 | 主体 | 归属维度 | 反向集合 |
|:---------|:-----|:---------|:---------|
| account_org_histories ⌛ | accounts | 公司/部门/岗位 | Account.orgHistory |
| customer_histories ⌛ | customers | 公司/地址 | Customer.history |
| product_price_histories ⌛ | product_offers | 价格 | ProductOffer.priceHistory |
| region_price_histories ⌛ | region_offers | 价格 | RegionOffer.priceHistory |
| dispatch_transfers ⌛ | dispatch_tickets | 师傅(改派) | DispatchTicket.transfers |
| resource_assignments ⌛ | resources | 公司/区域/地址 | Resource.assignments |
| asset_assignments ⌛ | assets | 师傅/地址 | Asset.assignments |

> 通用：`effective_from`/`effective_to`(null=至今) + `reason` + `operator_account_id`；主体 `@ManyToOne` + `@OneToMany` 反向；归属维度 `@ManyToOne` + `xxx_name` 快照。

> **企业锚点铁律**：归属到企业的业务事实/主单（orders/dispatch_tickets/complaints/bills/worker 各事实/assets/ports/lo_accounts）冗余 `legal_entity_id` + `legal_entity_name` 快照，跨企业对比 O(1) 锚点；集团共享数据与纯时间轴子记录不冗余（见 fields.md 8.1）。

## 3. 归属维度枚举（下钻视角的"锁定条件"）

| 维度 key | 含义 | 从哪个详情页下钻 | 命中的实体（extra 条件字段） |
|:---------|:-----|:-----------------|:---------------------------|
| region | 市/区/街道/小区/楼栋 | region/address 详情 | customers, ports, orders, expansions, transfers, workers |
| customer | 某客户 | 客户详情 | orders, bills, payments, arrears, lo-accounts, quad_link, complaints |
| worker | 某师傅 | 师傅详情 | tickets, materials, tools, schedules, performances, commissions, feedbacks, asset-returns |
| legal_entity | 某子公司 | 子公司详情 | departments, accounts |
| dept | 某部门 | 部门详情 | posts, accounts |
| post | 某岗位 | 岗位详情 | accounts |
| order | 某订单 | 订单详情 | order_stages, dispatch, callbacks, scan-logs, reserves, dismantles |
| asset | 某资产 | 资产详情 | tags, lifecycle, quad_link |
| port | 某端口 | 端口详情 | change-history, reserves, quad_link |

## 4. UI 经验沉淀（后续 agent 必读）

1. **全局共享样式必须命名空间隔离**：style.css 组件类一律带前缀（`dp-` 下钻面板、`pg-` 分页、`modal-` 弹框），
   禁止裸用 `.tabs/.desc/.detail-head` 等页内高频同名类。
2. **弹框只用于表单**（新建/编辑/批量+必填校验）；**详情一律下钻**（dp-panel：返回+Tabs+描述列表）。
3. **黄金模板**：列表 billing.html、详情 worker-detail.html（不动）；共享组件 common.js，禁止重复造轮子。
4. **XSS 一律 UI.esc**；错误 UI.toast(err) 禁止裸 alert；不臆造接口（缺的写操作 confirm+toast('演示')）。
5. 单文件 ≤300 行；列名/枚举对齐 fields.md；并行 agent 每批 ≤2 防限速。
6. 恢复文件先分清 工作区/暂存区/HEAD 三层，确认"好版本"在哪层再还原。

## 5. 使用规则

1. 新增实体先查本表与 fields.md，已定外键照抄；未定的按铁律回写本表。
2. 列表页过滤函数统一 `matches(row, extra)`：平铺 extra={}；下钻 extra={维度:值} 且 Tab 头显示锁定 tag。
3. 下钻列表能力不缩水（三态/分页/行操作齐全，仅范围收窄）。
4. 本表管数据关系，domain-map.md 管域边界，fields.md 管字段名——三者冲突时先在此对齐再改码。
