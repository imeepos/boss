# 2026-08-25 以用户为核心的数据完整性与关联正确性审计

> 工具:bossctl CLI(admin API key)直连 102 部署环境 http://192.168.0.102:28080;
> 表结构依据 migrations/*.up.sql(275 个迁移)与 docs/contract/terms.md、fields.md。

## 审计对象

- 核心用户:customer 213「采购经理·王」(realNameStatus=VERIFIED,serviceStatus=ACTIVE,addressId=288,legalEntityId=1)
- 周边数据域:注册/实名核验、订单(12 环节)、计费账单/缴费/发票、四码合一(资产-客户-端口-地址)、派单工单、投诉、用户端子表(user_accounts/addresses/plans/addons/messages/coupons/balances/usages/notify)、积分 LOY
- 全库孤岛扫描:服务端自带 `GET /db-patrol/orphans` 与 `GET /reports/recon/latest`

## 一、以客户 213 为中心的关联数据全景(实测)

| 数据域 | 数量 | 关联键 | 状态 |
|---|---|---|---|
| customers 主档 | 1 | id=213 | 完整(含 customerCode C-00000213) |
| customer_registrations | 1 | id=1 → customer_id=213 | APPROVED,链路完整 |
| real_name_verifications | 3 | customer_id=213 | PASS×2 + FAIL×1 |
| customer_real_name_verifications | 1 | customer_id=213 | PASS(id=5) |
| orders | 50 | customer_id=213 | DONE 2 / INSTALLING 8 / PENDING 27 / CANCELLED 13 |
| order_stages(12 环节) | 每单 12 行 | order_id | 见"异常"章节 |
| bills | 2 | customer_id=213 | 均 PAID(999 + 500) |
| payments | 5 | bill_id→bills | 2 退款 + 3 成功(见异常) |
| invoices | 1 | bill_id=1 | INV-00000001 ISSUED;bill 2 无发票(见异常) |
| quad_links | 1(自身)+1(同地址他人) | customer_id / address_id | UNLINKED,assetId=0(见异常) |
| dispatch_tickets | 9 | order_id | 2 真实工单 + 7 TK-TEST 测试工单 |
| complaints | 4 | customer_id=213 | 全部 OPEN,orderId=0(见异常) |
| order_ratings | 0 | order_no | DONE 订单无评价(业务缺失) |
| user_accounts / plans / addresses / notify / balances / usages | 各 1-2 | customer_id=213 | 基本齐全 |
| addon_subscriptions | 10 | customer_id=213 | subscribe/unsubscribe 齐全 |
| user_messages | 3 | customer_id=213 | 全部 unread |
| coupons | 1 | customer_id=213 | C-DEMO-001 ISSUED |
| loy_point_ledgers | 无 | customer_id=213 | 积分域未激活,balance=0 |

## 二、数据完整性结论(以用户为中心)

### 2.1 完整(链路闭合)
- 注册→审核→建主档→实名核验:registration 1 → customer 213 → realname PASS 全链路存在。
- 订单 12 环节时间轴:50 笔订单均生成 order_stages 行;DONE 订单 311 的 2~12 环节全 DONE。
- 账单-支付-发票主链:bill 1(999)→ payments(成功 999)→ invoice INV-00000001(1118.88=999×1.12)闭环。

### 2.2 缺失(完整性缺口)
1. **账单 2(BILL-E2E-DUNNING-001,500,PAID)无发票**:仅 bill 1 有发票,PAID 账单缺发票(可能属 E2E 未走开票)。
2. **DONE 订单无服务评价**:order 311/315 均 DONE,`/orders/:no/rate` 404,worker-feedbacks 空,评价环节完全未落库。
3. **投诉无关联订单**:4 条 complaints 全部 orderId=0,报障类投诉未挂到故障订单;且 messages 声称"报修已受理/催单已受理",complaints 却全 OPEN 且无事件流(events 空),状态与消息不一致。
4. **实名聚合口径断裂**:`GET /users/213` 的 verifyRecords=[](读 user_verify_records 表,000046),而 `/customers/213/verify-logs` 有 3 条(读 real_name_verifications 表,000026),`/customers/213/real-name` 读 customer_real_name_verifications(000051)——同一"实名核验"语义三张表,聚合视图漏数据。

### 2.3 不一致(正确性缺陷)
1. **实名核验身份与客户主档不符**:customer 213 主档 name=采购经理·王 / idNo=110101198811110011;而最新实名核验记录(id=5)realName=WangWu / idCardNo=110101199001011234——核验身份与主档身份不同,且 110101199001011234 恰是 worker 6 王测试的证件号(跨主体证件复用嫌疑)。
2. **订单环节时间轴乱序**:ORD-20260820-000315 状态 DONE/stage=12,但环节 2「资源核查」PENDING;ORD-20260821-000355 CANCELLED,环节 2 PENDING 而环节 3/4 DONE——环节推进跳过前置环节,状态机允许"跳步"。
3. **订单地址/归属与主档不一致**:order 359 用 addressId=2(用户地址簿 Green Park,非安装地址 288)、legalEntityId=6(平台总公司,客户主档是 legalEntityId=1);regionPath 部分订单为空(315)部分为 "root"(359/355),与客户区域不一致。
4. **同地址四码记录冲突**:address 288 下存在两条 quad_links——150(customer 213,assetId=0,UNLINKED,legalEntityId=6)与 225(customer 214,assetId=2,portId=317,LINKED,legalEntityId=6);客户 213 的四码缺失资产维度且归属主体错误。

## 三、孤岛数据(关联关系缺失)——全库扫描结果

服务端 `GET /db-patrol/orphans`(2026-08-25 09:51 快照)与 `GET /reports/recon/latest`(2026-08-23):

| 孤儿类型 | 数量 | 说明 | 涉及核心用户 |
|---|---|---|---|
| orders.customer_id 悬空 → customers | **140** | 订单引用不存在的客户;样本 id 402-411,最新一批 578 等 customerId=334/offerId=175/channelId=165 均不存在;`/orders` 列表 customer/product 列为空 | 否(E2E 造数) |
| orders.offer_id 悬空 → product_offers | **135** | 同上批订单的产品引用悬空 | 否 |
| orders.channel_id 悬空 → channels | **135** | 同上批订单渠道引用悬空 | 否 |
| lo_accounts.customer_id 悬空 → customers | **18** | LOID 账号绑定不存在客户(239/246/251/254...);19 条中仅 1 条(213)有效 | 213 自身有效 |
| reserve_records.port_id 悬空 → ports | **11** | 预占记录端口不存在 | 否 |
| transfers.resource_id 悬空 → resources | **22** | 调配单资源不存在(22 条调配单全悬空) | 否 |
| alarms.resource_id 悬空 → resources | **12** | 告警资源不存在 | 否 |
| 端口 RESERVED 但无在途 RESERVED 订单 | **58** | recon 检查 orphanReservedPorts;端口 314(P-E2E-VERIFY)RESERVED 但订单 381 已是 INSTALLING | 是(314) |
| payments.bill_id=0 且 customer_id 不可推导 | **2** | PAY-3/PAY-4 amount=1 无账单无客户 | 否(不可见) |
| 派单工单缺口 | **≥5** | 客户 213 有 10 笔 stage≥8 订单,仅 2 笔(358/359)有真实工单;330/331/332/333/350 环节 8「派单」DONE 但 dispatch_tickets 无记录 | **是** |

> Amended 2026-08-28:「端口 RESERVED 但无在途 RESERVED 订单」行内 314 判为孤儿有误——端口在环节 12
> （UpdateMap）才消费,INSTALLING 单端口保持 RESERVED 属合法在途;内嵌的"INSTALLING 僵尸单"
> 信号改由 recon 新检查 installingStuck 暴露。见 2026-08-28-port-reserved-installing-inflight.md。

## 四、根因分析

1. **E2E 造数污染**:140 笔孤儿订单、18 条孤儿 LOID、11 条孤儿预占、22 条孤儿调配单、12 条孤儿告警全部来自 E2E/联调测试数据,直接 INSERT 未走业务校验(软引用无 FK 约束),且未随 2026-08-21 死数据清理一并清除。
2. **软引用不设 FK**:orders.customer_id/offer_id/channel_id、lo_accounts.customer_id、reserve_records.port_id、transfers.resource_id、alarms.resource_id 均为"软引用"(注释标注,无 REFERENCES 约束),数据库层不拦截悬空。
3. **环节推进不校验前置**:charge(环节 4)自动推进 5-8,但环节 2 可为 PENDING 时环节 3/4 已 DONE;派单环节 DONE 未强制创建 dispatch_tickets。
4. **三张实名表并存**:real_name_verifications(000026,历史流水)与 customer_real_name_verifications(000051,1:1 当前态)与 user_verify_records(000046,门户步骤)语义重叠,聚合层只读其一。
5. **四码写入口径不一**:quad_links 的 assetId/legalEntityId 在 UNLINKED 态下未规范化(0/空/6),且同地址可并存多条记录,无唯一约束兜底(asset_id/customer_id/port_id/address_id 各自 UNIQUE,但 address_id=288 已出现两条 → 225 为后写覆盖未删旧)。

## 五、建议(按优先级)

1. **清理孤儿数据**:把 140 孤儿订单 + 18 孤儿 LOID + 11 孤儿预占 + 22 孤儿调配单 + 12 孤儿告警 + 2 孤儿支付纳入死数据清理任务(参照 2026-08-21 清理先例),并给 E2E 造数脚本加"用例结束自清理"。
2. **补 FK 或加守卫**:对高频软引用列(orders.customer_id/offer_id/channel_id)补数据库约束或写入层校验;至少把 db-patrol 孤儿巡检接入定时任务并在 CI 门禁卡 E2E 造数泄漏。
3. **环节推进加前置依赖**:charge 前校验环节 2/3 DONE;派单环节 DONE 必须伴随 dispatch_tickets 落库(同事务),否则环节状态与工单数据分叉。
4. **实名表归一**:裁定三张实名表职责(建议 real_name_verifications 保留历史流水、customer_real_name_verifications 为当前态、user_verify_records 只记门户自助步骤),聚合端点统一从当前态表取数,并补主档身份一致性校验(核验通过证件号必须与 customers.id_no 一致)。
5. **四码冲突治理**:同 address_id 四码记录迁移/合并,UNLINKED 态 assetId=0 与 legalEntityId 口径规范化,巡检 CONFLICT 时限(terms.md 已定义 4 小时)。
6. **投诉-订单-消息联动**:报障类投诉强制关联订单(orderId 必填),状态流转时同步消息;投诉办结后消息标记已读。

## 关联

- docs/contract/terms.md(12 环节/状态枚举)
- docs/contract/fields.md(§2.1 客户/§3.1-3.3 订单/计费/§4.1 资产/§5.1 四码)
- migrations/000004(客户)、000010/000017/000018(订单/工单)、000011/000024/000048(账务/发票)、000012(四码)、000046/000051(用户子表/实名)、000105(LOY)
- 服务端巡检端点:GET /db-patrol/orphans、GET /reports/recon/latest

## 六、Addendum(2026-08-29):接口门禁加固实施

按用户指令「接口加紧限制关联数据能不为空就不为空;检测关联数据是否存在/状态是否正常;
流程数据在状态流转中完善关联数据;孤儿数据优先从接口门禁出发优化」,本节落地了门禁层修复
(commit feat/api-gate-hardening)。原则:数据库层软引用不加 FK(历史裁定),改在写入域
(domain 层,handler 全部委托)加存在性/非空门禁,新数据不再产生孤儿;存量孤儿清理
(建议 §五.1)单列后续任务。

| 写入口 | 门禁 | 关闭的孤儿类 |
|---|---|---|
| order.Submit(环节1) | 已有:customer/address 存在 + offer PUBLISHED + channel ACTIVE | orders 三列悬空(140/135/135) |
| aaa.CreateLoAccount | 新增:customer_id/offer_id/legal_entity_id 必填且存在 | lo_accounts.customer_id(18) |
| resource.CreateTransfer | 收紧:resource_id/legal_entity_id 必填且存在(原 0 可放过) | transfers.resource_id(22) |
| resource.AppendReserveRecord | 新增:port_id/order_id 必填且存在 | reserve_records.port_id(11) |
| resource.AppendPortHistory | 新增:port_id 必填且存在 | port_change_history.port_id |
| billing.RecordPaymentWithCoupon | 新增:bill_id>0 时账单必须存在;bill/customer 双空拒收;INSERT 落 customer_id | payments 双空孤儿(2) |
| order.CreateComplaint | 收紧:customer_id 必填,未传时经订单推导(师傅端投诉只带 OrderID) | complaints.customer_id=0 |
| order.CreateDispatchTicket / AppendScanLog | 收紧:order_id 必填且存在 | dispatch_tickets/scan_logs 孤儿 |
| device.CreateAlarm | 新增:resource_id>0 时资源必须存在(NULL 保留平台告警) | alarms.resource_id 悬空(12) |
| order.DispatchOrder + Automation | 流程:派单幂等自愈——已到环节8 时重调仅补落缺失工单;updateMap 同为 selfHeal 环节 | 环节8 DONE 无工单(≥5,含 330/331/332/333/350 存量,重调自愈) |
| billing 读路径 | paymentCols 补 COALESCE(customer_id,0),API 不再恒返回 customerId=0 | 孤儿流水可观测 |

放弃的方案:对软引用列补 DB FK 约束(会拦历史脏数据导致迁移失败,且与「软引用」历史裁定
冲突);对存量孤儿做批量 DELETE(需单独死数据任务,不在门禁提交内)。门禁全部走
pgxmock 单测锁定 SQL 契约,`make check` 全绿。

## 七、Addendum(2026-08-25):存量孤儿数据清理落地

门禁(§六)只拦新数据;本节按建议 §五.1 清存量孤儿。执行通道:无本地 psql,经
`ssh imeepos@192.168.0.102` + `docker exec -i boss-infra-postgres-1 psql -U boss -d boss`
管道跑 SQL 文件(多层 shell 引号必炸,文件管道最稳)。流程:快照 → 预检(FK 依赖扫
描)→ pg_dump 备份(16 表,102 服务器 /tmp/boss_orphan_backup_20260825-045747.sql)→
单事务 ON_ERROR_STOP 清理 → 复扫=0 → API 冒烟。

| 类目 | 清理动作 | 结果 |
|---|---|---|
| orders 悬空(客户/产品/渠道缺失,E2E 造数) | 删 145(含从属:order_stages 18、reserve_records 11、其余 0) | orders 246→101 |
| lo_accounts 孤儿 | 删 18(仅剩 customer 213 的 LOID-E2E-RESUME-001) | 19→1 |
| cdrs 孤儿 loid | 删 95(76 存量 + 19 关联已删 LO 账号) | 95→0 |
| transfers 资源悬空 | 删 22 | 22→0 |
| reserve_records 端口/订单悬空 | 删 11+11 | →0 |
| alarms 悬空资源 | 删 12(NULL 平台告警 2 条保留) | 22→10 |
| 环节8+ 无工单(合法订单 336/352) | 补落工单 DT-20260820-000314/000330(DOING,与 INSTALLING 对齐) | 47→49 |
| 泄漏 RESERVED 端口(DONE/CANCELLED 订单) | 32→USED(环节12 语义)、1→IDLE(CANCELLED)、2 悬空 order_id 引用清空 | 33 归位 |
| payments PAY-3/PAY-4 | 不删——实际锚定 customer 213,此前读路径恒显示 customerId=0 是 §六 修复的观测假象 | 保持 |

清理后复扫各孤儿类=0;API 冒烟 /orders /ports /lo-accounts /transfers /alarms /payments
/complaints /dispatch/pool 全部正常,payments 已如实回显 customerId=213。
遗留:devseed/E2E 造数脚本加「用例结束自清理」(建议 §五.1 剩余项,防再泄漏)。
