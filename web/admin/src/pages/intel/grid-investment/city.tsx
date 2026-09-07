// 城市卷积视图(W5 顺序④):网格行按 (prv,city) 卷积 + 分光容量户级列(潜在/已接/可扩)。
import { useMemo, useState } from 'react'
import { useT } from '../../../i18n'
import { Pagination } from '../../../components/Pagination'
import { TableStateRow } from '../../../components/business'
import { pagerTexts } from '../../org/shared'
import { pageSlice } from '../types'
import type { CityInvestmentRow } from '../types'
import { CostCell, TD_CLS } from './head'

type CitySortKey =
  | 'facilitiesInService' | 'coverageServed'
  | 'settledCost' | 'plannedCost' | 'materialCost' | 'costPerServed'
  | 'potentialHomes' | 'connectedHomes' | 'expandableHomes' | 'costPerPotential'

const TH_CLS = 'h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]'

function cityLabel(r: CityInvestmentRow): string {
  return r.cityPrefix
}

function num(v: number | null): number | null {
  return v
}

export default function CityView({ rows, busy }: { rows: CityInvestmentRow[]; busy: boolean }) {
  const t = useT()
  const a = t.pages.gridInvestmentPage
  const [sort, setSort] = useState<{ key: CitySortKey; dir: 1 | -1 } | null>(null)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)

  const sorted = useMemo(() => {
    if (!sort) return rows
    return [...rows].sort((x, y) => {
      const vx = num(x[sort.key] as number | null)
      const vy = num(y[sort.key] as number | null)
      if (vx === null && vy === null) return 0
      if (vx === null) return 1
      if (vy === null) return -1
      return (vx - vy) * sort.dir
    })
  }, [rows, sort])
  const slice = pageSlice(sorted, page, pageSize)
  const onSort = (k: CitySortKey) =>
    setSort((s) => (s?.key === k ? { key: k, dir: s.dir === 1 ? -1 : 1 } : { key: k, dir: -1 }))
  const head = (label: string, k?: CitySortKey) => (
    <th className={TH_CLS}>
      {k ? (
        <button type="button" onClick={() => onSort(k)}
          className="flex cursor-pointer items-center gap-1 text-xs font-medium hover:text-[var(--shell-heading)]">
          {label}
          {sort?.key === k && <span aria-hidden>{sort.dir === 1 ? '↑' : '↓'}</span>}
        </button>
      ) : label}
    </th>
  )

  return (
    <>
      <div className="overflow-x-auto">
        <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
          <thead>
            <tr>
              {head(a.colCity)}
              {head(a.colGridCount)}
              {head(a.colInService, 'facilitiesInService')}
              {head(a.colServed, 'coverageServed')}
              {head(a.colSettledCost, 'settledCost')}
              {head(a.colPlannedCost, 'plannedCost')}
              {head(a.colMaterialCost, 'materialCost')}
              {head(a.colCostPerServed, 'costPerServed')}
              {head(a.colPotentialHomes, 'potentialHomes')}
              {head(a.colConnectedHomes, 'connectedHomes')}
              {head(a.colExpandableHomes, 'expandableHomes')}
              {head(a.colCostPerPotential, 'costPerPotential')}
            </tr>
          </thead>
          <tbody>
            {slice.map((r) => (
              <tr key={`${r.prvCode}-${r.cityPrefix}`}>
                <td className={TD_CLS}>{cityLabel(r)}</td>
                <td className={TD_CLS}>{r.gridCount}</td>
                <td className={TD_CLS}>{r.facilitiesInService}</td>
                <td className={TD_CLS}>{r.coverageServed}</td>
                <td className={TD_CLS}><CostCell v={r.settledCost} /></td>
                <td className={TD_CLS}><CostCell v={r.plannedCost} /></td>
                <td className={TD_CLS}><CostCell v={r.materialCost} /></td>
                <td className={TD_CLS}><CostCell v={r.costPerServed} /></td>
                <td className={TD_CLS}><CountCell v={r.potentialHomes} /></td>
                <td className={TD_CLS}><CountCell v={r.connectedHomes} /></td>
                <td className={TD_CLS}><CountCell v={r.expandableHomes} /></td>
                <td className={TD_CLS}><CostCell v={r.costPerPotential} /></td>
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

/** 户数单元格:null=无容量建模(显示未登记,禁止 0)。 */
function CountCell({ v }: { v: number | null }) {
  const unregistered = useT().pages.gridInvestmentPage.unregistered
  if (v === null) return <span className="text-[var(--shell-group-title)]">{unregistered}</span>
  return <>{v.toLocaleString()}</>
}