import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { TableStateRow } from '../../../components/business'
import { pageSlice, type CollectionTaskRow } from '../types'
import { fmtFee } from '../../../lib/format'

export default function CollectionTasksPage() {
  const t = useT()
  const a = t.pages.arrearsPage
  const [rows, setRows] = useState<CollectionTaskRow[]>([])
  const [status, setStatus] = useState('PENDING')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  const load = () => {
    setBusy(true)
    setError('')
    apiFetch<{ items: CollectionTaskRow[] }>(`/collection-tasks?status=${status}`)
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : a.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, [status])

  const update = async (task: CollectionTaskRow, next: 'DOING' | 'DONE' | 'FAILED') => {
    setBusy(true)
    try {
      await apiFetch(`/collection-tasks/${task.id}/status`, {
        method: 'POST', body: JSON.stringify({ status: next }),
      })
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : a.actionFail)
      setBusy(false)
    }
  }
  const slice = pageSlice(rows, page, pageSize)
  return <div>
    <PageHead title={t.menu.items['collection-tasks'] ?? 'Collection tasks'} desc={a.desc} />
    <div className="mb-4 flex items-center gap-2">
      {['PENDING', 'DOING', 'DONE', 'FAILED'].map((x) => <button key={x} className={`rounded-sm border px-3 py-1.5 text-xs ${status === x ? 'border-[var(--color-brand-gold-500)] bg-[var(--shell-menu-hover-bg)]' : 'border-[var(--shell-input-border)]'}`} onClick={() => { setStatus(x); setPage(1) }}>{x}</button>)}
      <button className="ml-auto rounded-sm border border-[var(--shell-input-border)] px-3 py-1.5 text-xs" onClick={load} disabled={busy}>{t.pages.audit.refresh}</button>
    </div>
    <div className="rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-4">
      {error && <div className="mb-3 text-sm text-[var(--color-danger)]">{error}</div>}
      <div className="overflow-x-auto"><table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]"><thead><tr>{['ID', a.columns[0], a.columns[1], 'Priority', 'Status', 'Due', 'Action'].map((x) => <th key={x} className="border-b border-[var(--shell-side-border)] px-3 py-2 text-left text-xs">{x}</th>)}</tr></thead><tbody>
        {slice.map((x) => <tr key={x.id}><td className="border-b border-[var(--shell-side-border)] px-3 py-2">{x.id}</td><td className="border-b border-[var(--shell-side-border)] px-3 py-2">{x.customer || `#${x.customerId}`}</td><td className="border-b border-[var(--shell-side-border)] px-3 py-2">{fmtFee(x.amount)} / {x.days}d</td><td className="border-b border-[var(--shell-side-border)] px-3 py-2">{x.priority}</td><td className="border-b border-[var(--shell-side-border)] px-3 py-2">{x.status}</td><td className="border-b border-[var(--shell-side-border)] px-3 py-2">{x.dueAt || '—'}</td><td className="border-b border-[var(--shell-side-border)] px-3 py-2"><span className="inline-flex gap-2">{x.status === 'PENDING' && <button onClick={() => update(x, 'DOING')}>DOING</button>}{x.status === 'DOING' && <><button onClick={() => update(x, 'DONE')}>DONE</button><button onClick={() => update(x, 'FAILED')}>FAILED</button></>}</span></td></tr>)}
        {!slice.length && <TableStateRow colSpan={7} loading={busy} text={a.empty} />}
      </tbody></table></div>
      <div className="flex justify-end pt-3"><Pagination total={rows.length} page={page} pageSize={pageSize} onPage={setPage} onSize={setPageSize} {...pagerTexts(a)} /></div>
    </div>
  </div>
}
