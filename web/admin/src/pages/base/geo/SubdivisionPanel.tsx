// 行政区划维护面板:卡片化列表 + 工具栏(国家筛选/搜索/主操作) + 抽屉式表单与译名。
import { useCallback, useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { Pagination } from '../../../components/Pagination'
import { useQueryInt, useQueryState } from '../../../lib/useQueryState'
import { StatusTag } from './CountryPanel'
import type { CountryRow } from './CountryForm'
import './geo.css'

export interface SubdivRow {
  code: string
  countryCode: string
  parentCode: string
  level: number
  category: string
  osmAdminLevel: number
  geonameId: number
  isActive: boolean
  displayName: string
}

interface NameRow { locale: string; name: string; nameType: string }

const EMPTY: SubdivRow = {
  code: '', countryCode: '', parentCode: '', level: 1, category: 'region',
  osmAdminLevel: 4, geonameId: 0, isActive: true, displayName: '',
}

export function SubdivisionPanel() {
  const t = useT()
  const g = t.pages.geo
  const [countries, setCountries] = useState<CountryRow[]>([])
  const [country, setCountry] = useQueryState('country', '')
  const [rows, setRows] = useState<SubdivRow[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useQueryState('kw', '')
  const [page, setPage] = useQueryInt('page', 1)
  const [pageSize, setPageSize] = useQueryInt('size', 20)
  const [form, setForm] = useState<SubdivRow | null>(null)
  const [editing, setEditing] = useState(false)
  const [namesOf, setNamesOf] = useState<string | null>(null)

  const filterCountry = (v: string) => { setCountry(v); setPage(1) }
  const search = (v: string) => { setKeyword(v); setPage(1) }
  const resize = (v: number) => { setPageSize(v); setPage(1) }

  useEffect(() => {
    apiFetch<CountryRow[]>('/geo/countries')
      .then((d) => setCountries(d ?? []))
      .catch(() => setError(g.loadFail))
  }, [g])

  const load = useCallback(() => {
    apiFetch<SubdivRow[]>(`/geo/subdivisions${country ? `?country=${country}` : ''}`)
      .then((d) => setRows(d ?? []))
      .catch(() => setError(g.loadFail))
  }, [country, g])

  useEffect(load, [load])

  const toggle = async (row: SubdivRow) => {
    await apiFetch(`/geo/subdivisions/${row.code}/active`, {
      method: 'PUT', body: { active: !row.isActive },
    }).catch(() => setError(g.saveFail))
    load()
  }

  const kw = keyword.trim().toLowerCase()
  const filtered = kw
    ? rows.filter((r) => [r.code, r.category, r.displayName].some((s) => s.toLowerCase().includes(kw)))
    : rows
  const pageCount = Math.max(1, Math.ceil(filtered.length / pageSize))
  const safePage = Math.min(page, pageCount)
  const paged = filtered.slice((safePage - 1) * pageSize, safePage * pageSize)

  return (
    <div className="geo-card">
      {error && <div className="geo-error" role="alert">{error}</div>}
      <div className="geo-toolbar">
        <select className="geo-select" style={{ width: 180 }} value={country}
          onChange={(e) => filterCountry(e.target.value)}>
          <option value="">{g.filterCountry}</option>
          {countries.map((c) => (
            <option key={c.alpha2} value={c.alpha2}>{c.alpha2} {c.displayName}</option>
          ))}
        </select>
        <input className="geo-input" style={{ width: 200 }} placeholder={g.searchPlaceholder}
          value={keyword} onChange={(e) => search(e.target.value)} />
        <div className="spacer" />
        <button className="geo-btn geo-btn-primary"
          onClick={() => { setForm({ ...EMPTY, countryCode: country }); setEditing(false) }}>
          + {g.add}
        </button>
      </div>
      <SubdivTable rows={paged} onEdit={(r) => { setForm({ ...r }); setEditing(true) }}
        onToggle={toggle} onNames={(code) => setNamesOf(namesOf === code ? null : code)} />
      <div className="geo-footer">
        <Pagination page={safePage} pageSize={pageSize} total={filtered.length}
          onPage={setPage} onSize={resize} totalText={g.total}
          prevText={g.prev} nextText={g.next} perPageText={g.perPage} />
      </div>
      {form && <SubdivForm initial={form} editing={editing} country={country}
        onDone={() => { setForm(null); load() }} onCancel={() => setForm(null)} />}
      {namesOf && <SubdivNames code={namesOf} onClose={() => setNamesOf(null)} />}
    </div>
  )
}

// SubdivTable 区划列表(design-spec §4.2)。
function SubdivTable({ rows, onEdit, onToggle, onNames }: {
  rows: SubdivRow[]
  onEdit: (r: SubdivRow) => void
  onToggle: (r: SubdivRow) => void
  onNames: (code: string) => void
}) {
  const g = useT().pages.geo
  if (!rows.length) return <div className="geo-empty">{g.empty}</div>
  return (
    <div className="geo-table-wrap">
      <table className="geo-table">
        <thead>
          <tr>{g.subdivColumns.map((c) => <th key={c}>{c}</th>)}</tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.code}>
              <td className="num">{r.code}</td>
              <td>{r.countryCode}</td>
              <td>{r.parentCode || '—'}</td>
              <td>{r.level}</td>
              <td>{r.category}</td>
              <td>{r.displayName}</td>
              <td><StatusTag on={r.isActive} /></td>
              <td>
                <div className="geo-act">
                  <button onClick={() => onEdit(r)}>{g.edit}</button><span className="sep">|</span>
                  <button onClick={() => onToggle(r)}>{r.isActive ? g.disable : g.enable}</button><span className="sep">|</span>
                  <button onClick={() => onNames(r.code)}>{g.names}</button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

// SubdivForm 区划新建/编辑抽屉表单。
function SubdivForm({ initial, editing, country, onDone, onCancel }: {
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
    } catch {
      setError(g.saveFail)
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
          {error && <span className="geo-error" style={{ margin: 0, marginRight: 'auto' }}>{error}</span>}
          <button className="geo-btn" onClick={onCancel}>{g.cancel}</button>
          <button className="geo-btn geo-btn-primary" onClick={save}>{g.save}</button>
        </>
      }>
      <div className="geo-form">
        {texts.map(([k, req]) => (
          <div key={k} className="geo-field full">
            <label>{req && <span className="req">*</span>}{g.subdivFields[k]}</label>
            <input className="geo-input" disabled={editing && k === 'code'}
              value={form[k] as string}
              onChange={(e) => setForm({ ...form, [k]: e.target.value })} />
          </div>
        ))}
        <NumField label={g.subdivFields.level} value={form.level}
          onChange={(v) => setForm({ ...form, level: v })} />
        <NumField label={g.subdivFields.osmAdminLevel} value={form.osmAdminLevel}
          onChange={(v) => setForm({ ...form, osmAdminLevel: v })} />
        <div className="geo-field full">
          <label>{g.subdivFields.geonameId}</label>
          <input className="geo-input" type="number" value={form.geonameId}
            onChange={(e) => setForm({ ...form, geonameId: Number(e.target.value) })} />
        </div>
      </div>
      {editing && <p className="hint" style={{ marginTop: 12, fontSize: 12, color: 'var(--shell-group-title)' }}>
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
    <div className="geo-field">
      <label>{label}</label>
      <input className="geo-input" type="number" value={value}
        onChange={(e) => onChange(Number(e.target.value))} />
    </div>
  )
}

// SubdivNames 区划译名维护抽屉。
function SubdivNames({ code, onClose }: { code: string; onClose: () => void }) {
  const t = useT()
  const g = t.pages.geo
  const [names, setNames] = useState<NameRow[]>([])
  const [locale, setLocale] = useState('zh-Hans')
  const [name, setName] = useState('')

  const load = useCallback(() => {
    apiFetch<NameRow[]>(`/geo/subdivisions/${code}/names`)
      .then((d) => setNames(d ?? []))
      .catch(() => setNames([]))
  }, [code])

  useEffect(load, [load])

  const add = async () => {
    if (!name.trim()) return
    await apiFetch(`/geo/subdivisions/${code}/names`, {
      method: 'POST', body: { locale, name, nameType: 'STANDARD' },
    }).catch(() => undefined)
    setName('')
    load()
  }

  const remove = async (loc: string, nameType: string) => {
    await apiFetch(`/geo/subdivisions/${code}/names/${loc}/${nameType}`, { method: 'DELETE' })
    load()
  }

  return (
    <Drawer title={`${g.names} · ${code}`} onClose={onClose}>
      {names.map((n) => (
        <div key={n.locale + n.nameType} className="geo-tag geo-tag-off"
          style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 6, padding: '2px 8px' }}>
          <span>{n.locale} · {n.nameType} · {n.name}</span>
          <button style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--color-danger)' }}
            onClick={() => remove(n.locale, n.nameType)}>×</button>
        </div>
      ))}
      <div style={{ display: 'flex', gap: 8, marginTop: 12 }}>
        <input className="geo-input" style={{ width: 110 }} value={locale}
          onChange={(e) => setLocale(e.target.value)} placeholder="locale" />
        <input className="geo-input" style={{ flex: 1 }} value={name}
          onChange={(e) => setName(e.target.value)} placeholder="name" />
        <button className="geo-btn" onClick={add}>{g.addName}</button>
      </div>
    </Drawer>
  )
}
