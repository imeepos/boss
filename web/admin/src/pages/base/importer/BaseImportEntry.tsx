// 页面级基础数据导入入口:按钮 + Drawer + 基础导入面板(addr/geo)。
// 只接收 kind/title/hint/endpoint,文案(useT)与权限由宿主页面装配;导入完成后回调 onImported。
import { useState } from 'react'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { ToolbarButton } from '../../../components/business/page-head'
import { ImportPanel } from './ImportPanel'
import type { ImportKind } from './preview'

export function BaseImportEntry({ kind, title, hint, endpoint, onImported, label }: {
  kind: ImportKind
  title: string
  hint: string
  endpoint: string
  /** 按钮文案;缺省为「导入」。 */
  label?: string
  /** 导入完成后回调,供宿主刷新树/面板。 */
  onImported: () => void
}) {
  const t = useT()
  const im = t.pages.importer
  const [open, setOpen] = useState(false)

  return (
    <>
      <ToolbarButton primary onClick={() => setOpen(true)}>
        {label ?? im.importBtn}
      </ToolbarButton>
      {open && (
        <Drawer title={title} onClose={() => setOpen(false)}>
          <ImportPanel kind={kind} title={title} hint={hint} endpoint={endpoint} text={im} onImported={onImported} />
        </Drawer>
      )}
    </>
  )
}
