// 设备更换单页:契约 GET /replacements、POST /replacements(单号后端自动生成 RPL-*)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { pageSlice, PRIORITIES, type ReplacementRow } from '../types'

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
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.replacementNo || `#${x.id}`}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">#{x.assetId}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.reason || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{priorityLabel(x.priority)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="task" value={x.status} /></td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={5} className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><div className="py-8 text-center text-[13px] text-[var(--shell-group-title)]">{r.empty}</div></td></tr>}
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
              <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy || !assetOk} onClick={submit}>
                {busy ? t.pages.account.submitting : t.pages.company.save}
              </button>
            </>
          }>
          <div className="flex flex-col gap-3.5">
            <div className="flex flex-col gap-1.5">
              <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{r.fAsset}</label>
              <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" type="number" value={assetId} placeholder={r.pAsset}
                onChange={(e) => setAssetId(e.target.value)} />
              {!assetOk && assetId !== '' && <span className="text-[11px] text-[var(--color-danger)]">{r.eAsset}</span>}
            </div>
            <div className="flex flex-col gap-1.5">
              <label>{r.fReason}</label>
              <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={reason} placeholder={r.pReason}
                onChange={(e) => setReason(e.target.value)} />
            </div>
            <div className="flex flex-col gap-1.5">
              <label>{r.fPriority}</label>
              <select className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 pr-6 text-[13px] text-[var(--shell-content-text)] outline-none focus:border-[var(--color-border-focus)]" value={priority} onChange={(e) => setPriority(e.target.value)}>
                {PRIORITIES.map((p, i) => <option key={p} value={p}>{r.priorities[i]}</option>)}
              </select>
            </div>
            {formError && <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]" style={{ margin: 0 }}>{formError}</div>}
          </div>
        </Drawer>
      )}
    </div>
  )
}
