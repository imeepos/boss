// 网格视图:W2 全列 + W5 规划/材料成本(W5 口径 fields.md 1.5.11 口径 5/6)。
import { useMemo, useState } from 'react'
import { useT } from '../../../i18n'
import { Pagination } from '../../../components/Pagination'
import { TableStateRow } from '../../../components/business'
import { pagerTexts } from '../../org/shared'
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

export default function GridView({ rows, busy }: { rows: GridInvestmentRow[]; busy: boolean }) {
  const t = useT()
  const a = t.pages.gridInvestmentPage
  const [sort, setSort] = useState<SortState | null>(null)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)

  const sorted = useMemo(
    () => (sort ? [...rows].sort((x, y) => compareRows(x, y, sort.key, sort.dir)) : rows),
    [rows, sort],
  )
  const slice = pageSlice(sorted, page, pageSize)
  const onSort = (k: SortKey) =>
    setSort((s) => (s?.key === k ? { key: k, dir: s.dir === 1 ? -1 : 1 } : { key: k, dir: -1 }))

  return (
    <>
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
                <td className={TD_CLS}><CostCell v={r.plannedCost} /></td>
                <td className={TD_CLS}><CostCell v={r.materialCost} /></td>
                <td className={TD_CLS}><CostCell v={r.costPerServed} /></td>
              </tr>
            ))}
            {!slice.length && <TableStateRow colSpan={12} loading={busy} text={a.empty} />}
          </tbody>
        </table>
      </div>
      <div className="flex justify-end pt-3 text-xs text-[var(--shell-group-title)]">
        <Pagination total={sorted.length} page={page} pageSize={pageSize}
          onPage={setPage} onSize={setPageSize} {...pagerTexts(a)} />
      </div>
    </>
  )
}
