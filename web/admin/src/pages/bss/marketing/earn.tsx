// 缴费送积分规则 Tab:当前规则摘要(单行 DataTable)+ 抽屉式编辑保存(每元积分/起缴门槛/有效期天数)。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { getEarnRule, saveEarnRule, type EarnRule } from '../../../api/marketing'
import { useT } from '../../../i18n'
import { Card } from '../../../components/ui/card'
import { RuleStatus } from './RuleStatus'
import {
  PageHead, ErrorBanner, ToolbarButton, FormField, DataTable, type ColumnDef,
} from '../../../components/business'
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
      toast.success(m.earnSaved)
      setSaved(true)
      closeForm()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : m.loadFail)
    } finally {
      setSaving(false)
    }
  }

  const columns: ColumnDef[] = [
    { key: 'status', label: m.colStatus, render: () => (rule
      ? <RuleStatus status={String(rule.status)} />
      : null) },
    { key: 'pointsPerYuan', label: m.earnPointsPerYuan, render: () => (rule ? String(rule.pointsPerYuan) + ' ' + m.earnPerYuan : '—') },
    { key: 'minCents', label: m.earnMinYuan, render: () => (rule ? (rule.minCents / 100).toFixed(2) : '—') },
    { key: 'expireDays', label: m.earnExpireDays, render: () => (rule ? (rule.expireDays > 0 ? String(rule.expireDays) : m.earnNever) : '—') },
  ]

  return (
    <div>
      <PageHead title={m.tabEarn} desc={m.desc} />
      <Card className="p-4">
        <div className="mb-3 flex items-center gap-2 text-xs text-[var(--shell-crumb-text)]">
          {m.earnCurrent}
          <span className="flex-1" />
          {saved && <span className="text-xs text-[var(--color-success)]">{m.earnSaved}</span>}
          <ToolbarButton primary onClick={() => { setSaved(false); setOpen(true) }}>{m.earnSave}</ToolbarButton>
        </div>
        {error ? <ErrorBanner message={error} /> : (
          <DataTable columns={columns} rows={rule ? [{ ...rule }] : []} emptyText={m.earnNone} />
        )}
      </Card>
      {open && (
        <Drawer title={m.earnSave} onClose={closeForm}
          footer={
            <>
              <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={closeForm}>
                {t.common.confirmDialog.cancel}
              </button>
              <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)] disabled:opacity-50" disabled={saving} onClick={submit}>
                {saving ? m.saving : m.earnSave}
              </button>
            </>
          }>
          <div className="grid gap-3">
            <FormField label={m.earnPointsPerYuan} required hint={m.earnPointsPerYuanHint}>
              <Input inputMode="numeric" value={form.pointsPerYuan} placeholder={m.earnPointsPh}
                onChange={(e) => setForm({ ...form, pointsPerYuan: e.target.value })} />
            </FormField>
            <FormField label={m.earnMinYuan}>
              <Input inputMode="decimal" value={form.minCentsYuan} placeholder={m.earnMinPh}
                onChange={(e) => setForm({ ...form, minCentsYuan: e.target.value })} />
            </FormField>
            <FormField label={m.earnExpireDays} hint={m.earnExpireDaysHint}>
              <Input inputMode="numeric" value={form.expireDays} placeholder={m.earnExpirePh}
                onChange={(e) => setForm({ ...form, expireDays: e.target.value })} />
            </FormField>
          </div>
          {formError && <div className="mt-3"><ErrorBanner message={formError} /></div>}
        </Drawer>
      )}
    </div>
  )
}
