# 发票 ARN 连续发号采用行锁计数表,不用 PG SEQUENCE

日期：2026-08-18

> Amended 2026-08-18: 属地裁定为多税管辖区并存（见 2026-08-18-tax-jurisdiction-china-shudian.md），
> 本表编号降格为**内部流水号**；法定票号以税局回执（tax_no）为准。发号机制本身不变。

## 决策

发票 ARN 编号用 `arn_sequences` 计数表（`doc_type` 一行一序列），发号方式为事务内 `UPDATE arn_sequences SET next_no = next_no + 1 ... RETURNING`，与发票 `INSERT` 同事务提交。发票/收据各自独立序列（TAX-004 建议分序列）。作废发票编号保留不回收（VOID），重开占新号。

## why

- 契约硬约束"连续无跳号、无重复"（TAX-004/GEN-002）。PG `SEQUENCE` 的 `nextval` 不回滚，事务失败即跳号，直接违反契约。
- 计数行 `UPDATE` 持行锁到事务结束，天然串行发号（同一时刻只允许一个单据占号，TAX-004 原文）；事务回滚号一并回退，配合"先查重后占号"顺序，幂等重跑不耗号。
- 与 order_no 的 DB 序列方案（2026-08-17 note）不同：订单号只要求唯一，发票号要求连续，二者约束不同故方案不同。

## 放弃了什么

- PG SEQUENCE：回滚跳号，违反 TAX-004。
- 应用内存发号：多实例/重启下重复，同 order_no 事故机理。
- 单条 SQL CTE（占号+插入合并）：并发竞态下冲突方已耗号产生跳号，且 CTE 执行顺序无保证。
- 断号告警兜底：保留为验收场景（TAX-TC1-S02），但方案层面不让断号发生。

## 关联

- docs/BOSS综合业务支撑平台/_artifacts/AG-04支付税务/交付物.md（TAX-004 串行发号）
- docs/contract/terms.md 第 4 节 invoice.status、第 5 节 ARN
- migrations/000048_tax_invoice.up.sql
- adopted/2026-08-17-order-no-db-sequence.md（对照:唯一性 vs 连续性）
