// 左侧文件分类侧栏:全部分类 + 9 类文件徽章入口,显示各类数量。
// 纯展示 + onSelect 回调,选中/翻页状态由父组件持有。
import { LayoutGrid } from 'lucide-react'
import type { useT } from '../../i18n'
import { FILE_CATEGORY_KEYS, countByCategory, type FileCategoryKey } from './logic'
import { FileTypeIcon } from './FileTypeIcon'

interface SidebarProps {
  items: { contentType: string; fileName: string }[]
  total: number
  selected: FileCategoryKey | ''
  onSelect: (category: FileCategoryKey | '') => void
  t: ReturnType<typeof useT>['attachmentManager']
}

const ACTIVE = 'border-l-[3px] border-[var(--shell-nav-line)] bg-[var(--shell-menu-active-bg)] font-semibold text-[var(--shell-menu-active-text)]'
const IDLE = 'text-[var(--shell-menu-text)] hover:bg-[var(--shell-menu-hover-bg)]'

function CountBadge({ n, active }: { n: number; active: boolean }) {
  return (
    <span className={`min-w-6 rounded-sm px-1.5 py-0.5 text-center text-[11px] tabular-nums ${active ? 'bg-[var(--shell-fab-bg)] text-[var(--shell-fab-icon)]' : 'bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]'}`}>{n}</span>
  )
}

export function CategorySidebar({ items, total, selected, onSelect, t }: SidebarProps) {
  const counts = countByCategory(items)
  const allActive = selected === ''
  return (
    <aside className="hidden w-56 shrink-0 border-r border-[var(--shell-side-border)] bg-[var(--shell-side-bg)] py-2 lg:block">
      <button
        type="button"
        className={`flex w-full items-center gap-2 px-4 py-2 text-left text-[13px] transition-colors ${allActive ? ACTIVE : IDLE}`}
        onClick={() => onSelect('')}
      >
        <span className={`afti-image inline-flex h-5 w-5 shrink-0 items-center justify-center rounded-sm ${allActive ? 'bg-[var(--shell-fab-bg)] text-[var(--shell-fab-icon)]' : ''}`}>
          <LayoutGrid className="h-3.5 w-3.5" />
        </span>
        <span className="flex-1">{t.fileCategory.all}</span>
        <CountBadge n={total} active={allActive} />
      </button>
      {FILE_CATEGORY_KEYS.map((k) => {
        const active = selected === k
        return (
          <button
            key={k}
            type="button"
            title={`${t.fileCategory[k]} · ${t.categoryBadgeTip}`}
            className={`flex w-full items-center gap-2 px-4 py-2 text-left text-[13px] transition-colors ${active ? ACTIVE : IDLE}`}
            onClick={() => onSelect(active ? '' : k)}
          >
            <FileTypeIcon category={k} size={20} />
            <span className="flex-1">{t.fileCategory[k]}</span>
            <CountBadge n={counts[k] ?? 0} active={active} />
          </button>
        )
      })}
    </aside>
  )
}
