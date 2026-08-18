// Drawer:右侧抽屉(antd 布局惯例),主题经 shell 令牌自适应;ESC/遮罩可关闭。
import { useEffect, type ReactNode } from 'react'
import './Drawer.css'

export function Drawer({
  title, onClose, footer, children,
}: {
  title: string
  onClose: () => void
  footer?: ReactNode
  children: ReactNode
}) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  return (
    <>
      <div className="dvr-overlay" onClick={onClose} />
      <aside className="dvr-panel" role="dialog" aria-label={title}>
        <header className="dvr-head">
          <h3 className="dvr-title">{title}</h3>
          <button className="dvr-close" onClick={onClose} aria-label="close">×</button>
        </header>
        <div className="dvr-body">{children}</div>
        {footer && <footer className="dvr-foot">{footer}</footer>}
      </aside>
    </>
  )
}
