// 挂接抽屉:国家下拉 + 同国家一级行政区下拉,保存调 PUT /addresses/:id/geo。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import type { CountryRow } from '../geo/CountryForm'
import type { SubdivRow } from '../geo/SubdivisionPanel'
import '../geo/geo.css'

export interface AddressRow {
  id: number
  parentId?: number | null
  path?: string
  level: number
  name: string
  countryCode: string
  adminCode: string
  hasChildren?: boolean
}

export type { CountryRow }

export function AddressGeoDrawer({ row, onDone, onCancel }: {
  row: AddressRow
  onDone: () => void
  onCancel: () => void
}) {
  const t = useT()
  const a = t.pages.address
  const g = t.pages.geo
  const [countries, setCountries] = useState<CountryRow[]>([])
  const [subdivs, setSubdivs] = useState<SubdivRow[]>([])
  const [country, setCountry] = useState(row.countryCode)
  const [admin, setAdmin] = useState(row.adminCode)
  const [error, setError] = useState('')

  useEffect(() => {
    apiFetch<CountryRow[]>('/geo/countries')
      .then((d) => setCountries(d ?? []))
      .catch(() => setError(g.loadFail))
  }, [g])

  useEffect(() => {
    if (!country) { setSubdivs([]); return }
    apiFetch<SubdivRow[]>(`/geo/subdivisions?country=${country}`)
      .then((d) => setSubdivs((d ?? []).filter((s) => s.level === 1 && s.isActive)))
      .catch(() => setSubdivs([]))
  }, [country])

  const save = async () => {
    try {
      await apiFetch(`/addresses/${row.id}/geo`, {
        method: 'PUT', body: { countryCode: country, adminCode: admin },
      })
      onDone()
    } catch {
      setError(g.saveFail)
    }
  }

  return (
    <Drawer title={`${a.attach} · ${row.name}`} onClose={onCancel}
      footer={
        <>
          {error && <span className="geo-error" style={{ margin: 0, marginRight: 'auto' }}>{error}</span>}
          <button className="geo-btn" onClick={onCancel}>{g.cancel}</button>
          <button className="geo-btn geo-btn-primary" disabled={!country} onClick={save}>{g.save}</button>
        </>
      }>
      <div className="geo-form">
        <div className="geo-field full">
          <label><span className="req">*</span>{a.country}</label>
          <Dropdown value={country} ariaLabel={a.country} onChange={setCountry}
            triggerStyle={{ width: '100%' }}
            options={[
              { value: '', label: g.filterCountry },
              ...countries.map((c) => ({ value: c.alpha2, label: `${c.alpha2} ${c.displayName}` })),
            ]} />
        </div>
        <div className="geo-field full">
          <label>{a.adminCode}</label>
          <Dropdown value={admin} ariaLabel={a.adminCode}
            onChange={setAdmin}
            triggerStyle={{ width: '100%' }}
            options={[
              { value: '', label: '—' },
              ...subdivs.map((s) => ({ value: s.code, label: `${s.code} ${s.displayName}` })),
            ]} />
        </div>
      </div>
    </Drawer>
  )
}
