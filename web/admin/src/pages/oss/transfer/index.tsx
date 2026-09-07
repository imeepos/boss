// 跨区域调配页:契约 GET /transfers、POST /transfers、POST /transfers/:transferNo/approve|reject。
import { IdRef } from '../../../components/business'
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { ResourcePicker } from '../../../components/ResourcePicker'
import { pageSlice, type RegionRefRow, type ResourceRow, type TransferRow } from '../types'
import { useConfirm } from '../../../components/ConfirmDialog'
import { TableStateRow, ErrorBanner } from '../../../components/business'

export default function TransferPage() {
  const t = useT()
  const confirmDialog = useConfirm()
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
    if (!(await confirmDialog(r.confirm.replace('{act}', act).replace('{no}', transferNo), { danger: action === 'reject' }))) return
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
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => setOpen(true)}>{r.create}</button>
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{r.columns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((x) => (
                  <tr key={x.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.transferNo || `#${x.id}`}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><IdRef value={x.resourceId} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{regionName(x.fromRegionId)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{regionName(x.toRegionId)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="task" value={x.status} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      {x.status === 'PENDING' ? (
                        <span className="inline-flex items-center">
                          <button disabled={busy} onClick={() => review(x.transferNo, 'approve')}>{r.approve}</button>
                          <span className="text-[var(--shell-side-border)]">|</span>
                          <button disabled={busy} onClick={() => review(x.transferNo, 'reject')}>{r.reject}</button>
                        </span>
                      ) : '—'}
                    </td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={6} loading={busy} text={r.empty} />}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(r)} />
        </div>
      </div>
      {open && (
        <Drawer title={r.createTitle} onClose={() => setOpen(false)}
          footer={
            <>
              <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={() => setOpen(false)}>{t.pages.company.cancel}</button>
              <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy || !resourceOk || !regionOk} onClick={submit}>
                {busy ? t.pages.account.submitting : t.pages.company.save}
              </button>
            </>
          }>
          <div className="flex flex-col gap-3.5">
            <div className="flex flex-col gap-1.5">
              <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{r.fResource}</label>
              <ResourcePicker
                value={resourceId}
                onChange={setResourceId}
                load={() => apiFetch<{ items: ResourceRow[] }>('/resources').then((x) => x?.items ?? [])}
                toOption={(x) => ({ value: String(x.id), label: `${x.name} (${x.code})` })}
                ariaLabel={r.fResource}
                searchPlaceholder={r.pResource}
                errorText={r.loadFail}
              />
              {!resourceOk && resourceId !== '' && <span className="text-[11px] text-[var(--color-danger)]">{r.eResource}</span>}
            </div>
            <div className="flex flex-col gap-1.5">
              <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{r.fFrom}</label>
              <Dropdown
                value={fromRegionId ? String(fromRegionId) : ''}
                options={[{ value: '', label: '—' }, ...regions.map((x) => ({ value: String(x.id), label: x.name }))]}
                onChange={(v) => setFromRegionId(Number(v) || 0)}
                ariaLabel={r.fFrom}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{r.fTo}</label>
              <Dropdown
                value={toRegionId ? String(toRegionId) : ''}
                options={[{ value: '', label: '—' }, ...regions.map((x) => ({ value: String(x.id), label: x.name }))]}
                onChange={(v) => setToRegionId(Number(v) || 0)}
                ariaLabel={r.fTo}
              />
              {!regionOk && (fromRegionId !== 0 || toRegionId !== 0) && <span className="text-[11px] text-[var(--color-danger)]">{r.eRegion}</span>}
            </div>
            {formError && <ErrorBanner message={formError} className="mx-0" />}
          </div>
        </Drawer>
      )}
    </div>
  )
}
