// 新增标签抽屉:契约 POST /tags {tagNo, epcCode, band};必填校验与载荷组装走 ./logic。
import { useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { ErrorBanner } from '../../../components/business/page-head'
import { Drawer } from '../../../components/Drawer'
import { buildTagPayload, emptyTagForm, tagFormErr, type TagFormState } from './logic'
import { TagFormFields } from './TagFormFields'

export function CreateTagDrawer({ onClose, onSaved }: { onClose: () => void; onSaved: () => void }) {
  const t = useT()
  const g = t.pages.tagPage
  const [form, setForm] = useState<TagFormState>(emptyTagForm)
  const [err, setErr] = useState('')
  const [apiError, setApiError] = useState('')
  const [busy, setBusy] = useState(false)

  const submit = async () => {
    if (busy) return
    const key = tagFormErr(form)
    setErr(key)
    if (key) return
    setBusy(true)
    setApiError('')
    try {
      await apiFetch('/tags', { method: 'POST', body: buildTagPayload(form) })
      toast.success(g.createOk)
      onSaved()
      onClose()
    } catch (e) {
      setApiError(e instanceof Error ? e.message : g.loadFail)
    } finally {
      setBusy(false)
    }
  }

  return (
    <Drawer title={g.createTitle} onClose={onClose}
      footer={
        <>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onClose}>{t.pages.company.cancel}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy} onClick={submit}>
            {busy ? t.pages.account.submitting : t.pages.company.save}
          </button>
        </>
      }>
      <TagFormFields form={form} error={err}
        onTagNo={(tagNo) => setForm({ ...form, tagNo })}
        onEpc={(epcCode) => setForm({ ...form, epcCode })}
        onBand={(band) => setForm({ ...form, band })} />
      {apiError && <ErrorBanner message={apiError} />}
    </Drawer>
  )
}
