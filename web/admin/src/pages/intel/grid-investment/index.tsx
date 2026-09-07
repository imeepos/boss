// 投资测算页:网格/城市/分光容量三视图(P-INFRA-1 W2 基础 + W5 深化,只读读模型)。
// 口径 docs/contract/fields.md 1.5.11/1.5.15;成本 null 显示「未登记」,禁止显示 0。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { CardShell } from '../../../components/business/charts'
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
      .catch((e) => setError(e instanceof Error ? e.message : a.loadFail))
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
      <CardShell>
        <div className="mb-3 flex items-center justify-between">
          <div className="flex items-center gap-2">
            {tabs.map((tb) => (
              <button key={tb.key} onClick={() => setView(tb.key)}
                className={view === tb.key ? TAB_ON : TAB_OFF}>
                {tb.label}
              </button>
            ))}
          </div>
          <button disabled={busy} onClick={load}
            className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]">
            {t.pages.audit.refresh}
          </button>
        </div>
        {error ? (
          <div className="mx-2 mb-2 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">
            {error}
          </div>
        ) : (
          <>
            {view === 'grid' && <GridView rows={gridRows} busy={busy} />}
            {view === 'city' && <CityView rows={cityRows} busy={busy} />}
            {view === 'capacity' && <CapacityView report={capacity} busy={busy} />}
          </>
        )}
      </CardShell>
    </div>
  )
}