// 登录/注册共用外壳:全屏背景 + 双栏面板(左品牌 382px,右表单 438px),文本走 i18n。
// 规格对齐 visual-design-prompts.md §3。
import type { CSSProperties, ReactNode } from 'react'
import bgFull from '../assets/brand/login-bg-full.png'
import logoMark from '../assets/brand/logo-mark-gradient.png'
import ornamentRing from '../assets/brand/ornament-ring.png'
import ornamentRibbon from '../assets/brand/ornament-ribbon.png'
import { useT } from '../i18n'

/* ── 品牌色(对齐 visual-design-prompts.md §2) ── */
export const BRAND_NAVY = '#0F1E3B'
export const BRAND_BLUE = '#273F70'
export const BRAND_BLUE_HOVER = '#1F355F'
export const BRAND_GOLD = '#D5A63A'
export const BRAND_GOLD_HOVER = '#BE8D25'

/* ── 表单组件(对齐 visual-design-prompts.md §6) ── */

export const inputStyle: CSSProperties = {
  width: '100%',
  height: 34,
  boxSizing: 'border-box',
  padding: '0 11px',
  marginBottom: 12,
  border: '1px solid var(--color-border-default)',
  borderRadius: 5,
  background: '#FFFFFF',
  color: '#4E5664',
  fontSize: 12,
  fontFamily: 'var(--font-family-base)',
  outline: 'none',
  transition: 'border-color 160ms',
}

export const buttonStyle: CSSProperties = {
  width: '100%',
  height: 37,
  border: 'none',
  borderRadius: 5,
  background: BRAND_BLUE,
  color: '#FFFFFF',
  fontSize: 14,
  fontWeight: 600,
  letterSpacing: 8,
  fontFamily: 'var(--font-family-base)',
  cursor: 'pointer',
  transition: 'background 160ms',
}

/** 左侧品牌区(装饰 + Logo + 标题 + 广告位)。 */
export function BrandAside({ children, tip }: { children?: ReactNode; tip?: string }) {
  const t = useT()
  return (
    <aside className="auth-aside" style={asideStyle}>
      <img src={ornamentRing} alt="" style={ringStyle} />
      <img src={logoMark} alt="Sphere Boss" style={asideLogoStyle} />
      <h1 style={asideTitleStyle}>Sphere Boss</h1>
      <p style={asideDescStyle}>{t.common.tagline}</p>
      {tip && <p style={asideTipStyle}>{tip}</p>}
      {children}
      <img src={ornamentRibbon} alt="" style={ribbonStyle} />
    </aside>
  )
}

/** 登录/注册页外壳(全屏背景 + 820×564 面板)。 */
export function AuthShell({ children, aside }: { children: ReactNode; aside?: ReactNode }) {
  const t = useT()
  return (
    <div style={{ ...pageStyle, backgroundImage: `url(${bgFull})` }}>
      <div className="auth-panel" style={panelStyle}>
        {aside}
        <section style={formPaneStyle}>
          <div style={formInnerStyle}>
            {children}
          </div>
          <div style={footerStyle}>{t.common.footer}</div>
        </section>
      </div>
    </div>
  )
}

/* ── 页面 ── */

const pageStyle: CSSProperties = {
  minHeight: '100vh',
  display: 'grid',
  placeItems: 'center',
  background: '#f5f6fa',
  backgroundSize: 'cover',
  backgroundPosition: 'center',
  fontFamily: 'var(--font-family-base)',
}

/* ── 面板 (820×564, 14px 圆角) ── */

const panelStyle: CSSProperties = {
  display: 'grid',
  gridTemplateColumns: '382px 438px',
  width: 820,
  minHeight: 564,
  borderRadius: 14,
  overflow: 'hidden',
  boxShadow: '0 20px 52px rgba(15, 30, 59, 0.18)',
}

/* ── 左侧品牌区 (382px) ── */

const asideStyle: CSSProperties = {
  position: 'relative',
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'center',
  background: BRAND_NAVY,
  color: '#fff',
  textAlign: 'center',
  padding: '96px 48px 0',
  overflow: 'hidden',
}

const ringStyle: CSSProperties = {
  position: 'absolute', top: -60, right: -60, width: 240, opacity: 0.55, pointerEvents: 'none',
}

const ribbonStyle: CSSProperties = {
  position: 'absolute', bottom: -30, left: -40, width: 280, opacity: 0.3,
  pointerEvents: 'none', transform: 'rotate(-6deg)',
}

const asideLogoStyle: CSSProperties = {
  width: 62, height: 62, zIndex: 1,
  filter: 'drop-shadow(0 4px 12px rgba(0,0,0,.3))',
}

const asideTitleStyle: CSSProperties = {
  margin: '20px 0 0', fontSize: 28, lineHeight: '36px', fontWeight: 700,
  letterSpacing: 1.2, fontFamily: 'var(--font-family-brand)', color: '#fff', zIndex: 1,
}

const asideDescStyle: CSSProperties = {
  margin: '6px 0 0', fontSize: 13, lineHeight: '20px', fontWeight: 500,
  letterSpacing: 4, color: BRAND_GOLD, zIndex: 1,
}

const asideTipStyle: CSSProperties = {
  margin: '16px 0 0', fontSize: 12, lineHeight: '18px', fontWeight: 500,
  letterSpacing: 1, color: 'rgba(255,255,255,.65)', zIndex: 1,
}

/* ── 右侧表单区 (438px) ── */

const formPaneStyle: CSSProperties = {
  position: 'relative',
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'center',
  justifyContent: 'flex-start',
  background: '#FCFCFD',
  paddingTop: 104,
}

const formInnerStyle: CSSProperties = {
  width: 274,
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'center',
}

const footerStyle: CSSProperties = {
  position: 'absolute', bottom: 16, fontSize: 12, color: '#A6B1C3',
  letterSpacing: 0.3, fontFamily: 'var(--font-family-base)',
}