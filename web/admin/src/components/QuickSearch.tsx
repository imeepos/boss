// 快捷搜索 + 聚合搜索命令面板:顶栏搜索入口 + Ctrl/Cmd+K 唤起。
// 空输入=菜单导航(按角色权限过滤);输入>=2 字符=菜单过滤 + 四域聚合检索(契约 /search)。
import { useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import type { Profile } from '../api/auth'
import { aggregateSearch, type SearchDomain, type SearchGroup, type SearchHit } from '../api/search'
import { Dialog, DialogContent } from './ui/dialog'
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from './ui/command'
import { useT } from '../i18n'
import { visibleGroupsForRole } from '../router/role-menu'
import { SearchIcon } from '../layouts/icons'

// 与 TopBar 工具按钮同款样式。
const TOOL_BTN = 'grid h-[34px] w-[34px] cursor-pointer place-items-center rounded-full border-0 bg-none text-white/80 hover:bg-[var(--shell-search-bg-focus)] hover:text-white'

// 各域结果点击后的落地页(与 menu.def.ts 路由 path 对齐)。
const DOMAIN_ROUTE: Record<SearchDomain, string> = {
  customer: '/bss/customer',
  user: '/bss/user',
  worker: '/boss/worker',
  order: '/boss/order',
}

const MIN_KEYWORD = 2
const DEBOUNCE_MS = 300

/** 聚合命中主键(各域 id 字段名不同)。 */
export function hitId(domain: SearchDomain, hit: SearchHit): number {
  switch (domain) {
    case 'customer':
    case 'worker':
    case 'order':
      return (hit as { id: number }).id
    case 'user':
      return (hit as { customerId: number }).customerId
  }
}

/** 聚合命中展示文案:title 主行,sub 次行(编号 · 联系方式)。 */
export function hitText(domain: SearchDomain, hit: SearchHit): { title: string; sub: string } {
  switch (domain) {
    case 'customer': {
      const h = hit as { name: string; customerCode: string; phone: string }
      return { title: h.name, sub: [h.customerCode, h.phone].filter(Boolean).join(' · ') }
    }
    case 'user': {
      const h = hit as { name: string; loginName: string; phone: string }
      return { title: h.name, sub: [h.loginName, h.phone].filter(Boolean).join(' · ') }
    }
    case 'worker': {
      const h = hit as { name: string; staffNo: string; phone: string }
      return { title: h.name, sub: [h.staffNo, h.phone].filter(Boolean).join(' · ') }
    }
    case 'order': {
      const h = hit as { orderNo: string; customer: string; product: string }
      return { title: h.orderNo, sub: [h.customer, h.product].filter(Boolean).join(' · ') }
    }
  }
}

export function QuickSearch({ profile }: { profile: Profile }) {
  const t = useT()
  const nav = useNavigate()
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [debounced, setDebounced] = useState('')
  const [groups, setGroups] = useState<SearchGroup[]>([])
  const [loading, setLoading] = useState(false)
  const seqRef = useRef(0)

  // Ctrl/Cmd+K 全局唤起。
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault()
        setOpen((v) => !v)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])

  // 防抖关键字;不足 MIN_KEYWORD 字符不发起聚合检索。
  useEffect(() => {
    const q = query.trim()
    if (q.length < MIN_KEYWORD) {
      setDebounced('')
      setGroups([])
      return
    }
    const timer = setTimeout(() => setDebounced(q), DEBOUNCE_MS)
    return () => clearTimeout(timer)
  }, [query])

  // 聚合检索:seq 防旧响应覆盖新输入。
  useEffect(() => {
    if (!debounced) return
    const seq = ++seqRef.current
    setLoading(true)
    aggregateSearch(debounced)
      .then((gs) => {
        if (seqRef.current === seq) setGroups(gs)
      })
      .catch(() => {
        if (seqRef.current === seq) setGroups([])
      })
      .finally(() => {
        if (seqRef.current === seq) setLoading(false)
      })
  }, [debounced])

  const close = () => {
    setOpen(false)
    setQuery('')
    setGroups([])
  }

  const goDomain = (domain: SearchDomain) => {
    // 客户/用户/订单页支持 ?kw= 预填;师傅页暂无关键字筛选,仅落地列表。
    const withKw = debounced !== '' && domain !== 'worker'
    nav(withKw ? `${DOMAIN_ROUTE[domain]}?kw=${encodeURIComponent(debounced)}` : DOMAIN_ROUTE[domain])
    close()
  }

  // 侧栏同口径的可见分组(内置角色静态映射 / 自定义角色按权限码)。
  const visible = useMemo(() => visibleGroupsForRole(profile.roleCode, profile.permissionCodes), [profile])
  const qLower = query.trim().toLowerCase()
  const menuHits = qLower
    ? visible
        .flatMap((g) => g.items.map((it) => ({ ...it, groupId: g.id })))
        .filter((it) => (t.menu.items[it.key] ?? it.label).toLowerCase().includes(qLower))
    : []

  const domainLabel = (d: SearchDomain) =>
    ({ customer: t.quickSearch.customers, user: t.quickSearch.users, worker: t.quickSearch.workers, order: t.quickSearch.orders })[d]

  return (
    <>
      <button className={TOOL_BTN} onClick={() => setOpen(true)} title={t.shell.searchPlaceholder} aria-label="quick search">
        <SearchIcon size={18} />
      </button>
      <Dialog open={open} onOpenChange={(v) => (v ? setOpen(true) : close())}>
        <DialogContent className="overflow-hidden p-0 shadow-lg">
          <Command shouldFilter={false} className="[&_[cmdk-group-heading]]:px-2 [&_[cmdk-group-heading]]:font-medium [&_[cmdk-group-heading]]:text-muted-foreground [&_[cmdk-group]]:px-2 [&_[cmdk-input-wrapper]_svg]:h-5 [&_[cmdk-input-wrapper]_svg]:w-5 [&_[cmdk-input]]:h-12 [&_[cmdk-item]]:px-2 [&_[cmdk-item]]:py-3 [&_[cmdk-item]_svg]:h-5 [&_[cmdk-item]_svg]:w-5">
            <CommandInput value={query} onValueChange={setQuery} placeholder={t.quickSearch.placeholder} autoFocus />
            <CommandList>
              {!qLower ? (
                visible.map((g) => (
                  <CommandGroup key={g.id} heading={t.menu.groups[g.id] ?? g.label}>
                    {g.items.map((it) => (
                      <CommandItem key={it.key} value={`menu:${it.key}`} onSelect={() => { nav(it.path); close() }}>
                        {t.menu.items[it.key] ?? it.label}
                      </CommandItem>
                    ))}
                  </CommandGroup>
                ))
              ) : (
                <>
                  {menuHits.length > 0 && (
                    <CommandGroup heading={t.quickSearch.jump}>
                      {menuHits.map((it) => (
                        <CommandItem key={it.key} value={`menu:${it.key}`} onSelect={() => { nav(it.path); close() }}>
                          {t.menu.items[it.key] ?? it.label}
                        </CommandItem>
                      ))}
                    </CommandGroup>
                  )}
                  {groups.map((g) => (
                    <CommandGroup key={g.domain} heading={domainLabel(g.domain)}>
                      {g.items.map((hit) => {
                        const text = hitText(g.domain, hit)
                        return (
                          <CommandItem key={`${g.domain}:${hitId(g.domain, hit)}`} value={`${g.domain}:${hitId(g.domain, hit)}`} onSelect={() => goDomain(g.domain)}>
                            <span>{text.title}</span>
                            <span className="ml-auto pl-3 text-xs text-[var(--shell-group-title)]">{text.sub}</span>
                          </CommandItem>
                        )
                      })}
                      <CommandItem key={`${g.domain}:all`} value={`${g.domain}:all`} onSelect={() => goDomain(g.domain)}>
                        <span>{t.quickSearch.viewAll}</span>
                      </CommandItem>
                    </CommandGroup>
                  ))}
                  {menuHits.length === 0 && groups.length === 0 && (
                    <CommandEmpty>
                      {loading ? t.common.loading : query.trim().length >= MIN_KEYWORD ? t.quickSearch.empty : t.quickSearch.hint}
                    </CommandEmpty>
                  )}
                </>
              )}
            </CommandList>
          </Command>
        </DialogContent>
      </Dialog>
    </>
  )
}
