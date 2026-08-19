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
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onClose}>{t.pages.company.cancel}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy || !values.accountId || !nameOk} onClick={onSubmit}>
            {busy ? t.pages.account.submitting : t.pages.company.save}
          </button>
        </>
      }>
      <div className="flex flex-col gap-3.5">
        <div className="flex flex-col gap-1.5">
          <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{t.pages.apikey.fAccount}</label>
          <select className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 pr-6 text-[13px] text-[var(--shell-content-text)] outline-none focus:border-[var(--color-border-focus)]" value={values.accountId ? String(values.accountId) : ''}
            onChange={(e) => onChange({ ...values, accountId: Number(e.target.value) || 0 })}>
            <option value="">{t.pages.apikey.pAccount}</option>
            {accounts.map((a) => (
              <option key={a.id} value={a.id}>{a.username} {a.realName}</option>
            ))}
          </select>
        </div>
        <div className="flex flex-col gap-1.5">
          <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{t.pages.apikey.fName}</label>
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={values.name} placeholder={t.pages.apikey.pName}
            onChange={(e) => onChange({ ...values, name: e.target.value })} />
          {!nameOk && values.name !== '' && <span className="text-[11px] text-[var(--color-danger)]">{t.pages.apikey.eName}</span>}
        </div>
        {submitError && <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]" style={{ margin: 0 }}>{submitError}</div>}
      </div>
    </Drawer>
  )
}
