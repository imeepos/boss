# Q3 账实核对（Ledger Reconciliation）设计

> 能力域：BIL+PAY+TAX（AG-03/AG-04）｜内部包：billing｜验收：账实核对差异可定位

## 定位

渠道对账（recon_batches）回答"渠道流水 vs 系统缴费"；账实核对回答"应收 vs 实收 vs 开票"。
两者正交：渠道对账以渠道为主体，账实核对以账单为主体。

## 口径

- 应收 = bills.amount（账期快照，含税前净额；与发票 net_amount 同源）
- 实收 = 该账单 SUCCESS payments 之和 − REFUNDED 之和（FAILED 不计）
- 开票 = 该账单 status='ISSUED' 的 invoices.total_amount（VOID 不计）

## 差异种类（可定位到 bill）

| diffKind | 含义 | 定位 |
|---|---|---|
| UNPAID | 实收 = 0 且账单未付/逾期 | 催收队列 |
| PARTIAL | 0 < 实收 < 应收 | 部分缴费 |
| OVERPAID | 实收 > 应收 | 退差/人工核 |
| PAID_NO_INVOICE | 账单 PAID 且实收足额，但无在发票 | 补开票 |
| INVOICE_NO_PAY | 有在发票但实收不足 | 开票与收款倒挂 |
| REFUNDED | 存在 REFUNDED 流水且无足额补收 | 退款追踪 |

一行账单只落一个最高优先级 kind：REFUNDED > PARTIAL/OVERPAID > UNPAID > INVOICE_NO_PAY > PAID_NO_INVOICE。

## 契约

```
GET /api/admin/v1/billing/ledger-recon?period=YYYY-MM&legalEntityId=&page=&pageSize=
permCode menu:paycheck
→ data: { items: [{billId,billNo,customerId,customerName,legalEntityName,period,
        billAmount, paidAmount, invoiceAmount, diffKind, invoiceNo, taxStatus}],
        total, summary: {billsTotal, paidTotal, invoiceTotal, byKind: {kind: count}} }
```

- period 必填（YYYY-MM）；缺省拒绝 42200。
- 排序：diffKind 优先级 → bill_id。
- 分页与 AAA 管理分页同一契约（page/pageSize/total）。
- 数据范围：沿当前账号 DataScope（legal_entity/region_scope）SQL 裁剪，与 AAA 收敛同一模式。
