// 客户成功实践区块:3 卡片,左侧圆形图标 + 标题/描述,右侧绿色指标面板。
import { MaskIcon } from '../../layouts/icons'
import { SectionHead } from './Features'

export interface CasesProps {
  cases: Array<{ icon: string; title: string; desc: string; metric: string }>
  title: string
  subtitle: string
}

const CARD = 'flex flex-col gap-4 rounded-xl border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-7 shadow-[var(--shell-card-shadow)] transition-all hover:-translate-y-1 hover:border-[var(--home-hero-badge-border)] hover:shadow-lg'
const ICON_CHIP = 'flex h-12 w-12 flex-none items-center justify-center rounded-full bg-[var(--home-icon-bg)] text-[var(--home-icon-fg)]'

export function Cases({ cases, title, subtitle }: CasesProps) {
  return (
    <section id="solutions" className="bg-gradient-to-b from-transparent to-[var(--home-section-alt)]">
      <div className="mx-auto max-w-6xl scroll-mt-20 px-4 py-16">
        <SectionHead title={title} subtitle={subtitle} />
        <div className="mt-12 grid gap-6 lg:grid-cols-3">
          {cases.map((c) => (
            <div key={c.icon} className={CARD}>
              <div className="flex items-center gap-4">
                <span className={ICON_CHIP}>
                  <MaskIcon url={`/icons/${c.icon}.svg`} size={20} />
                </span>
                <h3 className="font-brand min-w-0 flex-1 text-base font-semibold tracking-tight text-[var(--shell-heading)]">{c.title}</h3>
                <span className="flex flex-none items-center justify-center rounded-xl bg-[var(--home-case-bg)] px-3.5 py-2.5">
                  <span className="font-brand text-2xl font-bold leading-none text-[var(--home-case-fg)]">{c.metric}</span>
                </span>
              </div>
              <p className="text-sm leading-6 text-[var(--shell-content-text)]">{c.desc}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
