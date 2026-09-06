// 网络资源域行类型:对齐 internal/domain/{resource,device,aaa}。

export interface ResourceRow {
  id: number
  legalEntityId: number
  code: string // OLT-01/SPL-01
  name: string
  type: string // OLT/SPLITTER
  parentId: number
  addressId: number
  status: string // ONLINE/OFFLINE/FAULT
}

export interface PortRow {
  portId: number
  portCode: string
  quadCode: string
  resourceId: number
  legalEntityId: number
  legalEntityName: string
  addressId: number
  regionId: number
  regionName: string
  orderId: number // 0=空闲
  status: string // IDLE/RESERVED/USED/DISABLED
}

// 容量聚合行(fields.md §4.2.2):usageRate=USED/(USED+IDLE) 百分比两位小数。
export interface CapacityRow {
  resourceId: number
  code: string
  name: string
  type: string // OLT/SPLITTER
  totalPorts: number
  usedPorts: number
  usageRate: number
}

export interface PortHistoryRow {
  id: number
  portId: number
  status: string
  orderId: number
  changedAt: string
}

export interface ReserveRow {
  id: number
  portId: number
  orderId: number
  status: string // HELD/RELEASED/CONSUMED
}

export interface TransferRow {
  id: number
  transferNo: string
  resourceId: number
  legalEntityId: number
  legalEntityName: string
  fromRegionId: number
  toRegionId: number
  status: string // PENDING/DOING/DONE
}

export interface ExpansionRow {
  id: number
  legalEntityId: number
  expansionNo: string
  regionId: number
  expectedPorts: number
  status: string // PENDING/DOING/DONE
}

export interface DeviceMetricRow {
  id: number
  resourceId: number
  opticalPower?: number | null
  packetLoss?: number | null
  status: string // ONLINE/OFFLINE/FAULT
  collectedAt: string
}

export interface MaintenanceRow {
  id: number
  deviceNo: string
  deviceType: string
  healthScore: number
  faultCount: number
  ageYears?: number | null
  reason: string
  priority: string // MUST_REPLACE/SUGGEST/WATCH
}

export interface LoAccountRow {
  id: number
  loid: string
  customerId: number
  legalEntityId: number
  legalEntityName: string
  regionId: number
  regionName: string
  regionPath: string
  offerId: number
  qosTemplateId: number
  status: string // ACTIVE/SUSPENDED/CLOSED
  billingMode?: string // PREPAID/POSTPAID
}

export interface LegalEntityRow { id: number; name: string }
export interface RegionRefRow { id: number; name: string }

export function pageSlice<T>(rows: T[], page: number, pageSize: number): T[] {
  return rows.slice((page - 1) * pageSize, page * pageSize)
}
