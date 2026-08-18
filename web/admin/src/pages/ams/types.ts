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
