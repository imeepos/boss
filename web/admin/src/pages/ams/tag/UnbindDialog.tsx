// 解绑标签确认弹层(自 DangerOps 迁入接线,P2-T4 三要素):影响面清单 + 可恢复说明 +
// 原因必填(入 UNBIND 事件 detail)+ 红色确认键原因就绪前禁用。契约 POST /tags/{id}/unbind
// {expectedAssetId, reason};原因经 UnbindTag 落事件留痕。
import { useState } from 'react'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '../../../components/ui/dialog'
import { useT } from '../../../i18n'
import type { TagRow } from '../types'
import { canConfirmUnbind } from '../opsRules'

interface ImpactItem {
  label: string
  value: string
}

function ImpactList({ items }: { items: ImpactItem[] }) {
  return (
    <ul className="m-0 mb-3 list-none rounded-sm border border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] p-3 text-[13px] text-[var(--shell-content-text)]">
      {items.map((x) => (
        <li key={x.label} className="flex items-baseline justify-between gap-4 py-0.5">
          <span className="shrink-0 text-xs text-[var(--shell-group-title)]">{x.label}</span>
          <span className="text-right break-all">{x.value}</span>
        </li>
      ))}
    </ul>
  )
}

const fieldCls = "w-full rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 py-1.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]"

// UnbindDialog 解绑标签确认:影响面=标签状态/绑定资产/事件留痕;可重新绑定恢复。
export function UnbindDialog({ tag, busy, onClose, onConfirm }: { tag: TagRow; busy: boolean; onClose: () => void; onConfirm: (reason: string) => void }) {
  const t = useT()
  const g = t.pages.eventOps
  const [reason, setReason] = useState('')
  const ok = canConfirmUnbind(reason)
  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="max-w-md border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-0">
        <DialogHeader className="space-y-0 border-b border-[var(--shell-side-border)] px-5 py-4">
          <DialogTitle className="text-[15px] font-semibold text-[var(--shell-heading)]">{g.unbindTitle}</DialogTitle>
        </DialogHeader>
        <div className="px-5 py-4">
          <DialogDescription asChild>
            <ImpactList items={[
              { label: g.unbindImpactTag, value: tag.tagNo + ' · ' + tag.epcCode },
              { label: g.unbindImpactAsset, value: '#' + tag.boundAssetId },
              { label: g.unbindImpactStatus, value: g.unbindStatusTo },
              { label: g.unbindImpactEvent, value: g.unbindEventNote },
            ]} />
          </DialogDescription>
          <p className="mb-3 text-[13px] text-[var(--color-warning)]">{g.unbindRecover}</p>
          <label className="mb-3 block">
            <span className="mb-1 block text-xs text-[var(--shell-group-title)]">{g.unbindReason}</span>
            <textarea className={fieldCls} rows={2} value={reason} onChange={(ev) => setReason(ev.target.value)} />
            {!ok && reason.length > 0 && <span className="mt-1 block text-xs text-[var(--color-danger)]">{g.eReason}</span>}
          </label>
        </div>
        <DialogFooter className="gap-2 border-t border-[var(--shell-side-border)] px-5 py-3 sm:space-x-0">
          <button className="h-8 min-w-20 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onClose}>{t.pages.company.cancel}</button>
          <button className="h-8 min-w-20 rounded-sm border-none bg-[var(--color-danger)] px-4 text-[13px] text-white disabled:cursor-not-allowed disabled:opacity-45" disabled={!ok || busy} onClick={() => onConfirm(reason.trim())}>{g.confirm}</button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
