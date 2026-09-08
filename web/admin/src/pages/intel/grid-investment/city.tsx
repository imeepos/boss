// 城市卷积视图(W5 顺序④):网格行按 (prv,city) 卷积 + 分光容量户级列(潜在/已接/可扩)。
import { useMemo, useState } from 'react'
import { useT } from '../../../i18n'
import { Pagination } from '../../../components/Pagination'
import { TableStateRow } from '../../../components/business'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { CardFooter } from '../../../components/ui/card'
import { pagerTexts } from '../../org/shared'
import { pageSlice } from '../types'
import type { CityInvestmentRow } from '../types'
import { CostCell, TD_CLS } from './head'

type CitySortKey =
  | 'facilitiesInService' | 'coverageServed'
  | 'settledCost' | 'plannedCost' | 'materialCost' | 'costPerServed'
  | 'potentialHomes' | 'connectedHomes' | 'expandableHomes' | 'costPerPotential'

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
    <TableHead>
      {k ? (
        <button type="button" onClick={() => onSort(k)}
          className="flex cursor-pointer items-center gap-1 text-xs font-medium hover:text-[var(--shell-heading)]">
          {label}
          {sort?.key === k && <span aria-hidden>{sort.dir === 1 ? '↑' : '↓'}</span>}
        </button>
      ) : label}
    </TableHead>
  )

  return (
    <>
      <Table>
        <TableHeader>
          <TableRow>
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
          </TableRow>
        </TableHeader>
        <TableBody>
          {slice.map((r) => (
            <TableRow key={`${r.prvCode}-${r.cityPrefix}`}>
              <TableCell className={TD_CLS}>{cityLabel(r)}</TableCell>
              <TableCell className={TD_CLS}>{r.gridCount}</TableCell>
              <TableCell className={TD_CLS}>{r.facilitiesInService}</TableCell>
              <TableCell className={TD_CLS}>{r.coverageServed}</TableCell>
              <TableCell className={TD_CLS}><CostCell v={r.settledCost} /></TableCell>
              <TableCell className={TD_CLS}><CostCell v={r.plannedCost} /></TableCell>
              <TableCell className={TD_CLS}><CostCell v={r.materialCost} /></TableCell>
              <TableCell className={TD_CLS}><CostCell v={r.costPerServed} /></TableCell>
              <TableCell className={TD_CLS}><CountCell v={r.potentialHomes} /></TableCell>
              <TableCell className={TD_CLS}><CountCell v={r.connectedHomes} /></TableCell>
              <TableCell className={TD_CLS}><CountCell v={r.expandableHomes} /></TableCell>
              <TableCell className={TD_CLS}><CostCell v={r.costPerPotential} /></TableCell>
            </TableRow>
          ))}
          {!slice.length && <TableStateRow colSpan={12} loading={busy} text={a.empty} />}
        </TableBody>
      </Table>
      <CardFooter>
        <Pagination total={sorted.length} page={page} pageSize={pageSize}
          onPage={setPage} onSize={setPageSize} {...pagerTexts(a)} />
      </CardFooter>
    </>
  )
}

/** 户数单元格:null=无容量建模(显示未登记,禁止 0)。 */
function CountCell({ v }: { v: number | null }) {
  const unregistered = useT().pages.gridInvestmentPage.unregistered
  if (v === null) return <span className="text-[var(--shell-group-title)]">{unregistered}</span>
  return <>{v.toLocaleString()}</>
}
