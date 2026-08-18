// 对账与告警页:契约 GET /quad-conflicts + POST /quad-conflicts/:id/resolve + POST /quad-links/reconcile。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { pageSlice, type QuadLinkRow } from '../types'
import '../../org/org.css'

interface ReconReport { Total: number; Linked: number; Conflict: number; Unlinked: number }

export default function QuadCheckPage() {
  const t = useT()
  const c = t.pages.quadCheckPage
  const [rows, setRows] = useState<QuadLinkRow[]>([])
  const [error, setError] = useState('')
  const [hint, setHint] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: QuadLinkRow[] }>('/quad-conflicts')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : c.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const reconcile = async () => {
    if (busy || !window.confirm(c.reconcileConfirm)) return
    setBusy(true)
    setHint('')
    try {
      const rep = await apiFetch<ReconReport>('/quad-links/reconcile', { method: 'POST' })
      setHint(c.reconcileDone
        .replace('{total}', String(rep?.Total ?? 0))
        .replace('{linked}', String(rep?.Linked ?? 0))
        .replace('{conflict}', String(rep?.Conflict ?? 0))
        .replace('{unlinked}', String(rep?.Unlinked ?? 0)))
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : c.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const resolve = async (id: number) => {
    if (busy || !window.confirm(c.resolveConfirm)) return
    setBusy(true)
    try {
      await apiFetch(`/quad-conflicts/${id}/resolve`, { method: 'POST' })
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : c.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={c.title} desc={c.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
          <button className="org-btn org-btn-primary" disabled={busy} onClick={reconcile}>{c.reconcile}</button>
        </div>
        {hint && <div className="org-hint" style={{ padding: '4px 12px', color: '#1677ff', fontSize: 13 }}>{hint}</div>}
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{c.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((x) => (
                  <tr key={x.id}>
                    <td>#{x.id}</td>
                    <td>#{x.assetId}</td>
                    <td>#{x.customerId}</td>
                    <td>#{x.portId}</td>
                    <td>#{x.addressId}</td>
                    <td><StatusTag domain="quad" value={x.status} /></td>
                    <td>
                      <span className="org-act">
                        <button disabled={busy} onClick={() => resolve(x.id)}>{c.resolve}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={7}><div className="org-empty">{c.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(c)} />
        </div>
      </div>
    </div>
  )
}
