// 地址层级页:懒加载树 + 后端全树搜索(自动展开祖先链)+ 节点增删改 + 挂接抽屉(迁移 000040)。
import { useCallback, useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { useQueryState } from '../../../lib/useQueryState'
import { AddressGeoDrawer, type AddressRow, type CountryRow } from './AddressGeoDrawer'
import { AddressNodeDrawer } from './AddressNodeDrawer'
import '../geo/geo.css'

interface AddressHit { node: AddressRow; ancestors: AddressRow[] }

export default function AddressPage() {
  const t = useT()
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
      .then((d) => { setRoots(d ?? []); setChildrenOf({}); setExpanded(new Set()) })
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
    if (!window.confirm(`${a.deleteConfirm}: ${row.name}?`)) return
    await apiFetch(`/addresses/${row.id}`, { method: 'DELETE' })
      .catch(() => setError(a.deleteFail))
    loadRoots()
  }

  return (
    <div className="geo-card">
      {error && <div className="geo-error" role="alert">{error}</div>}
      <div className="geo-toolbar">
        <input className="geo-input" style={{ width: 200 }} placeholder={t.pages.geo.searchPlaceholder}
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter') search() }} />
        <button className="geo-btn" disabled={busy} onClick={search}>{a.searchAll}</button>
        <div className="spacer" />
        <button className={`geo-btn ${unlinked === '1' ? 'geo-btn-primary' : ''}`}
          onClick={() => setUnlinked(unlinked === '1' ? '' : '1')}>
          {unlinked === '1' ? a.unlinkedAll : a.unlinked}
        </button>
        <button className="geo-btn geo-btn-primary"
          onClick={() => setNodeForm({ mode: 'create', parent: undefined })}>+ {a.addRoot}</button>
      </div>
      <AddressTree rows={visible(roots)} childrenOf={childrenOf} expanded={expanded} depth={0}
        countryName={countryName} keyword={kw}
        onToggle={toggle} onAttach={setAttachOf}
        onAddChild={(r) => setNodeForm({ mode: 'create', parent: r })}
        onRename={(r) => setNodeForm({ mode: 'rename', row: r })}
        onDelete={remove} />
      {roots.length === 0 && <div className="geo-empty">{a.empty}</div>}
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
    <div className="addr-tree" style={{ paddingLeft: depth * 20 }}>
      {rows.map((r) => {
        const kids = childrenOf[r.id]
        const open = expanded.has(r.id)
        const leaf = kids !== undefined && kids.length === 0
        return (
          <div key={r.id} className="addr-node">
            <div className="addr-row">
              {r.level < 5 && !leaf
                ? <button className="addr-toggle" aria-label={open ? a.collapse : a.expand}
                  onClick={() => onToggle(r)}>{open ? '−' : '+'}</button>
                : <span className="addr-toggle addr-dot">·</span>}
              <span className="addr-name">{r.name}</span>
              <span className="geo-tag">{r.level}</span>
              <span className={`geo-tag ${r.countryCode ? '' : 'geo-tag-off'}`}>
                {countryName(r.countryCode)}
              </span>
              {r.adminCode && <span className="geo-tag">{r.adminCode}</span>}
              <span className="spacer" />
              <div className="geo-act">
                {r.level === 1 && <button onClick={() => onAttach(r)}>{a.attach}</button>}
                {r.level < 5 && <><span className="sep">|</span>
                  <button onClick={() => onAddChild(r)}>{a.addChild}</button></>}
                <span className="sep">|</span>
                <button onClick={() => onRename(r)}>{a.rename}</button>
                <span className="sep">|</span>
                <button onClick={() => onDelete(r)}>{a.delete}</button>
              </div>
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
function filterRows(rows: AddressRow[], kw: string, childrenOf: Record<number, AddressRow[]>): AddressRow[] {
  if (!kw) return rows
  const k = kw.toLowerCase()
  return rows.filter((r) => {
    const hit = [r.name, r.countryCode, r.adminCode].some((s) => s.toLowerCase().includes(k))
    return hit || filterRows(childrenOf[r.id] ?? [], kw, childrenOf).length > 0
  })
}
