// 工程应付台账行类型(契约 fields.md §1.5.8e;internal/domain/odn construction_payables)。
export interface Payable {
  id: number
  payableNo: string
  settlementId: number
  settlementNo: string
  projectId: number
  projectNo: string
  contractorId: number
  contractorName: string
  payableAmount: number
  deductedAmount: number
  paidAmount: number
  balance: number
  status: string
  voidReason: string
  createdAt: string
}

export interface PayablePayment { id: number; payableId: number; paymentNo: string; amount: number; method: string; paidAt: string; reference: string; note: string; createdAt: string }
export interface PayableDeduction { id: number; payableId: number; amount: number; reason: string; createdAt: string }
export interface PayableInvoice { id: number; payableId: number; invoiceNo: string; amount: number; invoicedAt?: string; note: string; createdAt: string }
export interface PayableDetail { payable: Payable; payments: PayablePayment[]; deductions: PayableDeduction[]; invoices: PayableInvoice[] }

export const PAYABLE_VARIANT: Record<string, 'default' | 'success' | 'warning' | 'danger'> = {
  OPEN: 'warning', PARTIAL: 'warning', PAID: 'success', VOIDED: 'danger',
}

export function fmtMoney(v: number | null | undefined): string {
  return (v ?? 0).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}
