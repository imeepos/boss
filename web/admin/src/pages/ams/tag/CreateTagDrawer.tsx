// 新增标签抽屉:契约 POST /tags {tagNo, epcCode, band};必填校验与载荷组装走 ./logic。
import { useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { SubmitButton } from '../../../components/business/submit-button'
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

  const submitState = busy ? 'loading' : (apiError ? 'failed' : 'idle')
  const submitLabels = { idle: t.pages.company.save, loading: t.pages.account.submitting, success: g.createOk, failed: g.loadFail }

  return (
    <Drawer title={g.createTitle} onClose={onClose}
      footer={
        <>
          <ToolbarButton onClick={onClose} disabled={busy}>{t.pages.company.cancel}</ToolbarButton>
          <SubmitButton state={submitState} labels={submitLabels} disabled={busy} onClick={submit} />
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
