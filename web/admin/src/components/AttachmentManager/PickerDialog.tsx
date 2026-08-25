// 通用附件选择器弹窗:AttachmentManager 选择模式圈选已上传附件(上传也在其内完成)。
// multiple 支持多选(正文插图),imageOnly 客户端过滤 image/*(封面/插图);
// 返回 AttachmentDTO(id/fileName/contentType),调用方自行决定引用形态(如 att/N)。
import { useMemo, useState } from 'react'
import { AttachmentManager } from './index'
import { ToolbarButton } from '../business/page-head'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '../ui/dialog'
import { useT } from '../../i18n'
import type { AttachmentDTO } from '../../api/attachments'

export function AttachmentPickerDialog({ open, onClose, onPick, multiple = false, imageOnly = false }: {
  open: boolean
  onClose: () => void
  onPick: (items: AttachmentDTO[]) => void
  multiple?: boolean
  imageOnly?: boolean
}) {
  const { attachmentManager: t, common } = useT()
  const [selected, setSelected] = useState<number[]>([])
  const [all, setAll] = useState<AttachmentDTO[]>([])

  // imageOnly 时裁掉非图片项,并同步修正已选集(选中后翻页再筛的边界)。
  const items = useMemo(() => (imageOnly ? all.filter((x) => x.contentType.startsWith('image/')) : all), [all, imageOnly])
  const byId = useMemo(() => new Map(all.map((x) => [x.id, x])), [all])
  const picked = selected.map((id) => byId.get(id)).filter((x): x is AttachmentDTO => !!x && (!imageOnly || x.contentType.startsWith('image/')))
  const ok = multiple ? picked.length > 0 : picked.length === 1

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onClose() }}>
      <DialogContent className="max-w-[min(72rem,calc(100vw-32px))] max-h-[90vh] grid-rows-[auto_minmax(0,1fr)_auto] gap-0 overflow-hidden p-0">
        <DialogHeader className="px-6 pt-5 pb-3">
          <DialogTitle>{multiple ? t.pickerTitleMulti : t.pickerTitle}</DialogTitle>
        </DialogHeader>
        <div className="min-h-0 overflow-auto px-6">
          {/* 全量快照仅在选择器打开期间用于回填 DTO;数量以服务端分页为准 */}
          <AttachmentManager selectable selectedIds={selected} onSelectionChange={(ids) => setSelected(multiple ? ids : ids.slice(-1))} onItemsLoaded={setAll} />
          {imageOnly && items.length === 0 && <div className="py-3 text-xs text-[var(--shell-group-title)]">{t.pickerImageOnlyHint}</div>}
        </div>
        <DialogFooter className="items-center gap-3 border-t border-[var(--shell-side-border)] px-6 py-3">
          <ToolbarButton
            onClick={() => { onPick(picked); setSelected([]); onClose() }}
            disabled={!ok}
          >
            {t.pickerUse}
          </ToolbarButton>
          <ToolbarButton onClick={onClose}>{common.confirmDialog.cancel}</ToolbarButton>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
