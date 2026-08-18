// 国家维护面板:列表 + 新增/编辑表单 + 启停 + 详情(译名/关联属性)。
import { useCallback, useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { btnPrimary, formBox, input, link, th, td } from './style'

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

export interface CountryDetail extends CountryRow {
  names: { locale: string; name: string; nameType: string }[]
  attrs: {
    timeZones: string[]
    currencies: { currency: string; isPrimary: boolean; minorUnit: number }[]
    callingCodes: string[]
  }
}

const EMPTY: CountryRow = {
  alpha2: '', alpha3: '', numericCode: '', shortName: '', fullName: '',
  status: 'INDEPENDENT', continentCode: 'AS', m49Region: '', postalRegex: '',
  isActive: true, displayName: '',
}

const CONTINENTS = ['AS', 'EU', 'NA', 'SA', 'AF', 'OC', 'AN']

export function CountryPanel() {
  const t = useT()
  const g = t.pages.geo
  const [rows, setRows] = useState<CountryRow[]>([])
  const [error, setError] = useState('')
  const [form, setForm] = useState<CountryRow | null>(null)
  const [editing, setEditing] = useState(false)
  const [detail, setDetail] = useState<CountryDetail | null>(null)

  const load = useCallback(() => {
    apiFetch<CountryRow[]>('/geo/countries')
      .then((d) => setRows(d ?? []))
      .catch(() => setError(g.loadFail))
  }, [g])

  useEffect(load, [load])

  const save = async () => {
    if (!form) return
    const body = { ...form }
    const path = editing ? `/geo/countries/${form.alpha2}` : '/geo/countries'
    try {
      await apiFetch(path, { method: editing ? 'PUT' : 'POST', body })
      setForm(null)
      setEditing(false)
      load()
    } catch {
      setError(g.saveFail)
    }
  }

  const toggle = async (row: CountryRow) => {
    await apiFetch(`/geo/countries/${row.alpha2}/active`, {
      method: 'PUT',
      body: { active: !row.isActive },
    }).catch(() => setError(g.saveFail))
    load()
  }

  const openDetail = async (alpha2: string) => {
    const d = await apiFetch<CountryDetail>(`/geo/countries/${alpha2}`).catch(() => null)
    setDetail(d ?? null)
    if (!d) setError(g.loadFail)
  }

  return (
    <div>
      {error && <div style={{ color: '#e54545' }}>{error}</div>}
      <button style={btnPrimary} onClick={() => { setForm({ ...EMPTY }); setEditing(false) }}>
        {g.add}
      </button>
      {form && (
        <div style={formBox}>
          {(
            [
              ['alpha2', 'ISO 3166-1 alpha-2'],
              ['alpha3', 'alpha-3'],
              ['numericCode', 'numeric'],
              ['shortName', 'short name'],
              ['fullName', 'full name'],
              ['m49Region', 'M49 region'],
              ['postalRegex', 'postal regex'],
            ] as const
          ).map(([k, label]) => (
            <input
              key={k}
              placeholder={label}
              disabled={editing && k === 'alpha2'}
              value={form[k]}
              onChange={(e) => setForm({ ...form, [k]: e.target.value })}
              style={input}
            />
          ))}
          <select
            value={form.continentCode}
            onChange={(e) => setForm({ ...form, continentCode: e.target.value })}
            style={input}
          >
            {CONTINENTS.map((c) => <option key={c}>{c}</option>)}
          </select>
          <select
            value={form.status}
            onChange={(e) => setForm({ ...form, status: e.target.value })}
            style={input}
          >
            <option>INDEPENDENT</option>
            <option>DISCONTINUED</option>
          </select>
          <button style={btnPrimary} onClick={save}>{g.save}</button>
          <button onClick={() => setForm(null)}>{g.cancel}</button>
        </div>
      )}
      <table style={{ width: '100%', background: '#fff', borderCollapse: 'collapse', fontSize: 13 }}>
        <thead>
          <tr>{g.countryColumns.map((c) => <th key={c} style={th}>{c}</th>)}</tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.alpha2}>
              <td style={td}>{r.alpha2}</td>
              <td style={td}>{r.alpha3}</td>
              <td style={td}>{r.numericCode}</td>
              <td style={td}>{r.displayName}</td>
              <td style={td}>{r.continentCode}</td>
              <td style={td}>{r.isActive ? g.active : g.inactive}</td>
              <td style={td}>
                <a style={link} onClick={() => { setForm({ ...r }); setEditing(true) }}>{g.edit}</a>
                <a style={link} onClick={() => toggle(r)}>{r.isActive ? g.disable : g.enable}</a>
                <a style={link} onClick={() => openDetail(r.alpha2)}>{g.detail}</a>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <p style={{ color: '#888' }}>{g.total.replace('{count}', String(rows.length))}</p>
      {detail && <CountryDetailCard detail={detail} onSaved={() => openDetail(detail.alpha2)} />}
    </div>
  )
}

// CountryDetailCard 译名与关联属性维护(整体替换式)。
function CountryDetailCard({ detail, onSaved }: { detail: CountryDetail; onSaved: () => void }) {
  const t = useT()
  const g = t.pages.geo
  const [locale, setLocale] = useState('zh-Hans')
  const [name, setName] = useState('')
  const [tz, setTz] = useState(detail.attrs.timeZones.join(','))
  const [cc, setCc] = useState(detail.attrs.callingCodes.join(','))
  const [cur, setCur] = useState(
    detail.attrs.currencies.map((c) => `${c.currency}:${c.minorUnit}:${c.isPrimary}`).join(','),
  )

  const addName = async () => {
    if (!name) return
    await apiFetch(`/geo/countries/${detail.alpha2}/names`, {
      method: 'POST', body: { locale, name, nameType: 'STANDARD' },
    }).catch(() => undefined)
    setName('')
    onSaved()
  }

  const removeName = async (loc: string, nameType: string) => {
    await apiFetch(`/geo/countries/${detail.alpha2}/names/${loc}/${nameType}`, { method: 'DELETE' })
    onSaved()
  }

  const saveAttrs = async () => {
    const currencies = cur.split(',').filter(Boolean).map((s) => {
      const [currency, minorUnit, primary] = s.split(':')
      return {
        currency: currency.trim(),
        minorUnit: Number(minorUnit ?? 2) || 0,
        isPrimary: (primary ?? 'true').trim() !== 'false',
      }
    })
    await apiFetch(`/geo/countries/${detail.alpha2}/attrs`, {
      method: 'PUT',
      body: {
        timeZones: tz.split(',').map((s) => s.trim()).filter(Boolean),
        callingCodes: cc.split(',').map((s) => s.trim()).filter(Boolean),
        currencies,
      },
    }).catch(() => undefined)
    onSaved()
  }

  return (
    <div style={{ ...formBox, flexDirection: 'column', alignItems: 'flex-start', gap: 6 }}>
      <b>{detail.alpha2} {g.names}</b>
      {detail.names.map((n) => (
        <div key={n.locale + n.nameType}>
          {n.locale} / {n.nameType} / {n.name}{' '}
          <a style={link} onClick={() => removeName(n.locale, n.nameType)}>x</a>
        </div>
      ))}
      <div style={{ display: 'flex', gap: 6 }}>
        <input style={input} value={locale} onChange={(e) => setLocale(e.target.value)} />
        <input style={input} value={name} onChange={(e) => setName(e.target.value)} />
        <button style={btnPrimary} onClick={addName}>{g.addName}</button>
      </div>
      <b>{g.attrs}</b>
      <div style={{ display: 'flex', gap: 6, width: '100%' }}>
        <input style={{ ...input, flex: 1 }} placeholder={g.timeZones} value={tz} onChange={(e) => setTz(e.target.value)} />
        <input style={{ ...input, flex: 1 }} placeholder={g.callingCodes} value={cc} onChange={(e) => setCc(e.target.value)} />
      </div>
      <input style={{ ...input, width: '100%' }} placeholder={g.currencies} value={cur} onChange={(e) => setCur(e.target.value)} />
      <button style={btnPrimary} onClick={saveAttrs}>{g.save}</button>
    </div>
  )
}

