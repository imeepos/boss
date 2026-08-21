// 列表/网格视图切换按钮组(高亮当前视图),纯展示组件。
import { LayoutGrid, List as ListIcon } from 'lucide-react'

interface ViewToggleProps {
  view: 'list' | 'grid'
  onChange: (view: 'list' | 'grid') => void
  listLabel: string
}

const BASE = 'inline-flex h-full w-8 items-center justify-center transition-colors text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]'
const ACTIVE = 'bg-[var(--shell-fab-bg)] text-[var(--shell-fab-icon)]'

export function ViewToggle({ view, onChange, listLabel }: ViewToggleProps) {
  return (
    <div className="inline-flex h-8 overflow-hidden rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)]">
      <button
        type="button"
        aria-label={listLabel + ' ' + view}
        className={`${BASE} ${view === 'list' ? ACTIVE : ''}`}
        onClick={() => onChange('list')}
      >
        <ListIcon className="h-4 w-4" />
      </button>
      <button
        type="button"
        aria-label="grid"
        className={`${BASE} border-l border-[var(--shell-input-border)] ${view === 'grid' ? ACTIVE : ''}`}
        onClick={() => onChange('grid')}
      >
        <LayoutGrid className="h-4 w-4" />
      </button>
    </div>
  )
}
