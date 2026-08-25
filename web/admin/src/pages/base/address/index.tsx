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
  const im = t.pages.importer
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

  const loadChildren = async (id: number): Promise<AddressRow[]> => {
    const kids = await apiFetch<AddressRow[]>(`/addresses?parentId=${id}`)
      .catch(() => { setError(a.loadFail); return [] as AddressRow[] })
    setChildrenOf((m) => ({ ...m, [id]: kids ?? [] }))
    return kids ?? []
  }
  const toggle = async (row: AddressRow) => {
    const next = new Set(expanded)
    if (next.has(row.id)) next.delete(row.id)
    else { next.add(row.id); if (childrenOf[row.id] === undefined) await loadChildren(row.id) }
    setExpanded(next)
  }

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
      setKeyword('')
      if (!hits?.length) setError(a.noHit)
    } finally { setBusy(false) }
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
    await apiFetch(`/addresses/${row.id}`, { method: 'DELETE' }).catch(() => setError(a.deleteFail))
    loadRoots()
  }

  return (
    <div className={CARD}>
      {error && <ErrorBanner message={error} className="mt-3" />}
      <div className={TOOLBAR}>
        <Input className="w-50" placeholder={t.pages.geo.searchPlaceholder} value={keyword}
          onChange={(e) => setKeyword(e.target.value)} onKeyDown={(e) => { if (e.key === 'Enter') search() }} />
        <ToolbarButton disabled={busy} onClick={search}>{a.searchAll}</ToolbarButton>
        <div className={SPACER} />
        <ToolbarButton primary={unlinked === '1'} onClick={() => setUnlinked(unlinked === '1' ? '' : '1')}>
          {unlinked === '1' ? a.unlinkedAll : a.unlinked}
        </ToolbarButton>
        <ToolbarButton primary onClick={() => setNodeForm({ mode: 'create', parent: undefined })}>+ {a.addRoot}</ToolbarButton>
        <BatchImportEntry kind="addr" onImported={loadRoots} />
      </div>
      <AddressTree rows={visible(roots)} childrenOf={childrenOf} expanded={expanded} depth={0}
        countryName={countryName} keyword={kw} onToggle={toggle} onAttach={setAttachOf}
        onAddChild={(r) => setNodeForm({ mode: 'create', parent: r })} onRename={(r) => setNodeForm({ mode: 'rename', row: r })} onDelete={remove} />
      {roots.length === 0 && <EmptyState text={a.empty} />}
      {nodeForm && <AddressNodeDrawer mode={nodeForm.mode} parent={nodeForm.parent} row={nodeForm.row}
        onDone={() => { setNodeForm(null); loadRoots() }} onCancel={() => setNodeForm(null)} />}
      {attachOf && <AddressGeoDrawer row={attachOf} onDone={() => { setAttachOf(null); loadRoots() }} onCancel={() => setAttachOf(null)} />}
    </div>
  )
}

// AddressTree 递归渲染一层节点列表;展开态读缓存。
function AddressTree({ rows, childrenOf, expanded, depth, countryName, keyword,
  onToggle, onAttach, onAddChild, onRename, onDelete }: {
  rows: AddressRow[]; childrenOf: Record<number, AddressRow[]>; expanded: Set<number>; depth: number
  countryName: (code: string) => string; keyword: string; onToggle: (r: AddressRow) => void
  onAttach: (r: AddressRow) => void; onAddChild: (r: AddressRow) => void; onRename: (r: AddressRow) => void
  onDelete: (r: AddressRow) => void
}) {
  return <div>{rows.map((r) => {
    const open = expanded.has(r.id); const kids = childrenOf[r.id] ?? []
    return <div key={r.id} className={ADDR_ROW} style={{ paddingLeft: `${depth * 20 + 8}px` }}>
      <button className={ADDR_TOGGLE} onClick={() => onToggle(r)}>{open ? '▾' : '▸'}</button>
      <span className={ADDR_NAME}>{keyword ? highlight(r.name, keyword) : r.name}</span>
      <span className="ml-auto text-xs text-[var(--shell-group-title)]">{countryName(r.countryCode)}</span>
      <button className={ACT_BTN} onClick={() => onAddChild(r)}>+</button><button className={ACT_BTN} onClick={() => onAttach(r)}>◉</button>
      <button className={ACT_BTN} onClick={() => onRename(r)}>✎</button><button className={ACT_BTN} onClick={() => onDelete(r)}>×</button>
      {open && <AddressTree rows={kids} childrenOf={childrenOf} expanded={expanded} depth={depth + 1} countryName={countryName} keyword={keyword}
        onToggle={onToggle} onAttach={onAttach} onAddChild={onAddChild} onRename={onRename} onDelete={onDelete} />}
    </div>
  })}</div>
}

function filterTopLevel(rows: AddressRow[]) { return rows.filter((r) => !r.parentId) }
function filterRows(rows: AddressRow[], kw: string, children: Record<number, AddressRow[]>) {
  if (!kw) return rows
  return rows.filter((r) => r.name.toLowerCase().includes(kw) || (children[r.id] ?? []).some((c) => c.name.toLowerCase().includes(kw)))
}
function highlight(name: string, kw: string) { const i = name.toLowerCase().indexOf(kw); return i < 0 ? name : <>{name.slice(0, i)}<mark>{name.slice(i, i + kw.length)}</mark>{name.slice(i + kw.length)}</> }
