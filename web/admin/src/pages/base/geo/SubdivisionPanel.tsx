// 行政区划维护面板:卡片化列表 + 工具栏(国家筛选/搜索/主操作) + 抽屉式表单与译名。
import { useCallback, useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { Pagination } from '../../../components/Pagination'
import { useQueryInt, useQueryState } from '../../../lib/useQueryState'
import { ErrorBanner, EmptyState, ToolbarButton } from '../../../components/business/page-head'
import { Input } from '../../../components/ui/input'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { StatusTag } from './CountryPanel'
import type { CountryRow } from './CountryForm'
import { CARD, TOOLBAR, SPACER, TABLE_WRAP, FOOTER, FORM, FIELD, FIELD_FULL, LABEL, REQ } from './styles'

export interface SubdivRow {
  code: string
  countryCode: string
  parentCode: string
  level: number
  category: string
  osmAdminLevel: number
  geonameId: number
  isActive: boolean
  displayName: string
}

interface NameRow { locale: string; name: string; nameType: string }

const EMPTY: SubdivRow = {
  code: '', countryCode: '', parentCode: '', level: 1, category: 'region',
  osmAdminLevel: 4, geonameId: 0, isActive: true, displayName: '',
}

const TAG_OFF = 'mb-1.5 inline-flex items-center justify-between rounded-[4px] border border-[color-mix(in_srgb,var(--shell-group-title)_35%,transparent)] bg-[color-mix(in_srgb,var(--shell-group-title)_10%,transparent)] px-2 py-0.5 text-xs leading-[22px] text-[var(--shell-group-title)]'

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
    await apiFetch(`/geo/subdivisions/${row.code}/active`, {
      method: 'PUT', body: { active: !row.isActive },
    }).catch(() => setError(g.saveFail))
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

// SubdivForm 区划新建/编辑抽屉表单。
function SubdivForm({ initial, editing, country, onDone, onCancel }: {
  initial: SubdivRow
  editing: boolean
  country: string
  onDone: () => void
  onCancel: () => void
}) {
  const t = useT()
  const g = t.pages.geo
  const [form, setForm] = useState(initial)
  const [error, setError] = useState('')

  const save = async () => {
    const path = editing ? `/geo/subdivisions/${form.code}` : '/geo/subdivisions'
    try {
      await apiFetch(path, { method: editing ? 'PUT' : 'POST', body: { ...form } })
      onDone()
    } catch {
      setError(g.saveFail)
    }
  }

  // 字段标签来自 i18n geo.subdivFields。
  const texts: [keyof typeof g.subdivFields, boolean][] = [
    ['code', true],
    ['countryCode', true],
    ['parentCode', false],
    ['category', true],
  ]

  return (
    <Drawer title={`${editing ? g.edit : g.add} · ${g.tabSubdiv}`} onClose={onCancel}
      footer={
        <>
          {error && <span className="mr-auto text-xs text-[var(--color-danger)]">{error}</span>}
          <ToolbarButton onClick={onCancel}>{g.cancel}</ToolbarButton>
          <ToolbarButton primary onClick={save}>{g.save}</ToolbarButton>
        </>
      }>
      <div className={FORM}>
        {texts.map(([k, req]) => (
          <div key={k} className={FIELD_FULL}>
            <label className={LABEL}>{req && <span className={REQ}>*</span>}{g.subdivFields[k]}</label>
            <Input disabled={editing && k === 'code'}
              value={form[k] as string}
              onChange={(e) => setForm({ ...form, [k]: e.target.value })} />
          </div>
        ))}
        <NumField label={g.subdivFields.level} value={form.level}
          onChange={(v) => setForm({ ...form, level: v })} />
        <NumField label={g.subdivFields.osmAdminLevel} value={form.osmAdminLevel}
          onChange={(v) => setForm({ ...form, osmAdminLevel: v })} />
        <div className={FIELD_FULL}>
          <label className={LABEL}>{g.subdivFields.geonameId}</label>
          <Input type="number" value={form.geonameId}
            onChange={(e) => setForm({ ...form, geonameId: Number(e.target.value) })} />
        </div>
      </div>
      {editing && <p className="mt-3 text-xs text-[var(--shell-group-title)]">
        {g.filterCountry}: {country || form.countryCode}
      </p>}
    </Drawer>
  )
}

// NumField 数字输入字段。
function NumField({ label, value, onChange }: {
  label: string
  value: number
  onChange: (v: number) => void
}) {
  return (
    <div className={FIELD}>
      <label className={LABEL}>{label}</label>
      <Input type="number" value={value}
        onChange={(e) => onChange(Number(e.target.value))} />
    </div>
  )
}

// SubdivNames 区划译名维护抽屉。
function SubdivNames({ code, onClose }: { code: string; onClose: () => void }) {
  const t = useT()
  const g = t.pages.geo
  const [names, setNames] = useState<NameRow[]>([])
  const [locale, setLocale] = useState('zh-Hans')
  const [name, setName] = useState('')

  const load = useCallback(() => {
    apiFetch<NameRow[]>(`/geo/subdivisions/${code}/names`)
      .then((d) => setNames(d ?? []))
      .catch(() => setNames([]))
  }, [code])

  useEffect(load, [load])

  const add = async () => {
    if (!name.trim()) return
    await apiFetch(`/geo/subdivisions/${code}/names`, {
      method: 'POST', body: { locale, name, nameType: 'STANDARD' },
    }).catch(() => undefined)
    setName('')
    load()
  }

  const remove = async (loc: string, nameType: string) => {
    await apiFetch(`/geo/subdivisions/${code}/names/${loc}/${nameType}`, { method: 'DELETE' })
    load()
  }

  return (
    <Drawer title={`${g.names} · ${code}`} onClose={onClose}>
      {names.map((n) => (
        <div key={n.locale + n.nameType} className={TAG_OFF}>
          <span>{n.locale} · {n.nameType} · {n.name}</span>
          <button className="cursor-pointer border-none bg-none text-[var(--color-danger)]"
            onClick={() => remove(n.locale, n.nameType)}>×</button>
        </div>
      ))}
      <div className="mt-3 flex gap-2">
        <Input className="w-27" value={locale}
          onChange={(e) => setLocale(e.target.value)} placeholder="locale" />
        <Input value={name}
          onChange={(e) => setName(e.target.value)} placeholder="name" />
        <ToolbarButton onClick={add}>{g.addName}</ToolbarButton>
      </div>
    </Drawer>
  )
}
