// 页面级批量导入入口:按钮 + Drawer + 业务实体导入面板。
// 只接收实体 kind,权限(useProfile)与文案(useT)内部装配;导入完成后回调 onImported。
import { useState } from 'react'
import { useT } from '../../../i18n'
import { useProfile } from '../../../layouts/profile'
import { Drawer } from '../../../components/Drawer'
import { ToolbarButton } from '../../../components/business/page-head'
import { EntityImportPanel } from './EntityImportPanel'
import { findEntity } from './entities'

export function BatchImportEntry({ kind, onImported, label }: {
  kind: string
  /** 按钮文案;缺省为「导入 + 实体名」。 */
  label?: string
  /** 导入完成(成功或失败行>0)后回调,供宿主刷新列表。 */
  onImported: () => void
}) {
  const t = useT()
  const im = t.pages.importer
  const profile = useProfile()
  const [open, setOpen] = useState(false)
  const def = findEntity(kind)
  if (!def) return null
  const name = im.entityNames[def.kind] ?? def.kind
  const noPerm = !(profile.permissionCodes ?? []).includes(def.perm)

  return (
    <>
      <ToolbarButton primary disabled={noPerm} onClick={() => setOpen(true)}>
        {label ?? `${im.importBtn} ${name}`}
      </ToolbarButton>
      {open && (
        <Drawer title={`${im.entityTitle} · ${name}`} onClose={() => setOpen(false)}>
          <EntityImportPanel def={def} noPerm={noPerm} text={im} onImported={onImported} />
        </Drawer>
      )}
    </>
  )
}
