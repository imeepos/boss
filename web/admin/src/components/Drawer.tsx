// Drawer:右侧抽屉(antd 布局惯例),主题经 shell 令牌自适应;ESC/遮罩可关闭。
// 样式:tailwind 原子类(原 Drawer.css 已删除),动画走 tailwindcss-animate。
// 层级阶梯:本组件遮罩 100/面板 101 < 页面临时遮罩 120 < 共享 ui/dialog 130 < Dropdown 等弹层 1000。
import { useEffect, type ReactNode } from 'react'

export function Drawer({
  title, onClose, footer, children, width = 520,
}: {
  title: string
  onClose: () => void
  footer?: ReactNode
  /** 抽屉宽度(px),受 92vw 上限约束;默认 520。 */
  width?: number
  children: ReactNode
}) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  return (
    <>
      <div className="fixed inset-0 z-[100] animate-in fade-in duration-200 bg-[rgba(15,30,59,0.45)]" onClick={onClose} />
      <aside className="fixed right-0 top-0 bottom-0 z-[101] flex animate-in slide-in-from-right duration-200 flex-col border-l border-[var(--shell-side-border)] bg-[var(--shell-card-bg)] shadow-[-8px_0_24px_rgba(3,13,31,0.18)]" style={{ width: `min(${width}px, 92vw)` }} role="dialog" aria-label={title}>
        <header className="flex items-center justify-between border-b border-[var(--shell-side-border)] px-5 py-4">
          <h3 className="m-0 text-base font-semibold text-[var(--shell-heading)]">{title}</h3>
          <button className="h-7 w-7 cursor-pointer rounded-sm border-none bg-none text-base leading-none text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)] hover:text-[var(--shell-heading)]" onClick={onClose} aria-label="close">×</button>
        </header>
        <div className="flex-1 overflow-y-auto p-5">{children}</div>
        {footer && <footer className="flex justify-end gap-2 border-t border-[var(--shell-side-border)] px-5 py-3.5">{footer}</footer>}
      </aside>
    </>
  )
}
