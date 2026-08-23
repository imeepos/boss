// 配置下发域行类型:字段口径对齐 internal/domain/provision(provision.go json 标签)。
export { pageSlice } from '../quad/types'

/** 下发任务:GET /provision-tasks(items)。 */
export interface ProvisionTaskRow {
  id: number
  taskNo: string
  orderId: number
  stageEvent: string
  loAccountId: number
  templateId: number
  status: string // PENDING/DOING/DONE/FAILED
}

/** 配置模板:GET /provision-templates(items)。 */
export interface ProvisionTemplateRow {
  id: number
  legalEntityId: number
  code: string
  name: string
  content: Record<string, unknown>
  version: number
  status: string
  updatedAt: string
}

/** 下发日志:GET /provision-logs?taskId(items)。 */
export interface ProvisionLogRow {
  id: number
  taskId: number
  resourceId: number
  resourceCode: string
  templateId: number
  templateCode: string
  result: string // SUCCESS/FAILED
  retries: number
  createdAt: string
}
