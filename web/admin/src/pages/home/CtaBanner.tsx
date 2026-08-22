// CTA 横幅:深藏青圆角卡片(双主题恒色) + 金色皇冠 + 标题/副文案 + 金色大按钮。
import { CrownIcon } from './icons'

export interface CtaBannerProps {
  title: string
  subtitle: string
  buttonLabel: string
  onClick: () => void
}

const BTN = 'inline-flex flex-none items-center justify-center rounded-md bg-[var(--color-brand-gold-500)] px-8 py-3.5 text-base font-semibold text-[var(--color-brand-navy-950)] transition-colors hover:bg-[var(--color-brand-gold-600)]'

export function CtaBanner({ title, subtitle, buttonLabel, onClick }: CtaBannerProps) {
  return (
    <section id="contact" className="mx-auto max-w-6xl scroll-mt-20 px-4 py-16">
      <div className="relative overflow-hidden rounded-2xl border border-white/10 bg-[var(--color-brand-navy-950)] px-6 py-14 md:px-14">
        <div className="pointer-events-none absolute inset-0 bg-[url('/images/cta-bg.png')] bg-cover bg-center opacity-30" />
        <div className="pointer-events-none absolute -right-10 -top-16 h-56 w-56 rounded-full bg-[var(--home-cta-glow)] blur-3xl" />
        <div className="pointer-events-none absolute -bottom-20 -left-10 h-56 w-56 rounded-full bg-[var(--home-cta-glow)] blur-3xl" />
        <div className="relative flex flex-col items-center gap-8 md:flex-row md:justify-between">
          <div className="flex items-center gap-6">
            <span className="relative flex-none">
              <span className="absolute inset-0 -m-3 rounded-full bg-[var(--home-cta-glow)] blur-xl" />
              <span className="relative block"><CrownIcon size={56} /></span>
            </span>
            <div>
              <h2 className="font-brand text-2xl font-bold tracking-tight text-white">{title}</h2>
              <p className="mt-2 text-sm text-[var(--shell-nav-text)]">{subtitle}</p>
            </div>
          </div>
          <button type="button" className={BTN} onClick={onClick}>{buttonLabel}</button>
        </div>
      </div>
    </section>
  )
}
