// 网格视图:附件卡片墙(选中描边/骨架屏/空态),由父组件传入数据与回调。
import type { useT } from '../../i18n'
import type { AttachmentDTO, UploaderType } from '../../api/attachments'
import { fmtTime } from '../../lib/format'
import { classifyAttachment, fmtBytes } from './logic'
import { FileTypeIcon } from './FileTypeIcon'
import { DANGER_BTN } from './styles'

interface GridProps {
  loading: boolean
  empty: string
  rows: AttachmentDTO[]
  selectable: boolean
  selectedIds: number[]
  fixedUploader: boolean
  uploaderLabel: Record<UploaderType, string>
  t: ReturnType<typeof useT>['attachmentManager']
  onToggle: (id: number, checked: boolean) => void
  onDelete: (row: AttachmentDTO) => void
}

export function GridView({ loading, empty, rows, selectable, selectedIds, fixedUploader, uploaderLabel, t, onToggle, onDelete }: GridProps) {
  if (loading) {
    return <div className="grid grid-cols-2 gap-3 p-4 md:grid-cols-3 xl:grid-cols-4">
      {Array.from({ length: 8 }).map((_, i) => (
        <div key={i} className="h-28 animate-pulse rounded-md bg-[var(--shell-menu-hover-bg)]" />
      ))}
    </div>
  }
  if (!rows.length) {
    return <div className="py-12 text-center text-[var(--shell-group-title)]">{empty}</div>
  }
  return (
    <div className="grid grid-cols-2 gap-3 p-4 md:grid-cols-3 xl:grid-cols-4">
      {rows.map((row) => {
        const cat = classifyAttachment(row.contentType, row.fileName)
        const selected = selectedIds.includes(row.id)
        return (
          <div
            key={row.id}
            className={`group relative flex flex-col gap-2 rounded-md border bg-[var(--shell-card-bg)] p-3 transition-colors ${selected ? 'border-[var(--shell-fab-bg)] ring-1 ring-[var(--shell-fab-bg)]' : 'border-[var(--shell-card-border)] hover:border-[var(--shell-input-border-hover)]'}`}
          >
            {selectable && (
              <input
                type="checkbox" aria-label={row.fileName}
                className="absolute right-2 top-2 accent-[var(--shell-fab-bg)]"
                checked={selected}
                onChange={(e) => onToggle(row.id, e.target.checked)}
              />
            )}
            <div className="flex items-center gap-2.5">
              <FileTypeIcon category={cat} size={36} />
              <div className="min-w-0 flex-1">
                <div className="truncate text-[13px] font-medium text-[var(--shell-heading)]" title={row.fileName}>{row.fileName || '—'}</div>
                <div className="mt-0.5 flex items-center gap-2 text-[11px] text-[var(--shell-content-text)]">
                  <span className="tabular-nums">{fmtBytes(row.sizeBytes)}</span>
                  <span>·</span>
                  <span className="truncate">{fmtTime(row.createdAt)}</span>
                </div>
              </div>
            </div>
            {!fixedUploader && (
              <div className="text-[11px] text-[var(--shell-content-text)]">
                {uploaderLabel[row.uploaderType] ?? row.uploaderType} #{row.uploaderId}
              </div>
            )}
            <div className="flex items-center justify-end gap-1 border-t border-[var(--shell-side-border)] pt-2">
              <button type="button" className={DANGER_BTN} title={t.download}>{t.download}</button>
              <span className="text-[var(--shell-side-border)]">|</span>
              <button type="button" className={DANGER_BTN} onClick={() => onDelete(row)}>{t.delete}</button>
            </div>
          </div>
        )
      })}
    </div>
  )
}
