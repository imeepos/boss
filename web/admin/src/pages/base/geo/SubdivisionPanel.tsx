// 行政区划维护面板:按国家筛选 + 列表 + 新增/编辑 + 启停 + 译名维护。
import { useCallback, useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { btnPrimary, formBox, input, link, th, td } from './style'
import type { CountryRow } from './CountryPanel'

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
  const [country, setCountry] = useState('')
  const [rows, setRows] = useState<SubdivRow[]>([])
  const [error, setError] = useState('')
  const [form, setForm] = useState<SubdivRow | null>(null)
  const [editing, setEditing] = useState(false)
  const [namesOf, setNamesOf] = useState<string | null>(null)

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

  const save = async () => {
    if (!form) return
    const path = editing ? `/geo/subdivisions/${form.code}` : '/geo/subdivisions'
    try {
      await apiFetch(path, { method: editing ? 'PUT' : 'POST', body: { ...form } })
      setForm(null)
      setEditing(false)
      load()
    } catch {
      setError(g.saveFail)
    }
  }

  const toggle = async (row: SubdivRow) => {
    await apiFetch(`/geo/subdivisions/${row.code}/active`, {
      method: 'PUT', body: { active: !row.isActive },
    }).catch(() => setError(g.saveFail))
    load()
  }

  return (
    <div>
      {error && <div style={{ color: '#e54545' }}>{error}</div>}
      <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
        <select value={country} onChange={(e) => setCountry(e.target.value)} style={input}>
          <option value="">{g.filterCountry}</option>
          {countries.map((c) => <option key={c.alpha2} value={c.alpha2}>{c.alpha2} {c.displayName}</option>)}
        </select>
        <button style={btnPrimary} onClick={() => { setForm({ ...EMPTY, countryCode: country }); setEditing(false) }}>
          {g.add}
        </button>
      </div>
      {form && (
        <div style={formBox}>
          {(
            [
              ['code', 'ISO 3166-2 code'],
              ['countryCode', 'country(alpha-2)'],
              ['parentCode', 'parent code'],
              ['category', 'category'],
            ] as const
          ).map(([k, label]) => (
            <input
              key={k}
              placeholder={label}
              disabled={editing && k === 'code'}
              value={form[k]}
              onChange={(e) => setForm({ ...form, [k]: e.target.value })}
              style={input}
            />
          ))}
          <input
            type="number" min={1} max={4} placeholder="level"
            value={form.level}
            onChange={(e) => setForm({ ...form, level: Number(e.target.value) })}
            style={input}
          />
          <input
            type="number" min={2} max={10} placeholder="osm_admin_level"
            value={form.osmAdminLevel}
            onChange={(e) => setForm({ ...form, osmAdminLevel: Number(e.target.value) })}
            style={input}
          />
          <input
            type="number" placeholder="geonameid"
            value={form.geonameId}
            onChange={(e) => setForm({ ...form, geonameId: Number(e.target.value) })}
            style={input}
          />
          <button style={btnPrimary} onClick={save}>{g.save}</button>
          <button onClick={() => setForm(null)}>{g.cancel}</button>
        </div>
      )}
      <table style={{ width: '100%', background: '#fff', borderCollapse: 'collapse', fontSize: 13 }}>
        <thead>
          <tr>{g.subdivColumns.map((c) => <th key={c} style={th}>{c}</th>)}</tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.code}>
              <td style={td}>{r.code}</td>
              <td style={td}>{r.countryCode}</td>
              <td style={td}>{r.parentCode || '—'}</td>
              <td style={td}>{r.level}</td>
              <td style={td}>{r.category}</td>
              <td style={td}>{r.displayName}</td>
              <td style={td}>{r.isActive ? g.active : g.inactive}</td>
              <td style={td}>
                <a style={link} onClick={() => { setForm({ ...r }); setEditing(true) }}>{g.edit}</a>
                <a style={link} onClick={() => toggle(r)}>{r.isActive ? g.disable : g.enable}</a>
                <a style={link} onClick={() => setNamesOf(namesOf === r.code ? null : r.code)}>
                  {g.names}
                </a>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <p style={{ color: '#888' }}>{g.total.replace('{count}', String(rows.length))}</p>
      {namesOf && <SubdivNamesCard code={namesOf} />}
    </div>
  )
}

// SubdivNamesCard 区划译名维护(列表 + 添加/删除)。
function SubdivNamesCard({ code }: { code: string }) {
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
    if (!name) return
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
    <div style={{ ...formBox, flexDirection: 'column', alignItems: 'flex-start', gap: 6 }}>
      <b>{code} {g.names}</b>
      {names.map((n) => (
        <div key={n.locale + n.nameType}>
          {n.locale} / {n.nameType} / {n.name}{' '}
          <a style={link} onClick={() => remove(n.locale, n.nameType)}>x</a>
        </div>
      ))}
      <div style={{ display: 'flex', gap: 6 }}>
        <input style={input} value={locale} onChange={(e) => setLocale(e.target.value)} />
        <input style={input} value={name} onChange={(e) => setName(e.target.value)} />
        <button style={btnPrimary} onClick={add}>{g.addName}</button>
      </div>
    </div>
  )
}
