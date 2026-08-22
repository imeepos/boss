# Q3 发票税局轨迹(invoice_tax_events)设计

> 能力域:TAX(AG-04)｜内部包:billing｜验收:发票状态与税局状态正交、可追踪

## 动机

000049 只落终态(tax_status/tax_no/tax_fail_reason),提交/回执/回填/作废/重开无历史可回放;
"可追踪"要求任一发票的税局交互轨迹可审计。

## 事件模型

| event | 触发 | tax_status_after | 说明 |
|---|---|---|---|
| RECEIPT | tax-submit 网关同步回执 | ISSUED/FAILED/SUBMITTED | 携带 taxNo 或 failReason |
| BACKFILL | tax-backfill 人工回填 | ISSUED | 携带 taxNo |
| VOID | void 作废 | 当时税局状态 | 发票维度事件 |
| REISSUE | reissue 重开 | '' | 事件挂原票,新票独立生命周期 |

operator_account_id=0 表示网关/系统自动;>0 为操作账号。

## 契约

```
GET /api/admin/v1/invoices/{id}/tax-events  (permCode menu:billing)
→ data.items: [{id,invoiceId,event,taxStatusAfter,taxNo,failReason,operatorAccountId,createdAt}]
   时间正序
```

## 取舍(记录于案)

- 轨迹追加为 best-effort:业务状态迁移成功后落痕,轨迹写失败不回滚业务动作;
  终态权威仍是 invoices 列,审计日志(httpx.RecordAudit)双轨兜底。
  后续如需强一致,可在 PGStore 内以同事务批量写入(需引入 tx 辅助)。
- 同步网关下 SUBMIT 与 RECEIPT 合一;异步网关落地时再拆分。
- 迁移 000110,纯增量表,不回填历史(历史发票无轨迹可考,从上线起留痕)。
