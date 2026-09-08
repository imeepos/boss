// 挂接抽屉:国家 + 行政区划级联选择(RegionCascadePicker),保存调 PUT /addresses/:id/geo。
// 旧实现为国家下拉 + 一级行政区下拉(仅 level=1);新组件支持多级下钻与直搜,编辑态按已存 code 回显展开。
import { useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { ToolbarButton } from '../../../components/business/page-head'
import { RegionCascadePicker, type RegionSelection } from '../../../components/pickers/RegionCascadePicker'
import { FORM, FIELD_FULL, LABEL, REQ, HINT } from '../geo/styles'

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

export type { CountryRow } from '../geo/CountryForm'

export function AddressGeoDrawer({ row, onDone, onCancel }: {
  row: AddressRow
  onDone: () => void
  onCancel: () => void
}) {
  const t = useT()
  const a = t.pages.address
  const g = t.pages.geo
  const [sel, setSel] = useState<RegionSelection | null>(null)
  const [error, setError] = useState('')
  // 未交互时沿用原挂接值；交互后以选择器回传为准（切换国家会回传 code:'' 清空区划）。
  const countryCode = sel?.countryCode || row.countryCode
  const adminCode = sel ? sel.code : row.adminCode

  const save = async () => {
    try {
      await apiFetch(`/addresses/${row.id}/geo`, {
        method: 'PUT', body: { countryCode, adminCode },
      })
      onDone()
    } catch (err) {
      console.warn('[address-drawer] attach geo save failed:', err)
      setError(err instanceof Error ? err.message : g.saveFail)
    }
  }

  return (
    <Drawer title={`${a.attach} · ${row.name}`} onClose={onCancel}
      footer={
        <>
          {error && <span className="mr-auto text-xs text-[var(--color-danger)]">{error}</span>}
          <ToolbarButton onClick={onCancel}>{g.cancel}</ToolbarButton>
          <ToolbarButton primary disabled={!countryCode} onClick={save}>{g.save}</ToolbarButton>
        </>
      }>
      <div className={FORM}>
        <div className={FIELD_FULL}>
          <label className={LABEL}><span className={REQ}>*</span>{a.country}</label>
          <RegionCascadePicker value={row.adminCode || undefined}
            countryCode={row.countryCode || undefined}
            onChange={setSel} />
          {sel?.code && <span className={HINT}>{sel.names.join(' / ')}</span>}
        </div>
      </div>
    </Drawer>
  )
}
