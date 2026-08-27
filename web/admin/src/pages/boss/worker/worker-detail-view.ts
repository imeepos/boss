// 师傅详情视图规格(纯数据,无 React/i18n 依赖):主档头 + 概览芯片 + 关联子集段。
// 数据源契约:GET /workers/{workerId}(主档,api/openapi/admin/worker.yaml) +
// /dispatch/my-tickets?workerId= /worker-messages?workerId= /worker-performances?workerId=
// /worker-feedbacks?workerId= 。列名对齐 docs/contract/fields.md §7.1 与 terms.md §4。
// 风格对齐 bss/user/detail-view.ts(用户详情规格),但师傅详情为多端点取子集,
// 与用户 14 段聚合不同源——两者不共享组件,不新增全站通用组件。

export type WColSpec =
  | { kind: 'time' }
  | { kind: 'tag'; domain: string } // StatusTag 注册域(ticket/message)
  | { kind: 'enum'; map: Record<string, string> } // 值 → workerPage.d 文案键
  | { kind: 'plain' }

export interface WDetailCol {
  /** 子集行字段名(JSON key)。 */
  key: string
  /** 列名文案键(workerPage.d 字典)。 */
  k: string
  spec?: WColSpec
}

export interface WDetailSection {
  /** 关联端点段键(= workerPage.sectionNames 的键)。 */
  key: string
  /** 取数子集端点路径。 */
  api: string
  cols: WDetailCol[]
  /** 分段列表展示上限(超限折叠,抽屉内可展开查看全部)。 */
  limit: number
}

/** 分段列表展示上限(默认 5 行,超限折叠——'有节制地展示')。 */
export const SECTION_LIMIT = 5

/** 在途(未完成)工单状态集:概览芯片「在途工单」计数口径。
 * 对齐 terms.md §4 派单工单枚举 PENDING/DOING/DONE/CANCELED。 */
export const ACTIVE_TICKET_STATUSES = ['PENDING', 'DOING'] as const

export const WORKER_DETAIL_SECTIONS: WDetailSection[] = [
  {
    key: 'tickets',
    api: '/dispatch/my-tickets',
    limit: SECTION_LIMIT,
    cols: [
      { key: 'ticketNo', k: 'dTicketNo' },
      { key: 'status', k: 'dStatus', spec: { kind: 'tag', domain: 'ticket' } },
      { key: 'regionName', k: 'dRegion', spec: { kind: 'plain' } },
      { key: 'createdAt', k: 'dCreatedAt', spec: { kind: 'time' } },
    ],
  },
  {
    key: 'messages',
    api: '/worker-messages',
    limit: SECTION_LIMIT,
    cols: [
      { key: 'level', k: 'dLevel', spec: { kind: 'tag', domain: 'message' } },
      { key: 'title', k: 'dTitle' },
      { key: 'read', k: 'dRead', spec: { kind: 'enum', map: { true: 'dYes', false: 'dNo' } } },
      { key: 'sentAt', k: 'dCreatedAt', spec: { kind: 'time' } },
    ],
  },
  {
    key: 'performances',
    api: '/worker-performances',
    limit: SECTION_LIMIT,
    cols: [
      { key: 'period', k: 'dPeriod' },
      { key: 'finished', k: 'dFinished' },
      { key: 'onTimeRate', k: 'dOnTimeRate' },
      { key: 'score', k: 'dScore' },
    ],
  },
  {
    key: 'feedbacks',
    api: '/worker-feedbacks',
    limit: SECTION_LIMIT,
    cols: [
      { key: 'customerName', k: 'dCustomer' },
      { key: 'score', k: 'dScoreStar' },
      { key: 'needReview', k: 'dNeedReview', spec: { kind: 'enum', map: { true: 'dYes', false: 'dNo' } } },
    ],
  },
]

/** 主档状态 → workerPage 文案键。terms.md §4 workers.status:1在职 / 0离职。 */
export const WORKER_STATUS_KEY: Record<number, string> = {
  1: 'active',
  0: 'left',
}

/** 抽屉级字面量文案键:worker-detail-drawer.tsx 直接引用、不在规格列 k / 字段 labelKey 内
 * (概览芯片/展开收起/段内布尔)。保持与 worker-detail-drawer.tsx 字面量同步,
 * 由 worker-detail-i18n.test.ts 兜底——打错引用键即门禁红灯,而非线上露出英文键。 */
export const DRAWER_W_KEYS = [
  'dStatTickets', 'dStatTotalTickets', 'dStatMessages', 'dStatFeedbacks',
  'dYes', 'dNo', 'dSeeAll', 'dCollapse', 'dLeftAt', 'dJoinedAt',
] as const

/** 主档展示字段:值取 GET /workers/{workerId} 主档;groupId → 组名由列表行传入,
 * regionId → 区域名尽力经 /regions 映射(无权限时降级显示 id)。 */
export const WORKER_PROFILE_FIELDS = [
  { key: 'phone', labelKey: 'dPhone' },
  { key: 'regionId', labelKey: 'dRegion' },
  { key: 'groupId', labelKey: 'dGroup' },
  { key: 'joinedAt', labelKey: 'dJoinedAt' },
  { key: 'leftAt', labelKey: 'dLeftAt' },
] as const