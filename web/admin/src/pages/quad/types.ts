// 四码/告警/AAA 日志行类型。ScanLog 后端无 json tag,键为 Go 字段名。

export interface QuadLinkRow {
  id: number
  assetId: number
  customerId: number
  portId: number
  addressId: number
  legalEntityId: number
  legalEntityName: string
  status: string // LINKED/CONFLICT/UNLINKED
}

export interface ScanLogRow {
  id: number
  orderId: number
  workerId: number
  workerName: string
  tagId: number
  result: string // MATCH/MISMATCH/OFFLINE_CACHED
}

export interface AlarmRow {
  id: number
  alarmNo: string
  level: string // CRITICAL/WARNING/INFO
  source: string // device/quadlink/aaa
  content: string
  resourceId: number
  status: string // OPEN/ACKED/CLOSED
  createdAt: string
}

export interface CdrRow {
  id: number
  loid: string
  username: string
  acctStatus: number // 1开始/2停止/3中间
  sessionId: string
  sessionTime: number
  inputOctets: number
  outputOctets: number
  nasIp: string
  billingStatus: string // UNBILLED/BILLED
  startedAt: string
}

export interface AuthLogRow {
  id: number
  loid: string
  result: string // SUCCESS/FAILED
  failReason: string // 失败原因码(fields.md §8A),空=成功或存量行
  createdAt: string
}

export function pageSlice<T>(rows: T[], page: number, pageSize: number): T[] {
  return rows.slice((page - 1) * pageSize, page * pageSize)
}
