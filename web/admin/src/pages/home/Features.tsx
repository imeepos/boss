// 核心能力区块:6 卡片三列网格,金色圆形图标 + 标题 + 域标签 + 描述。
// 卡片半透表面 + 细描边,hover 浮起并描边转金色。
import { MaskIcon } from '../../layouts/icons'

export interface FeaturesProps {
  features: Array<{ icon: string; title: string; tag: string; desc: string }>
  title: string
  subtitle: string
}

const CARD = 'rounded-xl border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-7 shadow-[var(--shell-card-shadow)] transition-all hover:-translate-y-1 hover:border-[var(--home-hero-badge-border)] hover:shadow-lg'
const ICON_CHIP = 'flex h-11 w-11 flex-none items-center justify-center rounded-full bg-[var(--home-gold-bg)] text-[var(--home-gold-fg)]'
const TAG = 'inline-flex items-center rounded-full bg-[var(--home-icon-bg)] px-2 py-0.5 text-[10px] font-medium text-[var(--home-icon-fg)]'

export function Features({ features, title, subtitle }: FeaturesProps) {
  return (
    <section id="features" className="mx-auto max-w-6xl scroll-mt-20 px-4 py-16">
      <SectionHead title={title} subtitle={subtitle} />
      <div className="mt-12 grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
        {features.map((f) => (
          <div key={f.icon} className={CARD}>
            <div className="flex items-start gap-4">
              <span className={ICON_CHIP}>
                <MaskIcon url={`/icons/${f.icon}.svg`} size={20} />
              </span>
              <div className="min-w-0 flex-1">
                <h3 className="font-brand text-lg font-semibold tracking-tight text-[var(--shell-heading)]">
                  {f.title}
                </h3>
                <span className={`mt-2 ${TAG}`}>{f.tag}</span>
              </div>
            </div>
            <p className="mt-4 text-sm leading-6 text-[var(--shell-content-text)]">{f.desc}</p>
          </div>
        ))}
      </div>
    </section>
  )
}

export function SectionHead({ title, subtitle }: { title: string; subtitle: string }) {
  return (
    <div className="text-center">
      <div className="flex items-center justify-center gap-3">
        <span className="h-px w-10 bg-gradient-to-r from-transparent to-[var(--home-hero-badge-border)]" />
        <span className="h-1.5 w-1.5 rotate-45 bg-[var(--color-brand-gold-500)]" />
        <span className="h-px w-10 bg-gradient-to-l from-transparent to-[var(--home-hero-badge-border)]" />
      </div>
      <h2 className="font-brand mt-4 text-3xl font-bold tracking-tight text-[var(--shell-heading)]">{title}</h2>
      <p className="mt-3 text-base text-[var(--shell-content-text)]">{subtitle}</p>
    </div>
  )
}
