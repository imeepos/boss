// 在线会话行类型:对齐 internal/domain/aaa SessionRecord 的 JSON 驼峰契约。
export interface SessionRow {
  id: number
  loid: string
  sessionId: string
  nasIp: string
  startedAt: string
  lastUpdate: string
  inputOctets: number
  outputOctets: number
  status: string // ONLINE/PENDING_OFFLINE(aaa.SessionOnline/SessionPendingOffline)
  disconnectAttempts: number
  closeReason: string
  closedAt: string | null
}
