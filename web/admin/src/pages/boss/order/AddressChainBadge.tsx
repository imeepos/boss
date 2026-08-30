// 待治理黄标:与地址管理页治理队列同口径的视觉标记;警示不阻断。
// 接入 ui Badge warning 变体(治理金色调),覆盖内边距保持行内微标记形态。
import { Badge } from '../../../components/ui/badge'

export function ReviewBadge({ text }: { text: string }) {
  return (
    <Badge variant="warning" className="shrink-0 rounded-sm px-1 py-px text-[10px] leading-none">
      {text}
    </Badge>
  )
}
