// 拆机管理页:契约 GET /dismantles(裸列表)+ POST /dismantles(无 json tag,键为 Go 字段名)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { pageSlice, type DismantleRow } from '../types'
import '../../org/org.css'

export default function DismantlePage() {
  const t = useT()
  const d = t.pages.dismantlePage
  const [rows, setRows] = useState<DismantleRow[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [open, setOpen] = useState(false)
  const [orderId, setOrderId] = useState('')
  const [assetId, setAssetId] = useState('')
  const [portId, setPortId] = useState('')
  const [formError, setFormError] = useState('')

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<DismantleRow[]>('/dismantles')
      .then((x) => setRows(Array.isArray(x) ? x : []))
      .catch((e) => setError(e instanceof Error ? e.message : d.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const submit = async () => {
    if (busy) return
    setBusy(true)
    setFormError('')
    try {
      await apiFetch('/dismantles', {
        method: 'POST',
        body: { OrderID: Number(orderId), AssetID: Number(assetId) || 0, PortID: Number(portId) || 0 },
      })
      setOpen(false)
      setOrderId('')
      setAssetId('')
      setPortId('')
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : d.saveFail)
    } finally {
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, page, pageSize)
  const idOk = (v: string) => v === '' || (/^\d+$/.test(v) && Number(v) > 0)
  const orderOk = /^\d+$/.test(orderId) && Number(orderId) > 0

  return (
    <div>
      <PageHead title={d.title} desc={d.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
          <button className="org-btn org-btn-primary" onClick={() => setOpen(true)}>{d.create}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{d.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((x) => (
                  <tr key={x.ID}>
                    <td>{x.DismantleNo || `#${x.ID}`}</td>
                    <td>#{x.OrderID}</td>
                    <td>{x.LegalEntityName || `#${x.LegalEntityID}`}</td>
                    <td>{x.AssetID ? `#${x.AssetID}` : '—'}</td>
                    <td>{x.PortID ? `#${x.PortID}` : '—'}</td>
                    <td><StatusTag domain="task" value={x.Status} /></td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={6}><div className="org-empty">{d.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(d)} />
        </div>
      </div>
      {open && (
        <Drawer title={d.createTitle} onClose={() => setOpen(false)}
          footer={
            <>
              <button className="org-btn" onClick={() => setOpen(false)}>{t.pages.company.cancel}</button>
              <button className="org-btn org-btn-primary" disabled={busy || !orderOk} onClick={submit}>
                {busy ? t.pages.account.submitting : t.pages.company.save}
              </button>
            </>
          }>
          <div className="org-form">
            <div className="org-field">
              <label><span className="req">*</span>{d.fOrder}</label>
              <input className="org-input" type="number" value={orderId}
                onChange={(e) => setOrderId(e.target.value)} />
              {!orderOk && orderId !== '' && <span className="acc-err">{d.eOrder}</span>}
            </div>
            <div className="org-field">
              <label>{d.fAsset}</label>
              <input className="org-input" type="number" value={assetId}
                onChange={(e) => setAssetId(e.target.value)} />
              {!idOk(assetId) && <span className="acc-err">{d.eOrder}</span>}
            </div>
            <div className="org-field">
              <label>{d.fPort}</label>
              <input className="org-input" type="number" value={portId}
                onChange={(e) => setPortId(e.target.value)} />
            </div>
            {formError && <div className="org-error" style={{ margin: 0 }}>{formError}</div>}
          </div>
        </Drawer>
      )}
    </div>
  )
}
