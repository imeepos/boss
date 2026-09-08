// 产品↔下发模板绑定抽屉:选该产品所属公司的启用模板,保存 PUT /products/{id}/provision-binding。
// 绑定后订单环节7 按下发模板开通;无带宽套餐(IPTV/增值包)必须绑定,否则环节7 显性失败。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { FormField } from '../../../components/business/form-field'
import { ErrorBanner } from '../../../components/business/page-head'
import { useConfirm } from '../../../components/ConfirmDialog'
import { useT } from '../../../i18n'
import type { ProvisionTemplateRow } from '../../provision/types'
import type { OfferBindingRow } from './types'

export function BindTemplateDrawer({ offerId, offerName, legalEntityId, onClose, onDone }: {
  offerId: number
  offerName: string
  legalEntityId: number
  onClose: () => void
  onDone: () => void
}) {
  const t = useT()
  const p = t.pages.product
  const confirmDialog = useConfirm()
  const [templates, setTemplates] = useState<ProvisionTemplateRow[]>([])
  const [bound, setBound] = useState(false)
  const [templateId, setTemplateId] = useState(0)
  const [remark, setRemark] = useState('')
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    apiFetch<{ items: ProvisionTemplateRow[] }>('/provision-templates')
      .then((d) => setTemplates(d?.items ?? []))
      .catch((e) => {
        setTemplates([])
        console.warn('[product] provision-templates 拉取失败:', e instanceof Error ? e.message : e)
      })
    apiFetch<OfferBindingRow>(`/products/${offerId}/provision-binding`)
      .then((b) => {
        if (b && b.templateId > 0) {
          setBound(true)
          setTemplateId(b.templateId)
          setRemark(b.remark ?? '')
        }
      })
      .catch((e) => {
        setBound(false)
        console.warn('[product] provision-binding 回显失败 offerId=' + offerId, e instanceof Error ? e.message : e)
      })
  }, [offerId]) // eslint-disable-line react-hooks/exhaustive-deps

  const opts = templates
    .filter((x) => x.legalEntityId === legalEntityId && x.status === 'ENABLED')
    .map((x) => ({ value: String(x.id), label: `${x.name} (${x.code})` }))

  const save = async () => {
    if (!templateId) { setErr(p.pTemplate); return }
    setBusy(true); setErr('')
    try {
      await apiFetch(`/products/${offerId}/provision-binding`, { method: 'PUT', body: { templateId, remark } })
      toast.success(p.bindSaved)
      onDone()
    } catch (e) {
      setErr(e instanceof Error ? e.message : p.saveFail)
      setBusy(false)
    }
  }

  const unbind = async () => {
    if (!(await confirmDialog(p.unbindConfirm, { danger: true }))) return
    setBusy(true); setErr('')
    try {
      await apiFetch(`/products/${offerId}/provision-binding`, { method: 'DELETE' })
      toast.success(p.unbindOk)
      onDone()
    } catch (e) {
      setErr(e instanceof Error ? e.message : p.saveFail)
      setBusy(false)
    }
  }

  return (
    <Drawer title={p.bindTemplateTitle} onClose={onClose}
      footer={
        <>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onClose}>{t.pages.company.cancel}</button>
          {bound && (
            <button className="h-8 cursor-pointer rounded-sm border border-[var(--color-danger)] px-4 text-[13px] text-[var(--color-danger)] hover:bg-[color-mix(in_srgb,var(--color-danger)_10%,transparent)]" disabled={busy} onClick={unbind}>{p.unbind}</button>
          )}
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy} onClick={save}>{t.pages.company.save}</button>
        </>
      }>
      <div className="flex flex-col gap-3.5">
        <div className="rounded-sm border border-[var(--shell-card-border)] bg-[var(--shell-menu-hover-bg)] px-3 py-2 text-[13px] text-[var(--shell-content-text)]">{offerName}</div>
        <FormField label={p.fTemplate} required>
          <Dropdown
            value={templateId ? String(templateId) : ''}
            options={[{ value: '', label: p.pTemplate }, ...opts]}
            onChange={(v) => setTemplateId(Number(v) || 0)}
            ariaLabel={p.fTemplate}
          />
        </FormField>
        <FormField label={p.fRemark}>
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={remark} placeholder={p.fRemark}
            onChange={(e) => setRemark(e.target.value)} />
        </FormField>
        <p className="text-xs text-[var(--shell-group-title)]">{p.bindTip}</p>
        {err && <ErrorBanner message={err} />}
      </div>
    </Drawer>
  )
}
