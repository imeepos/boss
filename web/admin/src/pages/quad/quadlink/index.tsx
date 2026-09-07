// 关联查询页:GET /quad-links 全量列表 + POST /quad-links 建链 + by-asset/customer/port/address 反查。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Dropdown } from '../../../components/Dropdown'
import { pageSlice, type QuadLinkRow } from '../types'
import { TableStateRow, ErrorBanner } from '../../../components/business'
import { CreateLinkDrawer } from './CreateLinkDrawer'

// 反查维度:by-asset/by-customer/by-port/by-address(后端四反查路由)。
const REVERSE_KEYS = ['by-asset', 'by-customer', 'by-port', 'by-address'] as const

export default function QuadLinkPage() {
  const t = useT()
  const q = t.pages.quadLinkPage
  const [rows, setRows] = useState<QuadLinkRow[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const [reverseBy, setReverseBy] = useState('by-asset')
  const [reverseId, setReverseId] = useState('')
  const [reverseNote, setReverseNote] = useState('')

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: QuadLinkRow[] }>('/quad-links')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : q.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  // 反查:命中把结果行置顶展示,未命中提示不臆造。
  const reverse = async () => {
    const id = Number(reverseId)
    if (!Number.isInteger(id) || id <= 0) return
    setError('')
    setBusy(true)
    setReverseNote('')
    try {
      const param = { 'by-asset': 'assetId', 'by-customer': 'customerId', 'by-port': 'portId', 'by-address': 'addressId' }[reverseBy as typeof REVERSE_KEYS[number]]
      const d = await apiFetch<QuadLinkRow | null>(`/quad-links/${reverseBy}`, { query: { [param]: id } })
      if (d && (d as QuadLinkRow).id) { setRows([(d as QuadLinkRow), ...rows.filter((x) => x.id !== (d as QuadLinkRow).id)]); setPage(1) }
      else setReverseNote(q.reverseEmpty)
    } catch (e) {
      setError(e instanceof Error ? e.message : q.loadFail)
    } finally {
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={q.title} desc={q.desc} />
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => setCreateOpen(true)}>{q.createBtn}</button>
          <span className="flex-1" />
          <div className="w-36">
            <Dropdown value={reverseBy} ariaLabel={q.reverseLabel} onChange={setReverseBy} options={REVERSE_KEYS.map((k) => ({ value: k, label: q.columns[REVERSE_KEYS.indexOf(k)] ?? k }))} />
          </div>
          <input className="h-8 w-32 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" type="number" placeholder={q.reverseIdPh} value={reverseId} onChange={(e) => setReverseId(e.target.value)} onKeyDown={(e) => { if (e.key === 'Enter') reverse() }} />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-3 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={reverse}>{q.reverseLabel}</button>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <ErrorBanner message={error} /> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{q.columns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((x) => (
                  <tr key={x.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><span className="font-mono text-xs text-[var(--shell-group-title)]">#{x.id}</span></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><span className="font-mono text-xs text-[var(--shell-group-title)]">#{x.assetId}</span></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><span className="font-mono text-xs text-[var(--shell-group-title)]">#{x.customerId}</span></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><span className="font-mono text-xs text-[var(--shell-group-title)]">#{x.portId}</span></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><span className="font-mono text-xs text-[var(--shell-group-title)]">#{x.addressId}</span></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.legalEntityName || `#${x.legalEntityId}`}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="quad" value={x.status} /></td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={7} loading={busy} text={q.empty} />}
              </tbody>
            </table>
          </div>
        )}
        {reverseNote && <p className="mx-4 mb-3 text-[13px] text-[var(--shell-group-title)]">{reverseNote}</p>}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(q)} />
        </div>
      </div>
      {createOpen && <CreateLinkDrawer onDone={() => { setCreateOpen(false); load() }} onClose={() => setCreateOpen(false)} />}
    </div>
  )
}