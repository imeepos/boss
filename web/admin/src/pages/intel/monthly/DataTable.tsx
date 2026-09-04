// 月×区域事实表(三页签共用)。列由 TableMeta 驱动;派生列=模板灰色「勿填」列:
// 灰字 + 角标只读展示,不提供任何编辑入口。
import { useT } from '../../../i18n'
import { TableStateRow } from '../../../components/business'
import type { FieldMeta, MonthlyRow, TableMeta } from './types'

const TD = 'h-10 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]'
const TD_DERIVED = 'h-10 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-input-placeholder)] hover:bg-[var(--shell-menu-hover-bg)]'
const TH = 'h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]'
const EDIT_BTN = 'h-7 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-3 text-xs text-[var(--shell-content-text)] text-center hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]'

function cellText(row: MonthlyRow, f: FieldMeta): string {
  const v = row[f.key]
  if (v === undefined || v === null || v === '') return '—'
  if (typeof v === 'number') return v.toLocaleString('en-US')
  return String(v)
}

interface DataTableProps {
  meta: TableMeta
  rows: MonthlyRow[]
  loading: boolean
  empty: string
  editLabel?: string
  onEdit?: (row: MonthlyRow) => void
}

export function DataTable({ meta, rows, loading, empty, editLabel, onEdit }: DataTableProps) {
  const t = useT()
  const m = t.pages.monthlyPage
  const cols = m[meta.columnsKey]
  const colCount = cols.length + (editLabel ? 1 : 0)
  return (
    <div className="overflow-x-auto">
      <table className="w-full border-collapse text-[13px]">
        <thead>
          <tr>
            {cols.map((x) => <th key={x} className={TH}>{x}</th>)}
            {editLabel && <th className={TH}>{editLabel}</th>}
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.month + '/' + r.region}>
              {meta.fields.map((f) => (
                <td key={f.key} className={f.derived ? TD_DERIVED : TD}>
                  {cellText(r, f)}
                  {f.derived && (
                    <span className="ml-1.5 rounded-sm bg-[var(--shell-menu-hover-bg)] px-1 py-0.5 text-[11px] text-[var(--shell-group-title)]">
                      {m.derivedBadge}
                    </span>
                  )}
                </td>
              ))}
              {editLabel && onEdit && (
                <td className={TD}>
                  <button type="button" className={EDIT_BTN} onClick={() => onEdit(r)}>{editLabel}</button>
                </td>
              )}
            </tr>
          ))}
          {!rows.length && <TableStateRow colSpan={colCount} loading={loading} text={empty} />}
        </tbody>
      </table>
    </div>
  )
}
