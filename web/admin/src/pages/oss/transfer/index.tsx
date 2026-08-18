// 跨区域调配页:契约 GET /transfers、POST /transfers、POST /transfers/:transferNo/approve|reject。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { pageSlice, type RegionRefRow, type TransferRow } from '../types'
import '../../org/org.css'

export default function TransferPage() {
  const t = useT()
  const r = t.pages.transferPage
  const [rows, setRows] = useState<TransferRow[]>([])
  const [regions, setRegions] = useState<RegionRefRow[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [open, setOpen] = useState(false)
  const [resourceId, setResourceId] = useState('')
  const [fromRegionId, setFromRegionId] = useState(0)
  const [toRegionId, setToRegionId] = useState(0)
  const [formError, setFormError] = useState('')

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: TransferRow[] }>('/transfers')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : r.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => {
    apiFetch<RegionRefRow[]>('/regions')
      .then((d) => setRegions(d ?? []))
      .catch(() => setRegions([]))
  }, [])

  const submit = async () => {
    if (busy) return
    setBusy(true)
    setFormError('')
    try {
      await apiFetch('/transfers', {
        method: 'POST',
        body: { resourceId: Number(resourceId), fromRegionId, toRegionId },
      })
      setOpen(false)
      setResourceId('')
      setFromRegionId(0)
      setToRegionId(0)
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : r.saveFail)
    } finally {
      setBusy(false)
    }
  }

  const review = async (transferNo: string, action: 'approve' | 'reject') => {
    if (busy) return
    const act = action === 'approve' ? r.approve : r.reject
    if (!window.confirm(r.confirm.replace('{act}', act).replace('{no}', transferNo))) return
    setBusy(true)
    try {
      await apiFetch(`/transfers/${encodeURIComponent(transferNo)}/${action}`, { method: 'POST' })
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : r.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const regionName = (id: number) => regions.find((x) => x.id === id)?.name ?? `#${id}`
  const slice = pageSlice(rows, page, pageSize)
  const resourceOk = /^\d+$/.test(resourceId) && Number(resourceId) > 0
  const regionOk = fromRegionId !== 0 && toRegionId !== 0 && fromRegionId !== toRegionId

  return (
    <div>
      <PageHead title={r.title} desc={r.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
          <button className="org-btn org-btn-primary" onClick={() => setOpen(true)}>{r.create}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{r.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((x) => (
                  <tr key={x.id}>
                    <td>{x.transferNo || `#${x.id}`}</td>
                    <td>#{x.resourceId}</td>
                    <td>{regionName(x.fromRegionId)}</td>
                    <td>{regionName(x.toRegionId)}</td>
                    <td><StatusTag domain="task" value={x.status} /></td>
                    <td>
                      {x.status === 'PENDING' ? (
                        <span className="org-act">
                          <button disabled={busy} onClick={() => review(x.transferNo, 'approve')}>{r.approve}</button>
                          <span className="sep">|</span>
                          <button disabled={busy} onClick={() => review(x.transferNo, 'reject')}>{r.reject}</button>
                        </span>
                      ) : '—'}
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={6}><div className="org-empty">{r.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(r)} />
        </div>
      </div>
      {open && (
        <Drawer title={r.createTitle} onClose={() => setOpen(false)}
          footer={
            <>
              <button className="org-btn" onClick={() => setOpen(false)}>{t.pages.company.cancel}</button>
              <button className="org-btn org-btn-primary" disabled={busy || !resourceOk || !regionOk} onClick={submit}>
                {busy ? t.pages.account.submitting : t.pages.company.save}
              </button>
            </>
          }>
          <div className="org-form">
            <div className="org-field">
              <label><span className="req">*</span>{r.fResource}</label>
              <input className="org-input" type="number" value={resourceId} placeholder={r.pResource}
                onChange={(e) => setResourceId(e.target.value)} />
              {!resourceOk && resourceId !== '' && <span className="acc-err">{r.eResource}</span>}
            </div>
            <div className="org-field">
              <label><span className="req">*</span>{r.fFrom}</label>
              <select className="org-select" value={fromRegionId ? String(fromRegionId) : ''}
                onChange={(e) => setFromRegionId(Number(e.target.value) || 0)}>
                <option value="">—</option>
                {regions.map((x) => <option key={x.id} value={x.id}>{x.name}</option>)}
              </select>
            </div>
            <div className="org-field">
              <label><span className="req">*</span>{r.fTo}</label>
              <select className="org-select" value={toRegionId ? String(toRegionId) : ''}
                onChange={(e) => setToRegionId(Number(e.target.value) || 0)}>
                <option value="">—</option>
                {regions.map((x) => <option key={x.id} value={x.id}>{x.name}</option>)}
              </select>
              {!regionOk && (fromRegionId !== 0 || toRegionId !== 0) && <span className="acc-err">{r.eRegion}</span>}
            </div>
            {formError && <div className="org-error" style={{ margin: 0 }}>{formError}</div>}
          </div>
        </Drawer>
      )}
    </div>
  )
}
