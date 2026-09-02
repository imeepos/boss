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
  boundOffers: number // 已绑定此模板的套餐数(方案B)
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

/** 下发日志详情:GET /provision-logs/{id};commands 为换行分隔的完整设备指令(空=无设备交互),deviceResponse 为设备原始应答(多段以 ---- 分隔)。order/template 关联数据已删时字段为空串/0。 */
export interface ProvisionLogDetail {
  log: {
    id: number
    taskId: number
    resourceId: number
    resourceCode: string
    templateId: number
    templateCode: string
    result: string
    retries: number
    commands: string
    deviceResponse: string
    createdAt: string
  }
  task: {
    id: number
    taskNo: string
    orderId: number
    stageEvent: string
    loAccountId: number
    templateId: number
    status: string
  }
  order: {
    orderNo: string
    status: string
    offerName: string
  }
  template: {
    code: string
    name: string
    status: string
    version: number
    content: Record<string, unknown>
  }
}
