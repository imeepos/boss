// DetailDrawer: read-only k-v list drawer, replaces DetailDrawer in org/shared
import { Drawer } from '../Drawer'
import type { DetailItem } from '../../pages/org/shared'

export interface DetailDrawerProps {
  title: string
  items: DetailItem[]
  onClose: () => void
  closeText: string
}

export function DetailDrawer({ title, items, onClose, closeText }: DetailDrawerProps) {
  return (
    <Drawer
      title={title}
      onClose={onClose}
      footer={
        <button className="h-8 px-4 text-xs rounded-sm cursor-pointer border-none text-[var(--shell-fab-icon)] bg-[var(--shell-fab-bg)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={onClose}>
          {closeText}
        </button>
      }
    >
      <div className="flex flex-col gap-2.5">
        {items.map((it) => (
          <div key={it.k} className="flex gap-3 text-xs">
            <span className="w-24 flex-none text-[var(--shell-group-title)]">{it.k}</span>
            <span className="text-[var(--shell-content-text)] break-all">{it.v || '—'}</span>
          </div>
        ))}
      </div>
    </Drawer>
  )
}