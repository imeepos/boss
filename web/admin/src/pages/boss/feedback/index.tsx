import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead } from '../../org/shared'
import { TableStateRow } from '../../../components/business'

type FeedbackRow = { id: number; workerName: string; customerName: string; ticketId: number; score: number; needReview: boolean }

export default function FeedbackPage() {
  const t = useT()
  const [rows, setRows] = useState<FeedbackRow[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const load = () => {
    setBusy(true); setError('')
    apiFetch<{ items: FeedbackRow[] }>('/worker-feedbacks')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : 'Failed to load'))
      .finally(() => setBusy(false))
  }
  useEffect(load, [])
  return <div><PageHead title={t.menu.items.feedback ?? 'Service feedback'} desc="Customer service ratings and review queue" />
    <div className="rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-4">
      <button className="mb-3 rounded-sm border border-[var(--shell-input-border)] px-3 py-1.5 text-xs" onClick={load} disabled={busy}>{t.pages.audit.refresh}</button>
      {error && <div className="mb-3 text-sm text-[var(--color-danger)]">{error}</div>}
      <div className="overflow-x-auto"><table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]"><thead><tr>{['ID', 'Customer', 'Worker', 'Ticket', 'Score', 'Review'].map((x) => <th key={x} className="border-b border-[var(--shell-side-border)] px-3 py-2 text-left text-xs">{x}</th>)}</tr></thead><tbody>{rows.map((x) => <tr key={x.id}><td className="border-b border-[var(--shell-side-border)] px-3 py-2">{x.id}</td><td className="border-b border-[var(--shell-side-border)] px-3 py-2">{x.customerName}</td><td className="border-b border-[var(--shell-side-border)] px-3 py-2">{x.workerName}</td><td className="border-b border-[var(--shell-side-border)] px-3 py-2">#{x.ticketId}</td><td className="border-b border-[var(--shell-side-border)] px-3 py-2">{x.score}/5</td><td className="border-b border-[var(--shell-side-border)] px-3 py-2">{x.needReview ? 'REVIEW' : '—'}</td></tr>)}{!rows.length && <TableStateRow colSpan={6} loading={busy} text="No feedback" />}</tbody></table></div>
    </div>
  </div>
}
