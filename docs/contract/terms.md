# 术语与状态权威字典（contract/terms）

> 版本 V1.0（2026-08-17）｜权威源：《BOSS综合业务支撑平台-需求全案》V1.2
> 定位：AI 开发时"只读不猜"的单一事实源。与他处表述冲突时，以本文件为准。

## 1. 订单业务环节（新装开通闭环，12 环节）

环节序号、名称、标识、触发条件固定，禁止增删改序。

| # | 名称 | 标识(方法/事件) | 前置 | 执行方 | 权威依据 |
|:-:|:-----|:----------------|:-----|:-------|:---------|
| 1 | 下单 | submitOrder | 客户有效 | 客户/客服 | 全案 3.3.1 |
| 2 | 资源核查 | checkResource | 订单已创建 | 系统 | 全案 3.3.1 |
| 3 | 端口预占 | reservePort | 核查有资源 | 系统 | 全案 3.3.1 |
| 4 | 合同收费 | chargeContract | 端口已预占 | 财务/客服 | 全案 3.3.1 |
| 5 | 标签预绑定 | applyTag | 已收费 | 系统 | 全案 3.3.1 |
| 6 | 创建账号 | createUserProfile | 标签已绑定 | 系统 | 全案 3.3.1 |
| 7 | 预下发配置 | preConfigOLT | 账号已创建 | 系统 | 全案 3.3.1 |
| 8 | 派单 | dispatchOrder | 配置就绪 | 调度/系统 | 全案 3.3.1 |
| 9 | 扫码绑定 | scanBind | 已派单 | 装维 | 全案 3.3.1 |
| 10 | 激活 | activateUser | 绑定成功 | 装维/系统 | 全案 3.3.1 |
| 11 | 激活回调 | notifyActivation | 激活成功 | 系统 | 全案 3.3.1 |
| 12 | 更新 GIS | updateMap | 订单完成 | 系统 | 全案 3.3.1 |

## 2. 历史矛盾裁定

历史文档曾并存"11 环节"与"12 环节"两版。**裁定：12 环节为唯一正确版本。**

- 差异点：11 环节版缺少"4 合同收费"，端口预占后直接标签预绑定。
- 裁定理由："合同收费"是硬约束（未收费不派单，见全案 REQ-CL-001），且权威源全案 V1.2 已将其固化为必经环节。
- 修改义务：任一历史文档出现"11 环节/11 个业务环节/11环节"且语义指向新装闭环时，一律按 12 环节修正，见 `terms.md` 附录 A 修改清单。

## 3. 订单状态枚举（interface 层）

| 状态码 | 中文 | 含义 | 环节关系 |
|:-------|:-----|:-----|:---------|
| PENDING | 待核查 | 订单创建待资源核查 | 1 下单完成 |
| RESERVED | 已预占 | 端口已锁定 | 3 端口预占完成 |
| INSTALLING | 装维中 | 派单后装维作业 | 8 派单 → 10 激活 |
| DONE | 已完成 | 激活回调后 | 11 激活回调完成 |
| CANCELLED | 已取消 | 订单取消/退订（任一未完成环节前） | — |

> 环节序号(1-12)与订单状态(PENDING/RESERVED/INSTALLING/DONE/CANCELLED)是两套正交枚举：环节表达"进行到第几步"，状态表达"订单当前状态"。禁止混用。

## 4. 通用状态枚举

| 域 | 状态码集 | 说明 |
|:---|:---------|:-----|
| 端口 port.status | IDLE / RESERVED / USED / DISABLED | IDLE 可预占；RESERVED 事务行锁兜底 |
| 资产 asset.status | IN_STOCK / DEPLOYED / MAINTENANCE / SCRAPPED | 生命周期状态 |
| 四码 quad_link | LINKED / CONFLICT / UNLINKED | 一致性对账用 |
| 标签 tag.status | UNBOUND / BOUND / DISABLED | 电子标签 |
| 报障 complaint.status | OPEN / PROCESSING / CLOSED | 受理中 / 处理中 / 已关闭 |
| 扫码 scan_log.result | MATCH / MISMATCH / OFFLINE_CACHED | 一致 / 不一致 / 离线缓存 |
| 激活回调 result | SUCCESS / FAILED | 成功 / 失败（RETRYING 重试中为展示态，由 FAILED+重试派生） |
| 激活(环节10) status | PENDING / SUCCESS / FAILED | 待激活 / 成功 / 失败（师傅端视图，非回调结果） |
| 消息 level | INFO / WARN / URGENT | 信息 / 警告 / 紧急 |
| 缴费 method | wechat / alipay / card / cash / offline | 微信 / 支付宝 / 银行卡（线上网关/柜面 POS 收单）/ 现金 / 线下收款（师傅现场当面收）。2026-09-05：师傅现场 CASH/QR/POS 统一记 offline，CARD 记 card。2026-08-28 柜面归类裁定（纪要 2026-08-28-柜面现金收款）：按资金入账通道归类不按物理动作——柜面现金记 cash、柜面扫码记 wechat/alipay、柜面 POS 记 card；offline 专属师傅个人代收，柜面不得占用；人员归因由 payments 网点/操作员列承载（正交） |
| 告警 alarm.level | CRITICAL / WARNING / INFO | 严重 / 警告 / 提示 |
| 告警 alarm.status | OPEN / ACKED / CLOSED | 待处理 / 已确认 / 已关闭 |
| 话单 cdr.billing_status | UNBILLED / BILLED | 未入账 / 已入账 |
| 认证日志 auth_log.result | SUCCESS / FAILED | 成功 / 失败 |
| 认证日志 auth_log.fail_reason | BAD_CREDENTIAL / LOCKED / NOT_FOUND / SUSPENDED / CLOSED（空=成功或存量行） | 认证失败原因码（000194，A1）；LOCKED=连续失败达阈值锁定窗口内（默认 5 次锁 15 分钟，可配），期间一律拒绝 |
| 设备 resource.status | ONLINE / OFFLINE / FAULT | 在线 / 离线 / 故障 |
| 认证账号 lo_account.status | ACTIVE / SUSPENDED / CLOSED | 在服 / 停服 / 注销 |
| 在线会话 aaa_online_sessions.status | ONLINE / PENDING_OFFLINE / OFFLINE / OFFLINE_FAILED | 在线 / 下线待确认(Disconnect 未确认,重试中) / 已下线(终态) / 下线失败(重试耗尽,终态);迁移 000195(AAA-A2,fields.md §8I) |
| NAS 客户端 aaa_nas_clients.vendor / enabled | HUAWEI / ZTE / GENERIC;true / false | 设备厂商(VSA 限速下发依据,GENERIC 不下发)/ 停用=认证、计费、CoA 一律拒绝;迁移 000196(AAA-A5,fields.md §8J) |
| 产品 product_offer.status | DRAFT / PUBLISHED / OFFLINE | 草稿 / 在售 / 下架 |
| 客户 service_status | ACTIVE / ARREARS / SUSPENDED | 在网 / 欠费 / 停机 |
| 客户 real_name_status | VERIFIED / PENDING | 已实名 / 待补登 |
| 派单工单 dispatch_ticket.status | PENDING / DOING / DONE / CANCELED | 待派 / 进行中 / 完成 / 取消 |
| 任务 task.status | PENDING / DOING / DONE / FAILED | 待执行 / 进行中 / 完成 / 失败 |
| 换新单 replacement.status | PENDING / DOING / DONE / FAILED / CANCELLED | 待执行 / 执行中 / 完成 / 失败 / 已取消（PENDING --派单→ DOING --完成→ DONE/FAILED；PENDING --取消→ CANCELLED(P2-W2-T1,2026-09-06)；终态不可再流转，adopted note 2026-08-27-replacement-ticket-flow） |
| 缴费流水 payment.status | SUCCESS / FAILED / REFUNDED | 成功 / 失败 / 已退款 |
| 发票 invoice.status | ISSUED / VOIDED | 已生成 / 已作废（编号保留不回收，TAX-004） |
| 发票 invoice.tax_status | PENDING / SUBMITTED / ISSUED / FAILED | 税局网关状态：待开具 / 已提交 / 已开具（税局票号回填）/ 失败可重试；与 invoice.status 正交 |
| 设备健康 priority | MUST_REPLACE / SUGGEST / WATCH | 必须更换 / 建议 / 观察 |
| 券模板 template.status | DRAFT / ENABLED / DISABLED | 草稿 / 启用 / 停用 |
| 券实例 coupon.status | ISSUED / USED / EXPIRED / DISABLED | 已发放 / 已使用 / 已过期 / 已停用；user 端展示态 available/used/expired/gifting 由其派生 |
| 券类型 coupon.type | FULL_CUT / DISCOUNT / CASH | 满减 / 折扣(万分比) / 代金券 |
| 券来源 coupon.source | ADMIN_ISSUE / CAMPAIGN / REDEEM / GIFT / INVITE / LOYALTY | 手发 / 活动 / 兑换码 / 转赠 / 邀请 / 积分兑换 |
| 兑换码 code.status | UNUSED / REDEEMED / DISABLED | 未兑换 / 已兑换 / 已停用 |
| 赠送规则 gift_rule.status | ENABLED / DISABLED | 启用 / 停用（时长阶梯 6送1/12送3/24送6） |
| 积分流水 entry.reason | ADMIN_ADJUST / EXCHANGE / EXCHANGE_REVERSAL | 手动调整 / 积分换券扣减 / 发券失败补偿回补 |
| 订购付费模式 billing_mode | PREPAID / POSTPAID | 预付费（办单即收，不进月度出账）/ 后付费（月度出账，存量默认）；挂 lo_accounts 与 orders 快照，不挂 product_offers（adopted note 2026-08-22） |
| 账实核对 ledger_recon.diffKind | UNPAID / PARTIAL / OVERPAID / REFUNDED / PAID_NO_INVOICE / MATCH | 未收 / 部分收 / 多收 / 退款未补收 / 已收未开票 / 三角一致；按账单定位（应收=bills.amount，实收=SUCCESS−REFUNDED，开票=ISSUED invoices.total_amount），见 docs/design/q3-ledger-recon.md |
| CS 工单优先级 cs_ticket_extensions.priority | LOW / NORMAL / HIGH / URGENT | 客服队列优先级；不改变 complaints.status |
| AR 信用等级 ar_credit_profiles.credit_level | STANDARD / WATCH / RESTRICTED / SUSPENDED | 可解释规则等级；停机仍经既有 stop_resume_tasks 编排 |
| 官网内容 cms_posts.status | DRAFT / PUBLISHED / OFFLINE | 草稿 / 已发布（公开读可见）/ 已下线；置 PUBLISHED 落 published_at |
| 官网内容 cms_posts.category | NEWS / ARTICLE | 动态新闻 / 文章 |
| 税局轨迹 invoice_tax_events.event | RECEIPT / BACKFILL / VOID / REISSUE | 网关回执 / 人工回填票号 / 发票作废 / 原票作废重开；轨迹与 invoice.status、tax_status 正交，时间正序回放，见 docs/design/q3-tax-trail.md |
| 客户端发版 client_releases.status | DRAFT / GRAY / PUBLISHED / ROLLED_BACK | 草稿 / 灰度（按比例+白名单分桶投放）/ 全量（公开可下载）/ 已回滚（不可再投放） |
| 客户端发版 client_releases.app | user / worker | 用户端 App / 师傅端 App |
| 师傅 workers.status | 1 / 0 | 1在职 / 0离职（后台师傅管理页 active/left 文案；`workerAssignable` 要求 status=1 且 left_at 为空才可接单） |
| 师傅注册 worker_registrations.status | PENDING / APPROVED / REJECTED | 待审核 / 已通过 / 已驳回（审核通过建 workers 主档 + 回填 worker_id） |
| 实名核验 verifications.result | PENDING / PASS / FAIL | 待核验 / 通过 / 不通过（000059 归一，subject_type=customer/worker；用户详情 verifyRecords 段三态） |
| 接单设置 worker_settings | accepting 布尔 + accept_types CSV | 后端当前仅识别工单作业类型 `INSTALL`（`internal/httpapi/worker/ticket_gate.go` portalTicketType）；P2 接单类型枚举待扩展 |
| 派单工单作业类型 dispatch_tickets.work_type（派生口径） | INSTALL / REPAIR | 报障单（complaints 联表有值）→ REPAIR，否则 INSTALL（`internal/httpapi/worker/ticket.go` portalTicketTypeOf；admin 看板只落 INSTALL） |
| 盘点任务 stocktake.status | DOING / DONE | 在盘 / 已关单（000156 起：差异明细全处置完才可关单，存在 OPEN 差异返回 40900；口径见 fields.md §4.2.1） |
| 盘点差异 stocktake_items.kind | PENDING / OK / MISMATCH / MISSING / EXTRA | 未扫 / 账实一致 / 状态不符 / 关单时仍未扫 / 计划外多扫 |
| 盘点处置 stocktake_items.resolution | OPEN / CONFIRMED / FIXED / ESCALATED | 待处置 / 确认差异(按实盘修正台账) / 现场核实台账为准 / 上报转人工 |
| 供应商 procurement_suppliers.status | ENABLED / DISABLED | 启用 / 停用（adopted 2026-08-28；迁移 000163） |
| 采购单 procurement_orders.status | DRAFT / SUBMITTED / PARTIAL / RECEIVED / CANCELLED | 草稿 / 已提交 / 部分到货 / 全部到货 / 已取消；CONFIRMED 自动从 SUBMITTED→PARTIAL→RECEIVED 推进；任意非 RECEIVED 状态可 CANCELLED（adopted 2026-08-28；迁移 000163） |
| 入库单 procurement_receipts.status | DRAFT / CONFIRMED / REJECTED | 草稿 / 已确认（建 asset_batches+逐台建 assets IN_STOCK 同事务）/ 拒收（adopted 2026-08-28；迁移 000163） |
| 施工回单 install_logs.status | OPEN / COMPLETED / REJECTED | 已提交待签收 / 已签收 / 已拒签；工单同一时刻最多一条 OPEN（uq_install_logs_ticket_open 部分唯一，adopted 2026-08-28；迁移 000164） |
| 派单工单到场 dispatch_tickets.arrived_at | TIMESTAMPTZ 可空 | 师傅到场打卡事实（不写回订单状态；GIS 施工实时图层读 arrive_lat/lng；adopted 2026-08-28；迁移 000164） |
| 派单工单站点坐标 dispatch_tickets.site_lat/site_lng | DOUBLE PRECISION 可空 | 派单时刻自 orders.address_id→addresses.geom 物化（快照口径同 price_snapshot；与到场打卡 arrive_lat/lng 师傅侧事实互不混用；半径闸门读此；adopted 2026-09-01；迁移 000174） |
| 派单区域匹配（派生口径） | regions 树祖先或自身 | 工单区域须落在师傅负责区域子树内（ltree path <@）；师傅区域 0=不限、工单区域 0=放行；adopted 2026-09-01（WorkerService.MatchedRegionIDs） |
| 授权类型 license_type | duration / lifetime / trial | 按时长 / 终身 / 试用（release-platform 发行契约，claims 内透传展示） |
| 授权 status（release-platform 侧） | issued / activated / consumed / expired / revoked | 已签发 / 已激活 / 已兑码 / 已过期 / 已吊销；boss 仅核验 `revoked`/`expired` 拒绝（verify.go checkStatus），其余透传展示 |
| ODN 设施生命周期 odn_facility.lifecycle_status | PLANNED / IN_BUILD / IN_SERVICE / RETIRED | 规划 / 施工中 / 在网 / 退役（site/device 同规，迁移 000198；RETIRED 终态与 status 双列同步） |
| 地址覆盖 address_coverage.status | SERVED / PENDING / UNSERVED | 可装 / 规划在建 / 未覆盖（迁移 000197；SERVED/PENDING 须挂服务设施或核心设备） |
| 供应商承建类型 procurement_suppliers.contractor_type | MATERIAL / CONSTRUCTION | 材料类（存量默认，既有语义不变）/ 施工类（含资质信息 qualification；联系人复用既有 contact 字段）。迁移 000205（原预分配 000203 让号，见 adopted 2026-09-07-contractor-settlement-model） |
| 工程结算 construction_settlements.status | PENDING / SETTLED / VOIDED | 待结算 / 已结算 / 已作废。发起前置：项目 ACCEPTED 且已指定施工类承包商；应付=发起时 SUM(construction_items.amount)（生成列，后端计算）。PENDING→SETTLED（确认）；PENDING/SETTLED→VOIDED（作废，原因必填）；VOIDED 终态。同项目同时最多一张有效结算单（部分唯一）；作废后重开以新结算单表达，原单保留历史。迁移 000206（原预分配 000204 让号顺延） |
| ODN 核心链路设备类型 odn_device.kind | SNW / OLT / ODF / OCC / ODB / OBD / SDB / SBD / PRT / TBP | 资源编码规范 2.2;OBD(一级分光器,归 ODB)/SBD(二级分光器,归 SDB)为箱内部件扩展(2.4 口径,迁移 000209,adopted 2026-09-07-odn-box-types-import-chain);导入域箱体设备可无城市(uq_odn_device_box 分域),SNW/OLT/PRT/TBP 仍强制城市 |
| ODN 资源链状态 odn_resource_chain | lifecycle_status 同设施四态;port_status IDLE/RESERVED/USED/DISABLED;laying_method AERIAL/UNDERGROUND/SUBMARINE/MICROTRENCH/INDOOR;row_status NOT_STARTED/PENDING/APPROVED/EXPIRED/NA;pece_status PENDING_SIGN/SIGNED/STAMPED/NA;分光比 1:2/1:4/1:8/1:16/1:32/1:64/1:128 | 迁移 000210(W3);模板中文标签映射:资源状态 规划→PLANNED/已安装·已测试→IN_BUILD/在用→IN_SERVICE/已报废→RETIRED/留空→PLANNED(绝不当作已安装/在网);总分光比=一级×二级;fields.md 1.5.12 |
| ODN 许可单 odn_permits.status | ROW: NOT_STARTED/PENDING/APPROVED/EXPIRED/NA;PECE: PENDING_SIGN/SIGNED/STAMPED/NA | ROW(路权)/PECE 许可状态机(000211,W4;字典与资源链 row_status/pece_status 对齐)。ROW 转移:未开始-待处理(提交)-已批准(批复号+有效期止必填)/未开始(驳回,原因必填);已批准-已过期(手动或开工门控自动回写);已过期-待处理(过期复验,重走批准);未开始与 NA 互转(标记/取消不适用)。PECE 转移:待签署-已签署-已盖章(开工门控满足态);已签署-待签署(退回补正,原因必填);待签署与 NA 互转(作废/恢复)。开工前置(F3):每类存在达标许可或 NA 方可开工(BOSS_ODN_PERMIT_GATE=on 灰度,默认关,与覆盖门控同模式);fields.md 1.5.13 |

## 5. 关键术语

| 术语 | 精确定义 | 边界说明 |
|:-----|:---------|:---------|
| 预占 | 端口锁定进入 RESERVED，未缴费前保留 | 超时释放（阈值走配置中心） |
| 预绑定 | 为订单预关联电子标签（端口/用户/地址） | 与现场扫码必须一致 |
| 扫码绑定 | 装维现场扫码，实物光猫与预绑定核对 | 不一致→换机/重绑 |
| 四码合一 | 资产/客户/端口/地址 四码关联（第 2 码=客户，fields.md 5.1 口径） | 任一码反查单表索引。Amended 2026-08-21：「唯一关联」不再绝对——000086 起 asset 可空（纯端口链路），000088 起 customer 可 1:N（一客户多链路）；asset/port/address 各至多一条非空活跃链路（部分唯一索引），UNLINKED 行保留为历史。Amended 2026-09-04：同客户同地址重装→scan-bind 刷新复用既有活跃行（更新端口/资产/状态并留审计）；跨客户→40920 族拒绝（adopted note 2026-09-04-quadlink-reinstall-reuse） |
| 未收费不派单 | 4 合同收费未成功，禁止进入 8 派单 | 硬约束，全案 REQ-CL-001 |
| 预付费 | 客户订购时选 PREPAID，环节 4 合同收费当场收款（缴费流水落账） | 不进月度出账；与后付费正交于套餐（同套餐可双卖法，adopted note 2026-08-22） |
| ARN | 对外单据（发票/收据）连续编号，发票 INV-、收据 OR- 各自成序列 | 占号行锁串行、回滚号回退；作废 VOID 保留编号不回收（TAX-004）。Amended 2026-08-18：降格为**内部流水号**，法定票号以税局回执（tax_no）为准（多属地网关，见 adopted note） |

## 附录 A：需按 12 环节修正的历史文件位置（已全部销项 ✅）

> 2026-08-18 复查：四文件均已完成 12 环节修正，无 "11 环节" 残留；清单保留作修正记录。

| 文件 | 位置 | 现表述 | 应改为 | 状态 |
|:-----|:-----|:-------|:-------|:----:|
| docs/archive/需求提示词-服务端.md | 六、标题及 199-209 行 | 11 个业务环节（无"合同收费"） | 12 环节，插入"4 合同收费" | ✅ 已改 |
| docs/archive/需求提示词-服务端-9阶段拆分.md | 「附：11 个业务环节」整表 | 11 行环节表 | 12 行，插入"合同收费" | ✅ 已改 |
| README.md | 阶段5 描述 | "订单 11 环节" | "订单 12 环节" | ✅ 已改 |
| docs/archive/技术栈方案-一步到位.md | 多处 | "订单 11 环节状态机" | "订单 12 环节状态机" | ✅ 已改 |
