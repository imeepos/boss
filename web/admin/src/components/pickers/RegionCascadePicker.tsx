// RegionCascadePicker（W-0904-UI U3）：国家 + 区划多级级联选择器。
// 懒加载下钻（parentCode）、每级本地搜索、关键字直搜（keyword 跨层级直达）、
// 默认国家自动选中（空值兜底 PH）、受控 value + 回显展开。列模型 rows[0]=一级区划，
// rows[j]=path[j-1] 之子；列组件见 RegionCascadeColumns.tsx；数据逻辑见 regionCascadeCore.ts。
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useT } from '../../i18n'
import { Input } from '../ui/input'
import {
  DEFAULT_COUNTRY_FALLBACK, DIRECT_SEARCH_DEBOUNCE_MS, RegionCascadeSource, debounce,
  type CountryLite, type FetchLike, type RegionSelection, type SubdivRow,
} from './regionCascadeCore'
import { CountryColumn, DirectResults, LevelColumn } from './RegionCascadeColumns'

export type { RegionSelection }

export interface RegionCascadePickerProps {
  /** 受控值：已存区划 code；编辑态据此逐级反查展开路径。 */
  value?: string
  /** 显式国家；未传时读默认国家端点（空值兜底 PH）。 */
  countryCode?: string
  onChange?: (sel: RegionSelection) => void
  disabled?: boolean
  /** 测试注入的数据源实现。 */
  fetchImpl?: FetchLike
}

const MAX_LEVELS = 4

export function RegionCascadePicker({ value, countryCode, onChange, disabled, fetchImpl }: RegionCascadePickerProps) {
  const rc = useT().pages.pickers.regionCascade
  const source = useMemo(() => new RegionCascadeSource(fetchImpl), [fetchImpl])
  const [countries, setCountries] = useState<CountryLite[]>([])
  const [country, setCountry] = useState<string>(countryCode ?? '')
  const [countryKw, setCountryKw] = useState('')
  const [path, setPath] = useState<SubdivRow[]>([])
  const [rows, setRows] = useState<SubdivRow[][]>([])
  const [kwOf, setKwOf] = useState<string[]>([])
  const [sel, setSel] = useState<RegionSelection | null>(null)
  const [error, setError] = useState('')
  const [directKw, setDirectKw] = useState('')
  const [directRows, setDirectRows] = useState<SubdivRow[]>([])
  const [directBusy, setDirectBusy] = useState(false)
  const booted = useRef(false)
  const colParent = useRef<(string | null)[]>([])
  const [expandedFor, setExpandedFor] = useState<string | null>(null)

  // 国家列表（地址页既有端点，menu:geo 门禁由页面持有）。
  useEffect(() => {
    let alive = true
    source.countries()
      .then((list) => { if (alive) setCountries(list) })
      .catch((err) => { if (alive) { console.warn('[region-picker] countries load failed:', err); setError(rc.loadFail) } })
    return () => { alive = false }
  }, [source, rc.loadFail])

  // 默认国家 boot：未显式传 countryCode 时读端点，空值/失败兜底 PH（契约 N3）。
  useEffect(() => {
    if (booted.current) return
    let alive = true
    const boot = async () => {
      const cc = countryCode && countryCode.trim() ? countryCode : await source.defaultCountry()
      if (alive) { setCountry(cc || DEFAULT_COUNTRY_FALLBACK); booted.current = true }
    }
    boot().catch((err) => {
      if (!alive) return
      console.warn('[region-picker] default country boot failed:', err)
      setCountry(DEFAULT_COUNTRY_FALLBACK)
      booted.current = true
    })
    return () => { alive = false }
  }, [countryCode, source])

  // 国家确定/切换：重置路径与列缓存，并懒加载一级区划（rows[0]）。
  useEffect(() => {
    if (!country) return
    let alive = true
    setPath([])
    setRows([])
    setKwOf([])
    setCountryKw('')
    colParent.current = [country]
    setSel(null)
    source.children(country, country)
      .then((list) => { if (alive) { setRows([list]); colParent.current = [country] } })
      .catch((err) => {
        if (!alive) return
        console.warn('[region-picker] level-1 load failed:', err)
        setError(rc.loadFail)
        setRows([[]])
      })
    return () => { alive = false }
  }, [country, source, rc.loadFail])

  // 编辑回显：value 变化且未展开过 → resolvePath 反查链并铺开路径（契约 N4）。
  useEffect(() => {
    if (!value || !booted.current || expandedFor === value) return
    setExpandedFor(value)
    const cc = country || countryCode || DEFAULT_COUNTRY_FALLBACK
    let alive = true
    source.resolvePath(cc, value)
      .then((chain) => {
        if (!alive) return
        if (!chain.length) {
          console.warn('[region-picker] edit value not found by reverse lookup:', value)
          setError(rc.loadFail)
          return
        }
        setPath(chain)
        setSel({ countryCode: cc, code: value, names: [countryName(cc), ...chain.map((n) => n.name)] })
      })
      .catch((err) => { if (alive) { console.warn('[region-picker] edit expansion failed:', err); setError(rc.loadFail) } })
    return () => { alive = false }
  }, [value, booted, country, countryCode, source, expandedFor, rc.loadFail])

  // 懒加载下钻：按 path 逐层补齐子级列（colParent 记父码，同父不重复拉取）。
  useEffect(() => {
    if (!country) return
    let alive = true
    for (let j = 1; j <= path.length && j <= MAX_LEVELS; j += 1) {
      const parentCode = path[j - 1].code
      if (colParent.current[j] === parentCode) continue
      colParent.current[j] = parentCode
      source.children(parentCode, country)
        .then((list) => { if (alive) setRows((prev) => { const next = [...prev]; next[j] = list; return next }) })
        .catch((err) => {
          if (!alive) return
          console.warn('[region-picker] column load failed:', err)
          setError(rc.loadFail)
          setRows((prev) => { const next = [...prev]; next[j] = []; return next })
        })
    }
    return () => { alive = false }
  }, [path, country, source, rc.loadFail])

  // 关键字直搜（契约 N2）：防抖 300ms 后走 keyword 参数，跨层级平铺结果。
  const runDirect = useCallback((kw: string) => {
    const k = kw.trim()
    if (!k) { setDirectRows([]); return }
    if (!country) return
    setDirectBusy(true)
    source.search(country, k)
      .then((list) => setDirectRows(list ?? []))
      .catch((err) => { console.warn('[region-picker] direct search failed:', err); setError(rc.searchFail); setDirectRows([]) })
      .finally(() => setDirectBusy(false))
  }, [country, source, rc.searchFail])
  const directDebounced = useMemo(() => debounce(runDirect, DIRECT_SEARCH_DEBOUNCE_MS), [runDirect])
  useEffect(() => () => directDebounced.cancel(), [directDebounced])

  const countryName = useCallback((code: string) => {
    const hit = countries.find((c) => c.alpha2 === code)
    return hit ? `${code} ${hit.displayName}` : code
  }, [countries])

  const emit = useCallback((next: RegionSelection) => { setSel(next); onChange?.(next) }, [onChange])

  const selectCountry = useCallback((code: string) => {
    if (code === country) return
    setCountry(code)
    setExpandedFor(value ?? '')
    setSel(null)
    onChange?.({ countryCode: code, code: '', names: [] })
  }, [country, value, onChange])

  // 点选 path 位置 i 的节点：截断更深列，下一列由 path 效应加载，回传各级名称（含国家名）。
  const selectRow = useCallback((i: number, node: SubdivRow) => {
    const names = [countryName(country), ...path.slice(0, i).map((p) => p.name), node.name]
    setPath((prev) => [...prev.slice(0, i), node])
    setRows((prev) => prev.slice(0, i + 1))
    setKwOf((prev) => prev.slice(0, i + 1))
    colParent.current = colParent.current.slice(0, i + 1)
    setExpandedFor(node.code)
    emit({ countryCode: country, code: node.code, names })
  }, [country, path, countryName, emit])

  const clear = useCallback(() => {
    setSel(null)
    setExpandedFor(value ?? '')
    onChange?.({ countryCode: country, code: '', names: [] })
  }, [country, onChange, value])

  // 直搜点选：反查完整链（回显各级名称），并按链铺开列。
  const pickDirect = useCallback((node: SubdivRow) => {
    const cc = country
    setExpandedFor(node.code)
    setDirectKw('')
    setDirectRows([])
    source.resolvePath(cc, node.code)
      .then((chain) => {
        if (chain.length) { setPath(chain); colParent.current = [cc] }
        emit({ countryCode: cc, code: node.code, names: [countryName(cc), ...(chain.length ? chain.map((n) => n.name) : [node.name])] })
      })
      .catch((err) => {
        console.warn('[region-picker] direct pick path failed:', err)
        emit({ countryCode: cc, code: node.code, names: [countryName(cc), node.name] })
      })
  }, [country, source, emit])

  const kw = directKw.trim()
  const selCode = sel?.code ?? value ?? ''
  return (
    <div className="flex flex-col gap-2" aria-label={rc.aria}>
      {error && <p className="text-xs text-[var(--color-danger)]">{error}</p>}
      <Input value={directKw} disabled={disabled} placeholder={rc.directSearch}
        onChange={(e) => { setDirectKw(e.target.value); directDebounced(e.target.value) }} />
      {sel && sel.code && (
        <div className="flex items-center justify-between gap-2 text-xs text-[var(--shell-content-text)]">
          <span className="truncate">{sel.names.join(' / ')}</span>
          <button type="button" disabled={disabled} onClick={clear}
            className="border-none bg-none px-1 text-[12px] text-[var(--color-text-link)] cursor-pointer hover:underline">{rc.clear}</button>
        </div>
      )}
      {kw !== ''
        ? <DirectResults rows={directRows} busy={directBusy} loadingText={rc.loading} emptyText={rc.empty}
          onPick={pickDirect} disabled={disabled} />
        : (
          <div className="flex max-w-full gap-2 overflow-x-auto pb-1">
            <CountryColumn countries={countries} value={country} kw={countryKw} label={rc.country}
              placeholder={rc.levelSearch} disabled={disabled}
              onKw={setCountryKw} onSelect={selectCountry} />
            {rows[0] && (
              <LevelColumn key={`${country}-l1`} label={countryName(country)} rows={rows[0]} loading={false}
                kw={kwOf[0] ?? ''} selected={path[0]?.code ?? selCode}
                placeholder={rc.levelSearch} emptyText={rc.empty} loadingText={rc.loading} disabled={disabled}
                onKw={(v) => setKwOf((prev) => { const next = [...prev]; next[0] = v; return next })}
                onSelect={(n) => selectRow(0, n)} />
            )}
            {path.map((node, i) => (
              <LevelColumn key={`${country}-${node.code}`} label={node.name} rows={rows[i + 1] ?? []}
                loading={rows[i + 1] === undefined} kw={kwOf[i + 1] ?? ''} selected={path[i + 1]?.code ?? selCode}
                placeholder={rc.levelSearch} emptyText={rc.empty} loadingText={rc.loading} disabled={disabled}
                onKw={(v) => setKwOf((prev) => { const next = [...prev]; next[i + 1] = v; return next })}
                onSelect={(n) => selectRow(i + 1, n)} />
            ))}
          </div>
        )}
    </div>
  )
}

