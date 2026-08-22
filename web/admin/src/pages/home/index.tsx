// 官网首页(公开落地页):根路径未登录时入口;聚合顶栏 + Hero + 数据条 +
// 核心能力(6 卡片)+ 客户成功实践(3 案例)+ CTA 横幅 + 页脚。
// 文案走 i18n,颜色走 brand/shell + home.css 页级令牌,子区块在 sections.tsx / Footer.tsx。
import { useNavigate } from 'react-router-dom'
import { useEffect } from 'react'
import { useT, useLang } from '../../i18n'
import { useTheme } from '../../theme/context'
import { getAuthToken } from '../../api/client'
import {
  TopNav, Hero, Stats, Features, Cases, CtaBanner,
} from './sections'
import { Footer } from './Footer'
import './home.css'

/** CTA 目标页 chunk 空闲预载:消除首击导航时顶层 Suspense 整页闪 Loading。 */
function usePreloadCtaTarget(signedIn: boolean): void {
  useEffect(() => {
    const idle = window.requestIdleCallback ?? ((cb: () => void) => window.setTimeout(cb, 300))
    const id = idle(() => {
      void import('../login')
      if (signedIn) void import('../dashboard')
    })
    return () => window.cancelIdleCallback?.(id)
  }, [signedIn])
}

export default function HomePage() {
  const t = useT()
  const nav = useNavigate()
  const { locale, setLocale } = useLang()
  const { theme, toggleTheme } = useTheme()
  const signedIn = !!getAuthToken()
  const ctaLabel = signedIn ? t.pages.home.enterConsole : t.pages.home.bookDemo
  const goCta = () => nav(signedIn ? '/dashboard' : '/login')
  usePreloadCtaTarget(signedIn)

  // 锚点平滑滚动(仅本页生效,卸载还原):原生 hash 跳转是瞬移,观感似闪烁。
  useEffect(() => {
    document.documentElement.classList.add('scroll-smooth')
    return () => document.documentElement.classList.remove('scroll-smooth')
  }, [])

  const h = t.pages.home
  return (
    <div className="min-h-screen bg-[var(--shell-content-bg)]">
      <TopNav
        locale={locale}
        setLocale={(v) => setLocale(v as typeof locale)}
        theme={theme}
        toggleTheme={toggleTheme}
        ctaLabel={ctaLabel}
        onCtaClick={goCta}
        t={{ heroBadge: h.heroBadge, navFeatures: h.navFeatures, navSolutions: h.navSolutions, navContact: h.navContact }}
        shellT={t.shell}
      />
      <Hero onCtaClick={goCta} t={{
        heroBadge: h.heroBadge, heroTitle: h.heroTitle, heroLine2: h.heroLine2,
        heroSubtitle: h.heroSubtitle, bookDemo: h.bookDemo, learnMore: h.learnMore,
      }} />
      <Stats stats={h.stats} />
      <Features features={h.features} title={h.featuresTitle} subtitle={h.featuresSubtitle} />
      <Cases cases={h.cases} title={h.casesTitle} subtitle={h.casesSubtitle} viewDetail={h.viewDetail} />
      <CtaBanner title={h.ctaBannerTitle} subtitle={h.ctaBannerSubtitle} buttonLabel={h.bookExclusive} onClick={goCta} />
      <Footer t={{
        tagline: h.footerTagline, navFeatures: h.navFeatures, navSolutions: h.navSolutions,
        navContact: h.navContact, login: h.login, copyright: h.footerCopyright,
      }} />
    </div>
  )
}