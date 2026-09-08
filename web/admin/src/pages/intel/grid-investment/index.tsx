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
  const [ver, setVer] = useState(0)

  useEffect(() => { void loadAll(setGridRows, setCityRows, setCapacity, setError, setBusy, a.loadFail) }, [ver]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div>
      <PageHead title={a.title} desc={a.desc} />
      <Card>
        <Toolbar
          view={view} setView={setView}
          busy={busy} onRefresh={() => setVer((v) => v + 1)}
          labels={{ grid: a.tabGrid, city: a.tabCity, capacity: a.tabCapacity, refresh: t.pages.audit.refresh }}
        />
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

async function loadAll(
  setGridRows: (v: GridInvestmentRow[]) => void,
  setCityRows: (v: CityInvestmentRow[]) => void,
  setCapacity: (v: SplitCapacityReport | null) => void,
  setError: (v: string) => void,
  setBusy: (v: boolean) => void,
  loadFailText: string,
) {
  setError('')
  setBusy(true)
  try {
    const [g, c, s] = await Promise.all([
      apiFetch<{ items: GridInvestmentRow[] }>('/odn/grid-investment'),
      apiFetch<{ items: CityInvestmentRow[] }>('/odn/city-investment'),
      apiFetch<SplitCapacityReport>('/odn/split-capacity'),
    ])
    setGridRows(g?.items ?? [])
    setCityRows(c?.items ?? [])
    setCapacity(s ?? null)
  } catch (e) {
    const msg = e instanceof Error ? e.message : loadFailText
    setError(msg)
    toast.error(loadFailText, { description: msg })
  } finally {
    setBusy(false)
  }
}

function Toolbar({
  view, setView, busy, onRefresh, labels,
}: {
  view: ViewKey
  setView: (v: ViewKey) => void
  busy: boolean
  onRefresh: () => void
  labels: { grid: string; city: string; capacity: string; refresh: string }
}) {
  const tabs: { key: ViewKey; label: string }[] = [
    { key: 'grid', label: labels.grid },
    { key: 'city', label: labels.city },
    { key: 'capacity', label: labels.capacity },
  ]
  return (
    <div className="mb-3 flex items-center justify-between">
      <div className="flex items-center gap-2">
        {tabs.map((tb) => (
          <button key={tb.key} onClick={() => setView(tb.key)}
            className={view === tb.key ? TAB_ON : TAB_OFF}>
            {tb.label}
          </button>
        ))}
      </div>
      <ToolbarButton onClick={onRefresh} disabled={busy}>{labels.refresh}</ToolbarButton>
    </div>
  )
}
