# 2026-08-29 客户中心数据审计收尾裁定(清单 1-6 闭环)

> 依据: docs/notes/adopted/2026-08-25-customer-centric-data-audit.md 及其 §六/§七
> (门禁加固 + 存量孤儿清理)。本文对剩余 6 项逐条给出结论;数据处置执行记录见 §8。

## 1. 实名核验链路(三表"消失"与聚合断裂)

- **表现状与迁移记录"不符"是 000059 的既定结果,不是缺陷**:000059 已把
  real_name_verifications(000026)/customer_real_name_verifications(000051)/
  worker_real_name_verifications(000050) 三表归一为 `verifications`
  (subject_type + subject_id),数据迁移后 DROP 原表。实测 102 库:
  `verifications` 6 行,客户 213 三行(id 1 PASS / id 4 FAIL / id 5 PASS)。
- `user_verify_records`(000046)保留门户步骤语义,但 000059 后已无写入方
  (门户自助提交走 SubmitRealName → verifications),恒为空属正常。
- **修复**:聚合端点 verifyRecords 与 admin user-verify-records 列表改读
  `verifications`(列形状不变),聚合与权威表一致(本分支 commit)。
- **一致性门禁**:后台 Verify PASS 前核对待核验单 id_card_no 与 customers.id_no,
  不一致拒绝(ErrRealNameMismatch);主档为空回填。跨主体证件复用从此进不来。
- **客户 213 处置结论**:id 4(FAIL)/id 5(PASS,WangWu/110101199001011234)
  系门户 E2E 自助提交造数(与 worker 6 王测试证件号重复),备份后删除;
  删除后最新记录回到 id 1(PASS,与主档 采购经理·王/110101198811110011 一致),
  real_name_status=VERIFIED 依据成立。

## 2. 环节乱序 5 单(335/336/337/377/383)

- **根因**(实测复核):advance 原语只校验 orders.stage 计数器(stage==N-1 即可推进),
  不校验 order_stages 日志行;CheckResource 资源不可用时置 stage=2/环节2 PENDING,
  计数器却已到 2,后续环节照常推进。5 单全部是同一形态:环节 2 PENDING、环节 ≥3 DONE。
- **代码修复**(本分支 commit):advance 推进到 N(≥3) 前要求 order_stages(N-1).result
  ='DONE';CheckResource 幂等条件收紧为"环节 2 已 DONE",PENDING 可重查自愈补正。
- **数据处置**:5 单环节 3+ 均已实际依赖资源完成(预占/收费/派单成立),环节 2
  PENDING 系核查器时序假象 → 备份后统一补正为 DONE(finished_at=补正时刻),
  不留悬案。335(进行中)补正后可继续合法推进。

## 3. 四码冲突(address 288 双记录 + UNLINKED 口径)

- **address 288 双记录裁定:合法,非冲突**。addresses 是区域树节点(非个人地址),
  客户 213 与 214 主档 address_id 同为 288;quad 150(c213,UNLINKED,asset_id NULL)
  为 213 历史预绑定留痕,quad 225(c214,LINKED,asset 2/port 317)为活跃链路。
  000110 裁定的部分唯一索引只约束 LINKED/CONFLICT,UNLINKED 留痕多行是
  postmortem/0009 固化的生命周期设计。
- **UNLINKED 口径裁定**:asset_id 允许 NULL(未绑定资产);legal_entity_id 为
  下单时订单主体快照(不回填主档主体,历史行不追溯);UNLINKED 行不参与活跃
  资产口径统计。巡检复核 SQL:
  `SELECT id,customer_id,port_id,status FROM quad_links WHERE address_id=288;`
  + 活跃唯一性由 uq_quad_links_address(LINKED/CONFLICT 谓词)兜底。
- 不做数据改动(0009 已在 e2e 用例锁定同地址两单并存语义)。

## 4. 造数自清理与巡检门禁

- scripts/ops/db-patrol-gate.sh(三通道:deploy-102 CI 步 / 验收收尾 / 102 每日
  cron,见 docs/ops/patrol-cron.md);scripts/ops/acceptance-cleanup.sh 按 acc_
  标记回收验收造数(备份+单事务),mainchain-acceptance.sh 收尾自动执行。
- 验证:--selftest 识别"140 孤儿订单"样例并拒绝;实测 102 当前 11 项巡检全 0。

## 5. 投诉-订单联动(orderId 可空)

- **裁定:维持可空,不强制关联**。门户报障按"地址+故障现象"受理(portalCreateFaultTicket
  无订单入参),用户侧本就无订单视角;建议类投诉更与订单无关。师傅端/管理端带
  OrderID 创建时经订单推导 customer_id(§六门禁已收口)。
- 存量 4 条(TKT-1..4,客户 213,门户验收产物)保留:引用有效、受理消息链引用其
  单号,删除收益为负。豁免记录于此,不再追溯。

## 6. 开票完整性与订单评价

- **bill 2(BILL-E2E-DUNNING-001,PAID)无发票**:经权威接口补开——POST /billing-runs
  按账期重跑会为该期未开票账单幂等补开(uninvoicedBillIDs),不直改库。
- **37 笔 DONE 订单无评价:豁免**。order_ratings 是用户自愿行为(POST /orders/:no/rate),
  系统不得代评;其中 31 笔系验收造数(随 acceptance-cleanup 回收后自然消减),
  其余为真实完成单,等用户自发评价,不属完整性缺陷。

## 7. 过程缺陷固化(本轮三项,随本 note 生效)

1. **数据核查先查库、再接口复核**:任何"疑似孤儿/不一致"结论必须先经 SQL 直查
   权威表,再以接口复核读路径(先例:PAY-3/4"孤儿支付"实为读路径假象)。
2. **改动文件前自查余量**:改前看目标文件行数(红线 300 行,超了先拆文件)与
   磁盘剩余空间(主分支曾越过红线无人察觉);make check C 项机械兜底行数。
3. **并行会话基线核对**:worktree 合并前反向同步并核对本地 main 与远端一致;
   测试运行与工作区改写严禁对同一 worktree 并发(见 worktree-merge-protocol)。

## 8. 数据处置执行记录(102,备份+单事务)

执行后回填:备份文件路径、事务结果、复扫结论(见 git log / 复盘)。
