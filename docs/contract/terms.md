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
| 缴费 method | wechat / alipay / card / cash | 微信 / 支付宝 / 银行卡 / 现金 |
| 告警 alarm.level | CRITICAL / WARNING / INFO | 严重 / 警告 / 提示 |
| 告警 alarm.status | OPEN / ACKED / CLOSED | 待处理 / 已确认 / 已关闭 |
| 话单 cdr.billing_status | UNBILLED / BILLED | 未入账 / 已入账 |
| 认证日志 auth_log.result | SUCCESS / FAILED | 成功 / 失败 |
| 设备 resource.status | ONLINE / OFFLINE / FAULT | 在线 / 离线 / 故障 |
| 认证账号 lo_account.status | ACTIVE / SUSPENDED / CLOSED | 在服 / 停服 / 注销 |
| 产品 product_offer.status | DRAFT / PUBLISHED / OFFLINE | 草稿 / 在售 / 下架 |
| 客户 service_status | ACTIVE / ARREARS / SUSPENDED | 在网 / 欠费 / 停机 |
| 客户 real_name_status | VERIFIED / PENDING | 已实名 / 待补登 |
| 派单工单 dispatch_ticket.status | PENDING / DOING / DONE / CANCELED | 待派 / 进行中 / 完成 / 取消 |
| 任务 task.status | PENDING / DOING / DONE / FAILED | 待执行 / 进行中 / 完成 / 失败 |
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

## 5. 关键术语

| 术语 | 精确定义 | 边界说明 |
|:-----|:---------|:---------|
| 预占 | 端口锁定进入 RESERVED，未缴费前保留 | 超时释放（阈值走配置中心） |
| 预绑定 | 为订单预关联电子标签（端口/用户/地址） | 与现场扫码必须一致 |
| 扫码绑定 | 装维现场扫码，实物光猫与预绑定核对 | 不一致→换机/重绑 |
| 四码合一 | 资产/客户/端口/地址 四码关联（第 2 码=客户，fields.md 5.1 口径） | 任一码反查单表索引。Amended 2026-08-21：「唯一关联」不再绝对——000086 起 asset 可空（纯端口链路），000088 起 customer 可 1:N（一客户多链路）；asset/port/address 各至多一条非空活跃链路（部分唯一索引），UNLINKED 行保留为历史 |
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
