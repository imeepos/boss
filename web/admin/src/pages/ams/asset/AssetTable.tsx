// 资产台账表格:列名以 fields.md §4.1 为准;操作列五动作,SCRAPPED 行隐藏报废/删除入口。
import { StatusTag } from '../../../components/StatusTag'
import { TableStateRow } from '../../../components/business'
import { useT } from '../../../i18n'
import type { AssetRow, TagRow } from '../types'

const td = 'h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]'
const th = 'h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]'

interface AssetTableProps {
  rows: AssetRow[]
  tagOf: (tagId: number) => TagRow | undefined
  busy: boolean
  onTrail: (r: AssetRow) => void
  onDetail: (r: AssetRow) => void
  onEdit: (r: AssetRow) => void
  onScrap: (r: AssetRow) => void
  onDelete: (r: AssetRow) => void
  /** P2-T4:打开事件时间轴抽屉(标签事件流,按 asset_id 查)。 */
  onEvents: (r: AssetRow) => void
}

export function AssetTable(p: AssetTableProps) {
  const t = useT()
  const a = t.pages.assetPage
  return (
    <div className="overflow-x-auto px-4 pb-4">
      <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
        <thead><tr>{a.columns.map((x) => <th key={x} className={th}>{x}</th>)}</tr></thead>
        <tbody>
          {p.rows.map((r) => (
            <tr key={r.assetId}>
              <td className={td}>{r.assetCode}</td>
              <td className={td}>{p.tagOf(r.tagId)?.tagNo ?? '—'}</td>
              <td className={td}>{p.tagOf(r.tagId)?.epcCode ?? '—'}</td>
              <td className={td}>{r.type || '—'}</td>
              <td className={td}>#{r.batchId}</td>
              <td className={td}>{r.addressId ? '#' + String(r.addressId) : '—'}</td>
              <td className={td}><StatusTag domain="asset" value={r.status} /></td>
              <td className={td}>
                <span className="inline-flex items-center gap-3">
                  <button disabled={p.busy} onClick={() => p.onTrail(r)}>{a.lifecycle}</button>
                  <button disabled={p.busy} onClick={() => p.onDetail(r)}>{a.detail}</button>
                  <button disabled={p.busy} onClick={() => p.onEdit(r)}>{a.edit}</button>
                  {r.status !== 'SCRAPPED' && <button disabled={p.busy} onClick={() => p.onScrap(r)}>{a.scrapAction}</button>}
                  {r.status !== 'SCRAPPED' && <button disabled={p.busy} onClick={() => p.onDelete(r)}>{a.deleteAction}</button>}
                  <button disabled={p.busy} onClick={() => p.onEvents(r)}>{t.pages.eventOps.actEvents}</button>
                </span>
              </td>
            </tr>
          ))}
          {!p.rows.length && <TableStateRow colSpan={8} loading={p.busy} text={a.empty} />}
        </tbody>
      </table>
    </div>
  )
}