// 容量视图页:列名以 fields.md §4.2.2 为准;契约 GET /resources/capacity + POST /resources/capacity/alert-scan。
// 使用率 >=80% 行预警高亮(与后端告警阈值 resource.CapacityWarnThresholdPct 对齐)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { Dropdown } from '../../../components/Dropdown'
import { Pagination } from '../../../components/Pagination'
import { pageSlice, type CapacityRow } from '../types'
import { TableStateRow } from '../../../components/business'

const ALERT_THRESHOLD_PCT = 80 // fields.md §4.2.2 预警阈值

export default function CapacityPage() {
  const t = useT()
  const r = t.pages.capacityPage
  const [rows, setRows] = useState<CapacityRow[]>([])
  const [dim, setDim] = useState('')
  const [error, setError] = useState('')
  const [scanMsg, setScanMsg] = useState('')
  const [busy, setBusy] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)

  const load = (d: string) => {
    setError('')
    setBusy(true)
    apiFetch<{ items: CapacityRow[] }>('/resources/capacity', { query: { dim: d || undefined, order: 'usageDesc' } })
      .then((x) => setRows(x?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : r.loadFail))
      .finally(() => setBusy(false))
  }

  const runScan = () => {
    setScanMsg('')
    setBusy(true)
    apiFetch<{ scanned: number; created: number; resolved: number }>('/resources/capacity/alert-scan', { method: 'POST' })
      .then((x) => {
        setScanMsg(r.scanDone.replace(
          '{scanned}', String(x?.scanned ?? 0),
        ).replace('{created}', String(x?.created ?? 0)).replace('{resolved}', String(x?.resolved ?? 0)))
        load(dim)
      })
      .catch((e) => setScanMsg(e instanceof Error ? e.message : r.scanFail))
      .finally(() => setBusy(false))
  }

  useEffect(() => { load('') }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const dimLabel = (d: string) => (d === 'OLT' ? r.dimOlt : r.dimSplitter)
  const dimOptions = [
    { value: '', label: r.dimAll },
    { value: 'OLT', label: r.dimOlt },
    { value: 'SPLITTER', label: r.dimSplitter },
  ]
  const slice = pageSlice(rows, page, pageSize)
  const th = "h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"
  const td = "h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"

  return (
    <div>
      <PageHead title={r.title} desc={r.desc} />
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <Dropdown
            value={dim}
            options={dimOptions}
            onChange={(v) => { setDim(v); setPage(1); load(v) }}
            ariaLabel={r.dimAll}
            triggerStyle={{ minWidth: 160 }}
          />
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={() => load(dim)}>
            {t.pages.audit.refresh}
          </button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy} onClick={runScan}>
            {r.scan}
          </button>
        </div>
        {scanMsg ? <div className="mx-4 mb-3 rounded-sm border border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] px-3 py-2 text-[13px] text-[var(--shell-content-text)]">{scanMsg}</div> : null}
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead><tr>{[r.colObject, r.colType, r.colTotal, r.colUsed, r.colUsage].map((x) => <th key={x} className={th}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((row) => (
                  <tr key={row.resourceId} style={row.usageRate >= ALERT_THRESHOLD_PCT ? { background: "color-mix(in srgb, var(--color-warning) 12%, transparent)" } : undefined}>
                    <td className={td}>{row.name} ({row.code})</td>
                    <td className={td}>{dimLabel(row.type)}</td>
                    <td className={td}>{row.totalPorts}</td>
                    <td className={td}>{row.usedPorts}</td>
                    <td className={td}>
                      <span className="inline-flex items-center gap-2">
                        <span>{row.usageRate.toFixed(2)}%</span>
                        {row.usageRate >= ALERT_THRESHOLD_PCT && (
                          <span title={r.alertHint} className="rounded-sm px-1.5 py-0.5 text-[11px] font-medium" style={{ background: "color-mix(in srgb, var(--color-warning) 16%, transparent)", color: "var(--color-warning)" }}>{r.alertMark}</span>
                        )}
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={5} loading={busy} text={r.empty} />}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(r)} />
        </div>
      </div>
    </div>
  )
}