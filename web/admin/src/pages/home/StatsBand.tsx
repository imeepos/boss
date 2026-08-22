// 关键数据条:圆角描边横条,三格均分,大金色数字 + 标签,紧随 Hero 之后。
export interface StatsBandProps {
  stats: Array<{ value: string; label: string }>
}

export function StatsBand({ stats }: StatsBandProps) {
  return (
    <section className="mx-auto max-w-6xl px-4 py-14">
      <div className="grid grid-cols-1 divide-y divide-[var(--shell-card-border)] rounded-2xl border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)] sm:grid-cols-3 sm:divide-x sm:divide-y-0">
        {stats.map((s) => (
          <div key={s.label} className="flex items-center justify-center gap-4 px-6 py-8">
            <span className="font-brand text-5xl font-bold leading-none tracking-tight text-[var(--color-brand-gold-500)]">
              {s.value}
            </span>
            <span className="text-sm text-[var(--shell-content-text)]">{s.label}</span>
          </div>
        ))}
      </div>
    </section>
  )
}
