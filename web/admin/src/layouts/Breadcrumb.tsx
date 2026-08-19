// 面包屑:首页 / 分组 / 当前页,分隔符金色。规格见 design-spec.md §2.3。
// 样式:tailwind 原子类(原 shell.css 已删除)。
import { Link, useLocation } from 'react-router-dom'
import { KEY_BY_PATH, PAGE_BY_KEY } from '../router/menu.def'
import { useT } from '../i18n'

export function Breadcrumb() {
  const t = useT()
  const { pathname } = useLocation()
  const key = KEY_BY_PATH.get(pathname)
  if (!key) return null
  const ref = PAGE_BY_KEY.get(key)
  if (!ref) return null
  const groupLabel = t.menu.groups[ref.groupId] ?? ref.groupId
  const itemLabel = t.menu.items[key] ?? ref.item.label
  return (
    <nav className="mb-4 flex items-center gap-2 text-xs text-[var(--shell-crumb-text)]" aria-label="breadcrumb">
      <Link className="text-inherit no-underline hover:text-[var(--color-brand-gold-500)]" to="/dashboard">{t.shell.home}</Link>
      <span className="text-[var(--color-brand-gold-500)]">/</span>
      <span>{groupLabel}</span>
      <span className="text-[var(--color-brand-gold-500)]">/</span>
      <span className="font-semibold text-[var(--shell-heading)]">{itemLabel}</span>
    </nav>
  )
}
