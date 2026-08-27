// StatusTag:<StatusTag domain="order" value="INSTALLING" />;未注册值灰色兜底。
// 标签走 i18n common.statusTags 平铺键 "domain.VALUE",缺键回退枚举原文;完备性由 StatusTag.test 双向锁定。
import { REGISTRY, type StatusDomain } from './registry'
import { useT } from '../../i18n'

type RegistryMap = Record<string, Record<string, string>>

export function statusTagMeta(domain: string, value: string): string | undefined {
  return (REGISTRY as RegistryMap)[domain]?.[value]
}

/** 纯函数便于 node 测试:标签 = 三语字典 → 枚举原文。 */
export function statusTagLabel(domain: string, value: string, statusTags?: Record<string, string>): string {
  if (!statusTagMeta(domain, value)) return value
  return statusTags?.[`${domain}.${value}`] ?? value
}

export function StatusTag({ domain, value }: { domain: StatusDomain | string; value: string }) {
  const t = useT()
  const color = statusTagMeta(domain, value)
  const label = statusTagLabel(domain, value, t.common.statusTags)
  return (
    <span
      className={color ? 'st-tag' : 'st-tag st-unknown'}
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
