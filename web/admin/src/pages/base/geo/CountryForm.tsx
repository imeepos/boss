// CountryForm:国家新建/编辑抽屉,antd Pro 表单惯例(标签 + 必填标记 + 两列栅格)。
import { useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import './geo.css'

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
          {error && <span className="geo-error" style={{ margin: 0, marginRight: 'auto' }}>{error}</span>}
          <button className="geo-btn" onClick={onCancel}>{g.cancel}</button>
          <button className="geo-btn geo-btn-primary" onClick={save}>{g.save}</button>
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
    <div className="geo-form">
      {TEXT_FIELDS.map(([k, req]) => (
        <div key={k} className={`geo-field ${k === 'shortName' || k === 'fullName' ? 'full' : ''}`}>
          <label>
            {req && <span className="req">*</span>}{labels[k]}
          </label>
          <input
            className="geo-input"
            disabled={editing && k === 'alpha2'}
            value={form[k] as string}
            onChange={(e) => onChange({ ...form, [k]: e.target.value })}
          />
        </div>
      ))}
      <div className="geo-field">
        <label><span className="req">*</span>{labels.continent}</label>
        <select className="geo-select" value={form.continentCode}
          onChange={(e) => onChange({ ...form, continentCode: e.target.value })}>
          {CONTINENTS.map((c) => <option key={c}>{c}</option>)}
        </select>
      </div>
      <div className="geo-field">
        <label><span className="req">*</span>{labels.status}</label>
        <select className="geo-select" value={form.status}
          onChange={(e) => onChange({ ...form, status: e.target.value })}>
          <option>INDEPENDENT</option>
          <option>DISCONTINUED</option>
        </select>
      </div>
    </div>
  )
}
