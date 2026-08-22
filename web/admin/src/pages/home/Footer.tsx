// 官网页脚:品牌区 + 快速导航 + 版权行;颜色走 brand/shell 令牌。
import { Link } from 'react-router-dom'
import logoFull from '../../assets/brand/logo-mark-gradient.png'

export function Footer({
  t,
}: {
  t: { tagline: string; navFeatures: string; navSolutions: string; navContact: string; login: string; copyright: string }
}) {
  return (
    <footer className="border-t border-white/10 bg-[var(--color-brand-navy-950)]">
      <div className="mx-auto flex max-w-6xl flex-col gap-6 px-4 py-10 md:flex-row md:items-start md:justify-between">
        <div className="flex items-center gap-3">
          <img src={logoFull} alt="" className="h-8 w-8" />
          <div className="flex flex-col leading-tight">
            <span className="text-base font-bold text-white">Sphere Boss</span>
            <span className="mt-0.5 text-xs text-[var(--shell-nav-text)]">{t.tagline}</span>
          </div>
        </div>
        <nav className="flex flex-wrap items-center gap-6">
          <a href="#features" className="text-sm text-[var(--shell-nav-text)] transition-colors hover:text-white">{t.navFeatures}</a>
          <a href="#solutions" className="text-sm text-[var(--shell-nav-text)] transition-colors hover:text-white">{t.navSolutions}</a>
          <a href="#contact" className="text-sm text-[var(--shell-nav-text)] transition-colors hover:text-white">{t.navContact}</a>
          <Link to="/login" className="text-sm text-[var(--shell-nav-text)] transition-colors hover:text-white">{t.login}</Link>
        </nav>
      </div>
      <div className="border-t border-white/5">
        <div className="mx-auto flex h-12 max-w-6xl items-center justify-center px-4 text-xs text-[var(--shell-nav-text)]">
          {t.copyright}
        </div>
      </div>
    </footer>
  )
}
