// 数据导入中心:地址层级 + ISO 地理数据导入面板 + 业务数据批量导入 + 导入任务历史
// (menu:importer;geo 面板另需 menu:geo,业务批量导入逐行走各域创建端点权限)。
import { useState } from 'react'
import { useT } from '../../../i18n'
import { ImportPanel } from './ImportPanel'
import { EntityImportPanel } from './EntityImportPanel'
import { ImportTaskList } from './TaskList'
import { IMPORT_ENTITIES, findEntity } from './entities'
import { Dropdown } from '../../../components/Dropdown'
import { CARD } from '../geo/styles'
import { PageHead } from '../../../components/business/page-head'

export default function ImporterPage() {
  const t = useT()
  const im = t.pages.importer
  const [taskRev, setTaskRev] = useState(0)
  const [entityKind, setEntityKind] = useState(IMPORT_ENTITIES[0].kind)
  const def = findEntity(entityKind) ?? IMPORT_ENTITIES[0]
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
        {/* 业务数据批量导入:下拉选择项目(账号/法人/部门/岗位/产品),复用附件选择器。 */}
        <div className="mx-4 mb-4 border-t border-[var(--shell-side-border)] pt-4">
          <div className="flex flex-wrap items-center gap-4">
            <label className="text-sm text-[var(--shell-content-text)]">{im.entityTitle}</label>
            <div className="w-56">
              <Dropdown
                value={def.kind}
                options={IMPORT_ENTITIES.map((e) => ({ value: e.kind, label: im.entityNames[e.kind] ?? e.kind }))}
                onChange={setEntityKind}
                ariaLabel={im.entityTitle}
              />
            </div>
          </div>
          <p className="m-0 mt-1.5 mb-1 text-xs text-[var(--shell-input-placeholder)]">{im.entityHint}</p>
          <EntityImportPanel key={def.kind} def={def} text={im} onImported={refreshTasks} />
        </div>
      </div>
      <div className={`${CARD} mt-4`}>
        <ImportTaskList refreshKey={taskRev} />
      </div>
    </div>
  )
}
