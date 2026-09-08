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
import type { MonthlyRegion, MonthlyRow, MonthlySummary } from './types'

const MONTH_RE = /^[0-9]{4}-(0[1-9]|1[0-2])$/

export default function MonthlyPage() {
  const t = useT()
  const m = t.pages.monthlyPage
  const [tab, setTab] = useQueryState('tab', MONTHLY_TABLES[0].key)
  const [month, setMonth] = useQueryState('month', '')
  const [region, setRegion] = useQueryState('region', '')
  const [page, setPage] = useQueryInt('page', 1)
  const [pageSize, setPageSize] = useQueryInt('pageSize', 20)
  const [monthDraft, setMonthDraft] = useState(month)
  const [rows, setRows] = useState<MonthlyRow[]>([])
  const [total, setTotal] = useState(0)
  const [regions, setRegions] = useState<MonthlyRegion[]>([])
  const [summary, setSummary] = useState<MonthlySummary | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [ver, setVer] = useState(0)
  const [editRow, setEditRow] = useState<MonthlyRow | null>(null)

  const meta = useMemo(() => MONTHLY_TABLES.find((x) => x.key === tab) ?? MONTHLY_TABLES[0], [tab])

  useEffect(() => { setMonthDraft(month) }, [month])
  useEffect(() => {
    fetchRegions().then(setRegions).catch(() => setRegions([]))
  }, [])
  useEffect(() => {
    fetchSummary(month).then(setSummary).catch(() => setSummary(null))
  }, [month, ver])

  const loadFailText = m.loadFail
  const reload = useCallback(() => setVer((v) => v + 1), [])
  const load = useCallback(() => {
    let alive = true
    setBusy(true)
    setError('')
    fetchRows(meta.key, { month, region, page, pageSize })
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
  }, [meta.key, month, region, page, pageSize, loadFailText])
  useEffect(() => load(), [load]) // eslint-disable-line react-hooks/exhaustive-deps

  const commitMonth = () => {
    const v = monthDraft.trim()
    if (v === month) return
    if (v === '' || MONTH_RE.test(v)) setMonth(v)
  }

  const monthInvalid = monthDraft.trim() !== '' && !MONTH_RE.test(monthDraft.trim())
  const regionOptions = [
    { value: '', label: m.filterAll },
    ...regions.filter((r) => r.active).map((r) => ({ value: r.region, label: r.region })),
  ]

  return (
    <div>
      <PageHead title={m.title} desc={m.desc} />
      <KpiCards summary={summary} />
      <Card>
        <nav className="mb-4 flex items-center gap-1" aria-label={m.title}>
          {MONTHLY_TABLES.map((x, i) => (
            <button
              key={x.key}
              type="button"
              aria-current={x.key === meta.key ? 'page' : undefined}
              className={monthlyTabClass(x.key === meta.key)}
              onClick={() => setTab(x.key)}
            >
              {m.tabs[i]}
            </button>
          ))}
          <span className="flex-1" />
          <ToolbarButton onClick={reload} disabled={busy}>{t.pages.audit.refresh}</ToolbarButton>
        </nav>
        <div className="mb-3 flex flex-wrap items-center gap-3">
          <label className="flex items-center gap-2 text-[13px] text-[var(--shell-content-text)]">
            {m.filterMonth}
            <span className="w-32">
              <Input
                aria-label={m.filterMonth}
                value={monthDraft}
                placeholder="YYYY-MM"
                onChange={(e) => setMonthDraft(e.target.value)}
                onBlur={commitMonth}
                onKeyDown={(e) => { if (e.key === 'Enter') commitMonth() }}
              />
            </span>
          </label>
          <span className="w-56">
            <Dropdown
              value={region}
              options={regionOptions}
              onChange={(v) => setRegion(v)}
              ariaLabel={m.filterRegion}
              searchable
              searchPlaceholder={m.filterRegion}
              placeholder={m.filterAll}
            />
          </span>
          {monthInvalid && <span className="text-xs text-[var(--color-danger)]">{m.filterMonthInvalid}</span>}
        </div>
        <div className="mb-3">
          <ImportExport table={meta.key} month={month} onImported={() => setVer((v) => v + 1)} />
        </div>
        {error ? (
          <ErrorBanner message={error} />
        ) : (
          <DataTable
            meta={meta}
            rows={rows}
            loading={busy}
            empty={m.empty}
            editLabel={m.edit}
            onEdit={setEditRow}
          />
        )}
        <CardFooter>
          <Pagination
            total={total}
            page={page}
            pageSize={pageSize}
            onPage={setPage}
            onSize={(s) => { setPageSize(s); setPage(1) }}
            {...pagerTexts(m)}
          />
        </CardFooter>
      </Card>
      {editRow && (
        <EditDrawer
          key={editRow.month + '/' + editRow.region}
          meta={meta}
          row={editRow}
          onClose={() => setEditRow(null)}
          onSaved={() => { setEditRow(null); setVer((v) => v + 1); load() }}
        />
      )}
    </div>
  )
}
