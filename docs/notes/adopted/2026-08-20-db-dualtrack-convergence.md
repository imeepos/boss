# 数据库双轨收敛与四码/账号口径裁定（db-design-review D1/D2/D3/D5）

日期：2026-08-20（裁定当天）

## 决策

对 `docs/review/db-design-review.md` 四项设计问题作出裁定：

1. **D1 用户侧双轨**：`portal(portal_*)` 为唯一权威运行态。user_* 中的双胞胎表
   （user_balances/user_messages/user_complaints/user_invoices/user_notify_settings.auto_pay）
   降级为视图或标注 deprecated，门户读写一律走 portal 域。
2. **D2 portal 合成 ID 号段**：portal_accounts.customer_id 的隔离空间合成 ID 一律取**负数段**，
   与真实 customers.id（正数 BIGSERIAL）物理隔离；加 CHECK 约束。
3. **D3 quad_links 复绑**：四列 UNIQUE 改为 `WHERE status='LINKED'` 的部分唯一索引，
   UNLINKED 行保留作链路历史，不再要求复绑 UPDATE 原行。
4. **D5 实名双轨**：合并为统一 verifications 表（subject_type=customer/worker + subject_id），
   000026 旧表数据迁移后废弃。

## why

- D1：portal 是线上门户实际运行态，双写不一致与报表口径分裂的代价大于合并成本；userdata 硬 FK 的规范性不足以抵消双轨债务。
- D2：负数段是零迁移成本、可 CHECK 强制的物理隔离；合并隔离空间时可安全映射，防止撞号。
- D3：部分唯一索引一个迁移即达成「复绑可 INSERT 新行 + 历史可追溯」，优于台账表（多一条维护路径）。
- D5：三处实名表命名/用途分裂，统一 subject 模型与归属台账、审计日志同构，长期最省。

## 放弃了什么（被否决项）

- D1 否决「user_* 为权威」：门户读写路径全改、迁移量最大，收益仅是 FK 规范性。
- D1 否决「仅写边界不合并」：双轨债务永久保留。
- D2 否决「映射表」：门户读写路径全改，短期不划算；负数段已够。
- D3 否决「补 quad_link_histories 台账」「索引+台账都做」：台账可由保留的 UNLINKED 行替代。
- D5 否决「保留两表废弃旧表」：既然要动，一步统一 subject 模型。

## 关联

- docs/review/db-design-review.md（D1/D2/D3/D5 条目追加裁定链接）
- docs/contract/data-relations.md §2.9/§2.10/§2.6/§2.3
