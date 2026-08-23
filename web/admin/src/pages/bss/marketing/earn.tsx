// 缴费送积分规则 Tab:单行生效规则(每元积分/起缴门槛/有效期天数)查看与保存。
import { useEffect, useState } from 'react'
import { getEarnRule, saveEarnRule, type EarnRule } from '../../../api/marketing'
import { useT } from '../../../i18n'
import { Card } from '../../../components/ui/card'
import { Badge } from '../../../components/ui/badge'
import { ErrorBanner, ToolbarButton, FormField } from '../../../components/business'

export default function EarnRuleTab() {
  const t = useT()
  const m = t.pages.marketing
  const [rule, setRule] = useState<EarnRule | null>(null)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)
  const [saving, setSaving] = useState(false)
  const [form, setForm] = useState({ pointsPerYuan: '', minCentsYuan: '', expireDays: '' })

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

  const submit = async () => {
    if (saving) return
    setSaved(false)
    setSaving(true)
    try {
      await saveEarnRule({
        pointsPerYuan: Number(form.pointsPerYuan || 0),
        minCents: Math.round(Number(form.minCentsYuan || 0) * 100),
        expireDays: Number(form.expireDays || 0),
      })
      setSaved(true)
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : m.loadFail)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div>
      <Card className="p-4">
        {error && <div className="mb-3"><ErrorBanner message={error} /></div>}
        <div className="mb-3 flex items-center gap-2 text-xs text-[var(--shell-crumb-text)]">
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
        </div>
        <div className="grid grid-cols-3 gap-3">
          <FormField label={m.earnPointsPerYuan} required hint={m.earnPointsPerYuanHint}>
            <input className="w-full" inputMode="numeric" value={form.pointsPerYuan}
              onChange={(e) => setForm({ ...form, pointsPerYuan: e.target.value })} />
          </FormField>
          <FormField label={m.earnMinYuan}>
            <input className="w-full" inputMode="decimal" value={form.minCentsYuan}
              onChange={(e) => setForm({ ...form, minCentsYuan: e.target.value })} />
          </FormField>
          <FormField label={m.earnExpireDays} hint={m.earnExpireDaysHint}>
            <input className="w-full" inputMode="numeric" value={form.expireDays}
              onChange={(e) => setForm({ ...form, expireDays: e.target.value })} />
          </FormField>
        </div>
        <div className="mt-3 flex items-center justify-end gap-3">
          {saved && <span className="text-xs text-[var(--color-success)]">{m.earnSaved}</span>}
          <ToolbarButton onClick={submit}>{saving ? m.creating : m.earnSave}</ToolbarButton>
        </div>
      </Card>
    </div>
  )
}
