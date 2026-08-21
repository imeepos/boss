// 订单与工单域行类型:对齐 internal/domain/{order,worker} 与 http_order*.go。

export interface OrderListRow {
  id: number
  orderNo: string
  customer: string
  product: string
  address: string
  stage: number
  stageLabel: string
  status: string // PENDING/RESERVED/INSTALLING/DONE/CANCELLED
  ops: string[]
  createdAt: string
}

export interface CheckDeviceRow {
  code: string
  name: string
  type: string
  status: string
  total: number
  idle: number
}

export interface CheckDetail {
  addressId: number
  total: number
  idle: number
  reserved: number
  used: number
  disabled: number
  idleCodes: string[]
  devices: CheckDeviceRow[]
}

export interface TimelineRow {
  stage: number
  name: string
  finishedAt?: string
  duration: string
  retries: number
  result: string // DONE/DOING/PENDING
}

export interface WorkerGroupRow {
  id: number
  legalEntityId: number
  code: string
  name: string
  leaderId: number
  leaderName: string
}

export interface WorkerRow {
  id: number
  staffNo: string
  name: string
  groupId: number
  regionId: number
  phone: string
  status: number // 1在职 0离职
  joinedAt: string
  leftAt?: string
}

export interface DispatchTicketRow {
  ticketId: number
  ticketNo: string
  orderId: number
  workerId: number // 0=未派
  workerName: string
  groupId: number
  groupName: string
  regionId: number
  regionName: string
  legalEntityId: number
  legalEntityName: string
  status: string // PENDING/DOING/DONE/CANCELED
}

export interface DispatchTransferRow {
  id: number
  ticketId: number
  fromWorkerId: number
  fromWorkerName: string
  toWorkerId: number
  toWorkerName: string
  reason: string
  operatorAccountId: number
  transferredAt: string
}

export interface ComplaintRow {
  id: number
  ticketNo: string
  customerId: number
  orderId: number
  legalEntityId: number
  legalEntityName: string
  type: string // NETWORK_FAULT/TARIFF_DISPUTE/SERVICE_COMPLAINT
  status: string // OPEN/PROCESSING/CLOSED
}

export interface DismantleRow {
  id: number
  dismantleNo: string
  orderId: number
  legalEntityId: number
  legalEntityName: string
  assetId: number
  portId: number
  status: string // PENDING/DOING/DONE/FAILED
}

export interface ActivationCallbackRow {
  id: number
  orderId: number
  result: string // SUCCESS/FAILED
  retries: number
}

export function pageSlice<T>(rows: T[], page: number, pageSize: number): T[] {
  return rows.slice((page - 1) * pageSize, page * pageSize)
}
