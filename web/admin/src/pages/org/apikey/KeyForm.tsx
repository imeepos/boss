// API key 签发 Drawer:选择绑定账号 + 用途名;密钥仅创建时返回一次。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { FormField } from '../../../components/business/form-field'
import { ErrorBanner } from '../../../components/business/page-head'
import { Input } from '../../../components/ui/input'

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
        <FormField label={t.pages.apikey.fAccount} required>
          <Dropdown
            value={values.accountId ? String(values.accountId) : ''}
            options={[
              { value: '', label: t.pages.apikey.pAccount },
              ...accounts.map((a) => ({ value: String(a.id), label: `${a.username} ${a.realName}` })),
            ]}
            onChange={(v) => onChange({ ...values, accountId: Number(v) || 0 })}
            ariaLabel={t.pages.apikey.pAccount}
          />
        </FormField>
        <FormField label={t.pages.apikey.fName} required>
          <Input value={values.name} placeholder={t.pages.apikey.pName}
            onChange={(e) => onChange({ ...values, name: e.target.value })} />
          {!nameOk && values.name !== '' && <span className="text-[11px] text-[var(--color-danger)]">{t.pages.apikey.eName}</span>}
        </FormField>
        {submitError && <ErrorBanner message={submitError} />}
      </div>
    </Drawer>
  )
}
