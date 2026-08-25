// 数据导入中心:地址层级 + ISO 地理数据导入面板 + 导入任务历史
// (menu:importer;geo 面板另需 menu:geo)。业务实体批量导入已下沉为页面级 BatchImportEntry,
// 各业务页自行挂载入口。
import { useState } from 'react'
import { useT } from '../../../i18n'
import { ImportPanel } from './ImportPanel'
import { ImportTaskList } from './TaskList'
import { CARD } from '../geo/styles'
import { PageHead } from '../../../components/business/page-head'

export default function ImporterPage() {
  const t = useT()
  const im = t.pages.importer
  const [taskRev, setTaskRev] = useState(0)
  const refreshTasks = () => setTaskRev((v) => v + 1)

  return (
    <div>
      <div className={CARD}>
        <div className="mx-4 mt-4 mb-3"><PageHead title={im.title} desc={im.addrHint} /></div>
        {/* 双面板并排各占一半;窄屏(md 以下)回落单列堆叠。 */}
        <div className="mx-4 mb-2 grid grid-cols-1 items-start gap-x-6 md:grid-cols-2">
          <ImportPanel kind="addr" title={im.addrTitle} hint={im.addrHint} endpoint="/addresses/import" text={im} onImported={refreshTasks} />
          <ImportPanel kind="geo" title={im.geoTitle} hint={im.geoHint} endpoint="/geo/import" text={im} onImported={refreshTasks} />
        </div>
      </div>
      <div className={`${CARD} mt-4`}>
        <ImportTaskList refreshKey={taskRev} />
      </div>
    </div>
  )
}
