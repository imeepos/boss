// Hero 区块:徽章 + 大标题(金色渐变第二行) + 副文案 + 双按钮,右侧控制台预览。
// 背景三层:主题渐变底 + 网格纹理图 + 金色光晕,营造纵深。
import { HeroConsoleMock } from './HeroConsoleMock'

export interface HeroProps {
  onCtaClick: () => void
  t: { heroBadge: string; heroTitle: string; heroLine2: string; heroSubtitle: string; bookDemo: string; learnMore: string }
}

const BTN_GOLD = 'inline-flex items-center justify-center gap-2 rounded-md bg-[var(--color-brand-gold-500)] px-7 py-3 text-sm font-semibold text-[var(--color-brand-navy-950)] transition-colors hover:bg-[var(--color-brand-gold-600)]'
const BTN_GHOST = 'inline-flex items-center justify-center gap-2 rounded-md border border-[var(--home-mock-border)] bg-[var(--color-brand-navy-950)] px-7 py-3 text-sm text-white transition-colors hover:border-white/40 hover:bg-white/5'

export function Hero({ onCtaClick, t }: HeroProps) {
  return (
    <section className="relative overflow-hidden bg-gradient-to-br from-[var(--home-hero-from)] via-[var(--home-hero-via)] to-[var(--home-hero-to)]">
      <div className="pointer-events-none absolute inset-0 bg-[url('/images/hero-bg.png')] bg-cover bg-center opacity-20 dark:opacity-30" />
      <div className="pointer-events-none absolute -top-24 right-[8%] h-72 w-72 rounded-full bg-[var(--home-cta-glow)] blur-3xl" />
      <div className="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-[var(--home-hero-badge-border)] to-transparent" />
      <div className="relative mx-auto grid max-w-7xl items-center gap-12 px-6 py-16 md:grid-cols-[1.05fr_1fr] md:py-24">
        <div className="relative z-10">
          <Badge label={t.heroBadge} />
          <h1 className="font-brand mt-6 text-[36px] font-bold leading-[1.15] tracking-tight text-[var(--color-brand-navy-950)] dark:text-[var(--shell-heading)] md:text-[40px]">
            <span className="block">{t.heroTitle}</span>
            <span className="mt-3 block bg-gradient-to-r from-[var(--color-brand-gold-500)] to-[var(--color-brand-gold-600)] bg-clip-text text-transparent">{t.heroLine2}</span>
          </h1>
          <p className="mt-6 max-w-lg text-base leading-7 text-[var(--home-hero-subtitle)]">{t.heroSubtitle}</p>
          <div className="mt-10 flex flex-wrap gap-4">
            <button type="button" className={BTN_GOLD} onClick={onCtaClick}>
              {t.bookDemo}
              <ArrowIcon />
            </button>
            <a href="#features" className={BTN_GHOST}>
              {t.learnMore}
              <ArrowIcon />
            </a>
          </div>
        </div>
        <div className="relative hidden min-h-[400px] md:block">
          <HeroConsoleMock />
        </div>
      </div>
    </section>
  )
}

function Badge({ label }: { label: string }) {
  return (
    <span className="inline-flex items-center gap-1.5 rounded-full border border-[var(--home-hero-badge-border)] bg-[var(--home-hero-badge-bg)] px-3.5 py-1.5 text-xs font-medium tracking-wide text-[var(--home-gold-fg)]">
      <span className="h-1.5 w-1.5 rounded-full bg-[var(--color-brand-gold-500)]" />
      {label}
    </span>
  )
}

function ArrowIcon() {
  return (
    <svg width={14} height={14} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
      <path d="M5 12h14M13 6l6 6-6 6" />
    </svg>
  )
}
