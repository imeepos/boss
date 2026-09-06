// 报告中心纯函数:周期人类可读名 + 列表客户端筛选(后端 /reports 无服务端筛选参数)。
// 抽离以便 vitest 覆盖(列表 UI 与筛选共用同一份逻辑)。
import type { ReportRow } from '../types'

/** 四个标准周期键(与后端 report periods 对齐)。 */
export const PERIODS = ['daily', 'weekly', 'monthly', 'quarterly'] as const

/** 周期名素材:四标准周期文案 + 两条特殊流水线快照的专名(字段名对齐 i18n key)。 */
export interface PeriodLabels {
  periods: readonly string[]
  periodRecon: string
  periodOssAudit: string
}

/**
 * 周期值 → 人类可读名:标准周期查文案表,recon-daily/oss-audit-daily 走专名,
 * 其余未知周期值原样透出(表格不吞真实数据)。
 */
export function periodLabel(p: string, labels: PeriodLabels): string {
  const i = (PERIODS as readonly string[]).indexOf(p)
  if (i >= 0) return labels.periods[i]
  if (p === 'recon-daily') return labels.periodRecon
  if (p === 'oss-audit-daily') return labels.periodOssAudit
  return p
}

/**
 * 客户端筛选:period 非空时按原始周期键等值过滤;
 * q 命中 ID / 原始周期键 / 本地化周期名(大小写不敏感,首尾空白忽略)。
 */
export function filterReports(
  rows: ReportRow[],
  period: string,
  q: string,
  labels: PeriodLabels,
): ReportRow[] {
  const kw = q.trim().toLowerCase()
  return rows.filter((x) => {
    if (period && x.period !== period) return false
    if (!kw) return true
    return (
      String(x.id).includes(kw) ||
      x.period.toLowerCase().includes(kw) ||
      periodLabel(x.period, labels).toLowerCase().includes(kw)
    )
  })
}