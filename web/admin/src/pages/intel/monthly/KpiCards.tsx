// 月度汇总 KPI 卡(GET /monthly/summary)。四个比率分母为 0 时后端返 null,
// 渲染前一律先判空显示「—」,严禁出现 NaN%。
import { useT } from '../../../i18n'
import { StatCard } from '../../../components/business/charts'
import type { MonthlySummary } from './types'

function fmtInt(v: number): string {
  return v.toLocaleString('en-US')
}

function fmtRate(v: number | null | undefined): string {
  if (v === null || v === undefined || !Number.isFinite(v)) return '—'
  return (v * 100).toFixed(2) + '%'
}

function fmtMoney(v: number | null | undefined): string {
  if (v === null || v === undefined || !Number.isFinite(v)) return '—'
  return v.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

export function KpiCards({ summary }: { summary: MonthlySummary | null }) {
  const t = useT()
  const k = t.pages.monthlyPage.kpi
  return (
    <section className="mb-4 grid grid-cols-2 gap-4 md:grid-cols-3 xl:grid-cols-6">
      <StatCard label={k.closingActive} value={summary ? fmtInt(summary.closingActiveTotal) : '—'} />
      <StatCard label={k.totalRevenue} value={summary ? fmtInt(summary.totalRevenueTotal) : '—'} />
      <StatCard label={k.ontimeRate} value={fmtRate(summary?.ontimeRate)} />
      <StatCard label={k.portUtilization} value={fmtRate(summary?.portUtilization)} />
      <StatCard label={k.collectionRate} value={fmtRate(summary?.collectionRate)} />
      <StatCard label={k.arpu} value={fmtMoney(summary?.arpu)} />
    </section>
  )
}
