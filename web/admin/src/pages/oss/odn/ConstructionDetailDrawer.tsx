// 施工单详情抽屉外壳:既有 ConstructionDetail 原样入壳(容器层重构,不改其内部实现)。
// 标题口径同模块既有详情抽屉(单号+名称,SurveysPanel 先例);宽度用 Drawer 默认 520。
import { Drawer } from '../../../components/Drawer'
import { ConstructionDetail } from './ConstructionDetail'

export function ConstructionDetailDrawer({ projectId, title, onClose, onChanged }: {
  projectId: number
  title: string
  onClose: () => void
  onChanged: () => void
}) {
  return <Drawer title={title} onClose={onClose}>
    <ConstructionDetail projectId={projectId} onChanged={onChanged} />
  </Drawer>
}
