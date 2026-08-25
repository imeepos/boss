// 地址层级页:懒加载树 + 后端全树搜索(自动展开祖先链)+ 节点增删改 + 挂接抽屉(迁移 000040)。
import { useCallback, useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { useQueryState } from '../../../lib/useQueryState'
import { ErrorBanner, EmptyState, ToolbarButton } from '../../../components/business/page-head'
import { Badge } from '../../../components/ui/badge'
import { Input } from '../../../components/ui/input'
import { AddressGeoDrawer, type AddressRow, type CountryRow } from './AddressGeoDrawer'
import { AddressNodeDrawer } from './AddressNodeDrawer'
import { BatchImportEntry } from '../importer/BatchImportEntry'
import { CARD, TOOLBAR, SPACER, ADDR_ROW, ADDR_TOGGLE, ADDR_NAME, ACT_BTN, SEP } from '../geo/styles'
import { useConfirm } from '../../../components/ConfirmDialog'

interface AddressHit { node: AddressRow; ancestors: AddressRow[] }

export default function AddressPage() {
  const t = useT()
  const confirmDialog = useConfirm()
  const a = t.pages.address
  const [roots, setRoots] = useState<AddressRow[]>([])
  const [countries, setCountries] = useState<CountryRow[]>([])
  const [childrenOf, setChildrenOf] = useState<Record<number, AddressRow[]>>({})
  const [expanded, setExpanded] = useState<Set<number>>(new Set())
  const [unlinked, setUnlinked] = useQueryState('unlinked', '')
  const [keyword, setKeyword] = useQueryState('kw', '')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [attachOf, setAttachOf] = useState<AddressRow | null>(null)
  const [nodeForm, setNodeForm] =
    useState<{ mode: 'create' | 'rename'; parent?: AddressRow; row?: AddressRow } | null>(null)

  useEffect(() => {
    apiFetch<CountryRow[]>('/geo/countries')
      .then((d) => setCountries(d ?? []))
      .catch(() => setError(a.loadFail))
  }, [a])

  const loadRoots = useCallback(() => {
    const q = unlinked === '1' ? 'unlinked=1' : 'parentId=0'
    apiFetch<AddressRow[]>(`/addresses?${q}`)
      .then((d) => { setRoots(filterTopLevel(d ?? [])); setChildrenOf({}); setExpanded(new Set()) })
      .catch(() => setError(a.loadFail))
  }, [unlinked, a])
  useEffect(loadRoots, [loadRoots])

  // toggle 懒展开:首次展开按 parentId 拉子级并缓存;无子级则标记叶节点(children=[])。
  const loadChildren = async (id: number): Promise<AddressRow[]> => {
    const kids = await apiFetch<AddressRow[]>(`/addresses?parentId=${id}`)
      .catch(() => { setError(a.loadFail); return [] as AddressRow[] })
    setChildrenOf((m) => ({ ...m, [id]: kids ?? [] }))
    return kids ?? []
  }
  const toggle = async (row: AddressRow) => {
    const next = new Set(expanded)
    if (next.has(row.id)) {
      next.delete(row.id)
    } else {
      next.add(row.id)
      if (childrenOf[row.id] === undefined) await loadChildren(row.id)
    }
    setExpanded(next)
  }

  // 全树搜索:后端返回命中+祖先链,逐层拉子级并展开,命中路径即完整可见。
  const search = async () => {
    const kw = keyword.trim()
    if (!kw) return
    setBusy(true)
    try {
      const hits = await apiFetch<AddressHit[]>(`/addresses/search?q=${encodeURIComponent(kw)}`)
        .catch(() => { setError(a.loadFail); return [] as AddressHit[] })
      const next = new Set(expanded)
      for (const h of hits ?? []) {
        for (const anc of h.ancestors) {
          if (childrenOf[anc.id] === undefined) await loadChildren(anc.id)
          next.add(anc.id)
        }
      }
      setExpanded(next)
      setKeyword('') // 清空本地过滤,展示整条命中路径
      if (!hits?.length) setError(a.noHit)
    } finally {
      setBusy(false)
    }
  }

  const countryName = (code: string) => {
    if (!code) return a.none
    const hit = countries.find((c) => c.alpha2 === code)
    return hit ? `${code} ${hit.displayName}` : code
  }

  const kw = keyword.trim().toLowerCase()
  const visible = (rows: AddressRow[]) => filterRows(rows, kw, childrenOf)

  const remove = async (row: AddressRow) => {
    if (!(await confirmDialog(`${a.deleteConfirm}: ${row.name}?`, { danger: true }))) return
    await apiFetch(`/addresses/${row.id}`, { method: 'DELETE' })
      .catch(() => setError(a.deleteFail))
    loadRoots()
  }

  return (
    <div className={CARD}>
      {error && <ErrorBanner message={error} className="mt-3" />}
      <div className={TOOLBAR}>
        <Input className="w-50" placeholder={t.pages.geo.searchPlaceholder}
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter') search() }} />
        <ToolbarButton disabled={busy} onClick={search}>{a.searchAll}</ToolbarButton>
        <div className={SPACER} />
        <ToolbarButton primary={unlinked === '1'}
          onClick={() => setUnlinked(unlinked === '1' ? '' : '1')}>
          {unlinked === '1' ? a.unlinkedAll : a.unlinked}
        </ToolbarButton>
        <ToolbarButton primary
          onClick={() => setNodeForm({ mode: 'create', parent: undefined })}>+ {a.addRoot}</ToolbarButton>
        <BatchImportEntry kind="addr" onImported={loadRoots} />
      </div>
      <AddressTree rows={visible(roots)} childrenOf={childrenOf} expanded={expanded} depth={0}
        countryName={countryName} keyword={kw}
        onToggle={toggle} onAttach={setAttachOf}
        onAddChild={(r) => setNodeForm({ mode: 'create', parent: r })}
        onRename={(r) => setNodeForm({ mode: 'rename', row: r })}
        onDelete={remove} />
      {roots.length === 0 && <EmptyState text={a.empty} />}
      {nodeForm && (
        <AddressNodeDrawer mode={nodeForm.mode} parent={nodeForm.parent} row={nodeForm.row}
          onDone={() => { setNodeForm(null); loadRoots() }} onCancel={() => setNodeForm(null)} />
      )}
      {attachOf && (
        <AddressGeoDrawer row={attachOf} onDone={() => { setAttachOf(null); loadRoots() }}
          onCancel={() => setAttachOf(null)} />
      )}
    </div>
  )
}

// AddressTree 递归渲染一层节点列表;展开态读缓存。
function AddressTree({ rows, childrenOf, expanded, depth, countryName, keyword,
  onToggle, onAttach, onAddChild, onRename, onDelete }: {
  rows: AddressRow[]
  childrenOf: Record<number, AddressRow[]>
  expanded: Set<number>
  depth: number
  countryName: (code: string) => string
  keyword: string
  onToggle: (row: AddressRow) => void
  onAttach: (row: AddressRow) => void
  onAddChild: (row: AddressRow) => void
  onRename: (row: AddressRow) => void
  onDelete: (row: AddressRow) => void
}) {
  const t = useT()
  const a = t.pages.address
  if (rows.length === 0) return null
  return (
    <div className="mx-4 my-3 flex flex-col gap-0.5" style={{ paddingLeft: depth * 20 }}>
      {rows.map((r) => {
        const kids = childrenOf[r.id]
        const open = expanded.has(r.id)
        const leaf = kids !== undefined ? kids.length === 0 : r.hasChildren === false
        return (
          <div key={r.id} className="flex flex-col">
            <div className={ADDR_ROW}>
              {r.level < 5 && !leaf
                ? <button className={ADDR_TOGGLE} aria-label={open ? a.collapse : a.expand}
                  onClick={() => onToggle(r)}>{open ? '−' : '+'}</button>
                : <span className={ADDR_TOGGLE + ' cursor-default border-none text-[var(--shell-group-title)]'}>·</span>}
              <span className={ADDR_NAME}>{r.name}</span>
              <Badge>{r.level}</Badge>
              <Badge>{countryName(r.countryCode)}</Badge>
              {r.adminCode && <Badge>{r.adminCode}</Badge>}
              <div className={SPACER} />
              <span className="inline-flex items-center">
                {r.level === 1 && <button className={ACT_BTN} onClick={() => onAttach(r)}>{a.attach}</button>}
                {r.level < 5 && <><span className={SEP}>|</span>
                  <button className={ACT_BTN} onClick={() => onAddChild(r)}>{a.addChild}</button></>}
                <span className={SEP}>|</span>
                <button className={ACT_BTN} onClick={() => onRename(r)}>{a.rename}</button>
                <span className={SEP}>|</span>
                <button className={ACT_BTN} onClick={() => onDelete(r)}>{a.delete}</button>
              </span>
            </div>
            {open && (
              <AddressTree rows={filterRows(kids ?? [], keyword, childrenOf)} childrenOf={childrenOf}
                expanded={expanded} depth={depth + 1} countryName={countryName} keyword={keyword}
                onToggle={onToggle} onAttach={onAttach} onAddChild={onAddChild}
                onRename={onRename} onDelete={onDelete} />
            )}
          </div>
        )
      })}
    </div>
  )
}

// filterRows 本地关键字过滤(递归):命中节点保留整棵子树;未命中但已加载子孙命中则保留自身作路径。
function filterTopLevel(rows: AddressRow[]): AddressRow[] {
  const ids = new Set(rows.map((r) => r.id))
  return rows.filter((r) => !hasParentInRows(r, ids, rows))
}

function hasParentInRows(row: AddressRow, ids: Set<number>, rows: AddressRow[]): boolean {
  if (row.parentId != null) return ids.has(row.parentId)
  if (!row.path) return false
  const parts = row.path.split('.')
  if (parts.length < 2) return false
  const parentPath = parts.slice(0, -1).join('.')
  return rows.some((candidate) => candidate.path === parentPath)
}

function filterRows(rows: AddressRow[], kw: string, childrenOf: Record<number, AddressRow[]>): AddressRow[] {
  if (!kw) return rows
  const k = kw.toLowerCase()
  return rows.filter((r) => {
    const hit = [r.name, r.countryCode, r.adminCode].some((s) => s.toLowerCase().includes(k))
    return hit || filterRows(childrenOf[r.id] ?? [], kw, childrenOf).length > 0
  })
}
