// 月度填报域类型与三表元数据。列序对齐 docs/books 三张标准 CSV 表头;
// derived=true 的列为模板灰色「公式勿填」列:响应含、PUT 请求体无、前端禁编辑。
export type TableKey = 'user-revenue' | 'network-delivery' | 'finance-cost'

export interface PageResult<T> {
  items: T[]
  total: number
  page: number
  pageSize: number
}

export interface MonthlyRegion {
  region: string
  active: boolean
}

export interface MonthlySummary {
  month: string
  closingActiveTotal: number
  totalRevenueTotal: number
  ontimeRate: number | null
  portUtilization: number | null
  collectionRate: number | null
  arpu: number | null
}

export interface MonthlyRowError {
  line: number
  reason: string
}

export interface ImportResult {
  total: number
  imported: number
  failed: number
  errors?: MonthlyRowError[]
}

export interface MonthlyRow {
  month: string
  region: string
  [field: string]: string | number | boolean | undefined
}

/** 字段元:derived=服务端派生列(GENERATED),只读展示。 */
export interface FieldMeta {
  key: string
  derived?: boolean
}

export interface TableMeta {
  key: TableKey
  /** i18n 列头数组 key(pages.monthlyPage 下),长度与 fields 一致。 */
  columnsKey: 'columnsUserRevenue' | 'columnsNetwork' | 'columnsFinance'
  fields: FieldMeta[]
}

// 三表元数据:列序与 fields 顺序逐一对齐 i18n 列头数组;派生列位置同模板灰列。
export const MONTHLY_TABLES: TableMeta[] = [
  {
    key: 'user-revenue',
    columnsKey: 'columnsUserRevenue',
    fields: [
      { key: 'month' }, { key: 'region' },
      { key: 'openingActive' }, { key: 'newUsers' }, { key: 'churnedUsers' }, { key: 'adjustedUsers' },
      { key: 'closingActive', derived: true },
      { key: 'broadbandRevenue' }, { key: 'valueAddedRevenue' }, { key: 'onetimeCharge' },
      { key: 'discountAmount' }, { key: 'refundReversal' },
      { key: 'totalRevenue', derived: true },
    ],
  },
  {
    key: 'network-delivery',
    columnsKey: 'columnsNetwork',
    fields: [
      { key: 'month' }, { key: 'region' },
      { key: 'installRequests' }, { key: 'ontimeCompletions' },
      { key: 'portsDeployed' }, { key: 'portsActive' },
      { key: 'faultReports' }, { key: 'repairHours' },
    ],
  },
  {
    key: 'finance-cost',
    columnsKey: 'columnsFinance',
    fields: [
      { key: 'month' }, { key: 'region' },
      { key: 'invoicedAmount' }, { key: 'collectedAmount' }, { key: 'receivableEnding' },
      { key: 'directCost' }, { key: 'fixedCost' }, { key: 'capexInvest' },
    ],
  },
]
