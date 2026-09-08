// SubdivForm 区划新建/编辑抽屉表单(NumField 数字字段内含)。
import { useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { ToolbarButton } from '../../../components/business/page-head'
import { Input } from '../../../components/ui/input'
import { FORM, FIELD, FIELD_FULL, LABEL, REQ } from './styles'
import type { SubdivRow } from './subdiv-shared'

export function SubdivForm({ initial, editing, country, onDone, onCancel }: {
  initial: SubdivRow
  editing: boolean
  country: string
  onDone: () => void
  onCancel: () => void
}) {
  const t = useT()
  const g = t.pages.geo
  const [form, setForm] = useState(initial)
  const [error, setError] = useState('')

  const save = async () => {
    const path = editing ? `/geo/subdivisions/${form.code}` : '/geo/subdivisions'
    try {
      await apiFetch(path, { method: editing ? 'PUT' : 'POST', body: { ...form } })
      onDone()
    } catch (e) {
      setError(e instanceof Error ? e.message : g.saveFail)
    }
  }

  // 字段标签来自 i18n geo.subdivFields。
  const texts: [keyof typeof g.subdivFields, boolean][] = [
    ['code', true],
    ['countryCode', true],
    ['parentCode', false],
    ['category', true],
  ]

  return (
    <Drawer title={`${editing ? g.edit : g.add} · ${g.tabSubdiv}`} onClose={onCancel}
      footer={
        <>
          {error && <span className="mr-auto text-xs text-[var(--color-danger)]">{error}</span>}
          <ToolbarButton onClick={onCancel}>{g.cancel}</ToolbarButton>
          <ToolbarButton primary onClick={save}>{g.save}</ToolbarButton>
        </>
      }>
      <div className={FORM}>
        {texts.map(([k, req]) => (
          <div key={k} className={FIELD_FULL}>
            <label className={LABEL}>{req && <span className={REQ}>*</span>}{g.subdivFields[k]}</label>
            <Input disabled={editing && k === 'code'}
              value={form[k] as string}
              onChange={(e) => setForm({ ...form, [k]: e.target.value })} />
          </div>
        ))}
        <NumField label={g.subdivFields.level} value={form.level}
          onChange={(v) => setForm({ ...form, level: v })} />
        <NumField label={g.subdivFields.osmAdminLevel} value={form.osmAdminLevel}
          onChange={(v) => setForm({ ...form, osmAdminLevel: v })} />
        <div className={FIELD_FULL}>
          <label className={LABEL}>{g.subdivFields.geonameId}</label>
          <Input type="number" value={form.geonameId}
            onChange={(e) => setForm({ ...form, geonameId: Number(e.target.value) })} />
        </div>
      </div>
      {editing && <p className="mt-3 text-xs text-[var(--shell-group-title)]">
        {g.filterCountry}: {country || form.countryCode}
      </p>}
    </Drawer>
  )
}

// NumField 数字输入字段。
function NumField({ label, value, onChange }: {
  label: string
  value: number
  onChange: (v: number) => void
}) {
  return (
    <div className={FIELD}>
      <label className={LABEL}>{label}</label>
      <Input type="number" value={value}
        onChange={(e) => onChange(Number(e.target.value))} />
    </div>
  )
}
