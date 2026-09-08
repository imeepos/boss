// 创建开放应用 Drawer:名称 + 限流/配额 + 沙箱标记;Secret 创建后仅显示一次。
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { FormField } from '../../../components/business/form-field'
import { ErrorBanner } from '../../../components/business/page-head'
import { Input } from '../../../components/ui/input'

export interface AppFormValues {
  name: string
  rateLimitRpm: number
  dailyQuota: number
  sandbox: boolean
}

export function AppFormDrawer({
  open, values, onChange, onClose, onSubmit, busy, submitError,
}: {
  open: boolean
  values: AppFormValues
  onChange: (v: AppFormValues) => void
  onClose: () => void
  onSubmit: () => void
  busy: boolean
  submitError: string
}) {
  const t = useT()
  if (!open) return null
  const nameOk = values.name.trim().length > 0 && values.name.trim().length <= 64
  const rpmOk = Number.isFinite(values.rateLimitRpm) && values.rateLimitRpm >= 0
  const quotaOk = Number.isFinite(values.dailyQuota) && values.dailyQuota >= 0

  return (
    <Drawer title={t.pages.openplat.createTitle} onClose={onClose}
      footer={
        <>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onClose}>{t.pages.company.cancel}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy || !nameOk || !rpmOk || !quotaOk} onClick={onSubmit}>
            {busy ? t.pages.account.submitting : t.pages.company.save}
          </button>
        </>
      }>
      <div className="flex flex-col gap-3.5">
        <FormField label={t.pages.openplat.fName} required>
          <Input value={values.name} maxLength={64}
            placeholder={t.pages.openplat.pName} onChange={(e) => onChange({ ...values, name: e.target.value })} />
        </FormField>
        <FormField label={t.pages.openplat.fRpm}>
          <Input type="number" min={0} value={values.rateLimitRpm}
            onChange={(e) => onChange({ ...values, rateLimitRpm: Number(e.target.value) })} />
        </FormField>
        <FormField label={t.pages.openplat.fQuota}>
          <Input type="number" min={0} value={values.dailyQuota}
            onChange={(e) => onChange({ ...values, dailyQuota: Number(e.target.value) })} />
        </FormField>
        <FormField label={t.pages.openplat.fSandbox}>
          <label className="flex items-center gap-2 text-[13px]">
            <input type="checkbox" checked={values.sandbox}
              onChange={(e) => onChange({ ...values, sandbox: e.target.checked })} />
            {t.pages.openplat.sandboxHint}
          </label>
        </FormField>
        {submitError && <ErrorBanner message={submitError} />}
      </div>
    </Drawer>
  )
}
