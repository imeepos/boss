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
  modelId: number // 0=未挂型号(P1-T3)
  status: string // IN_STOCK/DEPLOYED/MAINTENANCE/SCRAPPED
  sn: string // 序列号,''=未登记(P3-T2,全网唯一)
  mac: string // MAC 地址,''=未登记(P3-T2,全网唯一)
  loid: string // 电信 LOID,''=未登记(P3-T2,全网唯一)
}

// 型号字典(契约 GET /asset-models,含停用;建档/编辑下拉仅取 isActive 项)。
export interface AssetModelRow {
  id: number
  vendor: string
  model: string
  category: string // 类型权威来源:选中型号后 type 派生自 category
  partNumber: string
  isActive: boolean
}

// 入库批次(契约 GET /assets/batches,建档下拉数据源)。
export interface AssetBatchRow {
  id: number
  code: string
  name: string
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

// 标签事件流(契约 GET /tags/{tagId}/events;action: BIND/UNBIND/RECYCLE)。
// 形状对齐 internal/domain/asset.TagEvent 全量行(append-only 审计流)。
export interface TagEventRow {
  id: number
  eventId: string
  tagId: number
  assetId: number
  action: string
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
  receivedQty?: number // PUT /procurement/orders/{id} 整体替换时忽略,详情展示用
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
  remark?: string
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
