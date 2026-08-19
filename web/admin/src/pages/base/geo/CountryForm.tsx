// CountryForm:国家新建/编辑抽屉,antd Pro 表单惯例(标签 + 必填标记 + 两列栅格)。
import { useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { ToolbarButton } from '../../../components/business/page-head'
import { Input } from '../../../components/ui/input'
import { FORM, FIELD, FIELD_FULL, LABEL, REQ } from './styles'

export interface CountryRow {
  alpha2: string
  alpha3: string
  numericCode: string
  shortName: string
  fullName: string
  status: string
  continentCode: string
  m49Region: string
  postalRegex: string
  isActive: boolean
  displayName: string
}

export const EMPTY: CountryRow = {
  alpha2: '', alpha3: '', numericCode: '', shortName: '', fullName: '',
  status: 'INDEPENDENT', continentCode: 'AS', m49Region: '', postalRegex: '',
  isActive: true, displayName: '',
}

const CONTINENTS = ['AS', 'EU', 'NA', 'SA', 'AF', 'OC', 'AN']

// 字段定义:[key, 标签键, 必填];标签文案来自 i18n geo.countryFields。
type CountryFieldKey = 'alpha2' | 'alpha3' | 'numericCode' | 'shortName' | 'fullName' | 'm49Region' | 'postalRegex'

const TEXT_FIELDS: [CountryFieldKey, boolean][] = [
  ['alpha2', true],
  ['alpha3', true],
  ['numericCode', true],
  ['shortName', true],
  ['fullName', false],
  ['m49Region', false],
  ['postalRegex', false],
]

export function CountryForm({ initial, editing, onDone, onCancel }: {
  initial: CountryRow
  editing: boolean
  onDone: () => void
  onCancel: () => void
}) {
  const t = useT()
  const g = t.pages.geo
  const [form, setForm] = useState(initial)
  const [error, setError] = useState('')

  const save = async () => {
    const path = editing ? `/geo/countries/${form.alpha2}` : '/geo/countries'
    try {
      await apiFetch(path, { method: editing ? 'PUT' : 'POST', body: { ...form } })
      onDone()
    } catch {
      setError(g.saveFail)
    }
  }

  return (
    <Drawer
      title={`${editing ? g.edit : g.add} · ${g.tabCountry}`}
      onClose={onCancel}
      footer={
        <>
          {error && <span className="mr-auto text-xs text-[var(--color-danger)]">{error}</span>}
          <ToolbarButton onClick={onCancel}>{g.cancel}</ToolbarButton>
          <ToolbarButton primary onClick={save}>{g.save}</ToolbarButton>
        </>
      }
    >
      <FormGrid form={form} editing={editing} labels={g.countryFields} onChange={setForm} />
    </Drawer>
  )
}

// FormGrid 两列表单栅格;编辑态主键 alpha2 只读。
function FormGrid({ form, editing, labels, onChange }: {
  form: CountryRow
  editing: boolean
  labels: ReturnType<typeof useT>['pages']['geo']['countryFields']
  onChange: (f: CountryRow) => void
}) {
  return (
    <div className={FORM}>
      {TEXT_FIELDS.map(([k, req]) => (
        <div key={k} className={k === 'shortName' || k === 'fullName' ? FIELD_FULL : FIELD}>
          <label className={LABEL}>
            {req && <span className={REQ}>*</span>}{labels[k]}
          </label>
          <Input
            disabled={editing && k === 'alpha2'}
            value={form[k] as string}
            onChange={(e) => onChange({ ...form, [k]: e.target.value })}
          />
        </div>
      ))}
      <div className={FIELD}>
        <label className={LABEL}><span className={REQ}>*</span>{labels.continent}</label>
        <Dropdown
          value={form.continentCode}
          ariaLabel={labels.continent}
          onChange={(continentCode) => onChange({ ...form, continentCode })}
          options={CONTINENTS.map((c) => ({ value: c, label: c }))}
        />
      </div>
      <div className={FIELD}>
        <label className={LABEL}><span className={REQ}>*</span>{labels.status}</label>
        <Dropdown
          value={form.status}
          ariaLabel={labels.status}
          onChange={(status) => onChange({ ...form, status })}
          options={['INDEPENDENT', 'DISCONTINUED'].map((status) => ({ value: status, label: status }))}
        />
      </div>
    </div>
  )
}
