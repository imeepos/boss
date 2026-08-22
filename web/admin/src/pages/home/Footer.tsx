// 官网页脚:全幅两段式(对齐 MiniMax 官网结构)——上段超链组(品牌 + 三栏
// 链接),下段版权条(版权 + 语言切换)。全幅深底大留白,内层限宽居中。
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

const COL_TITLE = 'font-brand text-sm font-semibold tracking-wide text-white'
const LINK = 'text-sm text-[var(--shell-nav-text)] transition-colors hover:text-white'

export function Footer({ t, featureLinks, caseLinks, locale, setLocale }: FooterProps) {
  return (
    <footer className="bg-[var(--color-brand-navy-950)] px-4 md:px-[60px]">
      {/* 第一段:超链组(品牌 + 产品/方案/快速入口),全幅铺满仅留边距 */}
      <div>
        <div className="grid grid-cols-1 gap-12 py-16 sm:grid-cols-2 lg:grid-cols-[1.6fr_1fr_1fr_1fr] lg:gap-8">
          <div>
            <div className="flex items-center gap-3">
              <img src={logoFull} alt="" className="h-10 w-10" />
              <div className="flex flex-col leading-tight">
                <span className="font-brand text-xl font-bold tracking-tight text-white">Sphere Boss</span>
                <span className="mt-1 text-xs text-[var(--shell-nav-text)]">{t.tagline}</span>
              </div>
            </div>
            <div className="mt-6 max-w-sm text-sm leading-6 text-[var(--shell-nav-text)]">{t.taglineDesc}</div>
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
        {/* 第二段:版权条 */}
        <div className="flex flex-col items-center gap-4 border-t border-white/10 py-6 sm:flex-row sm:justify-between">
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
    <div className="flex flex-col gap-4">
      <h3 className={COL_TITLE}>{title}</h3>
      {children}
    </div>
  )
}
