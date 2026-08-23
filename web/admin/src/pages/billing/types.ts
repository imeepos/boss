// 计费账务域行类型:对齐 internal/domain/billing(Bill/Payment/ArrearsItem/StopResumeTask/ReconBatch)。

export interface BillRow {
  billId: number
  billNo: string
  customerId: number
  customerName: string
  legalEntityId: number
  legalEntityName: string
  regionId: number
  regionName: string
  period: string // 账期,如 2026-08
  amount: number
  status: string // UNPAID/PAID/OVERDUE
}

// 契约:internal/domain/billing tax.go Invoice(GET /invoices → items)。
export interface InvoiceRow {
  id: number
  invoiceNo: string // 内部流水号 INV-00000001
  billId: number
  billNo: string
  customerId: number
  customerName: string
  netAmount: number
  vatRate: number
  vatAmount: number
  totalAmount: number
  status: string // ISSUED/VOIDED
  taxJurisdiction: string // CN/PH,空=未定
  taxChannel: string // manual/leqi/bir_eis
  taxStatus: string // PENDING/SUBMITTED/ISSUED/FAILED
  taxNo: string
  issuedAt: string
}

export interface PaymentRow {
  id: number
  payNo: string
  billId: number
  amount: number
  method: string // wechat/alipay/card/cash
  status: string // SUCCESS/FAILED/REFUNDED
}

export interface ArrearsRow {
  customerId: number
  customer: string
  amount: number
  days: number
  status: string // 催收中/已停机等
}

export interface CollectionTaskRow {
  id: number
  customerId: number
  customer: string
  taskType: string
  priority: string
  status: string
  dueAt: string
  amount: number
  days: number
  note: string
}

export interface ARMetrics {
  totalAmount: number
  customerCount: number
  stoppedCount: number
  overdueBillCount: number
  agingBuckets: { d0To15: number; d16To30: number; d31To60: number; d61To90: number; d90Plus: number }
  lastRunAt: string
}

export interface StopResumeTaskRow {
  id: number
  customerId: number
  loAccountId: number
  action: string // STOP/RESUME
  status: string // PENDING/DOING/DONE/FAILED
}

export interface ReconRow {
  id: number
  batchNo: string
  channel: string
  channelAmount: number
  systemAmount: number
  diff: number // 渠道-系统
  status: string // DIFF_PENDING/SETTLED
  createdAt: string
  settledAt?: string
}

// 账实核对行(契约 GET /billing/ledger-recon;枚举见 terms.md ledger_recon.diffKind)。
export interface LedgerReconRow {
  billId: number
  billNo: string
  customerId: number
  customerName: string
  legalEntityId: number
  legalEntityName: string
  period: string
  billAmount: number
  paidAmount: number
  invoiceAmount: number
  refundAmount: number
  diffKind: string // UNPAID/PARTIAL/OVERPAID/REFUNDED/PAID_NO_INVOICE/MATCH
  invoiceNo: string
  taxStatus: string
}

export interface LedgerReconSummary {
  billsTotal: number
  paidTotal: number
  invoiceTotal: number
  byKind: Record<string, number>
}

export function pageSlice<T>(rows: T[], page: number, pageSize: number): T[] {
  return rows.slice((page - 1) * pageSize, page * pageSize)
}
