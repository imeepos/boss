// 实名代录抽屉:客户无法自助完成时的后台补录通道。
// 回显 GET /customers/:id/real-name(40410=暂无核验单);提交 POST 同端点落 PENDING,
// 二要素通道启用时即时自动判定(fields.md §7.6);证件照预览复用实名审核中心的 AttachmentPreview。
import { useEffect, useState, type ReactNode } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { ApiError } from '../../../api/envelope'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { useT } from '../../../i18n'
import { fmtTime } from '../../../lib/format'
import { AttachmentPreview } from '../../base/realname-review/AttachmentPreview'

interface LatestRealName {
  id: number
  method: string
  realName: string
  idCardNo: string
  result: string
  rejectReason: string
  idCardFrontId: number
  idCardBackId: number
  verifiedAt: string
}

const CODE_NOT_FOUND = 40410

export function RealNameDrawer({
  customerId, customerName, onClose, onSubmitted,
}: { customerId: number; customerName: string; onClose: () => void; onSubmitted: () => void }) {
  const t = useT()
  const c = t.pages.customer
  const [latest, setLatest] = useState<LatestRealName | null>(null)
  const [loaded, setLoaded] = useState(false)
  const [method, setMethod] = useState('')
  const [realName, setRealName] = useState(customerName)
  const [idCardNo, setIdCardNo] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [previewId, setPreviewId] = useState<number | null>(null)
  const [verifying, setVerifying] = useState(false)
  const [reason, setReason] = useState('')

  useEffect(() => {
    apiFetch<LatestRealName>(`/customers/${customerId}/real-name`)
      .then((d) => setLatest(d))
      .catch((e) => { if (!(e instanceof ApiError && e.code === CODE_NOT_FOUND)) setError(c.rnLoadFail) })
      .finally(() => setLoaded(true))
  }, [customerId]) // eslint-disable-line react-hooks/exhaustive-deps

  const submit = async () => {
    if (busy || !realName.trim() || !idCardNo.trim() || !method) return
    setBusy(true); setError(''); setNotice('')
    try {
      const d = await apiFetch<{ result: string }>(`/customers/${customerId}/real-name`, {
        method: 'POST',
        body: { realName: realName.trim(), idCardNo: idCardNo.trim(), method },
      })
      const text = c.rnSubmitted.replace('{result}', c.verifyResultLabels[d?.result ?? ''] ?? d?.result ?? '')
      setNotice(text)
      toast.success(text)
      onSubmitted()
      apiFetch<LatestRealName>(`/customers/${customerId}/real-name`).then((v) => setLatest(v))
    } catch (e) {
      setError(e instanceof Error ? e.message : c.rnFail)
    } finally { setBusy(false) }
  }

  // 后台核验(latest PENDING 时):PASS 直改;FAIL 必填理由。核验后刷新 latest + 外层列表。
  const verify = async (result: 'PASS' | 'FAIL') => {
    if (busy) return
    if (result === 'FAIL' && !reason.trim()) { setError(c.rnReasonRequired); return }
    setBusy(true); setError(''); setNotice('')
    try {
      await apiFetch(`/customers/${customerId}/real-name/verify`, {
        method: 'POST', body: { result, reason: reason.trim() || undefined },
      })
      const text = c.verifyResultLabels[result] ?? result
      setNotice(text)
      toast.success(text)
      setVerifying(false); setReason('')
      onSubmitted()
      apiFetch<LatestRealName>(`/customers/${customerId}/real-name`).then((v) => setLatest(v))
    } catch (e) {
      setError(e instanceof Error ? e.message : c.rnFail)
    } finally { setBusy(false) }
  }

  return (
    <Drawer title={c.rnTitle.replace('{name}', customerName)} onClose={onClose}
      footer={
        <>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)]" onClick={onClose}>{t.common.confirmDialog.cancel}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)] disabled:opacity-50"
            disabled={busy || !realName.trim() || !idCardNo.trim() || !method} onClick={submit}>{c.rnSubmit}</button>
        </>
      }>
      {error && (
        <div className="mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div>
      )}
      {notice && (
        <div className="mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-success)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-success)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-success)]">{notice}</div>
      )}

      <div className="mb-1 text-[13px] font-medium text-[var(--shell-group-title)]">{c.rnLatestHead}</div>
      {!loaded ? <p className="text-[13px] text-[var(--shell-group-title)]">{t.common.loading}</p>
        : latest == null ? <p className="text-[13px] text-[var(--shell-group-title)]">{c.rnLatestNone}</p>
          : <LatestCard row={latest} c={c} onPreview={setPreviewId} />}
      {loaded && latest?.result === 'PENDING' && (
        <div className="mt-3 rounded-md border border-[var(--shell-side-border)] p-3">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-[13px] font-medium text-[var(--shell-heading)]">{c.rnVerifyHead}</span>
            <span className="spacer" />
            <button type="button" className="h-8 cursor-pointer rounded-sm border-none bg-[var(--color-success)] px-4 text-[13px] text-white disabled:opacity-50" disabled={busy}
              onClick={() => verify('PASS')}>{c.rnVerifyPass}</button>
            <button type="button" className="h-8 cursor-pointer rounded-sm border border-[var(--color-danger)] bg-transparent px-4 text-[13px] text-[var(--color-danger)] hover:bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)]" onClick={() => setVerifying((v) => !v)}>{c.rnVerifyReject}</button>
          </div>
          {verifying && (
            <div className="mt-2">
              <textarea className="mb-2 h-16 w-full rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] p-2 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={reason} placeholder={c.rnReasonPh} onChange={(e) => setReason(e.target.value)} />
              <button type="button" className="h-8 cursor-pointer rounded-sm border-none bg-[var(--color-danger)] px-4 text-[13px] text-white disabled:opacity-50" disabled={busy} onClick={() => verify('FAIL')}>{c.rnVerifyReject}</button>
            </div>
          )}
        </div>
      )}

      <div className="mt-5 mb-2 text-[13px] font-medium text-[var(--shell-group-title)]">{c.rnFormHead}</div>
      <div className="flex flex-col gap-3">
        <Field label={c.rnMethod}>
          <Dropdown value={method} ariaLabel={c.rnMethod}
            options={[{ value: '', label: c.rnMethodAll }, ...c.rnMethods.map((m) => ({ value: m, label: m }))]}
            onChange={(v) => setMethod(v)} />
        </Field>
        <Field label={c.rnRealName}>
          <input className="h-9 w-full rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-3 text-[13px] text-[var(--shell-content-text)] outline-none focus:border-[var(--color-border-focus)]"
            value={realName} placeholder={c.rnRealNamePh} onChange={(e) => setRealName(e.target.value)} />
        </Field>
        <Field label={c.rnIdCardNo}>
          <input className="h-9 w-full rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-3 text-[13px] text-[var(--shell-content-text)] outline-none focus:border-[var(--color-border-focus)]"
            value={idCardNo} placeholder={c.rnIdCardNoPh} onChange={(e) => setIdCardNo(e.target.value)} />
        </Field>
      </div>

      {previewId != null && <AttachmentPreview attachmentId={previewId} onClose={() => setPreviewId(null)} />}
    </Drawer>
  )
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="flex items-center gap-3">
      <span className="w-20 shrink-0 text-right text-[13px] text-[var(--shell-group-title)]">{label}</span>
      {children}
    </label>
  )
}

function LatestCard({ row, c, onPreview }: {
  row: LatestRealName
  c: ReturnType<typeof useT>['pages']['customer']
  onPreview: (id: number) => void
}) {
  const color = row.result === 'PASS' ? 'var(--color-success)' : row.result === 'FAIL' ? 'var(--color-danger)' : 'var(--color-brand-gold-600)'
  const kv: Array<[string, string]> = [
    ['ID', String(row.id)],
    [c.verifyColumns[0], row.method],
    [c.rnRealName, row.realName],
    [c.rnIdCardNo, row.idCardNo],
    [c.verifyColumns[1], fmtTime(row.verifiedAt)],
  ]
  return (
    <div className="rounded-md border border-[var(--shell-side-border)] p-3 text-[13px] text-[var(--shell-content-text)]">
      {kv.map(([k, v]) => (
        <div key={k} className="flex gap-2 py-0.5">
          <span className="w-24 shrink-0 text-[var(--shell-group-title)]">{k}</span>
          <span className="break-all">{v || '—'}</span>
        </div>
      ))}
      <div className="flex items-center gap-2 py-0.5">
        <span className="w-24 shrink-0 text-[var(--shell-group-title)]">{c.verifyColumns[2]}</span>
        <span className="inline-flex h-6 items-center rounded-full px-2 text-[11px] font-medium"
          style={{ color, background: `color-mix(in srgb, ${color} 15%, transparent)` }}>
          {c.verifyResultLabels[row.result] ?? row.result}
        </span>
      </div>
      {row.result === 'FAIL' && row.rejectReason && (
        <div className="flex gap-2 py-0.5">
          <span className="w-24 shrink-0 text-[var(--shell-group-title)]">{c.rnRejectReason}</span>
          <span>{row.rejectReason}</span>
        </div>
      )}
      {(row.idCardFrontId > 0 || row.idCardBackId > 0) && (
        <div className="flex gap-2 py-0.5">
          <span className="w-24 shrink-0 text-[var(--shell-group-title)]">{c.rnAttachments}</span>
          <span className="flex gap-3">
            {row.idCardFrontId > 0 && <button className="cursor-pointer text-[var(--color-text-link)] hover:underline" onClick={() => onPreview(row.idCardFrontId)}>{c.rnFront}</button>}
            {row.idCardBackId > 0 && <button className="cursor-pointer text-[var(--color-text-link)] hover:underline" onClick={() => onPreview(row.idCardBackId)}>{c.rnBack}</button>}
          </span>
        </div>
      )}
    </div>
  )
}
