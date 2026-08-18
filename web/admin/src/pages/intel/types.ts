// 经营分析域行类型:对齐 internal/domain/{gis,analytics,report}。
export { pageSlice } from '../quad/types'

/** 八级下钻节点:GET /gis/drill?level&parentId。 */
export interface GisNode {
  id: number
  name: string
  level: number
  count: number
}

/** 资源实时详情:GET /gis/resources/:resourceId/detail。 */
export interface GisResourceDetail {
  id: number
  code: string
  name: string
  type: string
  status: string
  addressId: number
  opticalPower?: number | null
  packetLoss?: number | null
  customerName: string
  portsUsed: number
  portsTotal: number
  collectedAt?: string | null
}

/** 五大指标:GET /analytics/indicators(indicators)。 */
export interface IndicatorRow {
  key: string
  name: string
  value: number
  detail: string
}

/** 区域 ROI:GET /analytics/indicators(regionROI)。 */
export interface RegionRoiRow {
  regionId: number
  regionName: string
  revenue: number
  investment: number
  roi: number
}

/** 热力格子:GET /analytics/heatmap(items)。 */
export interface HeatCellRow {
  addressId: number
  name: string
  level: number // 4 小区 / 5 楼栋
  portsTotal: number
  portsUsed: number
  utilization: number // 0~1
}

/** 维护清单行:GET /analytics/maintenance(items)。 */
export interface AnalyticsMaintRow {
  deviceNo: string
  deviceType: string
  healthScore: number
  faultCount: number
  ageYears: number
  reason: string
  priority: string // MUST_REPLACE/SUGGEST/WATCH
}

/** 报告快照:GET /reports(items)。 */
export interface ReportRow {
  id: number
  period: string // daily/weekly/monthly/quarterly
  windowStart: string
  windowEnd: string
  createdAt: string
}

/** 报告正文:GET /reports/latest?period(payload)。 */
export interface ReportPayload {
  generatedAt: string
  indicators: IndicatorRow[]
  regionROI: RegionRoiRow[]
  heatmap: HeatCellRow[]
  maintenance: AnalyticsMaintRow[]
  conclusion?: string
}
