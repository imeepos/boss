// CountryDetail:国家详情抽屉,antd Descriptions 惯例(键值分区:译名列表 + 关联属性编辑)。
import { useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import type { CountryRow } from './CountryForm'
import './geo.css'

export interface CountryDetailData extends CountryRow {
  names: { locale: string; name: string; nameType: string }[]
  attrs: {
    timeZones: string[]
    currencies: { currency: string; isPrimary: boolean; minorUnit: number }[]
    callingCodes: string[]
  }
}

export function CountryDetail({ data, onChanged, onClose }: {
  data: CountryDetailData
  onChanged: () => void
  onClose: () => void
}) {
  const t = useT()
  const g = t.pages.geo
  const [locale, setLocale] = useState('zh-Hans')
  const [name, setName] = useState('')
  const [tz, setTz] = useState(data.attrs.timeZones.join(', '))
  const [cc, setCc] = useState(data.attrs.callingCodes.join(', '))
  const [cur, setCur] = useState(
    data.attrs.currencies.map((c) => `${c.currency}:${c.minorUnit}:${c.isPrimary}`).join(', '))

  const addName = async () => {
    if (!name.trim()) return
    await apiFetch(`/geo/countries/${data.alpha2}/names`, {
      method: 'POST', body: { locale, name, nameType: 'STANDARD' },
    }).catch(() => undefined)
    setName('')
    onChanged()
  }

  const removeName = async (loc: string, nameType: string) => {
    await apiFetch(`/geo/countries/${data.alpha2}/names/${loc}/${nameType}`, { method: 'DELETE' })
    onChanged()
  }

  const saveAttrs = async () => {
    const currencies = cur.split(',').filter((s) => s.trim()).map((s) => {
      const [currency, minorUnit, primary] = s.split(':')
      return {
        currency: currency.trim(),
        minorUnit: Number(minorUnit ?? 2) || 0,
        isPrimary: (primary ?? 'true').trim() !== 'false',
      }
    })
    await apiFetch(`/geo/countries/${data.alpha2}/attrs`, {
      method: 'PUT',
      body: {
        timeZones: tz.split(',').map((s) => s.trim()).filter(Boolean),
        callingCodes: cc.split(',').map((s) => s.trim()).filter(Boolean),
        currencies,
      },
    }).catch(() => undefined)
    onChanged()
  }

  return (
    <Drawer title={`${g.detail} · ${data.alpha2} ${data.displayName}`} onClose={onClose}>
      <DescGrid data={data} />
      <NamesSection names={data.names} locale={locale} name={name}
        setLocale={setLocale} setName={setName} onAdd={addName} onRemove={removeName} />
      <AttrsSection tz={tz} cc={cc} cur={cur}
        setTz={setTz} setCc={setCc} setCur={setCur} onSave={saveAttrs} />
    </Drawer>
  )
}

// DescGrid 基础信息键值区。
function DescGrid({ data }: { data: CountryDetailData }) {
  const items: [string, string][] = [
    ['alpha-3', data.alpha3], ['numeric', data.numericCode],
    ['continent', data.continentCode], ['M49', data.m49Region || '—'],
    ['status', data.status], ['postal regex', data.postalRegex || '—'],
  ]
  return (
    <div className="geo-form" style={{ marginBottom: 8 }}>
      {items.map(([k, v]) => (
        <div key={k} className="geo-field">
          <label>{k}</label>
          <span style={{ fontSize: 13, color: 'var(--shell-heading)' }}>{v}</span>
        </div>
      ))}
    </div>
  )
}

// NamesSection 译名维护:标签行 + 增删。
function NamesSection(p: {
  names: { locale: string; name: string; nameType: string }[]
  locale: string; name: string
  setLocale: (v: string) => void; setName: (v: string) => void
  onAdd: () => void; onRemove: (loc: string, nameType: string) => void
}) {
  const g = useT().pages.geo
  return (
    <section style={{ marginTop: 16 }}>
      <h4 style={{ margin: '0 0 8px', fontSize: 13, color: 'var(--shell-heading)' }}>{g.names}</h4>
      {p.names.map((n) => (
        <div key={n.locale + n.nameType} className="geo-tag geo-tag-off"
          style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 6, padding: '2px 8px' }}>
          <span>{n.locale} · {n.nameType} · {n.name}</span>
          <button style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--color-danger)' }}
            onClick={() => p.onRemove(n.locale, n.nameType)}>×</button>
        </div>
      ))}
      <div style={{ display: 'flex', gap: 8 }}>
        <input className="geo-input" style={{ width: 110 }} value={p.locale}
          onChange={(e) => p.setLocale(e.target.value)} placeholder="locale" />
        <input className="geo-input" style={{ flex: 1 }} value={p.name}
          onChange={(e) => p.setName(e.target.value)} placeholder="name" />
        <button className="geo-btn" onClick={p.onAdd}>{g.addName}</button>
      </div>
    </section>
  )
}

// AttrsSection 关联属性编辑(整体替换)。
function AttrsSection(p: {
  tz: string; cc: string; cur: string
  setTz: (v: string) => void; setCc: (v: string) => void; setCur: (v: string) => void
  onSave: () => void
}) {
  const g = useT().pages.geo
  return (
    <section style={{ marginTop: 16 }}>
      <h4 style={{ margin: '0 0 8px', fontSize: 13, color: 'var(--shell-heading)' }}>{g.attrs}</h4>
      <div className="geo-form">
        <div className="geo-field full">
          <label>{g.timeZones}</label>
          <input className="geo-input" value={p.tz} onChange={(e) => p.setTz(e.target.value)} />
        </div>
        <div className="geo-field full">
          <label>{g.callingCodes}</label>
          <input className="geo-input" value={p.cc} onChange={(e) => p.setCc(e.target.value)} />
        </div>
        <div className="geo-field full">
          <label>{g.currencies}</label>
          <input className="geo-input" value={p.cur} onChange={(e) => p.setCur(e.target.value)} />
        </div>
      </div>
      <button className="geo-btn geo-btn-primary" style={{ marginTop: 12 }} onClick={p.onSave}>{g.save}</button>
    </section>
  )
}
