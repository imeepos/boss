// 坐标抽屉:地图选点(MapLocationPicker,WGS84)+ 保存走既有 PUT /addresses/:id/geom。
// 后端列表响应不含坐标,抽屉从无值起步;选点交互全部由 MapLocationPicker 承担。
import { useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { ToolbarButton } from '../../../components/business/page-head'
import { MapLocationPicker, type LatLng } from '../../../components/business/maps'
import { FORM, FIELD_FULL, LABEL, HINT } from '../geo/styles'
import type { AddressRow } from './AddressGeoDrawer'

export function AddressGeomDrawer({ row, onDone, onCancel }: {
  row: AddressRow
  onDone: () => void
  onCancel: () => void
}) {
  const t = useT()
  const a = t.pages.address
  const g = t.pages.geo
  const [value, setValue] = useState<LatLng | null>(null)
  const [error, setError] = useState('')

  const save = async () => {
    if (!value) return
    try {
      await apiFetch('/addresses/' + row.id + '/geom', {
        method: 'PUT', body: { lat: value.lat, lng: value.lng },
      })
      onDone()
    } catch (e) {
      setError(e instanceof Error ? e.message : g.saveFail)
    }
  }

  return (
    <Drawer title={a.setCoord + ' · ' + row.name} onClose={onCancel}
      footer={
        <>
          {error && <span className="mr-auto text-xs text-[var(--color-danger)]">{error}</span>}
          <ToolbarButton onClick={onCancel}>{g.cancel}</ToolbarButton>
          <ToolbarButton primary disabled={!value} onClick={save}>{g.save}</ToolbarButton>
        </>
      }>
      <div className={FORM}>
        <div className={FIELD_FULL}>
          <label className={LABEL}>{a.coordLabel}</label>
          <MapLocationPicker value={value} onChange={setValue}
            height={320} clearLabel={a.coordClear} hint={a.coordHint} placeholder={a.coordEmpty} />
          <span className={HINT}>{a.coordSaveHint}</span>
        </div>
      </div>
    </Drawer>
  )
}
