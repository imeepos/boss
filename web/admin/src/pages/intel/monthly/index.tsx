// 月度填报页(intel 组,menu:monthly)。三页签对齐三事实表(月×区域,分页缺省 1/20);
// 顶部 KPI 汇总卡(分母 0 时后端返 null,渲染判空);筛选=月份(YYYY-MM)+区域(白名单下拉);
// 派生列只读;CSV 导入/导出。
import { useCallback, useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import { useT } from '../../../i18n'
import { useQueryInt, useQueryState } from '../../../lib/useQueryState'
import { PageHead, pagerTexts } from '../../org/shared'
import { ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { Pagination } from '../../../components/Pagination'
import { Dropdown } from '../../../components/Dropdown'
import { Input } from '../../../components/ui/input'
import { Card, CardFooter } from '../../../components/ui/card'
import { fetchRegions, fetchRows, fetchSummary } from './api'
import { KpiCards } from './KpiCards'
import { DataTable } from './DataTable'
import { EditDrawer } from './EditDrawer'
import { ImportExport } from './ImportExport'
import { MONTHLY_TABLES } from './types'
import { monthlyTabClass } from './tabClass'
import type { MonthlyRegion, MonthlyRow, MonthlySummary, TableKey } from './types'

const MONTH_RE = /^[0-9]{4}-(0[1-9]|1[0-2])$/

export default function MonthlyPage() {
  const t = useT()
  const m = t.pages.monthlyPage
  const [tab, setTab] = useQueryState('tab', MONTHLY_TABLES[0].key)
  const [month, setMonth] = useQueryState('month', '')
  const [region, setRegion] = useQueryState('region', '')
  const [page, setPage] = useQueryInt('page', 1)
  const [pageSize, setPageSize] = useQueryInt('pageSize', 20)
  const [rows, setRows] = useState<MonthlyRow[]>([])
  const [total, setTotal] = useState(0)
  const [regions, setRegions] = useState<MonthlyRegion[]>([])
  const [summary, setSummary] = useState<MonthlySummary | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [ver, setVer] = useState(0)
  const [editRow, setEditRow] = useState<MonthlyRow | null>(null)

  const meta = useMemo(() => MONTHLY_TABLES.find((x) => x.key === tab) ?? MONTHLY_TABLES[0], [tab])
  const reload = useCallback(() => setVer((v) => v + 1), [])
  const load = useMonthlyRowsLoader(meta.key, month, region, page, pageSize, m.loadFail, setRows, setTotal, setError, setBusy)
  useEffect(() => load(), [load]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    fetchRegions().then(setRegions).catch(() => setRegions([]))
  }, [])
  useEffect(() => {
    fetchSummary(month).then(setSummary).catch(() => setSummary(null))
  }, [month, ver])

  return (
    <div>
      <PageHead title={m.title} desc={m.desc} />
      <KpiCards summary={summary} />
      <MonthlyCard
        meta={meta} month={month} setMonth={setMonth}
        region={region} setRegion={setRegion} regions={regions}
        rows={rows} total={total} page={page} setPage={setPage}
        pageSize={pageSize} setPageSize={setPageSize}
        busy={busy} error={error} editRow={editRow} setEditRow={setEditRow}
        onRefresh={reload} onImported={() => setVer((v) => v + 1)}
        setVer={setVer} load={load} labels={m} refreshLabel={t.pages.audit.refresh}
        onTabChange={setTab}
      />
    </div>
  )
}

type MonthlyCardProps = {
  meta: typeof MONTHLY_TABLES[number]
  month: string
  setMonth: (v: string) => void
  region: string
  setRegion: (v: string) => void
  regions: MonthlyRegion[]
  rows: MonthlyRow[]
  total: number
  page: number
  setPage: (n: number) => void
  pageSize: number
  setPageSize: (n: number) => void
  busy: boolean
  error: string
  editRow: MonthlyRow | null
  setEditRow: (r: MonthlyRow | null) => void
  onRefresh: () => void
  onImported: () => void
  setVer: React.Dispatch<React.SetStateAction<number>>
  load: () => void
  labels: ReturnType<typeof useT>['pages']['monthlyPage']
  refreshLabel: string
  onTabChange: (k: TableKey) => void
}

function MonthlyCard({
  meta, month, setMonth, region, setRegion, regions,
  rows, total, page, setPage, pageSize, setPageSize,
  busy, error, editRow, setEditRow,
  onRefresh, onImported, setVer, load, labels, refreshLabel, onTabChange,
}: MonthlyCardProps) {
  return (
    <Card>
      <MonthlyToolbar
        activeKey={meta.key} onTabChange={onTabChange}
        onRefresh={onRefresh} busy={busy} refreshLabel={refreshLabel}
      />
      <MonthlyFilters
        month={month} setMonth={setMonth}
        region={region} setRegion={setRegion}
        regions={regions}
      />
      <div className="mb-3">
        <ImportExport table={meta.key} month={month} onImported={onImported} />
      </div>
      <MonthlyTableAndPager
        meta={meta} rows={rows} busy={busy} error={error}
        empty={labels.empty} editLabel={labels.edit}
        onEdit={setEditRow}
        total={total} page={page} setPage={setPage}
        pageSize={pageSize} setPageSize={setPageSize}
        labels={labels}
      />
      {editRow && (
        <EditDrawer
          key={editRow.month + '/' + editRow.region}
          meta={meta}
          row={editRow}
          onClose={() => setEditRow(null)}
          onSaved={() => { setEditRow(null); setVer((v) => v + 1); load() }}
        />
      )}
    </Card>
  )
}

function MonthlyToolbar({
  activeKey, onTabChange, onRefresh, busy, refreshLabel,
}: {
  activeKey: TableKey
  onTabChange: (k: TableKey) => void
  onRefresh: () => void
  busy: boolean
  refreshLabel: string
}) {
  return (
    <MonthlyTabs active={activeKey} onChange={onTabChange} onRefresh={onRefresh} busy={busy} refreshLabel={refreshLabel} />
  )
}

function MonthlyTableAndPager({
  meta, rows, busy, error, empty, editLabel, onEdit,
  total, page, setPage, pageSize, setPageSize, labels,
}: {
  meta: typeof MONTHLY_TABLES[number]
  rows: MonthlyRow[]
  busy: boolean
  error: string
  empty: string
  editLabel: string
  onEdit: (r: MonthlyRow) => void
  total: number
  page: number
  setPage: (n: number) => void
  pageSize: number
  setPageSize: (n: number) => void
  labels: ReturnType<typeof useT>['pages']['monthlyPage']
}) {
  return (
    <>
      {error ? (
        <ErrorBanner message={error} />
      ) : (
        <DataTable
          meta={meta}
          rows={rows}
          loading={busy}
          empty={empty}
          editLabel={editLabel}
          onEdit={onEdit}
        />
      )}
      <CardFooter>
        <Pagination
          total={total}
          page={page}
          pageSize={pageSize}
          onPage={setPage}
          onSize={(s) => { setPageSize(s); setPage(1) }}
          {...pagerTexts(labels)}
        />
      </CardFooter>
    </>
  )
}

/** 拉取月度行 + 失败 toast 提示(用 useCallback 闭包保住 setter 引用稳定)。 */
function useMonthlyRowsLoader(
  key: TableKey,
  month: string,
  region: string,
  page: number,
  pageSize: number,
  loadFailText: string,
  setRows: (v: MonthlyRow[]) => void,
  setTotal: (v: number) => void,
  setError: (v: string) => void,
  setBusy: (v: boolean) => void,
) {
  return useCallback(() => {
    let alive = true
    setBusy(true)
    setError('')
    fetchRows(key, { month, region, page, pageSize })
      .then((d) => {
        if (!alive) return
        setRows(d?.items ?? [])
        setTotal(d?.total ?? 0)
      })
      .catch((e) => {
        if (!alive) return
        const msg = e instanceof Error ? e.message : loadFailText
        setError(msg)
        toast.error(loadFailText, { description: msg })
      })
      .finally(() => { if (alive) setBusy(false) })
    return () => { alive = false }
  }, [key, month, region, page, pageSize, loadFailText, setRows, setTotal, setError, setBusy])
}

function MonthlyTabs({
  active, onChange, onRefresh, busy, refreshLabel,
}: {
  active: TableKey
  onChange: (k: TableKey) => void
  onRefresh: () => void
  busy: boolean
  refreshLabel: string
}) {
  const t = useT()
  const m = t.pages.monthlyPage
  return (
    <nav className="mb-4 flex items-center gap-1" aria-label={m.title}>
      {MONTHLY_TABLES.map((x, i) => (
        <button
          key={x.key}
          type="button"
          aria-current={x.key === active ? 'page' : undefined}
          className={monthlyTabClass(x.key === active)}
          onClick={() => onChange(x.key)}
        >
          {m.tabs[i]}
        </button>
      ))}
      <span className="flex-1" />
      <ToolbarButton onClick={onRefresh} disabled={busy}>{refreshLabel}</ToolbarButton>
    </nav>
  )
}

function MonthlyFilters({
  month, setMonth, region, setRegion, regions,
}: {
  month: string
  setMonth: (v: string) => void
  region: string
  setRegion: (v: string) => void
  regions: MonthlyRegion[]
}) {
  const t = useT()
  const m = t.pages.monthlyPage
  const [draft, setDraft] = useState(month)
  useEffect(() => { setDraft(month) }, [month])
  const commit = () => {
    const v = draft.trim()
    if (v === month) return
    if (v === '' || MONTH_RE.test(v)) setMonth(v)
  }
  const invalid = draft.trim() !== '' && !MONTH_RE.test(draft.trim())
  const options = [
    { value: '', label: m.filterAll },
    ...regions.filter((r) => r.active).map((r) => ({ value: r.region, label: r.region })),
  ]
  return (
    <div className="mb-3 flex flex-wrap items-center gap-3">
      <label className="flex items-center gap-2 text-[13px] text-[var(--shell-content-text)]">
        {m.filterMonth}
        <span className="w-32">
          <Input
            aria-label={m.filterMonth}
            value={draft}
            placeholder="YYYY-MM"
            onChange={(e) => setDraft(e.target.value)}
            onBlur={commit}
            onKeyDown={(e) => { if (e.key === 'Enter') commit() }}
          />
        </span>
      </label>
      <span className="w-56">
        <Dropdown
          value={region}
          options={options}
          onChange={(v) => setRegion(v)}
          ariaLabel={m.filterRegion}
          searchable
          searchPlaceholder={m.filterRegion}
          placeholder={m.filterAll}
        />
      </span>
      {invalid && <span className="text-xs text-[var(--color-danger)]">{m.filterMonthInvalid}</span>}
    </div>
  )
}
