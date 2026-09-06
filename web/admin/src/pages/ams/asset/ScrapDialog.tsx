// 报废弹窗(P3-F 三要素确认):展示资产编码/SN/标签号参考值(select-none 不可复制,核对实物铭牌),
// 三个确认输入按「有无 SN/有无绑标签」动态必填;reason 保留;提交前本地预校验,
// 服务端同规则强校验兜底(不符 422,信息只指明要素不回显现值)。
import { useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '../../../components/ui/dialog'
import { useT } from '../../../i18n'
import type { AssetRow, TagRow } from '../types'
import { scrapConfirmErr, scrapReasonErr, type ScrapConfirmField } from './logic'

const input = 'h-8 w-full rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'
const refKey = 'text-[12px] text-[var(--shell-content-text)] shrink-0'
const refVal = 'font-mono text-[13px] text-[var(--shell-heading)] break-all text-right'
const refRow = 'flex items-center justify-between gap-3 border-b border-dashed border-[var(--shell-side-border)] py-1 text-[13px]'
const errText = 'text-[11px] text-[var(--color-danger)]'
const apiErrBox = 'rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]'

type ATexts = ReturnType<typeof useT>['pages']['assetPage']
export type ErrField = ScrapConfirmField | 'reason'

// ScrapRefBlock 三要素参考值(select-none 防直接复制,须对照实物铭牌人工誊录)。
export function ScrapRefBlock(p: { title: string; codeLabel: string; code: string; snLabel: string; sn: string; tagLabel: string; tagNo: string }) {
  return (
    <div className='rounded-sm border border-[var(--shell-side-border)] px-3 py-1.5 select-none'>
      <div className='pb-0.5 text-[11px] text-[var(--shell-group-title)]'>{p.title}</div>
      <div className={refRow}><span className={refKey}>{p.codeLabel}</span><span className={refVal}>{p.code}</span></div>
      {p.sn && <div className={refRow}><span className={refKey}>{p.snLabel}</span><span className={refVal}>{p.sn}</span></div>}
      {p.tagNo && <div className={refRow}><span className={refKey}>{p.tagLabel}</span><span className={refVal}>{p.tagNo}</span></div>}
    </div>
  )
}

// ConfirmField 确认输入行:必填星号+输入框+预校验错误文案;输入即清全局错误态。
function ConfirmField(p: { label: string; ph: string; value: string; on: (v: string) => void; err: boolean; errText: string; clear: () => void }) {
  return (
    <div className='flex flex-col gap-0.5'>
      <label className='text-[13px] text-[var(--shell-content-text)]'><span className='mr-0.5 text-[var(--color-danger)]'>*</span>{p.label}</label>
      <input className={input} value={p.value} maxLength={64} placeholder={p.ph}
        onChange={(e) => { p.on(e.target.value); p.clear() }} />
      {p.err && <span className={errText}>{p.errText}</span>}
    </div>
  )
}

// ScrapFields reason + 三要素确认输入(无 SN/未绑标签的确认框按动态规则不渲染)。
// 导出供组件测试直测:Radix Dialog 在 renderToStaticMarkup 下渲染为空。
export function ScrapFields(p: { a: ATexts; hasSn: boolean; hasTag: boolean; errField: ErrField; clear: () => void;
  reason: string; onReason: (v: string) => void; code: string; onCode: (v: string) => void;
  sn: string; onSn: (v: string) => void; tagNo: string; onTagNo: (v: string) => void }) {
  return (
    <>
      <div className='flex flex-col gap-0.5'>
        <label className='text-[13px] text-[var(--shell-content-text)]'><span className='mr-0.5 text-[var(--color-danger)]'>*</span>{p.a.fReason}</label>
        <input className={input} value={p.reason} maxLength={64} placeholder={p.a.pReason}
          onChange={(e) => { p.onReason(e.target.value); p.clear() }} />
        {p.errField === 'reason' && <span className={errText}>{p.a.eReason}</span>}
      </div>
      <ConfirmField label={p.a.dCode} ph={p.a.pConfirmCode} value={p.code} on={p.onCode} err={p.errField === 'code'} errText={p.a.eConfirmCode} clear={p.clear} />
      {p.hasSn && <ConfirmField label={p.a.fSn} ph={p.a.pConfirmSn} value={p.sn} on={p.onSn} err={p.errField === 'sn'} errText={p.a.eConfirmSn} clear={p.clear} />}
      {p.hasTag && <ConfirmField label={p.a.dTagNo} ph={p.a.pConfirmTag} value={p.tagNo} on={p.onTagNo} err={p.errField === 'tagNo'} errText={p.a.eConfirmTag} clear={p.clear} />}
    </>
  )
}

// ScrapDialog 报废确认弹窗:本地预校验不过不发起请求;载荷恒带三要素(无则空串)。
export function ScrapDialog({ asset, tag, onClose, onSaved }: {
  asset: AssetRow
  tag?: TagRow
  onClose: () => void
  onSaved: () => void
}) {
  const t = useT()
  const a = t.pages.assetPage
  const hasSn = Boolean(asset.sn)
  const hasTag = Boolean(tag)
  const [reason, setReason] = useState('')
  const [code, setCode] = useState('')
  const [sn, setSn] = useState('')
  const [tagNo, setTagNo] = useState('')
  const [errField, setErrField] = useState<ErrField>('')
  const [apiError, setApiError] = useState('')
  const [busy, setBusy] = useState(false)

  const submit = async () => {
    if (busy) return
    const bad = scrapReasonErr(reason) ? 'reason' : scrapConfirmErr({ code, sn, tagNo }, hasSn, hasTag)
    setErrField(bad)
    if (bad) return
    setBusy(true)
    setApiError('')
    try {
      await apiFetch('/assets/' + String(asset.assetId) + '/scrap', {
        method: 'POST',
        body: { reason: reason.trim(), confirmAssetCode: code.trim(), confirmSn: sn.trim(), confirmTagNo: tagNo.trim() },
      })
      toast.success(a.scrapOk)
      onSaved()
      onClose()
    } catch (err) {
      setApiError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className='max-w-sm border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-0'>
        <DialogHeader className='space-y-0 border-b border-[var(--shell-side-border)] px-5 py-4'>
          <DialogTitle className='text-[15px] font-semibold text-[var(--shell-heading)]'>{a.scrapTitle}</DialogTitle>
        </DialogHeader>
        <div className='flex flex-col gap-2 px-5 py-4'>
          <ScrapRefBlock title={a.scrapRefTitle} codeLabel={a.dCode} code={asset.assetCode}
            snLabel={a.fSn} sn={asset.sn ?? ''} tagLabel={a.dTagNo} tagNo={tag?.tagNo ?? ''} />
          <ScrapFields a={a} hasSn={hasSn} hasTag={hasTag} errField={errField} clear={() => setErrField('')}
            reason={reason} onReason={setReason} code={code} onCode={setCode}
            sn={sn} onSn={setSn} tagNo={tagNo} onTagNo={setTagNo} />
          {apiError && <div className={apiErrBox}>{apiError}</div>}
        </div>
        <DialogFooter className='gap-2 border-t border-[var(--shell-side-border)] px-5 py-3 sm:space-x-0'>
          <button className='h-8 min-w-20 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]' onClick={onClose}>{t.pages.company.cancel}</button>
          <button className='h-8 min-w-20 cursor-pointer rounded-sm border-none bg-[var(--color-danger)] px-4 text-[13px] text-white hover:opacity-90' disabled={busy} onClick={submit}>
            {busy ? t.pages.account.submitting : a.scrapAction}
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
