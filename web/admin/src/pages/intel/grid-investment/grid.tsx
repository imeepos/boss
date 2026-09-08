// 网格视图:W2 全列 + W5 规划/材料成本(W5 口径 fields.md 1.5.11 口径 5/6)。
// 表头走 InvestTableHead(两行 rowSpan/colspan,保留自定义),body 走 ui-table/TableCell.
import { useMemo, useState } from 'react'
import { useT } from '../../../i18n'
import { Pagination } from '../../../components/Pagination'
import { TableStateRow } from '../../../components/business'
import { Table, TableBody, TableCell, TableRow } from '../../../components/ui/table'
import { CardFooter } from '../../../components/ui/card'
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
      <Table>
        <InvestTableHead sort={sort} onSort={onSort} />
        <TableBody>
          {slice.map((r) => (
            <TableRow key={`${r.prvCode}-${r.cityPrefix}-${r.gridCode}`}>
              <TableCell className={TD_CLS}>{gridLabel(r)}</TableCell>
              <TableCell className={TD_CLS}>{r.facilitiesPlanned}</TableCell>
              <TableCell className={TD_CLS}>{r.facilitiesInBuild}</TableCell>
              <TableCell className={TD_CLS}>{r.facilitiesInService}</TableCell>
              <TableCell className={TD_CLS}>{r.facilitiesRetired}</TableCell>
              <TableCell className={TD_CLS}>{r.coverageServed}</TableCell>
              <TableCell className={TD_CLS}>{r.coveragePending}</TableCell>
              <TableCell className={TD_CLS}>{r.coverageUnserved}</TableCell>
              <TableCell className={TD_CLS}><CostCell v={r.settledCost} /></TableCell>
              <TableCell className={TD_CLS}><CostCell v={r.plannedCost} /></TableCell>
              <TableCell className={TD_CLS}><CostCell v={r.materialCost} /></TableCell>
              <TableCell className={TD_CLS}><CostCell v={r.costPerServed} /></TableCell>
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
