// 节点抽屉:新建根/子级(label+名称,根可带锚点)与重命名。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import type { CountryRow } from '../geo/CountryForm'
import type { SubdivRow } from '../geo/SubdivisionPanel'
import type { AddressRow } from './AddressGeoDrawer'
import '../geo/geo.css'

export function AddressNodeDrawer({ mode, parent, row, onDone, onCancel }: {
  mode: 'create' | 'rename'
  parent?: AddressRow
  row?: AddressRow
  onDone: () => void
  onCancel: () => void
}) {
  const t = useT()
  const a = t.pages.address
  const g = t.pages.geo
  const isRoot = mode === 'create' && !parent
  const [label, setLabel] = useState('')
  const [name, setName] = useState(mode === 'rename' ? row?.name ?? '' : '')
  const [country, setCountry] = useState('')
  const [admin, setAdmin] = useState('')
  const [countries, setCountries] = useState<CountryRow[]>([])
  const [subdivs, setSubdivs] = useState<SubdivRow[]>([])
  const [error, setError] = useState('')

  useEffect(() => {
    if (!isRoot) return
    apiFetch<CountryRow[]>('/geo/countries')
      .then((d) => setCountries(d ?? []))
      .catch(() => setError(g.loadFail))
  }, [isRoot, g])

  useEffect(() => {
    if (!country) { setSubdivs([]); return }
    apiFetch<SubdivRow[]>(`/geo/subdivisions?country=${country}`)
      .then((d) => setSubdivs((d ?? []).filter((s) => s.level === 1 && s.isActive)))
      .catch(() => setSubdivs([]))
  }, [country])

  const save = async () => {
    try {
      if (mode === 'rename') {
        await apiFetch(`/addresses/${row!.id}`, { method: 'PUT', body: { name } })
      } else {
        await apiFetch('/addresses', {
          method: 'POST',
          body: { parentId: parent?.id ?? 0, label, name, countryCode: country, adminCode: admin },
        })
      }
      onDone()
    } catch {
      setError(g.saveFail)
    }
  }

  const valid = mode === 'rename' ? name.trim() !== '' : label.trim() !== '' && name.trim() !== ''

  return (
    <Drawer title={`${mode === 'create'
      ? (parent ? a.addChild : a.addRoot) : a.rename}${parent ? ` · ${parent.name}` : ''}`}
      onClose={onCancel}
      footer={
        <>
          {error && <span className="geo-error" style={{ margin: 0, marginRight: 'auto' }}>{error}</span>}
          <button className="geo-btn" onClick={onCancel}>{g.cancel}</button>
          <button className="geo-btn geo-btn-primary" disabled={!valid} onClick={save}>{g.save}</button>
        </>
      }>
      <div className="geo-form">
        {mode === 'create' && (
          <div className="geo-field full">
            <label><span className="req">*</span>{a.pathLabel}</label>
            <input className="geo-input" value={label} placeholder="如 nanshan"
              onChange={(e) => setLabel(e.target.value.toLowerCase())} />
            <span className="hint">{a.pathHint}{parent ? ` · ${parent.name}` : ''}</span>
          </div>
        )}
        <div className="geo-field full">
          <label><span className="req">*</span>{a.nameLabel}</label>
          <input className="geo-input" value={name}
            onChange={(e) => setName(e.target.value)} />
        </div>
        {isRoot && (
          <>
            <div className="geo-field full">
              <label>{a.country}</label>
              <Dropdown value={country} ariaLabel={a.country} onChange={setCountry}
                triggerStyle={{ width: '100%' }}
                options={[
                  { value: '', label: g.filterCountry },
                  ...countries.map((c) => ({ value: c.alpha2, label: `${c.alpha2} ${c.displayName}` })),
                ]} />
            </div>
            <div className="geo-field full">
              <label>{a.adminCode}</label>
              <Dropdown value={admin} ariaLabel={a.adminCode} onChange={setAdmin}
                triggerStyle={{ width: '100%' }}
                options={[
                  { value: '', label: '—' },
                  ...subdivs.map((s) => ({ value: s.code, label: `${s.code} ${s.displayName}` })),
                ]} />
            </div>
          </>
        )}
      </div>
    </Drawer>
  )
}
