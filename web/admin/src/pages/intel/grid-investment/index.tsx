// 投资测算页:网格/城市/分光容量三视图(P-INFRA-1 W2 基础 + W5 深化,只读读模型)。
// 口径 docs/contract/fields.md 1.5.11/1.5.15;成本 null 显示「未登记」,禁止显示 0。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Card } from '../../../components/ui/card'
import { ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { PageHead } from '../../org/shared'
import type { CityInvestmentRow, GridInvestmentRow, SplitCapacityReport } from '../types'
import GridView from './grid'
import CityView from './city'
import CapacityView from './capacity'

type ViewKey = 'grid' | 'city' | 'capacity'

const TAB_ON = 'h-8 cursor-pointer rounded-sm border px-4 text-[13px] font-medium border-[var(--color-border-hover)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-heading)]'
const TAB_OFF = 'h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]'

export default function GridInvestmentPage() {
  const t = useT()
  const a = t.pages.gridInvestmentPage
  const [view, setView] = useState<ViewKey>('grid')
  const [gridRows, setGridRows] = useState<GridInvestmentRow[]>([])
  const [cityRows, setCityRows] = useState<CityInvestmentRow[]>([])
  const [capacity, setCapacity] = useState<SplitCapacityReport | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    Promise.all([
      apiFetch<{ items: GridInvestmentRow[] }>('/odn/grid-investment'),
      apiFetch<{ items: CityInvestmentRow[] }>('/odn/city-investment'),
      apiFetch<SplitCapacityReport>('/odn/split-capacity'),
    ])
      .then(([g, c, s]) => {
        setGridRows(g?.items ?? [])
        setCityRows(c?.items ?? [])
        setCapacity(s ?? null)
      })
      .catch((e) => {
        const msg = e instanceof Error ? e.message : a.loadFail
        setError(msg)
        toast.error(a.loadFail, { description: msg })
      })
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const tabs: { key: ViewKey; label: string }[] = [
    { key: 'grid', label: a.tabGrid },
    { key: 'city', label: a.tabCity },
    { key: 'capacity', label: a.tabCapacity },
  ]

  return (
    <div>
      <PageHead title={a.title} desc={a.desc} />
      <Card>
        <div className="mb-3 flex items-center justify-between">
          <div className="flex items-center gap-2">
            {tabs.map((tb) => (
              <button key={tb.key} onClick={() => setView(tb.key)}
                className={view === tb.key ? TAB_ON : TAB_OFF}>
                {tb.label}
              </button>
            ))}
          </div>
          <ToolbarButton onClick={load} disabled={busy}>{t.pages.audit.refresh}</ToolbarButton>
        </div>
        {error ? (
          <div className="px-4 pb-3"><ErrorBanner message={error} /></div>
        ) : (
          <>
            {view === 'grid' && <GridView rows={gridRows} busy={busy} />}
            {view === 'city' && <CityView rows={cityRows} busy={busy} />}
            {view === 'capacity' && <CapacityView report={capacity} busy={busy} />}
          </>
        )}
      </Card>
    </div>
  )
}