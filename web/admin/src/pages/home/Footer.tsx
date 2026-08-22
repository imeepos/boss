// 官网页脚:品牌区 + 产品/方案/快速入口三栏 + 语言切换 + 版权行。
// 链接文案复用 h.features/h.cases 标题与既有 i18n key,颜色走 brand/shell 令牌。
import { Link } from 'react-router-dom'
import { Dropdown } from '../../components/Dropdown'
import { localeOptions, type Locale } from '../../i18n'
import logoFull from '../../assets/brand/logo-mark-gradient.png'

export interface FooterProps {
  t: {
    tagline: string
    taglineDesc: string
    productTitle: string
    solutionsTitle: string
    quickTitle: string
    navContact: string
    bookDemo: string
    login: string
    copyright: string
    language: string
  }
  featureLinks: Array<{ title: string }>
  caseLinks: Array<{ title: string }>
  locale: Locale
  setLocale: (v: Locale) => void
}

const COL_TITLE = 'text-sm font-semibold text-white'
const LINK = 'text-sm text-[var(--shell-nav-text)] transition-colors hover:text-white'

export function Footer({ t, featureLinks, caseLinks, locale, setLocale }: FooterProps) {
  return (
    <footer className="border-t border-white/10 bg-[var(--color-brand-navy-950)]">
      <div className="mx-auto grid max-w-6xl grid-cols-1 gap-10 px-4 py-12 sm:grid-cols-2 lg:grid-cols-[1.4fr_1fr_1fr_1fr]">
        <div>
          <div className="flex items-center gap-3">
            <img src={logoFull} alt="" className="h-9 w-9" />
            <div className="flex flex-col leading-tight">
              <span className="text-base font-bold text-white">Sphere Boss</span>
              <span className="mt-0.5 text-xs text-[var(--shell-nav-text)]">{t.tagline}</span>
            </div>
          </div>
          <div className="mt-5 max-w-xs text-xs leading-5 text-[var(--shell-nav-text)]">{t.taglineDesc}</div>
        </div>
        <FooterColumn title={t.productTitle}>
          {featureLinks.map((f) => (
            <a key={f.title} href="#features" className={LINK}>{f.title}</a>
          ))}
        </FooterColumn>
        <FooterColumn title={t.solutionsTitle}>
          {caseLinks.map((c) => (
            <a key={c.title} href="#solutions" className={LINK}>{c.title}</a>
          ))}
        </FooterColumn>
        <FooterColumn title={t.quickTitle}>
          <Link to="/login" className={LINK}>{t.login}</Link>
          <a href="#contact" className={LINK}>{t.bookDemo}</a>
          <a href="#contact" className={LINK}>{t.navContact}</a>
        </FooterColumn>
      </div>
      <div className="border-t border-white/5">
        <div className="mx-auto flex h-14 max-w-6xl flex-col items-center gap-3 px-4 sm:flex-row sm:justify-between">
          <span className="text-xs text-[var(--shell-nav-text)]">{t.copyright}</span>
          <div className="flex items-center gap-2">
            <span className="text-xs text-[var(--shell-nav-text)]">{t.language}</span>
            <Dropdown
              value={locale}
              options={localeOptions()}
              onChange={(v) => setLocale(v as Locale)}
              ariaLabel="language"
              triggerStyle={{ height: 32 }}
            />
          </div>
        </div>
      </div>
    </footer>
  )
}

function FooterColumn({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-3">
      <h3 className={COL_TITLE}>{title}</h3>
      {children}
    </div>
  )
}
