// 扩容申请页:契约 GET /expansions、POST /expansions;公司下拉源 GET /expansions/qos-templates(同权限组)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { pageSlice, type ExpansionRow, type LegalEntityRow, type RegionRefRow } from '../types'
import '../../org/org.css'

export default function ExpandPage() {
  const t = useT()
  const e = t.pages.expandPage
  const [rows, setRows] = useState<ExpansionRow[]>([])
  const [companies, setCompanies] = useState<LegalEntityRow[]>([])
  const [regions, setRegions] = useState<RegionRefRow[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [open, setOpen] = useState(false)
  const [legalEntityId, setLegalEntityId] = useState(0)
  const [regionId, setRegionId] = useState(0)
  const [expectedPorts, setExpectedPorts] = useState('')
  const [formError, setFormError] = useState('')

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: ExpansionRow[] }>('/expansions')
      .then((d) => setRows(d?.items ?? []))
      .catch((x) => setError(x instanceof Error ? x.message : e.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => {
    // 公司选项取自 QoS 模板(模板按公司归属,与 /expansions 同 menu:transfer 权限组)。
    apiFetch<{ items: { legalEntityId: number; legalEntityName?: string }[] }>('/expansions/qos-templates')
      .then((d) => {
        const seen = new Map<number, string>()
        for (const q of d?.items ?? []) seen.set(q.legalEntityId, q.legalEntityName ?? '')
        setCompanies([...seen].map(([id, name]) => ({ id, name: name || `#${id}` })))
      })
      .catch(() => setCompanies([]))
    apiFetch<RegionRefRow[]>('/regions')
      .then((d) => setRegions(d ?? []))
      .catch(() => setRegions([]))
  }, [])

  const submit = async () => {
    if (busy) return
    setBusy(true)
    setFormError('')
    try {
      await apiFetch('/expansions', {
        method: 'POST',
        body: { legalEntityId, regionId, expectedPorts: Number(expectedPorts) },
      })
      setOpen(false)
      setLegalEntityId(0)
      setRegionId(0)
      setExpectedPorts('')
      load()
    } catch (x) {
      setFormError(x instanceof Error ? x.message : e.saveFail)
    } finally {
      setBusy(false)
    }
  }

  const companyName = (id: number) => companies.find((c) => c.id === id)?.name ?? `#${id}`
  const regionName = (id: number) => regions.find((x) => x.id === id)?.name ?? `#${id}`
  const slice = pageSlice(rows, page, pageSize)
  const portsOk = /^\d+$/.test(expectedPorts) && Number(expectedPorts) > 0

  return (
    <div>
      <PageHead title={e.title} desc={e.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
          <button className="org-btn org-btn-primary" onClick={() => setOpen(true)}>{e.create}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{e.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((x) => (
                  <tr key={x.id}>
                    <td>{x.expansionNo || `#${x.id}`}</td>
                    <td>{companyName(x.legalEntityId)}</td>
                    <td>{regionName(x.regionId)}</td>
                    <td>{x.expectedPorts}</td>
                    <td><StatusTag domain="task" value={x.status} /></td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={5}><div className="org-empty">{e.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(e)} />
        </div>
      </div>
      {open && (
        <Drawer title={e.createTitle} onClose={() => setOpen(false)}
          footer={
            <>
              <button className="org-btn" onClick={() => setOpen(false)}>{t.pages.company.cancel}</button>
              <button className="org-btn org-btn-primary"
                disabled={busy || !legalEntityId || !regionId || !portsOk} onClick={submit}>
                {busy ? t.pages.account.submitting : t.pages.company.save}
              </button>
            </>
          }>
          <div className="org-form">
            <div className="org-field">
              <label><span className="req">*</span>{e.fCompany}</label>
              <select className="org-select" value={legalEntityId ? String(legalEntityId) : ''}
                onChange={(ev) => setLegalEntityId(Number(ev.target.value) || 0)}>
                <option value="">{e.pCompany}</option>
                {companies.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
              </select>
            </div>
            <div className="org-field">
              <label><span className="req">*</span>{e.fRegion}</label>
              <select className="org-select" value={regionId ? String(regionId) : ''}
                onChange={(ev) => setRegionId(Number(ev.target.value) || 0)}>
                <option value="">{e.pRegion}</option>
                {regions.map((x) => <option key={x.id} value={x.id}>{x.name}</option>)}
              </select>
            </div>
            <div className="org-field">
              <label><span className="req">*</span>{e.fPorts}</label>
              <input className="org-input" type="number" value={expectedPorts} placeholder="0"
                onChange={(ev) => setExpectedPorts(ev.target.value)} />
              {!portsOk && expectedPorts !== '' && <span className="text-[11px] text-[var(--color-danger)]">{e.ePorts}</span>}
            </div>
            {formError && <div className="org-error" style={{ margin: 0 }}>{formError}</div>}
          </div>
        </Drawer>
      )}
    </div>
  )
}
