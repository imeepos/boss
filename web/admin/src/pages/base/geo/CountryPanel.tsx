// 国家维护面板:卡片化列表 + 工具栏(搜索/主操作) + 抽屉式新建编辑与详情。
import { useCallback, useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { useQueryInt, useQueryState } from '../../../lib/useQueryState'
import { Pagination } from '../../../components/Pagination'
import { ErrorBanner, EmptyState, ToolbarButton } from '../../../components/business/page-head'
import { Badge } from '../../../components/ui/badge'
import { Input } from '../../../components/ui/input'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { CountryForm, type CountryRow, EMPTY } from './CountryForm'
import { CountryDetail, type CountryDetailData } from './CountryDetail'
import { CARD, TOOLBAR, SPACER, TABLE_WRAP, FOOTER } from './styles'

export function CountryPanel() {
  const t = useT()
  const g = t.pages.geo
  const [rows, setRows] = useState<CountryRow[]>([])
  const [error, setError] = useState('')
  const [urlKeyword, setUrlKeyword] = useQueryState('kw', '')
  const [keyword, setKeyword] = useState(urlKeyword)
  const [draftKeyword, setDraftKeyword] = useState(urlKeyword)
  const [urlPage, setUrlPage] = useQueryInt('page', 1)
  const [page, setPage] = useState(urlPage)
  const [urlPageSize, setUrlPageSize] = useQueryInt('size', 20)
  const [pageSize, setPageSize] = useState(urlPageSize)
  const [form, setForm] = useState<CountryRow | null>(null)
  const [editing, setEditing] = useState(false)
  const [detail, setDetail] = useState<CountryDetailData | null>(null)

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

  const load = useCallback(() => {
    apiFetch<CountryRow[]>('/geo/countries')
      .then((d) => setRows(d ?? []))
      .catch(() => setError(g.loadFail))
  }, [g])

  useEffect(load, [load])

  const openDetail = async (alpha2: string) => {
    const d = await apiFetch<CountryDetailData>(`/geo/countries/${alpha2}`).catch(() => null)
    setDetail(d ?? null)
    if (!d) setError(g.loadFail)
  }

  const toggle = async (row: CountryRow) => {
    await apiFetch(`/geo/countries/${row.alpha2}/active`, {
      method: 'PUT',
      body: { active: !row.isActive },
    }).catch(() => setError(g.saveFail))
    load()
  }

  const kw = keyword.trim().toLowerCase()
  const filtered = kw
    ? rows.filter((r) =>
        [r.alpha2, r.alpha3, r.shortName, r.displayName].some((s) => s.toLowerCase().includes(kw)))
    : rows
  const pageCount = Math.max(1, Math.ceil(filtered.length / pageSize))
  const safePage = Math.min(page, pageCount)
  const paged = filtered.slice((safePage - 1) * pageSize, safePage * pageSize)

  return (
    <div className={CARD}>
      {error && <ErrorBanner role="alert" message={error} className="mt-3" />}
      <div className={TOOLBAR}>
        <Input
          className="w-60"
          placeholder={g.searchPlaceholder}
          value={draftKeyword}
          onChange={(e) => search(e.target.value)}
        />
        <div className={SPACER} />
        <ToolbarButton primary onClick={() => { setForm({ ...EMPTY }); setEditing(false) }}>
          + {g.add}
        </ToolbarButton>
      </div>
      <CountryTable rows={paged} onEdit={(r) => { setForm({ ...r }); setEditing(true) }}
        onToggle={toggle} onDetail={openDetail} />
      <div className={FOOTER}>
        <Pagination page={safePage} pageSize={pageSize} total={filtered.length}
          onPage={(v) => { setPage(v); setUrlPage(v) }} onSize={resize} rangeText={g.rangeText}
          prevText={g.prev} nextText={g.next} perPageText={g.perPage}
          jumpText={g.jumpText} pageUnitText={g.pageUnit} />
      </div>
      {form && <CountryForm initial={form} editing={editing}
        onDone={() => { setForm(null); load() }} onCancel={() => setForm(null)} />}
      {detail && (
        <CountryDetail
          data={detail}
          onChanged={() => openDetail(detail.alpha2)}
          onClose={() => setDetail(null)}
        />
      )}
    </div>
  )
}

// CountryTable 国家列表;启停按语义 Tag 呈现(design-spec §4.2 表格规格)。
function CountryTable({ rows, onEdit, onToggle, onDetail }: {
  rows: CountryRow[]
  onEdit: (r: CountryRow) => void
  onToggle: (r: CountryRow) => void
  onDetail: (alpha2: string) => void
}) {
  const g = useT().pages.geo
  if (!rows.length) return <EmptyState className="py-8" text={g.empty} />
  return (
    <div className={TABLE_WRAP}>
      <Table>
        <TableHeader>
          <TableRow>{g.countryColumns.map((c) => <TableHead key={c}>{c}</TableHead>)}</TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((r) => (
            <TableRow key={r.alpha2}>
              <TableCell className="font-semibold text-[var(--shell-heading)]">{r.alpha2}</TableCell>
              <TableCell>{r.alpha3}</TableCell>
              <TableCell>{r.numericCode}</TableCell>
              <TableCell>{r.displayName}</TableCell>
              <TableCell>{r.continentCode}</TableCell>
              <TableCell><StatusTag on={r.isActive} /></TableCell>
              <TableCell>
                <span className="inline-flex items-center">
                  <ActBtn label={g.edit} onClick={() => onEdit(r)} />
                  <ActBtn label={r.isActive ? g.disable : g.enable} onClick={() => onToggle(r)} />
                  <ActBtn label={g.detail} onClick={() => onDetail(r.alpha2)} last />
                </span>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}

function ActBtn({ label, onClick, last }: { label: string; onClick: () => void; last?: boolean }) {
  return (
    <>
      {!last && <span className="text-[var(--shell-side-border)]">|</span>}
      <button className="border-none bg-none px-1 text-[13px] text-[var(--color-text-link)] cursor-pointer hover:text-[var(--color-brand-gold-600)] hover:underline" onClick={onClick}>
        {label}
      </button>
    </>
  )
}

// StatusTag 启用/停用语义标签。
export function StatusTag({ on }: { on: boolean }) {
  const g = useT().pages.geo
  return <Badge variant={on ? 'success' : 'default'}>{on ? g.active : g.inactive}</Badge>
}
