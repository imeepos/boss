// 节点抽屉:新建根/子级(label+名称,根可带锚点)与重命名。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { ToolbarButton } from '../../../components/business/page-head'
import { Input } from '../../../components/ui/input'
import type { CountryRow } from '../geo/CountryForm'
import type { SubdivRow } from '../geo/subdiv-shared'
import type { AddressRow } from './AddressGeoDrawer'
import { FORM, FIELD_FULL, LABEL, REQ, HINT } from '../geo/styles'

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
          {error && <span className="mr-auto text-xs text-[var(--color-danger)]">{error}</span>}
          <ToolbarButton onClick={onCancel}>{g.cancel}</ToolbarButton>
          <ToolbarButton primary disabled={!valid} onClick={save}>{g.save}</ToolbarButton>
        </>
      }>
      <div className={FORM}>
        {mode === 'create' && (
          <div className={FIELD_FULL}>
            <label className={LABEL}><span className={REQ}>*</span>{a.pathLabel}</label>
            <Input value={label} placeholder="如 nanshan"
              onChange={(e) => setLabel(e.target.value.toLowerCase())} />
            <span className={HINT}>{a.pathHint}{parent ? ` · ${parent.name}` : ''}</span>
          </div>
        )}
        <div className={FIELD_FULL}>
          <label className={LABEL}><span className={REQ}>*</span>{a.nameLabel}</label>
          <Input value={name}
            onChange={(e) => setName(e.target.value)} />
        </div>
        {isRoot && (
          <>
            <div className={FIELD_FULL}>
              <label className={LABEL}>{a.country}</label>
              <Dropdown value={country} ariaLabel={a.country} onChange={setCountry}
                triggerStyle={{ width: '100%' }}
                options={[
                  { value: '', label: g.filterCountry },
                  ...countries.map((c) => ({ value: c.alpha2, label: `${c.alpha2} ${c.displayName}` })),
                ]} />
            </div>
            <div className={FIELD_FULL}>
              <label className={LABEL}>{a.adminCode}</label>
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
