// 分光容量视图(W5):设备容量清单+全网汇总卡(「PON 口还能接几户」容量视角,fields.md 1.5.15)。
import { useMemo, useState } from 'react'
import { useT } from '../../../i18n'
import { Pagination } from '../../../components/Pagination'
import { TableStateRow } from '../../../components/business'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { CardFooter } from '../../../components/ui/card'
import { pagerTexts } from '../../org/shared'
import { pageSlice } from '../types'
import type { SplitCapacityReport, SplitCapacityRow } from '../types'
import { TD_CLS } from './head'

type CapSortKey = 'ratio' | 'usedPorts' | 'expandable' | 'chainRows'

const CHIP_CLS = 'rounded-sm border border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] px-3 py-2'

export default function CapacityView({ report, busy }: { report: SplitCapacityReport | null; busy: boolean }) {
  const t = useT()
  const a = t.pages.gridInvestmentPage
  const [sort, setSort] = useState<{ key: CapSortKey; dir: 1 | -1 } | null>(null)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)

  const items = report?.items ?? []
  const sorted = useMemo(() => {
    if (!sort) return items
    return [...items].sort((x, y) => (x[sort.key] - y[sort.key]) * sort.dir)
  }, [items, sort])
  const slice = pageSlice(sorted, page, pageSize)
  const onSort = (k: CapSortKey) =>
    setSort((s) => (s?.key === k ? { key: k, dir: s.dir === 1 ? -1 : 1 } : { key: k, dir: -1 }))
  const head = (label: string, k?: CapSortKey) => (
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
  const s = report?.summary
  const chips: { label: string; value: number }[] = s
    ? [
        { label: a.summaryDevices, value: s.devices },
        { label: a.summaryPotentialHomes, value: s.potentialHomes },
        { label: a.summaryConnectedHomes, value: s.connectedHomes },
        { label: a.summaryExpandableHomes, value: s.expandableHomes },
      ]
    : []

  return (
    <>
      <div className="mb-3 flex flex-wrap gap-2">
        {chips.map((c) => (
          <div key={c.label} className={CHIP_CLS}>
            <div className="text-xs text-[var(--shell-group-title)]">{c.label}</div>
            <div className="text-base font-semibold text-[var(--shell-heading)]">{c.value.toLocaleString()}</div>
          </div>
        ))}
      </div>
      <Table>
        <TableHeader>
          <TableRow>
            {head(a.colDevice)}
            {head(a.colKind)}
            {head(a.colLevel)}
            {head(a.colRatio, 'ratio')}
            {head(a.colChainRows, 'chainRows')}
            {head(a.colUsedPorts, 'usedPorts')}
            {head(a.colExpandable, 'expandable')}
            {head(a.colSecondary)}
            {head(a.colScope)}
            {head(a.colLifecycle)}
          </TableRow>
        </TableHeader>
        <TableBody>
          {slice.map((r) => (
            <CapRow key={r.deviceId} r={r} />
          ))}
          {!slice.length && <TableStateRow colSpan={10} loading={busy} text={a.emptyCapacity} />}
        </TableBody>
      </Table>
      <CardFooter>
        <Pagination total={sorted.length} page={page} pageSize={pageSize}
          onPage={setPage} onSize={setPageSize} {...pagerTexts(a)} />
      </CardFooter>
    </>
  )
}

function CapRow({ r }: { r: SplitCapacityRow }) {
  const t = useT()
  const a = t.pages.gridInvestmentPage
  const scope = r.prvCode && r.cityPrefix
    ? `${r.prvCode}-${r.cityPrefix}`
    : a.scopeImport
  return (
    <TableRow>
      <TableCell className={TD_CLS}>{r.code}</TableCell>
      <TableCell className={TD_CLS}>{r.kind}</TableCell>
      <TableCell className={TD_CLS}>{r.splitLevel === 1 ? a.level1 : a.level2}</TableCell>
      <TableCell className={TD_CLS}>{r.ratio}</TableCell>
      <TableCell className={TD_CLS}>{r.chainRows}</TableCell>
      <TableCell className={TD_CLS}>{r.usedPorts}</TableCell>
      <TableCell className={TD_CLS}>{r.expandable}</TableCell>
      <TableCell className={TD_CLS}>{r.hasSecondary ? a.yes : a.no}</TableCell>
      <TableCell className={TD_CLS}>{scope}</TableCell>
      <TableCell className={TD_CLS}>{r.lifecycleStatus}</TableCell>
    </TableRow>
  )
}
