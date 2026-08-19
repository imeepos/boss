// 盘点管理页:契约 GET /stocktakes、POST /stocktakes、POST /stocktakes/:taskId/diff-handle。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { pageSlice, type StocktakeRow } from '../types'
import '../../org/org.css'

export default function StockPage() {
  const t = useT()
  const s = t.pages.stock
  const [rows, setRows] = useState<StocktakeRow[]>([])
  const [companies, setCompanies] = useState<{ id: number; name: string }[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [open, setOpen] = useState(false)
  const [legalEntityId, setLegalEntityId] = useState(0)
  const [scope, setScope] = useState('')
  const [formError, setFormError] = useState('')

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: StocktakeRow[] }>('/stocktakes')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : s.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => {
    apiFetch<{ id: number; name: string }[]>('/legal-entities')
      .then((d) => setCompanies(d ?? []))
      .catch(() => setCompanies([]))
  }, [])

  const submit = async () => {
    if (busy) return
    setBusy(true)
    setFormError('')
    try {
      await apiFetch('/stocktakes', {
        method: 'POST',
        body: { legalEntityId, scope: scope.trim() },
      })
      setOpen(false)
      setScope('')
      setLegalEntityId(0)
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : s.saveFail)
    } finally {
      setBusy(false)
    }
  }

  const diffHandle = async (taskId: number) => {
    if (busy || !window.confirm(s.diffConfirm)) return
    setBusy(true)
    try {
      await apiFetch(`/stocktakes/${taskId}/diff-handle`, { method: 'POST' })
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : s.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, page, pageSize)
  const scopeOk = scope.trim().length > 0 && scope.trim().length <= 64

  return (
    <div>
      <PageHead title={s.title} desc={s.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
          <button className="org-btn org-btn-primary" onClick={() => setOpen(true)}>{s.create}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{s.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.id}>
                    <td>#{r.id}</td>
                    <td>{r.scope}</td>
                    <td>{s.progress.replace('{n}', String(r.progress))}</td>
                    <td>{r.diffCount}</td>
                    <td><StatusTag domain="task" value={r.status} /></td>
                    <td>
                      {r.status === 'DOING' ? (
                        <span className="org-act">
                          <button disabled={busy} onClick={() => diffHandle(r.id)}>{s.diffHandle}</button>
                        </span>
                      ) : '—'}
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={6}><div className="org-empty">{s.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(s)} />
        </div>
      </div>
      {open && (
        <Drawer title={s.createTitle} onClose={() => setOpen(false)}
          footer={
            <>
              <button className="org-btn" onClick={() => setOpen(false)}>{t.pages.company.cancel}</button>
              <button className="org-btn org-btn-primary" disabled={busy || !legalEntityId || !scopeOk} onClick={submit}>
                {busy ? t.pages.account.submitting : t.pages.company.save}
              </button>
            </>
          }>
          <div className="org-form">
            <div className="org-field">
              <label><span className="req">*</span>{s.fCompany}</label>
              <select className="org-select" value={legalEntityId ? String(legalEntityId) : ''}
                onChange={(e) => setLegalEntityId(Number(e.target.value) || 0)}>
                <option value="">{s.pCompany}</option>
                {companies.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
              </select>
            </div>
            <div className="org-field">
              <label><span className="req">*</span>{s.fScope}</label>
              <input className="org-input" value={scope} placeholder={s.pScope}
                onChange={(e) => setScope(e.target.value)} />
              {!scopeOk && scope !== '' && <span className="text-[11px] text-[var(--color-danger)]">{s.eScope}</span>}
            </div>
            {formError && <div className="org-error" style={{ margin: 0 }}>{formError}</div>}
          </div>
        </Drawer>
      )}
    </div>
  )
}
