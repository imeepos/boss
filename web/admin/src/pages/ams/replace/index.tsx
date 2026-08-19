// 设备更换单页:契约 GET /replacements、POST /replacements(单号后端自动生成 RPL-*)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { pageSlice, PRIORITIES, type ReplacementRow } from '../types'
import '../../org/org.css'

export default function ReplacePage() {
  const t = useT()
  const r = t.pages.replacePage
  const [rows, setRows] = useState<ReplacementRow[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [open, setOpen] = useState(false)
  const [assetId, setAssetId] = useState('')
  const [reason, setReason] = useState('')
  const [priority, setPriority] = useState<string>('MEDIUM')
  const [formError, setFormError] = useState('')

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: ReplacementRow[] }>('/replacements')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : r.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const submit = async () => {
    if (busy) return
    setBusy(true)
    setFormError('')
    try {
      await apiFetch('/replacements', {
        method: 'POST',
        body: { assetId: Number(assetId), reason: reason.trim(), priority },
      })
      setOpen(false)
      setAssetId('')
      setReason('')
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : r.saveFail)
    } finally {
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, page, pageSize)
  const priorityLabel = (v: string) => r.priorities[PRIORITIES.indexOf(v as typeof PRIORITIES[number])] ?? v
  const assetOk = /^\d+$/.test(assetId) && Number(assetId) > 0

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
                    <td>{x.replacementNo || `#${x.id}`}</td>
                    <td>#{x.assetId}</td>
                    <td>{x.reason || '—'}</td>
                    <td>{priorityLabel(x.priority)}</td>
                    <td><StatusTag domain="task" value={x.status} /></td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={5}><div className="org-empty">{r.empty}</div></td></tr>}
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
              <button className="org-btn org-btn-primary" disabled={busy || !assetOk} onClick={submit}>
                {busy ? t.pages.account.submitting : t.pages.company.save}
              </button>
            </>
          }>
          <div className="org-form">
            <div className="org-field">
              <label><span className="req">*</span>{r.fAsset}</label>
              <input className="org-input" type="number" value={assetId} placeholder={r.pAsset}
                onChange={(e) => setAssetId(e.target.value)} />
              {!assetOk && assetId !== '' && <span className="text-[11px] text-[var(--color-danger)]">{r.eAsset}</span>}
            </div>
            <div className="org-field">
              <label>{r.fReason}</label>
              <input className="org-input" value={reason} placeholder={r.pReason}
                onChange={(e) => setReason(e.target.value)} />
            </div>
            <div className="org-field">
              <label>{r.fPriority}</label>
              <select className="org-select" value={priority} onChange={(e) => setPriority(e.target.value)}>
                {PRIORITIES.map((p, i) => <option key={p} value={p}>{r.priorities[i]}</option>)}
              </select>
            </div>
            {formError && <div className="org-error" style={{ margin: 0 }}>{formError}</div>}
          </div>
        </Drawer>
      )}
    </div>
  )
}
