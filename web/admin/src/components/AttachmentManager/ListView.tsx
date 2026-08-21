// 列表视图:附件表格(loading/空态/数据行/批量选择列),由父组件传入数据与回调。
import type { useT } from '../../i18n'
import type { AttachmentDTO, UploaderType } from '../../api/attachments'
import { fmtTime } from '../../lib/format'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../ui/table'
import { classifyAttachment, fmtBytes } from './logic'
import { FileTypeIcon } from './FileTypeIcon'
import { DANGER_BTN } from './styles'

interface ListProps {
  loading: boolean
  rows: AttachmentDTO[]
  selectable: boolean
  selectedIds: number[]
  fixedUploader: boolean
  colSpan: number
  allChecked: boolean
  uploaderLabel: Record<UploaderType, string>
  t: ReturnType<typeof useT>['attachmentManager']
  commonLoading: string
  onToggle: (id: number, checked: boolean) => void
  onToggleAll: (checked: boolean) => void
  onDelete: (row: AttachmentDTO) => void
}

export function ListView({
  loading, rows, selectable, selectedIds, fixedUploader, colSpan, allChecked,
  uploaderLabel, t, commonLoading, onToggle, onToggleAll, onDelete,
}: ListProps) {
  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            {selectable && (
              <TableHead className="w-9">
                <input type="checkbox" aria-label={t.title} className="accent-[var(--shell-fab-bg)]" checked={allChecked} onChange={(e) => onToggleAll(e.target.checked)} />
              </TableHead>
            )}
            <TableHead>{t.colFileName}</TableHead>
            <TableHead>{t.colType}</TableHead>
            <TableHead>{t.colSize}</TableHead>
            {!fixedUploader && <TableHead>{t.colUploader}</TableHead>}
            <TableHead>{t.colTime}</TableHead>
            <TableHead className="w-28">{t.colActions}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {loading && (
            <TableRow><TableCell colSpan={colSpan} className="py-8 text-center text-[var(--shell-group-title)]">{commonLoading}</TableCell></TableRow>
          )}
          {!loading && rows.length === 0 && (
            <TableRow><TableCell colSpan={colSpan} className="py-8 text-center text-[var(--shell-group-title)]">{t.empty}</TableCell></TableRow>
          )}
          {!loading && rows.map((row) => {
            const cat = classifyAttachment(row.contentType, row.fileName)
            return (
              <TableRow key={row.id}>
                {selectable && (
                  <TableCell>
                    <input
                      type="checkbox" aria-label={row.fileName}
                      className="accent-[var(--shell-fab-bg)]"
                      checked={selectedIds.includes(row.id)}
                      onChange={(e) => onToggle(row.id, e.target.checked)}
                    />
                  </TableCell>
                )}
                <TableCell className="max-w-72 text-[var(--shell-heading)]">
                  <div className="flex items-center gap-2.5">
                    <FileTypeIcon category={cat} size={28} />
                    <span className="truncate" title={row.fileName}>{row.fileName || '—'}</span>
                  </div>
                </TableCell>
                <TableCell className="text-[var(--shell-content-text)]">{t.fileCategory[cat]}</TableCell>
                <TableCell className="text-[var(--shell-content-text)] tabular-nums">{fmtBytes(row.sizeBytes)}</TableCell>
                {!fixedUploader && (
                  <TableCell className="text-[var(--shell-content-text)]">
                    {uploaderLabel[row.uploaderType] ?? row.uploaderType} #{row.uploaderId}
                  </TableCell>
                )}
                <TableCell className="text-[var(--shell-content-text)]">{fmtTime(row.createdAt)}</TableCell>
                <TableCell>
                  <div className="inline-flex items-center gap-1">
                    <button type="button" className={DANGER_BTN} title={t.download}>{t.download}</button>
                    <span className="text-[var(--shell-side-border)]">|</span>
                    <button type="button" className={DANGER_BTN} onClick={() => onDelete(row)}>{t.delete}</button>
                  </div>
                </TableCell>
              </TableRow>
            )
          })}
        </TableBody>
      </Table>
    </div>
  )
}
