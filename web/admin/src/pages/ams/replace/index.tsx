// 设备更换单页:契约 GET /replacements、POST /replacements、POST /replacements/{id}/assign
// (单号后端自动生成 RPL-*;派单 PENDING→DOING,adopted note 2026-08-27-replacement-ticket-flow)。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useConfirm } from '../../../components/ConfirmDialog'
import { useT } from '../../../i18n'
import { Dropdown } from '../../../components/Dropdown'
import { ResourcePicker } from '../../../components/ResourcePicker'
import { PageHead, pagerTexts } from '../../org/shared'
import { ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { pageSlice, PRIORITIES, type AssetRow, type ReplacementRow } from '../types'
import { ActionLink, ActionLinks, ActionSep, TableStateRow } from '../../../components/business'
import { FormField } from '../../../components/business/form-field'
import { SubmitButton } from '../../../components/business/submit-button'
import { Card, CardContent, CardFooter } from '../../../components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { WorkerPicker, type PickedWorker } from '../../boss/dispatch/WorkerPicker'
import { canCancelReplacement, cancelPath } from './logic'

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
  const [dispatchRow, setDispatchRow] = useState<ReplacementRow | null>(null)
  const [picked, setPicked] = useState<PickedWorker | null>(null)
  const [dispatchError, setDispatchError] = useState('')
  const confirm = useConfirm()

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
      toast.success(r.createOk)
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : r.saveFail)
      setBusy(false)
    }
  }

  const submitDispatch = async () => {
    if (busy || !dispatchRow || !picked) return
    setBusy(true)
    setDispatchError('')
    try {
      await apiFetch(`/replacements/${dispatchRow.id}/assign`, {
        method: 'POST',
        body: { workerId: picked.id },
      })
      setDispatchRow(null)
      setPicked(null)
      toast.success(r.dispatchOk)
      load()
    } catch (e) {
      setDispatchError(e instanceof Error ? e.message : r.dispatchFail)
      setBusy(false)
    }
  }

  // 取消:PENDING 行专属,确认后 POST /replacements/{id}/cancel,终态不可再流转。
  const cancelRow = async (x: ReplacementRow) => {
    const no = x.replacementNo || '#' + x.id
    if (!(await confirm(r.cancelConfirm.replace('{no}', no), { danger: true, title: r.cancel }))) return
    setBusy(true)
    try {
      await apiFetch(cancelPath(x.id), { method: 'POST' })
      toast.success(r.cancelOk)
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : r.cancelFail)
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, page, pageSize)
  const priorityLabel = (v: string) => r.priorities[PRIORITIES.indexOf(v as typeof PRIORITIES[number])] ?? v
  const assetOk = /^\d+$/.test(assetId) && Number(assetId) > 0

  const createState = busy ? 'loading' : (formError ? 'failed' : 'idle')
  const createLabels = { idle: t.pages.company.save, loading: t.pages.account.submitting, success: r.createOk, failed: r.saveFail }
  const dispatchState = busy ? 'loading' : (dispatchError ? 'failed' : 'idle')
  const dispatchLabels = { idle: r.dispatchConfirm, loading: t.pages.account.submitting, success: r.dispatchOk, failed: r.dispatchFail }

  return (
    <div>
      <PageHead title={r.title} desc={r.desc} />
      <Card>
        <CardContent>
          <div className="flex flex-wrap items-center gap-2">
            <span className="spacer" />
            <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
            <ToolbarButton primary onClick={() => setOpen(true)}>{r.create}</ToolbarButton>
          </div>
        </CardContent>
        {error && <ErrorBanner message={error} />}
        <div className="px-4 pb-4">
          <Table>
            <TableHeader>
              <TableRow>{r.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
            </TableHeader>
            <TableBody>
              {slice.map((x) => (
                <TableRow key={x.id}>
                  <TableCell className="font-mono">{x.replacementNo || `#${x.id}`}</TableCell>
                  <TableCell className="font-mono">#{x.assetId}</TableCell>
                  <TableCell>{x.reason || '—'}</TableCell>
                  <TableCell>{priorityLabel(x.priority)}</TableCell>
                  <TableCell><StatusTag domain="task" value={x.status} /></TableCell>
                  <TableCell>
                    {x.status === 'PENDING' && (
                      <ActionLinks>
                        <ActionLink onClick={() => { setDispatchRow(x); setPicked(null); setDispatchError('') }} label={r.dispatch} testId={`replace-dispatch-${x.id}`} />
                        {canCancelReplacement(x.status) && (
                          <>
                            <ActionSep />
                            <ActionLink onClick={() => cancelRow(x)} label={r.cancel} testId={`replace-cancel-${x.id}`} />
                          </>
                        )}
                      </ActionLinks>
                    )}
                    {x.status !== 'PENDING' && <span>—</span>}
                  </TableCell>
                </TableRow>
              ))}
              {!slice.length && <TableStateRow colSpan={6} loading={busy} text={r.empty} />}
            </TableBody>
          </Table>
        </div>
        <CardFooter>
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(r)} />
        </CardFooter>
      </Card>
      {open && (
        <Drawer title={r.createTitle} onClose={() => setOpen(false)}
          footer={
            <>
              <ToolbarButton onClick={() => setOpen(false)} disabled={busy}>{t.pages.company.cancel}</ToolbarButton>
              <SubmitButton state={createState} labels={createLabels} disabled={busy || !assetOk} onClick={submit} />
            </>
          }>
          <div className="flex flex-col gap-3.5">
            <FormField label={r.fAsset} required>
              <ResourcePicker
                value={assetId}
                onChange={setAssetId}
                load={() => apiFetch<{ items: AssetRow[] }>('/assets').then((x) => x?.items ?? [])}
                toOption={(a) => ({ value: String(a.assetId), label: `${a.assetCode} (#${a.assetId})` })}
                ariaLabel={r.fAsset}
                searchPlaceholder={r.pickSearch}
                errorText={r.loadFail}
              />
            </FormField>
            <FormField label={r.fReason}>
              <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={reason} placeholder={r.pReason}
                onChange={(e) => setReason(e.target.value)} />
            </FormField>
            <FormField label={r.fPriority}>
              <Dropdown
                value={priority}
                options={PRIORITIES.map((p, i) => ({ value: p, label: r.priorities[i] }))}
                onChange={(v) => setPriority(v)}
                ariaLabel={r.fPriority}
              />
            </FormField>
            {formError && <ErrorBanner message={formError} />}
          </div>
        </Drawer>
      )}
      {dispatchRow && (
        <Drawer title={r.dispatchTitle.replace('{no}', dispatchRow.replacementNo || `#${dispatchRow.id}`)} onClose={() => setDispatchRow(null)}
          footer={
            <>
              <ToolbarButton onClick={() => setDispatchRow(null)} disabled={busy}>{t.pages.company.cancel}</ToolbarButton>
              <SubmitButton state={dispatchState} labels={dispatchLabels} disabled={busy || !picked} onClick={submitDispatch} />
            </>
          }>
          <div className="flex flex-col gap-3.5">
            <WorkerPicker selectedId={picked ? String(picked.id) : ''} onSelect={setPicked} />
            {dispatchError && <ErrorBanner message={dispatchError} />}
          </div>
        </Drawer>
      )}
    </div>
  )
}
