// 许可单详情抽屉外壳:既有 PermitDetail 原样入壳(容器层重构,不改其内部实现)。
// 标题口径同模块既有详情抽屉(单号+名称,SurveysPanel 先例);宽度用 Drawer 默认 520。
import { Drawer } from '../../../components/Drawer'
import { PermitDetail } from './PermitDetail'

export function PermitDetailDrawer({ permitId, title, onClose, onChanged }: {
  permitId: number
  title: string
  onClose: () => void
  onChanged: () => void
}) {
  return <Drawer title={title} onClose={onClose}>
    <PermitDetail permitId={permitId} onChanged={onChanged} />
  </Drawer>
}
