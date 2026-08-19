// 外壳内联 SVG 图标(描边随 currentColor)+ 原型 SVG 的遮罩图标组件。
import type { CSSProperties } from 'react'

interface IconProps {
  size?: number
}

const svgProps = {
  viewBox: '0 0 24 24',
  fill: 'none',
  stroke: 'currentColor',
  strokeWidth: 1.8,
  strokeLinecap: 'round',
  strokeLinejoin: 'round',
} as const

export function SearchIcon({ size = 16 }: IconProps) {
  return (
    <svg {...svgProps} width={size} height={size} aria-hidden>
      <circle cx="11" cy="11" r="7" />
      <path d="m20 20-3.2-3.2" />
    </svg>
  )
}

export function BellIcon({ size = 18 }: IconProps) {
  return (
    <svg {...svgProps} width={size} height={size} aria-hidden>
      <path d="M18 8a6 6 0 1 0-12 0c0 7-3 8-3 8h18s-3-1-3-8" />
      <path d="M13.7 20a2 2 0 0 1-3.4 0" />
    </svg>
  )
}

export function SunIcon({ size = 18 }: IconProps) {
  return (
    <svg {...svgProps} width={size} height={size} aria-hidden>
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2m0 16v2M4.9 4.9l1.4 1.4m11.4 11.4 1.4 1.4M2 12h2m16 0h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" />
    </svg>
  )
}

export function MoonIcon({ size = 18 }: IconProps) {
  return (
    <svg {...svgProps} width={size} height={size} aria-hidden>
      <path d="M21 12.8A9 9 0 1 1 11.2 3 7 7 0 0 0 21 12.8z" />
    </svg>
  )
}

export function GlobeIcon({ size = 18 }: IconProps) {
  return (
    <svg {...svgProps} width={size} height={size} aria-hidden>
      <circle cx="12" cy="12" r="9" />
      <path d="M3 12h18M12 3c2.7 2.6 4 5.6 4 9s-1.3 6.4-4 9c-2.7-2.6-4-5.6-4-9s1.3-6.4 4-9z" />
    </svg>
  )
}

export function CheckIcon({ size = 14 }: IconProps) {
  return (
    <svg {...svgProps} width={size} height={size} strokeWidth={2.2} aria-hidden>
      <path d="m5 12.5 4.5 4.5L19 7.5" />
    </svg>
  )
}

export function LogoutIcon({ size = 16 }: IconProps) {
  return (
    <svg {...svgProps} width={size} height={size} aria-hidden>
      <path d="M9 21H6a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h3" />
      <path d="m16 17 5-5-5-5M21 12H9" />
    </svg>
  )
}

export function UserIcon({ size = 16 }: IconProps) {
  return (
    <svg {...svgProps} width={size} height={size} aria-hidden>
      <circle cx="12" cy="8" r="4" />
      <path d="M4 21c0-4 3.6-6.5 8-6.5s8 2.5 8 6.5" />
    </svg>
  )
}

export function PlusIcon({ size = 20 }: IconProps) {
  return (
    <svg {...svgProps} width={size} height={size} strokeWidth={2.2} aria-hidden>
      <path d="M12 5v14M5 12h14" />
    </svg>
  )
}

export function MenuIcon({ size = 20 }: IconProps) {
  return (
    <svg {...svgProps} width={size} height={size} aria-hidden>
      <path d="M4 6h16M4 12h16M4 18h16" />
    </svg>
  )
}

export function CollapseIcon({ collapsed }: { collapsed: boolean }) {
  return (
    <svg {...svgProps} width={18} height={18} aria-hidden>
      {collapsed ? (
        <path d="m9 6 6 6-6 6M4 6l6 6-6 6" transform="translate(-1 0)" />
      ) : (
        <path d="m15 6-6 6 6 6M20 6l-6 6 6 6" transform="translate(1 0)" />
      )}
    </svg>
  )
}

/** 遮罩图标:引用 public/icons 下原型 SVG,颜色随 currentColor(亮/暗/激活自适应)。 */
export function MaskIcon({ url, size = 18 }: { url: string; size?: number }) {
  // mask 样式原在 shell.css .mask-icon,现内联(shell.css 已删除)。
  const style = {
    '--icon-url': `url("${url}")`,
    width: size,
    height: size,
    display: 'inline-block',
    flex: 'none',
    background: 'currentColor',
    WebkitMask: 'var(--icon-url) center / contain no-repeat',
    mask: 'var(--icon-url) center / contain no-repeat',
  } as CSSProperties
  return <span style={style} aria-hidden />
}
