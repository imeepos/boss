// 投资测算表头与成本单元格:两行表头(网格 | 设施数×生命周期 | 覆盖地址数 | 成本两列)。
// 排序交互:点击数值列头切换升降序;未登记(null)恒排尾部。口径 fields.md 1.5.11。
// 备注:表头两行含 rowSpan/colspan,保留原生 <thead>/<tr>/<th> 写法(非「裸 table」模式);
// body 走 ui-table/TableCell,TD_CLS 仍导出供各 view 注入 className 保留 hover 反馈.
import { useT } from '../../../i18n'
import { formatCurrency } from '../../../components/business/charts/format-value'
import type { GridInvestmentRow } from '../types'

export type SortKey =
  | 'facilitiesPlanned' | 'facilitiesInBuild' | 'facilitiesInService' | 'facilitiesRetired'
  | 'coverageServed' | 'coveragePending' | 'coverageUnserved'
  | 'settledCost' | 'plannedCost' | 'materialCost' | 'costPerServed'

export interface SortState { key: SortKey; dir: 1 | -1 }

export function sortValue(r: GridInvestmentRow, k: SortKey): number | null {
  return r[k] as number | null
}

export function gridLabel(r: GridInvestmentRow): string {
  const code = `${r.cityPrefix}-${String(r.gridCode).padStart(2, '0')}`
  return r.gridName ? `${code} ${r.gridName}` : code
}

const TH_CLS = 'h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]'
/** 表格行单元格样式(供各 view 在 TableCell className 注入),保留 hover 反馈。 */
export const TD_CLS = 'h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]'

function SortHead({ label, sortKey, sort, onSort, span }: {
  label: string
  sortKey: SortKey
  sort: SortState | null
  onSort: (k: SortKey) => void
  span?: number
}) {
  const active = sort?.key === sortKey
  return (
    <th colSpan={span} className={TH_CLS}>
      <button type="button" onClick={() => onSort(sortKey)}
        className="flex cursor-pointer items-center gap-1 text-xs font-medium hover:text-[var(--shell-heading)]">
        {label}
        {active && <span aria-hidden>{sort?.dir === 1 ? '↑' : '↓'}</span>}
      </button>
    </th>
  )
}

/** 表头两行:网格 | 设施数(生命周期) | 覆盖地址数 | 投资成本(四列,W5 扩规划/材料);数值列可排序。 */
export function InvestTableHead({ sort, onSort }: { sort: SortState | null; onSort: (k: SortKey) => void }) {
  const a = useT().pages.gridInvestmentPage
  return (
    <thead>
      <tr>
        <th rowSpan={2} className={TH_CLS}>{a.colGrid}</th>
        <SortHead label={a.groupFacilities} sortKey="facilitiesPlanned" sort={sort} onSort={onSort} span={4} />
        <SortHead label={a.groupCoverage} sortKey="coverageServed" sort={sort} onSort={onSort} span={3} />
        <SortHead label={a.groupCost} sortKey="settledCost" sort={sort} onSort={onSort} span={4} />
      </tr>
      <tr>
        <SortHead label={a.colPlanned} sortKey="facilitiesPlanned" sort={sort} onSort={onSort} />
        <SortHead label={a.colInBuild} sortKey="facilitiesInBuild" sort={sort} onSort={onSort} />
        <SortHead label={a.colInService} sortKey="facilitiesInService" sort={sort} onSort={onSort} />
        <SortHead label={a.colRetired} sortKey="facilitiesRetired" sort={sort} onSort={onSort} />
        <SortHead label={a.colServed} sortKey="coverageServed" sort={sort} onSort={onSort} />
        <SortHead label={a.colPending} sortKey="coveragePending" sort={sort} onSort={onSort} />
        <SortHead label={a.colUnserved} sortKey="coverageUnserved" sort={sort} onSort={onSort} />
        <SortHead label={a.colSettledCost} sortKey="settledCost" sort={sort} onSort={onSort} />
        <SortHead label={a.colPlannedCost} sortKey="plannedCost" sort={sort} onSort={onSort} />
        <SortHead label={a.colMaterialCost} sortKey="materialCost" sort={sort} onSort={onSort} />
        <SortHead label={a.colCostPerServed} sortKey="costPerServed" sort={sort} onSort={onSort} />
      </tr>
    </thead>
  )
}

/** 未登记渲染(null 语义,禁止显示 0)。 */
export function CostCell({ v }: { v: number | null }) {
  const unregistered = useT().pages.gridInvestmentPage.unregistered
  if (v === null) return <span className="text-[var(--shell-group-title)]">{unregistered}</span>
  return <>{formatCurrency(v)}</>
}