// 广告素材轮播:登录/注册页左侧品牌区,规格对齐 visual-design-prompts.md §6.6,文本走 i18n。
import { useEffect, useState, type CSSProperties } from 'react'
import adNetwork from '../assets/brand/ad-network.png'
import adDashboard from '../assets/brand/ad-dashboard.png'
import adService from '../assets/brand/ad-service.png'
import { BRAND_GOLD } from './auth-shell'
import { useT } from '../i18n'

const INTERVAL_MS = 5000

const AD_IMAGES = [adNetwork, adDashboard, adService]

/** 广告卡:256×170,8px 圆角,金色指示器,5s 自动轮播。 */
export function AdCarousel({ width = 256 }: { width?: number }) {
  const t = useT()
  const [idx, setIdx] = useState(0)
  useEffect(() => {
    const interval = setInterval(() => setIdx((i) => (i + 1) % t.ads.length), INTERVAL_MS)
    return () => clearInterval(interval)
  }, [t.ads.length])
  const ad = t.ads[idx]
  return (
    <div style={{ ...boxStyle, width }}>
      <img src={AD_IMAGES[idx]} alt={ad.title} style={imgStyle} />
      <div style={captionStyle}>
        <span style={titleStyle}>{ad.title}</span>
        <span style={subStyle}>{ad.sub}</span>
      </div>
      <div style={dotsStyle}>
        {t.ads.map((_, i) => (
          <span key={i} style={i === idx ? dotOnStyle : dotOffStyle} />
        ))}
      </div>
    </div>
  )
}

const boxStyle: CSSProperties = {
  position: 'relative',
  width: 256,
  height: 170,
  borderRadius: 8,
  overflow: 'hidden',
  boxShadow: '0 8px 24px rgba(3, 13, 31, 0.18)',
  marginTop: 32,
  flexShrink: 0,
}

const imgStyle: CSSProperties = {
  display: 'block', width: '100%', height: '100%', objectFit: 'cover',
}

const captionStyle: CSSProperties = {
  position: 'absolute', inset: 'auto 0 0 0', padding: '24px 14px 10px',
  background: 'linear-gradient(transparent, rgba(15,30,59,.82))', color: '#fff',
  display: 'flex', flexDirection: 'column', gap: 2,
}

const titleStyle: CSSProperties = { fontSize: 12, fontWeight: 600, letterSpacing: 0.5 }
const subStyle: CSSProperties = { fontSize: 11, color: BRAND_GOLD }

const dotsStyle: CSSProperties = {
  position: 'absolute', top: 10, right: 10, display: 'flex', gap: 4,
}

const dotOnStyle: CSSProperties = {
  width: 12, height: 3, borderRadius: 2, background: BRAND_GOLD,
}

const dotOffStyle: CSSProperties = {
  width: 6, height: 3, borderRadius: 2, background: 'rgba(255,255,255,.55)',
}