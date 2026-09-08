// 行政区划维护面板:卡片化列表 + 工具栏(国家筛选/搜索/主操作) + 抽屉式表单与译名。
// 表单在 SubdivForm.tsx,译名在 SubdivNames.tsx,共享类型在 subdiv-shared.ts。
import { useCallback, useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Dropdown } from '../../../components/Dropdown'
import { Pagination } from '../../../components/Pagination'
import { useQueryInt, useQueryState } from '../../../lib/useQueryState'
import { ErrorBanner, EmptyState, ToolbarButton } from '../../../components/business/page-head'
import { Input } from '../../../components/ui/input'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { StatusTag } from './CountryPanel'
import type { CountryRow } from './CountryForm'
import { CARD, TOOLBAR, SPACER, TABLE_WRAP, FOOTER } from './styles'
import { SubdivForm } from './SubdivForm'
import { SubdivNames } from './SubdivNames'
import { EMPTY, type SubdivRow } from './subdiv-shared'

export function SubdivisionPanel() {
  const t = useT()
  const g = t.pages.geo
  const [countries, setCountries] = useState<CountryRow[]>([])
  const [urlCountry, setUrlCountry] = useQueryState('country', '')
  const [country, setCountry] = useState(urlCountry)
  const [rows, setRows] = useState<SubdivRow[]>([])
  const [error, setError] = useState('')
  const [urlKeyword, setUrlKeyword] = useQueryState('kw', '')
  const [keyword, setKeyword] = useState(urlKeyword)
  const [draftKeyword, setDraftKeyword] = useState(urlKeyword)
  const [urlPage, setUrlPage] = useQueryInt('page', 1)
  const [page, setPage] = useState(urlPage)
  const [urlPageSize, setUrlPageSize] = useQueryInt('size', 20)
  const [pageSize, setPageSize] = useState(urlPageSize)
  const [form, setForm] = useState<SubdivRow | null>(null)
  const [editing, setEditing] = useState(false)
  const [namesOf, setNamesOf] = useState<string | null>(null)

  const filterCountry = (v: string) => {
    setCountry(v)
    setUrlCountry(v)
    setPage(1)
    setUrlPage(1)
  }
  const search = (v: string) => {
    setDraftKeyword(v)
    setKeyword(v)
    setUrlKeyword(v)
    setPage(1)
    setUrlPage(1)
  }
  const resize = (v: number) => {
    setPageSize(v)
    setUrlPageSize(v)
    setPage(1)
    setUrlPage(1)
  }

  useEffect(() => {
    apiFetch<CountryRow[]>('/geo/countries')
      .then((d) => setCountries(d ?? []))
      .catch(() => setError(g.loadFail))
  }, [g])

  const load = useCallback(() => {
    apiFetch<SubdivRow[]>(`/geo/subdivisions${country ? `?country=${country}` : ''}`)
      .then((d) => setRows(d ?? []))
      .catch(() => setError(g.loadFail))
  }, [country, g])

  useEffect(load, [load])

  const toggle = async (row: SubdivRow) => {
    try {
      await apiFetch(`/geo/subdivisions/${row.code}/active`, {
        method: 'PUT', body: { active: !row.isActive },
      })
    } catch (e) {
      setError(e instanceof Error ? e.message : g.saveFail)
      return
    }
    load()
  }

  const kw = keyword.trim().toLowerCase()
  const filtered = kw
    ? rows.filter((r) => [r.code, r.category, r.displayName].some((s) => s.toLowerCase().includes(kw)))
    : rows
  const pageCount = Math.max(1, Math.ceil(filtered.length / pageSize))
  const safePage = Math.min(page, pageCount)
  const paged = filtered.slice((safePage - 1) * pageSize, safePage * pageSize)

  return (
    <div className={CARD}>
      {error && <ErrorBanner message={error} className="mt-3" />}
      <div className={TOOLBAR}>
        <Dropdown
          value={country}
          ariaLabel={g.filterCountry}
          onChange={filterCountry}
          triggerStyle={{ width: 180 }}
          options={[
            { value: '', label: g.filterCountry },
            ...countries.map((c) => ({
              value: c.alpha2,
              label: `${c.alpha2} ${c.displayName}`,
            })),
          ]}
        />
        <Input className="w-50" placeholder={g.searchPlaceholder}
          value={draftKeyword} onChange={(e) => search(e.target.value)} />
        <div className={SPACER} />
        <ToolbarButton primary
          onClick={() => { setForm({ ...EMPTY, countryCode: country }); setEditing(false) }}>
          + {g.add}
        </ToolbarButton>
      </div>
      <SubdivTable rows={paged} onEdit={(r) => { setForm({ ...r }); setEditing(true) }}
        onToggle={toggle} onNames={(code) => setNamesOf(namesOf === code ? null : code)} />
      <div className={FOOTER}>
        <Pagination page={safePage} pageSize={pageSize} total={filtered.length}
          onPage={(v) => { setPage(v); setUrlPage(v) }} onSize={resize} rangeText={g.rangeText}
          prevText={g.prev} nextText={g.next} perPageText={g.perPage}
          jumpText={g.jumpText} pageUnitText={g.pageUnit} />
      </div>
      {form && <SubdivForm initial={form} editing={editing} country={country}
        onDone={() => { setForm(null); load() }} onCancel={() => setForm(null)} />}
      {namesOf && <SubdivNames code={namesOf} onClose={() => setNamesOf(null)} />}
    </div>
  )
}

// SubdivTable 区划列表(design-spec §4.2)。
function SubdivTable({ rows, onEdit, onToggle, onNames }: {
  rows: SubdivRow[]
  onEdit: (r: SubdivRow) => void
  onToggle: (r: SubdivRow) => void
  onNames: (code: string) => void
}) {
  const g = useT().pages.geo
  if (!rows.length) return <EmptyState text={g.empty} />
  return (
    <div className={TABLE_WRAP}>
      <Table>
        <TableHeader>
          <TableRow>{g.subdivColumns.map((c) => <TableHead key={c}>{c}</TableHead>)}</TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((r) => (
            <TableRow key={r.code}>
              <TableCell className="font-semibold text-[var(--shell-heading)]">{r.code}</TableCell>
              <TableCell>{r.countryCode}</TableCell>
              <TableCell>{r.parentCode || '—'}</TableCell>
              <TableCell>{r.level}</TableCell>
              <TableCell>{r.category}</TableCell>
              <TableCell>{r.displayName}</TableCell>
              <TableCell><StatusTag on={r.isActive} /></TableCell>
              <TableCell>
                <span className="inline-flex items-center">
                  <button className="border-none bg-none px-1 text-[13px] text-[var(--color-text-link)] cursor-pointer hover:text-[var(--color-brand-gold-600)] hover:underline" onClick={() => onEdit(r)}>{g.edit}</button>
                  <span className="text-[var(--shell-side-border)]">|</span>
                  <button className="border-none bg-none px-1 text-[13px] text-[var(--color-text-link)] cursor-pointer hover:text-[var(--color-brand-gold-600)] hover:underline" onClick={() => onToggle(r)}>{r.isActive ? g.disable : g.enable}</button>
                  <span className="text-[var(--shell-side-border)]">|</span>
                  <button className="border-none bg-none px-1 text-[13px] text-[var(--color-text-link)] cursor-pointer hover:text-[var(--color-brand-gold-600)] hover:underline" onClick={() => onNames(r.code)}>{g.names}</button>
                </span>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
