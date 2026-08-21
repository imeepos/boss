// CTA 横幅金色皇冠图标 + 主题切换图标(亮=月亮/暗=太阳)。
export function CrownIcon({ size = 56 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 64 64" fill="none" aria-hidden>
      <path d="M8 22l8 16h32l8-16-12 8L32 14 20 30z" fill="#D5A63A" />
      <rect x="10" y="42" width="44" height="6" rx="2" fill="#D5A63A" />
      <circle cx="8" cy="22" r="3" fill="#D5A63A" />
      <circle cx="32" cy="14" r="3" fill="#D5A63A" />
      <circle cx="56" cy="22" r="3" fill="#D5A63A" />
    </svg>
  )
}

export function ThemeIcon({ dark }: { dark: boolean }) {
  if (dark) return <SunIcon />
  return <MoonIcon />
}

function SunIcon() {
  return (
    <svg width={16} height={16} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41" />
    </svg>
  )
}

function MoonIcon() {
  return (
    <svg width={16} height={16} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
    </svg>
  )
}