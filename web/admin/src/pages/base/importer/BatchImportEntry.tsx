// 页面级批量导入入口:按钮 + Drawer + 导入面板。
// 实体 kind(account/legal_entity/...)走 EntityImportPanel(逐行调用创建端点);
// addr/geo 复用 ImportPanel(服务端信封导入 /addresses/import、/geo/import)。
// 权限(useProfile)与文案(useT)内部装配;导入完成后回调 onImported。
import { useState } from 'react'
import { useT } from '../../../i18n'
import { useProfile } from '../../../layouts/profile'
import { Drawer } from '../../../components/Drawer'
import { ToolbarButton } from '../../../components/business/page-head'
import { EntityImportPanel } from './EntityImportPanel'
import { ImportPanel } from './ImportPanel'
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
  const panel = kind === 'addr' || kind === 'geo'
  if (!def && !panel) return null
  const name = def ? im.entityNames[def.kind] ?? def.kind : (kind === 'addr' ? im.entryAddr : im.entryGeo)
  const btnLabel = def ? `${im.importBtn} ${name}` : name
  const drawerTitle = def ? `${im.entityTitle} · ${name}` : (kind === 'addr' ? im.addrTitle : im.geoTitle)
  const perm = def?.perm ?? (kind === 'addr' ? 'menu:importer' : 'menu:geo')
  const noPerm = !(profile.permissionCodes ?? []).includes(perm)

  return (
    <>
      <span title={noPerm ? im.entityNoPerm.replace('{perm}', perm) : undefined}>
        <ToolbarButton primary disabled={noPerm} onClick={() => setOpen(true)}>
          {label ?? btnLabel}
        </ToolbarButton>
      </span>
      {open && (
        <Drawer title={drawerTitle} onClose={() => setOpen(false)}>
          {def ? (
            <EntityImportPanel def={def} noPerm={noPerm} text={im} onImported={onImported} />
          ) : kind === 'addr' ? (
            <ImportPanel kind="addr" title={im.addrTitle} hint={im.addrHint} endpoint="/addresses/import" noPerm={noPerm} text={im} onImported={onImported} />
          ) : (
            <ImportPanel kind="geo" title={im.geoTitle} hint={im.geoHint} endpoint="/geo/import" noPerm={noPerm} text={im} onImported={onImported} />
          )}
        </Drawer>
      )}
    </>
  )
}
