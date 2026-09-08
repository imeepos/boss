// 跨区域调配页:契约 GET /transfers、POST /transfers、POST /transfers/:transferNo/approve|reject。
import { IdRef } from '../../../components/business'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { ResourcePicker } from '../../../components/ResourcePicker'
import { Card, CardFooter } from '../../../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { pageSlice, type RegionRefRow, type ResourceRow, type TransferRow } from '../types'
import { useConfirm } from '../../../components/ConfirmDialog'
import { TableStateRow, ErrorBanner, ToolbarButton } from '../../../components/business'

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
      .catch((e) => {
        const msg = e instanceof Error ? e.message : r.loadFail
        setError(msg)
        toast.error(r.loadFail, { description: msg })
      })
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
      toast.success(r.create)
      load()
    } catch (e) {
      const msg = e instanceof Error ? e.message : r.saveFail
      setFormError(msg)
      toast.error(r.saveFail, { description: msg })
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
      toast.success(act)
      load()
    } catch (e) {
      const msg = e instanceof Error ? e.message : r.actionFail
      setError(msg)
      toast.error(r.actionFail, { description: msg })
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
      <Card className="mb-4">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <span className="flex-1" />
          <ToolbarButton onClick={load} disabled={busy}>{t.pages.audit.refresh}</ToolbarButton>
          <ToolbarButton primary onClick={() => setOpen(true)}>{r.create}</ToolbarButton>
        </div>
        {error ? <div className="px-4 pb-3"><ErrorBanner message={error} /></div> : (
          <div className="px-4 pb-4">
            <Table>
              <TableHeader>
                <TableRow>
                  {r.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}
                </TableRow>
              </TableHeader>
              <TableBody>
                {slice.map((x) => (
                  <TableRow key={x.id}>
                    <TableCell>{x.transferNo || `#${x.id}`}</TableCell>
                    <TableCell><IdRef value={x.resourceId} /></TableCell>
                    <TableCell>{regionName(x.fromRegionId)}</TableCell>
                    <TableCell>{regionName(x.toRegionId)}</TableCell>
                    <TableCell><StatusTag domain="task" value={x.status} /></TableCell>
                    <TableCell>
                      {x.status === 'PENDING' ? (
                        <span className="inline-flex items-center gap-2">
                          <button type="button" disabled={busy} onClick={() => review(x.transferNo, 'approve')}>{r.approve}</button>
                          <span className="text-[var(--shell-side-border)]">|</span>
                          <button type="button" disabled={busy} onClick={() => review(x.transferNo, 'reject')}>{r.reject}</button>
                        </span>
                      ) : '—'}
                    </TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={6} loading={busy} text={r.empty} />}
              </TableBody>
            </Table>
          </div>
        )}
        <CardFooter>
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(r)} />
        </CardFooter>
      </Card>
      {open && (
        <Drawer title={r.createTitle} onClose={() => setOpen(false)}
          footer={
            <>
              <ToolbarButton onClick={() => setOpen(false)}>{t.pages.company.cancel}</ToolbarButton>
              <ToolbarButton primary disabled={busy || !resourceOk || !regionOk} onClick={submit}>
                {busy ? t.pages.account.submitting : t.pages.company.save}
              </ToolbarButton>
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
