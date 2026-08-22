// 容器卡:全站统计卡/图表区块通用外壳(对齐 dashboard 既有 rounded-md border shadow)。
import type { ReactNode } from 'react'

export function CardShell({
  title: titleStr, className = '', children,
}: {
  title?: string
  className?: string
  children: ReactNode
}) {
  return (
    <section
      className={`rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-5 shadow-[var(--shell-card-shadow)] ${className}`}
    >
      {titleStr && <h3 className="mb-4 text-base font-semibold text-[var(--shell-heading)]">{titleStr}</h3>}
      {children}
    </section>
  )
}