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

export function pageSlice<T>(rows: T[], page: number, pageSize: number): T[] {
  return rows.slice((page - 1) * pageSize, page * pageSize)
}
