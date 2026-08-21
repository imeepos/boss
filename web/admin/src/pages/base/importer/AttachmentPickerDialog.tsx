// 附件选择器弹窗:AttachmentManager 选择模式圈选 1 个已上传附件,下载字节交给导入链路。
import { useState } from 'react'
import { AttachmentManager } from '../../../components/AttachmentManager'
import { ToolbarButton } from '../../../components/business/page-head'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '../../../components/ui/dialog'
import { fetchAttachmentFile } from '../../../api/attachments'
import { useT } from '../../../i18n'

export function AttachmentPickerDialog({ open, onClose, onPick }: {
  open: boolean
  onClose: () => void
  onPick: (file: File) => void
}) {
  const { pages: { importer: t }, common } = useT()
  const [selected, setSelected] = useState<number[]>([])
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')

  const pick = async () => {
    if (selected.length !== 1 || busy) return
    setBusy(true)
    setErr('')
    try {
      const file = await fetchAttachmentFile(selected[0])
      onPick(file)
      setSelected([])
      onClose()
    } catch (e) {
      setErr(e instanceof Error ? e.message : t.pickFetchFail)
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onClose() }}>
      <DialogContent className="max-w-[min(72rem,calc(100vw-32px))] max-h-[90vh] grid-rows-[auto_minmax(0,1fr)_auto] gap-0 overflow-hidden p-0">
        <DialogHeader className="px-6 pt-5 pb-3">
          <DialogTitle>{t.pickTitle}</DialogTitle>
        </DialogHeader>
        <div className="min-h-0 overflow-auto px-6">
          <AttachmentManager selectable selectedIds={selected} onSelectionChange={setSelected} />
        </div>
        <DialogFooter className="items-center gap-3 border-t border-[var(--shell-side-border)] px-6 py-3">
          {err && <span className="mr-auto text-xs text-[var(--color-danger)]">{err}</span>}
          <ToolbarButton onClick={() => void pick()} disabled={busy || selected.length !== 1}>
            {busy ? t.pickFetching : t.pickUse}
          </ToolbarButton>
          <ToolbarButton onClick={onClose}>{common.confirmDialog.cancel}</ToolbarButton>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
