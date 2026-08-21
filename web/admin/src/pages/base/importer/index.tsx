// 数据导入中心:地址层级 + ISO 地理数据导入面板 + 导入任务历史(menu:importer;geo 面板另需 menu:geo)。
import { useState } from 'react'
import { useT } from '../../../i18n'
import { ImportPanel } from './ImportPanel'
import { ImportTaskList } from './TaskList'
import { CARD } from '../geo/styles'

export default function ImporterPage() {
  const t = useT()
  const im = t.pages.importer
  const [taskRev, setTaskRev] = useState(0)
  return (
    <div className={CARD}>
      <h2 className="mx-4 mt-4 mb-3 text-base text-[var(--shell-content-text)]">{im.title}</h2>
      {/* 双面板并排各占一半;窄屏(md 以下)回落单列堆叠。 */}
      <div className="mx-4 mb-2 grid grid-cols-1 items-start gap-x-6 md:grid-cols-2">
        <ImportPanel kind="addr" title={im.addrTitle} hint={im.addrHint} endpoint="/addresses/import" text={im} onImported={() => setTaskRev((v) => v + 1)} />
        <ImportPanel kind="geo" title={im.geoTitle} hint={im.geoHint} endpoint="/geo/import" text={im} onImported={() => setTaskRev((v) => v + 1)} />
      </div>
      <ImportTaskList refreshKey={taskRev} />
    </div>
  )
}
