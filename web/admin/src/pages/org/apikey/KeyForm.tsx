// API key 签发 Drawer:选择绑定账号 + 用途名;密钥仅创建时返回一次。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'

export interface ApiKeyFormValues {
  accountId: number
  name: string
}

interface AccountOption {
  id: number
  username: string
  realName: string
  status: number
}

export function ApiKeyFormDrawer({
  open, values, onChange, onClose, onSubmit, busy, submitError,
}: {
  open: boolean
  values: ApiKeyFormValues
  onChange: (v: ApiKeyFormValues) => void
  onClose: () => void
  onSubmit: () => void
  busy: boolean
  submitError: string
}) {
  const t = useT()
  const [accounts, setAccounts] = useState<AccountOption[]>([])

  useEffect(() => {
    if (!open) return
    apiFetch<AccountOption[]>('/accounts')
      .then((d) => setAccounts((d ?? []).filter((a) => a.status === 1)))
      .catch(() => setAccounts([]))
  }, [open])

  if (!open) return null
  const nameOk = values.name.trim().length > 0 && values.name.trim().length <= 64
  return (
    <Drawer title={t.pages.apikey.createTitle} onClose={onClose}
      footer={
        <>
          <button className="org-btn" onClick={onClose}>{t.pages.company.cancel}</button>
          <button className="org-btn org-btn-primary" disabled={busy || !values.accountId || !nameOk} onClick={onSubmit}>
            {busy ? t.pages.account.submitting : t.pages.company.save}
          </button>
        </>
      }>
      <div className="org-form">
        <div className="org-field">
          <label><span className="req">*</span>{t.pages.apikey.fAccount}</label>
          <select className="org-select" value={values.accountId ? String(values.accountId) : ''}
            onChange={(e) => onChange({ ...values, accountId: Number(e.target.value) || 0 })}>
            <option value="">{t.pages.apikey.pAccount}</option>
            {accounts.map((a) => (
              <option key={a.id} value={a.id}>{a.username} {a.realName}</option>
            ))}
          </select>
        </div>
        <div className="org-field">
          <label><span className="req">*</span>{t.pages.apikey.fName}</label>
          <input className="org-input" value={values.name} placeholder={t.pages.apikey.pName}
            onChange={(e) => onChange({ ...values, name: e.target.value })} />
          {!nameOk && values.name !== '' && <span className="text-[11px] text-[var(--color-danger)]">{t.pages.apikey.eName}</span>}
        </div>
        {submitError && <div className="org-error" style={{ margin: 0 }}>{submitError}</div>}
      </div>
    </Drawer>
  )
}
