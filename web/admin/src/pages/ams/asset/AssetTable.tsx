// 资产台账表格:列名以 fields.md §4.1 为准;操作列五动作,SCRAPPED 行隐藏报废/删除入口。
import { StatusTag } from '../../../components/StatusTag'
import { ActionLink, ActionLinks, ActionSep, TableStateRow } from '../../../components/business'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { useT } from '../../../i18n'
import type { AssetRow, TagRow } from '../types'
import { batchLabel } from './logic'

interface AssetTableProps {
  rows: AssetRow[]
  tagOf: (tagId: number) => TagRow | undefined
  batchOf: (batchId: number) => { code: string; name: string } | undefined
  busy: boolean
  onTrail: (r: AssetRow) => void
  onDetail: (r: AssetRow) => void
  onEdit: (r: AssetRow) => void
  onScrap: (r: AssetRow) => void
  onDelete: (r: AssetRow) => void
}

export function AssetTable(p: AssetTableProps) {
  const t = useT()
  const a = t.pages.assetPage
  return (
    <div className="px-4 pb-4">
      <Table>
        <TableHeader>
          <TableRow>
            {a.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}
          </TableRow>
        </TableHeader>
        <TableBody>
          {p.rows.map((r) => (
            <TableRow key={r.assetId}>
              <TableCell>{r.assetCode}</TableCell>
              <TableCell>{p.tagOf(r.tagId)?.tagNo ?? '—'}</TableCell>
              <TableCell>{p.tagOf(r.tagId)?.epcCode ?? '—'}</TableCell>
              <TableCell>{r.type || '—'}</TableCell>
              <TableCell>{batchLabel(p.batchOf(r.batchId), r.batchId)}</TableCell>
              <TableCell>{r.addressId ? '#' + String(r.addressId) : '—'}</TableCell>
              <TableCell><StatusTag domain="asset" value={r.status} /></TableCell>
              <TableCell>
                <ActionLinks>
                  <ActionLink onClick={() => p.onTrail(r)} label={a.lifecycle} testId={`asset-trail-${r.assetId}`} />
                  <ActionSep />
                  <ActionLink onClick={() => p.onDetail(r)} label={a.detail} testId={`asset-detail-${r.assetId}`} />
                  <ActionSep />
                  <ActionLink onClick={() => p.onEdit(r)} label={a.edit} testId={`asset-edit-${r.assetId}`} />
                  {r.status !== 'SCRAPPED' && (
                    <>
                      <ActionSep />
                      <ActionLink onClick={() => p.onScrap(r)} label={a.scrapAction} testId={`asset-scrap-${r.assetId}`} />
                    </>
                  )}
                  {r.status !== 'SCRAPPED' && (
                    <>
                      <ActionSep />
                      <ActionLink onClick={() => p.onDelete(r)} label={a.deleteAction} testId={`asset-del-${r.assetId}`} />
                    </>
                  )}
                </ActionLinks>
              </TableCell>
            </TableRow>
          ))}
          {!p.rows.length && <TableStateRow colSpan={8} loading={p.busy} text={a.empty} />}
        </TableBody>
      </Table>
    </div>
  )
}