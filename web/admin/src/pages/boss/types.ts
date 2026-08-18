// 订单与工单域行类型:对齐 internal/domain/{order,worker} 与 http_order*.go。
// 注:complaints/dismantles/activation-callbacks 后端结构体无 json tag,键为 Go 字段名。

export interface OrderListRow {
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

// 无 json tag 的 Go 结构体(键为 Go 字段名)。
export interface ComplaintRow {
  ID: number
  TicketNo: string
  CustomerID: number
  OrderID: number
  LegalEntityID: number
  LegalEntityName: string
  Type: string // NETWORK_FAULT/TARIFF_DISPUTE/SERVICE_COMPLAINT
  Status: string // OPEN/PROCESSING/CLOSED
}

export interface DismantleRow {
  ID: number
  DismantleNo: string
  OrderID: number
  LegalEntityID: number
  LegalEntityName: string
  AssetID: number
  PortID: number
  Status: string // PENDING/DOING/DONE/FAILED
}

export interface ActivationCallbackRow {
  ID: number
  OrderID: number
  Result: string // SUCCESS/FAILED
  Retries: number
}

export function pageSlice<T>(rows: T[], page: number, pageSize: number): T[] {
  return rows.slice((page - 1) * pageSize, page * pageSize)
}
