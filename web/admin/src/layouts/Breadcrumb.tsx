// 面包屑:首页 / 分组 / 当前页,分隔符金色。规格见 design-spec.md §2.3。
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
    <nav className="shell-crumb" aria-label="breadcrumb">
      <Link to="/dashboard">{t.shell.home}</Link>
      <span className="shell-crumb-sep">/</span>
      <span>{groupLabel}</span>
      <span className="shell-crumb-sep">/</span>
      <span className="shell-crumb-current">{itemLabel}</span>
    </nav>
  )
}
