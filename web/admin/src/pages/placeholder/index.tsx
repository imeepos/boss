// A1 前占位页:菜单全集路由可直达,A1 批次逐页替换为正式实现,文本走 i18n。
import { useT } from '../../i18n'

export function PlaceholderPage({ title }: { title: string }) {
  const t = useT()
  return (
    <div>
      <h2>{title}</h2>
      <p className="text-[var(--color-text-tertiary)]">{t.pages.placeholder.building}</p>
    </div>
  )
}