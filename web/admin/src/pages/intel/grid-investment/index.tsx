// 投资测算页:契约 GET /odn/grid-investment(P-INFRA-1 W2 只读读模型)。
// 指标口径 docs/contract/fields.md 1.5.11;成本 null 显示「未登记」,禁止显示 0。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Pagination } from '../../../components/Pagination'
import { TableStateRow } from '../../../components/business'
import { CardShell } from '../../../components/business/charts'
import { PageHead, pagerTexts } from '../../org/shared'
import { pageSlice } from '../types'
import type { GridInvestmentRow } from '../types'
import { CostCell, InvestTableHead, TD_CLS, gridLabel, sortValue } from './head'
import type { SortKey, SortState } from './head'

/** 未登记值恒排尾部,其余按方向排序。 */
function compareRows(x: GridInvestmentRow, y: GridInvestmentRow, key: SortKey, dir: 1 | -1): number {
  const vx = sortValue(x, key)
  const vy = sortValue(y, key)
  if (vx === null && vy === null) return 0
  if (vx === null) return 1
  if (vy === null) return -1
  return (vx - vy) * dir
}

export default function GridInvestmentPage() {
  const t = useT()
  const a = t.pages.gridInvestmentPage
  const [rows, setRows] = useState<GridInvestmentRow[]>([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [sort, setSort] = useState<SortState | null>(null)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: GridInvestmentRow[] }>('/odn/grid-investment')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : a.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const sorted = useMemo(
    () => (sort ? [...rows].sort((x, y) => compareRows(x, y, sort.key, sort.dir)) : rows),
    [rows, sort],
  )
  const slice = pageSlice(sorted, page, pageSize)
  const onSort = (k: SortKey) =>
    setSort((s) => (s?.key === k ? { key: k, dir: s.dir === 1 ? -1 : 1 } : { key: k, dir: -1 }))

  return (
    <div>
      <PageHead title={a.title} desc={a.desc} />
      <CardShell>
        <div className="mb-3 flex items-center justify-between">
          <h3 className="m-0 text-base font-semibold text-[var(--shell-heading)]">{a.title}</h3>
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
          <div className="overflow-x-auto">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <InvestTableHead sort={sort} onSort={onSort} />
              <tbody>
                {slice.map((r) => (
                  <tr key={`${r.prvCode}-${r.cityPrefix}-${r.gridCode}`}>
                    <td className={TD_CLS}>{gridLabel(r)}</td>
                    <td className={TD_CLS}>{r.facilitiesPlanned}</td>
                    <td className={TD_CLS}>{r.facilitiesInBuild}</td>
                    <td className={TD_CLS}>{r.facilitiesInService}</td>
                    <td className={TD_CLS}>{r.facilitiesRetired}</td>
                    <td className={TD_CLS}>{r.coverageServed}</td>
                    <td className={TD_CLS}>{r.coveragePending}</td>
                    <td className={TD_CLS}>{r.coverageUnserved}</td>
                    <td className={TD_CLS}><CostCell v={r.settledCost} /></td>
                    <td className={TD_CLS}><CostCell v={r.costPerServed} /></td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={11} loading={busy} text={a.empty} />}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end pt-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={sorted.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(a)} />
        </div>
      </CardShell>
    </div>
  )
}
