// 危险操作二次确认弹层(P2-T4):确认三要素 = 影响面清单 + 不可逆/恢复路径说明 +
// 红色确认键默认禁用(原因必填;报废另需输入资产编码精确匹配)。影响面口径与事后
// 可查回的事件字段对齐:解绑写 UNBIND、报废写 RECYCLE+SCRAPPED 轨迹,原因均入 detail。
import { useState, type ReactNode } from 'react'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '../../components/ui/dialog'
import { useT } from '../../i18n'
import type { TagRow } from './types'
import { canConfirmScrap, canConfirmUnbind } from './opsRules'

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

function Field({ label, error, children }: { label: string; error?: string; children: ReactNode }) {
  return (
    <label className="mb-3 block">
      <span className="mb-1 block text-xs text-[var(--shell-group-title)]">{label}</span>
      {children}
      {error ? <span className="mt-1 block text-xs text-[var(--color-danger)]">{error}</span> : null}
    </label>
  )
}

// UnbindConfirmDialog 解绑标签确认:影响面=标签状态/绑定资产/事件留痕;可重新绑定恢复。
export function UnbindConfirmDialog({ tag, busy, onClose, onConfirm }: { tag: TagRow; busy: boolean; onClose: () => void; onConfirm: (reason: string) => void }) {
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
          <Field label={g.unbindReason} error={reason.length > 0 && !ok ? g.eReason : undefined}>
            <textarea className={fieldCls} rows={2} value={reason} onChange={(ev) => setReason(ev.target.value)} />
          </Field>
        </div>
        <DialogFooter className="gap-2 border-t border-[var(--shell-side-border)] px-5 py-3 sm:space-x-0">
          <button className="h-8 min-w-20 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onClose}>{t.pages.company.cancel}</button>
          <button className="h-8 min-w-20 rounded-sm border-none px-4 text-[13px] text-white"
            style={{ background: 'var(--color-danger)', opacity: ok && !busy ? 1 : 0.45, cursor: ok && !busy ? 'pointer' : 'not-allowed' }}
            disabled={!ok || busy} onClick={() => onConfirm(reason.trim())}>{g.confirm}</button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ScrapConfirmDialog 报废资产确认:影响面=资产终态/标签强回收/轨迹留痕;报废不可逆,
// 需输入资产编码 + 原因,红色确认键在两者就绪前保持禁用。
export function ScrapConfirmDialog({ assetCode, assetId, tag, busy, onClose, onConfirm }: { assetCode: string; assetId: number; tag?: TagRow; busy: boolean; onClose: () => void; onConfirm: (reason: string) => void }) {
  const t = useT()
  const g = t.pages.eventOps
  const [reason, setReason] = useState('')
  const [code, setCode] = useState('')
  const ok = canConfirmScrap(reason, code, assetCode)
  const codeMismatch = code.trim() !== '' && code.trim() !== assetCode
  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="max-w-md border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-0">
        <DialogHeader className="space-y-0 border-b border-[var(--shell-side-border)] px-5 py-4">
          <DialogTitle className="text-[15px] font-semibold text-[var(--shell-heading)]">{g.scrapTitle}</DialogTitle>
        </DialogHeader>
        <div className="px-5 py-4">
          <DialogDescription asChild>
            <ImpactList items={[
              { label: g.scrapImpactAsset, value: assetCode + ' (#' + assetId + ')' },
              { label: g.scrapImpactStatus, value: g.scrapStatusTo },
              { label: g.scrapImpactTag, value: tag ? tag.tagNo : '-' },
              { label: g.scrapImpactEvent, value: g.scrapTagNote },
              { label: g.scrapImpactTrail, value: g.scrapTrailNote },
            ]} />
          </DialogDescription>
          <p className="mb-3 text-[13px] font-medium text-[var(--color-danger)]">{g.scrapIrreversible}</p>
          <Field label={g.scrapReason}>
            <textarea className={fieldCls} rows={2} value={reason} onChange={(ev) => setReason(ev.target.value)} />
          </Field>
          <Field label={g.scrapCode} error={codeMismatch ? g.eCode : undefined}>
            <input className={fieldCls} value={code} onChange={(ev) => setCode(ev.target.value)} placeholder={assetCode} />
          </Field>
        </div>
        <DialogFooter className="gap-2 border-t border-[var(--shell-side-border)] px-5 py-3 sm:space-x-0">
          <button className="h-8 min-w-20 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onClose}>{t.pages.company.cancel}</button>
          <button className="h-8 min-w-20 rounded-sm border-none px-4 text-[13px] text-white"
            style={{ background: 'var(--color-danger)', opacity: ok && !busy ? 1 : 0.45, cursor: ok && !busy ? 'pointer' : 'not-allowed' }}
            disabled={!ok || busy} onClick={() => onConfirm(reason.trim())}>{g.confirm}</button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
