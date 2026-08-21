// 扩容申请页:契约 GET /expansions、POST /expansions;公司下拉源 GET /expansions/qos-templates(同权限组)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { pageSlice, type ExpansionRow, type LegalEntityRow, type RegionRefRow } from '../types'
import { TableStateRow } from '../../../components/business'

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
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => setOpen(true)}>{e.create}</button>
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{e.columns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((x) => (
                  <tr key={x.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.expansionNo || `#${x.id}`}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{companyName(x.legalEntityId)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{regionName(x.regionId)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.expectedPorts}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="task" value={x.status} /></td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={5} loading={busy} text={e.empty} />}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(e)} />
        </div>
      </div>
      {open && (
        <Drawer title={e.createTitle} onClose={() => setOpen(false)}
          footer={
            <>
              <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={() => setOpen(false)}>{t.pages.company.cancel}</button>
              <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]"
                disabled={busy || !legalEntityId || !regionId || !portsOk} onClick={submit}>
                {busy ? t.pages.account.submitting : t.pages.company.save}
              </button>
            </>
          }>
          <div className="flex flex-col gap-3.5">
            <div className="flex flex-col gap-1.5">
              <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{e.fCompany}</label>
              <Dropdown
                value={legalEntityId ? String(legalEntityId) : ''}
                options={[{ value: '', label: e.pCompany }, ...companies.map((c) => ({ value: String(c.id), label: c.name }))]}
                onChange={(v) => setLegalEntityId(Number(v) || 0)}
                ariaLabel={e.pCompany}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{e.fRegion}</label>
              <Dropdown
                value={regionId ? String(regionId) : ''}
                options={[{ value: '', label: e.pRegion }, ...regions.map((x) => ({ value: String(x.id), label: x.name }))]}
                onChange={(v) => setRegionId(Number(v) || 0)}
                ariaLabel={e.pRegion}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{e.fPorts}</label>
              <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" type="number" value={expectedPorts} placeholder="0"
                onChange={(ev) => setExpectedPorts(ev.target.value)} />
              {!portsOk && expectedPorts !== '' && <span className="text-[11px] text-[var(--color-danger)]">{e.ePorts}</span>}
            </div>
            {formError && <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]" style={{ margin: 0 }}>{formError}</div>}
          </div>
        </Drawer>
      )}
    </div>
  )
}
