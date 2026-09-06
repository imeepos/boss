// 资产域行类型:对齐 internal/domain/asset(Asset/Tag/Stocktake/Replacement/AssetLifecycle/AssetAssignment)。

export interface AssetRow {
  assetId: number
  assetCode: string
  batchId: number
  legalEntityId: number
  legalEntityName: string
  tagId: number // 0=未绑定
  addressId: number // 0=未部署
  regionId: number
  regionName: string
  type: string // 光猫/ONU/路由器
  status: string // IN_STOCK/DEPLOYED/MAINTENANCE/SCRAPPED
}

export interface TagRow {
  tagId: number
  legalEntityId: number
  tagNo: string
  epcCode: string
  band: string
  boundAssetId: number // 0=未绑定
  status: string // UNBOUND/BOUND/DISABLED
  battery: string
}

// TagEventRow 标签绑定事件行(P2-T4):对齐 internal/domain/asset.TagEvent,append-only。
export interface TagEventRow {
  id: number
  eventId: string
  tagId: number
  assetId: number
  action: string // BIND/UNBIND/RECYCLE
  actorAccountId: number
  detail: string
  changed?: Record<string, unknown>
  createdAt: string
}

export interface LifecycleRow {
  id: number
  assetId: number
  status: string
  addressId: number
  addressName: string
  workerId: number
  workerName: string
  changedAt: string
}

export interface AssignmentRow {
  id: number
  assetId: number
  workerId: number
  workerName: string
  addressId: number
  addressName: string
  reason: string
  operatorAccountId: number
  effectiveFrom: string
  effectiveTo?: string
}

export interface StocktakeRow {
  id: number
  legalEntityId: number
  scope: string
  progress: number // 0~100
  diffCount: number
  status: string // DOING/DONE
}

export interface StocktakeItemRow {
  id: number
  taskId: number
  assetId: number
  expectedStatus: string // ''=计划外(EXTRA)
  scannedStatus: string // ''=未扫
  scannedAt?: string
  kind: string // PENDING/OK/MISMATCH/MISSING/EXTRA
  resolution: string // OPEN/CONFIRMED/FIXED/ESCALATED
  handledBy: number // 0=未处置
  handledAt?: string
  note: string
}

// 采购-库存域(决策 1:internal/domain/procurement,迁移 000163)。
export interface SupplierRow {
  id: number
  code: string
  name: string
  contactName: string
  contactPhone: string
  legalEntityId: number
  status: 'ENABLED' | 'DISABLED'
  remark: string
}

export interface OrderItemRow {
  materialCode: string
  spec: string
  quantity: number
  unitAmount: number
}

export interface OrderRow {
  id: number
  procurementNo: string
  legalEntityId: number
  legalEntityName: string
  supplierId: number
  supplierName: string
  status: 'DRAFT' | 'SUBMITTED' | 'PARTIAL' | 'RECEIVED' | 'CANCELLED'
  totalAmount: number
  expectedDate?: string
  remark: string
  createdAt: string
}

export interface ReceiptRow {
  id: number
  receiptNo: string
  orderId: number
  orderNo: string
  batchId: number
  legalEntityId: number
  legalEntityName: string
  status: 'DRAFT' | 'CONFIRMED' | 'REJECTED'
  receivedAt: string
}

export interface InventoryRow {
  materialCode: string
  batchId: number
  inStockQty: number
}

export const SCAN_STATUSES = ['IN_STOCK', 'DEPLOYED', 'MAINTENANCE', 'SCRAPPED'] as const
export const STOCKTAKE_ACTIONS = ['CONFIRM', 'FIX', 'ESCALATE'] as const

export interface ReplacementRow {
  id: number
  replacementNo: string
  assetId: number
  legalEntityId: number
  legalEntityName: string
  reason: string
  priority: string // HIGH/MEDIUM/LOW
  status: string // PENDING/DOING/DONE/FAILED
}

export const PRIORITIES = ['HIGH', 'MEDIUM', 'LOW'] as const

export function pageSlice<T>(rows: T[], page: number, pageSize: number): T[] {
  return rows.slice((page - 1) * pageSize, page * pageSize)
}
