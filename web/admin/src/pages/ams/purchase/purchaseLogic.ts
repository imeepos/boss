// 采购域纯逻辑:供应商/订单/入库单的表单校验、载荷组装、操作列状态显隐。
// 契约:PUT /procurement/suppliers/{id}、POST /procurement/suppliers/{id}/enable、
// PUT /procurement/orders/{id}(明细整体替换)、POST /procurement/receipts/{id}/reject。
import type { OrderItemRow } from '../types'

export interface SupplierFormState {
  code: string
  name: string
  contactName: string
  contactPhone: string
  legalEntityId: number
  remark: string
}

export const emptySupplierForm: SupplierFormState = {
  code: '', name: '', contactName: '', contactPhone: '', legalEntityId: 1, remark: '',
}

export function supplierFormErr(form: SupplierFormState): string {
  if (!form.name.trim() || !form.code.trim()) return 'supErrName'
  return ''
}

export function buildSupplierPayload(form: SupplierFormState) {
  return {
    code: form.code.trim(),
    name: form.name.trim(),
    contactName: form.contactName.trim(),
    contactPhone: form.contactPhone.trim(),
    legalEntityId: form.legalEntityId,
    remark: form.remark.trim(),
  }
}

// 显隐规则:编辑仅 DRAFT 订单;驳回仅 DRAFT 入库单;启用仅 DISABLED 供应商。
export const canEditOrder = (status: string): boolean => status === 'DRAFT'
export const canRejectReceipt = (status: string): boolean => status === 'DRAFT'
export const canEnableSupplier = (status: string): boolean => status === 'DISABLED'

// PUT /procurement/orders/{id}:明细整体替换,剥掉只读的 receivedQty 再上行。
export interface OrderEditFormState {
  supplierId: number
  legalEntityId: number
  remark: string
  items: OrderItemRow[]
}

export function buildOrderEditPayload(form: OrderEditFormState) {
  return {
    supplierId: form.supplierId,
    legalEntityId: form.legalEntityId,
    remark: form.remark,
    items: form.items.map((i) => ({
      materialCode: i.materialCode,
      spec: i.spec,
      quantity: i.quantity,
      unitAmount: i.unitAmount,
    })),
  }
}

export function orderEditErr(form: OrderEditFormState): string {
  if (!form.supplierId || form.supplierId <= 0) return 'errSupplier'
  if (form.items.some((i) => !i.materialCode || i.quantity <= 0)) return 'errItems'
  return ''
}

export interface RejectFormState {
  reason: string
}

export function rejectReasonErr(form: RejectFormState): string {
  const v = form.reason.trim()
  if (!v || v.length > 255) return 'eRejectReason'
  return ''
}

export function buildRejectPayload(form: RejectFormState) {
  return { reason: form.reason.trim() }
}
