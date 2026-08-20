# 数据库表结构设计评审（db-design-review）

> 评审对象：migrations 000001-000055（117 表）｜评审日期：2026-08-19
> 方法：第 1 轮 逐表对账（DDL ↔ docs/contract/data-relations.md ↔ ER 图规格），第 2 轮 设计问题评估（引用完整性/生命周期/双轨风险）。
> 关联：docs/contract/data-relations.md V1.2、scripts/gen-er-drawio.mjs、docs/boss-entities-er.drawio。

## 第 1 轮：对账结论

- 117 张 CREATE TABLE（含 audit_logs 2 个分区）全部收录于 data-relations.md V1.2 与 ER 图规格，双向无缺漏。
- 发现并已修正：coupons.customer_id 实为可空硬 FK（原文档/图误标软引用）。
- 硬外键（REFERENCES）113 处，集中在组织/主档/同域子表；跨域主单全部软引用 + 快照列（口径见 data-relations.md §0.5）。

## 迁移实跑验证（2026-08-20，PG 16 + PostGIS 3.4 临时容器）

- 61 个 up 迁移按序全量应用，0 失败（122 表含分区；注意 postgres:16-alpine 无 postgis/ltree 扩展，验证需 postgis 镜像）。
- D3 部分唯一索引语义实测：LINKED 占位 → 同码第二活跃行被 uq_quad_links_*_active 拒绝 → UNLINKED 释放槽位 → 复绑 INSERT 新行成功 → 历史行保留。
- 000056-000060 五个 down 迁移在干净态全部可逆，down 后 re-up 无失败。000056.down 在存在同码多行历史时 UNIQUE 重建会失败（预期，历史行即设计产物）。

## 第 2 轮：设计问题评估

按严重度排序；D# = design finding（编号沿用本文件，不与 D/A/B/E/G 契约编号混用）。

### D1【高】用户侧双轨：userdata(user_*) 与 portal(portal_*) 职责重叠

- 同一客户存在两套余额（user_balances / portal_wallets）、两套消息（user_messages / portal_messages）、
  两套投诉（user_complaints / complaints）、两套发票（user_invoices / invoices）、两套缴费偏好
  （user_accounts.auto_pay / portal_billing_prefs.auto_pay）。
- 风险：双写不一致、报表口径分裂、运维排查需查两处。
- 建议：裁定单一权威表（portal 为运行时态、user_* 为档案态亦需明示），写入 domain-map.md；其余表降级为视图或标注 deprecated。
- **已裁定 2026-08-20**：portal 为唯一权威，见 docs/notes/adopted/2026-08-20-db-dualtrack-convergence.md。
- **已落地 2026-08-20**：admin userdata 端点切权威表 + 停写回归测试（commit 38512b4）；user_invoices 开票走 billing 域为二阶段遗留。

### D2【高】三套客户账号体系 customer_id 口径不统一

- customers（主档）、user_accounts（硬 FK）、portal_accounts（UNIQUE 软引用，隔离空间可合成 ID）。
- portal_accounts 若用合成 ID 与真实 customers.id 同列混存，无任何约束防止撞号；合并隔离空间时不可迁移。
- 建议：约定合成 ID 独立号段（如负数/超高位）并写进 data-relations.md §2.10；或建 portal↔customers 映射表。
- **已裁定 2026-08-20**：合成 ID 一律负数段 + CHECK 约束，同上 note。
- **已落地 2026-08-20**：migrations/000057 + pg/memory 双实现（commit b132243）。

### D3【中】quad_links 四列 UNIQUE + UNLINKED 行不删除 → 复绑只能 UPDATE 原行

- 拆机解绑仅 `UPDATE status='UNLINKED'`（internal/domain/quadlink/pg_scan.go:103），四码列 UNIQUE 槽位仍被占用。
- 新链路必须复用历史行 UPDATE，导致链路变迁无台账（何时解绑/复绑不可追溯）；若实现误走 INSERT 会撞 UNIQUE。
- 建议：UNIQUE 改为「status='LINKED' 时的部分唯一索引」`CREATE UNIQUE INDEX ... WHERE status='LINKED'`，
  UNLINKED 行保留为历史；或补 quad_link_histories 台账。
- **已裁定 2026-08-20**：采用部分唯一索引方案，UNLINKED 行即历史，不另建台账。
- **已落地 2026-08-20**：migrations/000056 + quadlink getBy 取最新行（commit 5097a1e）。

### D4【中】orders.channel_id / region_path 无索引

- orders 仅有 customer/offer 索引；渠道维度下钻（channel_id）与数据权限裁剪（region_path ∈ scope 子树）都会全表扫。
- 建议：`CREATE INDEX idx_orders_channel ON orders(channel_id)`、`CREATE INDEX idx_orders_region ON orders(region_path)`（或 text pattern opclass）。
- **已落地 2026-08-20**：migrations/000058（commit a78156a）。

### D5【中】实名表双轨并存

- real_name_verifications(000026) 与 customer_real_name_verifications(000051) 并存，字段/用途相近；
  worker 侧命名又是 worker_real_name_verifications。
- 建议：合并为统一 verifications（subject_type+subject_id），或明确废弃旧表并在 data-relations.md 标注 deprecated。
- **已裁定 2026-08-20**：合并为统一 verifications(subject_type+subject_id)，旧表数据迁移后废弃。
- **已落地 2026-08-20**：migrations/000059 + 三域改造（commit df155cd）。

### D6【低】reconciliation_batches 不挂 payments 行级

- 对账批次只有渠道侧/系统侧总额对比，差异时无法定位到具体 payment 行。
- 建议：加 reconciliation_items(batch_id, payment_id, diff_amount) 明细表（后续迭代）。
- **已落地 2026-08-20**：migrations/000060 + 四类比对 + admin 路由（commit 9ff5f69）。

### D7【低】软引用无孤儿兜底任务

- 113 硬 FK 之外的跨域软引用（orders.customer_id、lo_accounts.customer_id、reserve_records.order_id 等）
- 无 DB 约束，删主档可留孤儿。建议：应用层定期孤儿检测（复用 report_snapshots 或 admin 巡检接口），先报表不阻断。
- **已落地 2026-08-20**：GET /db-patrol/orphans 11 项巡检，只读不阻断（commit ba85172）。

### D8【低】worker 事件级事实表 group_id 快照口径需显式化

- worker_materials/tools/feedbacks/asset_returns 均带 group_id FK；事件发生时师傅若正在换组，
- 口径应为「事发时班组」（与工单一致）。已按此理解落文档；建议在 fields.md 7.2 补一句明确。
- **已落地 2026-08-20**：fields.md 7.3 第 4 条已补事件级事实表口径（commit de0ed7a）。

## 维护约定

1. 表结构变更迁移合入时，同步改 data-relations.md + gen-er-drawio.mjs 规格，跑 `node scripts/gen-er-drawio.mjs` 再导出 png/svg。
2. 新增 D# 编号续接本文件；问题修复后在原条目追加「已修复：commit」不删原文。
