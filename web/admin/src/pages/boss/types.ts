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

export interface WorkerLocationRow {
  workerId: number
  lat: number
  lng: number
  accuracyM: number
  speedMps: number
  bearing: number
  reportedAt: string
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
  memberCount: number // 在职成员数(000141)
}

// 装维队成员月度业绩(000141 GET /worker-groups/{id}/performance)
export interface TeamPerfRow {
  workerId: number
  staffNo: string
  name: string
  phone: string
  isLeader: boolean
  finished: number
  onTimeRate: number
  score: number
}

export interface WorkerRow {
  id: number
  staffNo: string
  name: string
  groupId: number
  groupName: string // 班组名(读取时 JOIN 现值,data-relations §6.2)
  regionId: number
  regionName: string // 主区域名(读取时 JOIN 现值,data-relations §6.2)
  regionIds: number[] // 全部负责区域(000175;主区域首位;未配置回退 [regionId])
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
  customerName: string // 客户名(读取时 JOIN 现值,data-relations §6.2 同批)
  orderId: number
  legalEntityId: number
  legalEntityName: string
  type: string // NETWORK_FAULT/TARIFF_DISPUTE/SERVICE_COMPLAINT
  status: string // OPEN/PROCESSING/CLOSED
}

export interface CSMetrics {
  openCount: number
  processingCount: number
  closedCount: number
  slaBreachedOpen: number
  slaOnTimeClosed: number
  slaOverdueClosed: number
  avgCloseHours: number
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

/** worker_feedbacks 行(GET /worker-feedbacks,fields.md §7.3/§7.5 隶属师傅事件级事实)。
 * 评价低分(<3)由事件记录侧写入 needReview=true,后台复核后置 false(events.go)。 */
export interface FeedbackRow {
  id: number
  workerId: number
  workerName: string
  groupId: number
  groupName: string
  legalEntityId: number
  legalEntityName: string
  regionId: number
  regionName: string
  ticketId: number
  customerId: number
  customerName: string
  score: number // 1~5
  needReview: boolean
}

export function pageSlice<T>(rows: T[], page: number, pageSize: number): T[] {
  return rows.slice((page - 1) * pageSize, page * pageSize)
}
