// 数据导入中心:地址层级 + ISO 地理数据导入面板 + 导入任务历史
// (menu:importer;geo 面板另需 menu:geo)。业务实体批量导入已下沉为页面级 BatchImportEntry,
// 各业务页自行挂载入口。
import { useT } from '../../../i18n'
import { ImportTaskList } from './TaskList'
import { CARD } from '../geo/styles'
import { PageHead } from '../../../components/business/page-head'

export default function ImporterPage() {
  const t = useT()
  const im = t.pages.importer

  return (
    <div>
      <PageHead title={im.title} desc={im.tasksTitle} />
      <div className={CARD}>
        <ImportTaskList />
      </div>
    </div>
  )
}
