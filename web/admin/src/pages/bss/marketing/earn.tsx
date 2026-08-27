// 缴费送积分规则 Tab:当前规则摘要 + 抽屉式编辑保存(每元积分/起缴门槛/有效期天数)。
import { useEffect, useState } from 'react'
import { getEarnRule, saveEarnRule, type EarnRule } from '../../../api/marketing'
import { useT } from '../../../i18n'
import { Card } from '../../../components/ui/card'
import { Badge } from '../../../components/ui/badge'
import { ErrorBanner, ToolbarButton, FormField } from '../../../components/business'
import { Drawer } from '../../../components/Drawer'
import { Input } from '../../../components/ui/input'

const EMPTY_FORM = { pointsPerYuan: '', minCentsYuan: '', expireDays: '' }

export default function EarnRuleTab() {
  const t = useT()
  const m = t.pages.marketing
  const [rule, setRule] = useState<EarnRule | null>(null)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)
  const [saving, setSaving] = useState(false)
  const [open, setOpen] = useState(false)
  const [formError, setFormError] = useState('')
  const [form, setForm] = useState(EMPTY_FORM)

  const load = () => {
    setError('')
    getEarnRule()
      .then((d) => {
        const r = d?.rule ?? null
        setRule(r)
        setForm({
          pointsPerYuan: r ? String(r.pointsPerYuan) : '',
          minCentsYuan: r ? (r.minCents / 100).toFixed(2) : '',
          expireDays: r ? String(r.expireDays) : '',
        })
      })
      .catch((e) => setError(e instanceof Error ? e.message : m.loadFail))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const closeForm = () => {
    setOpen(false)
    setFormError('')
    load()
  }

  const submit = async () => {
    if (saving) return
    setFormError('')
    if (form.pointsPerYuan === '') {
      setFormError(m.formIncomplete); return
    }
    setSaving(true)
    try {
      await saveEarnRule({
        pointsPerYuan: Number(form.pointsPerYuan || 0),
        minCents: Math.round(Number(form.minCentsYuan || 0) * 100),
        expireDays: Number(form.expireDays || 0),
      })
      setSaved(true)
      closeForm()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : m.loadFail)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div>
      <Card className="p-4">
        {error && <div className="mb-3"><ErrorBanner message={error} /></div>}
        <div className="flex items-center gap-2 text-xs text-[var(--shell-crumb-text)]">
          {m.earnCurrent}
          {rule ? (
            <>
              <Badge variant="success">{rule.status}</Badge>
              <span>
                {rule.pointsPerYuan} {m.earnPerYuan} / {m.earnMin} {(rule.minCents / 100).toFixed(2)} /
                {' '}{m.earnExpire} {rule.expireDays > 0 ? rule.expireDays : m.earnNever}
              </span>
            </>
          ) : (
            <span>{m.earnNone}</span>
          )}
          <span className="flex-1" />
          {saved && <span className="text-xs text-[var(--color-success)]">{m.earnSaved}</span>}
          <ToolbarButton primary onClick={() => { setSaved(false); setOpen(true) }}>{m.earnSave}</ToolbarButton>
        </div>
      </Card>
      {open && (
        <Drawer title={m.earnSave} onClose={closeForm}
          footer={
            <>
              <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={closeForm}>
                {t.common.confirmDialog.cancel}
              </button>
              <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={saving} onClick={submit}>
                {saving ? m.creating : m.earnSave}
              </button>
            </>
          }>
          <div className="grid gap-3">
            <FormField label={m.earnPointsPerYuan} required hint={m.earnPointsPerYuanHint}>
              <Input inputMode="numeric" value={form.pointsPerYuan}
                onChange={(e) => setForm({ ...form, pointsPerYuan: e.target.value })} />
            </FormField>
            <FormField label={m.earnMinYuan}>
              <Input inputMode="decimal" value={form.minCentsYuan}
                onChange={(e) => setForm({ ...form, minCentsYuan: e.target.value })} />
            </FormField>
            <FormField label={m.earnExpireDays} hint={m.earnExpireDaysHint}>
              <Input inputMode="numeric" value={form.expireDays}
                onChange={(e) => setForm({ ...form, expireDays: e.target.value })} />
            </FormField>
          </div>
          {formError && <div className="mt-3"><ErrorBanner message={formError} /></div>}
        </Drawer>
      )}
    </div>
  )
}
