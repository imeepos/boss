// StatusTag:<StatusTag domain="order" value="INSTALLING" />;未注册值灰色兜底。
import { REGISTRY, type StatusDomain } from './registry'

export function StatusTag({ domain, value }: { domain: StatusDomain | string; value: string }) {
  const meta = (REGISTRY as Record<string, Record<string, { label: string; color: string }>>)[domain]?.[value]
  const label = meta?.label ?? value
  const color = meta?.color ?? '#8c8c8c'
  return (
    <span
      className={meta ? 'st-tag' : 'st-tag st-unknown'}
      style={{
        display: 'inline-block',
        padding: '0 8px',
        borderRadius: 4,
        fontSize: 12,
        lineHeight: '22px',
        color,
        background: `${color}1a`,
        border: `1px solid ${color}55`,
      }}
    >
      {label}
    </span>
  )
}
