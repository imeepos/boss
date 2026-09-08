// 师傅注册审核页:GET /worker-registrations?status=(空=全部) + 通过/驳回动作。
// B7:状态页签(待审核/已通过/已驳回/全部)+ 分页;审批对话框班组/区域走选择器。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { ErrorBanner, ToolbarButton, TableStateRow } from '../../../components/business'
import { Pagination } from '../../../components/Pagination'
import { ResourcePicker } from '../../../components/ResourcePicker'
import { TabBar } from '../../../components/business/tab-bar'
import { Card } from '../../../components/ui/card'
import { Textarea } from '../../../components/ui/textarea'
import { Button } from '../../../components/ui/button'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { fmtTime } from '../../../lib/format'
import { pageSlice } from '../types'
import { Shell, Err, compact } from '../worker/TeamDialogs'

const compactBtnCls = compact

type RegStatus = 'PENDING' | 'APPROVED' | 'REJECTED'

interface WorkerRegistrationRow {
  id: number
  name: string
  phone: string
  idCardNo: string
  groupId: number
  regionId: number
  status: string
  reviewNote: string
  submittedAt: string
  reviewedAt: string
}

type TabKey = 'ALL' | RegStatus

export default function WorkerRegistrationPage() {
  const t = useT()
  const w = t.pages.workerRegPage
  const [rows, setRows] = useState<WorkerRegistrationRow[]>([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [tab, setTab] = useState<TabKey>('PENDING')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [actId, setActId] = useState<number | null>(null)
  const [actMode, setActMode] = useState<'approve' | 'reject' | null>(null)
  const [groupId, setGroupId] = useState('')
  const [regionId, setRegionId] = useState('')
  const [rejectNote, setRejectNote] = useState('')
  const [formError, setFormError] = useState('')

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: WorkerRegistrationRow[] }>('/worker-registrations', {
      query: { status: tab === 'ALL' ? undefined : tab },
    })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : w.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(() => { load() }, [tab]) // eslint-disable-line react-hooks/exhaustive-deps

  const closeDialog = () => {
    setActId(null)
    setActMode(null)
    setGroupId('')
    setRegionId('')
    setRejectNote('')
    setFormError('')
  }

  const handleApprove = async () => {
    if (actId == null || busy) return
    if (!/^\d+$/.test(groupId) || Number(groupId) <= 0) { setFormError(w.eGroupRequired); return }
    if (!/^\d+$/.test(regionId) || Number(regionId) <= 0) { setFormError(w.eRegionRequired); return }
    setBusy(true)
    setFormError('')
    try {
      await apiFetch('/worker-registrations/' + actId + '/approve', {
        method: 'POST', body: { groupId: Number(groupId), regionId: Number(regionId) },
      })
      toast.success(w.toastApproveOk)
      closeDialog()
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : w.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const handleReject = async () => {
    if (actId == null || busy) return
    if (!rejectNote.trim()) { setFormError(w.eNoteRequired); return }
    setBusy(true)
    setFormError('')
    try {
      await apiFetch('/worker-registrations/' + actId + '/reject', {
        method: 'POST', body: { note: rejectNote.trim() },
      })
      toast.success(w.toastRejectOk)
      closeDialog()
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : w.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const tabs: Array<{ key: TabKey; label: string }> = [
    { key: 'PENDING', label: w.tabPending },
    { key: 'APPROVED', label: w.tabApproved },
    { key: 'REJECTED', label: w.tabRejected },
    { key: 'ALL', label: w.tabAll },
  ]
  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={w.title} desc={w.desc} />
      <Card>
        <div className="p-4 pb-0">
          <TabBar tabs={tabs} value={tab} onChange={(k) => { setTab(k as TabKey); setPage(1) }} />
        </div>
        <div className="flex items-center gap-2 p-4">
          <span className="text-sm text-[var(--shell-group-title)]">{w.total.replace('{count}', String(rows.length))}</span>
          <span className="spacer" />
          <ToolbarButton disabled={busy} onClick={load}>{w.refresh}</ToolbarButton>
        </div>
        {error && <ErrorBanner message={error} />}
        <div className="px-4 pb-4">
          <Table>
            <TableHeader>
              <TableRow>
                {w.columns.map((x: string) => <TableHead key={x}>{x}</TableHead>)}
              </TableRow>
            </TableHeader>
            <TableBody>
              {slice.map((r) => (
                <TableRow key={r.id}>
                  <TableCell><span title={'registrationId=' + r.id}>#{r.id}</span></TableCell>
                  <TableCell>{r.name}</TableCell>
                  <TableCell>{r.phone}</TableCell>
                  <TableCell>{r.idCardNo}</TableCell>
                  <TableCell>{fmtTime(r.submittedAt)}</TableCell>
                  <TableCell>
                    {r.status === 'PENDING' ? (
                      <div className="flex gap-2">
                        <button className="h-7 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-3 text-[12px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)] disabled:cursor-not-allowed disabled:opacity-50" disabled={busy} onClick={() => { setActId(r.id); setActMode('approve') }}>{w.approve}</button>
                        <button className="h-7 cursor-pointer rounded-sm border border-[var(--color-danger)] bg-transparent px-3 text-[12px] text-[var(--color-danger)] hover:bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] disabled:cursor-not-allowed disabled:opacity-50" disabled={busy} onClick={() => { setActId(r.id); setActMode('reject') }}>{w.reject}</button>
                      </div>
                    ) : '—'}
                  </TableCell>
                </TableRow>
              ))}
              {!rows.length && <TableStateRow colSpan={6} loading={busy} text={w.empty} />}
            </TableBody>
          </Table>
        </div>
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={(s) => { setPageSize(s); setPage(1) }} {...pagerTexts(t.pages.company)} />
        </div>
      </Card>

      {actId != null && actMode && (
        <Shell title={actMode === 'approve' ? w.approveTitle : w.rejectTitle} onClose={closeDialog}>
          {actMode === 'approve' ? (
            <>
              <label className="mb-1 block text-sm text-[var(--shell-group-title)]">{w.groupId}</label>
              <div className="mb-3">
                <ResourcePicker
                  value={groupId}
                  onChange={setGroupId}
                  load={() => apiFetch<{ items: { id: number; name: string }[] }>('/worker-groups').then((x) => x?.items ?? [])}
                  toOption={(g) => ({ value: String(g.id), label: g.name })}
                  ariaLabel={w.groupId}
                  searchPlaceholder={w.groupIdPlaceholder}
                  errorText={w.loadFail}
                />
              </div>
              <label className="mb-1 block text-sm text-[var(--shell-group-title)]">{w.regionId}</label>
              <div className="mb-3">
                <ResourcePicker
                  value={regionId}
                  onChange={setRegionId}
                  load={() => apiFetch<{ id: number; name: string }[]>('/regions').then((x) => (Array.isArray(x) ? x : []))}
                  toOption={(r) => ({ value: String(r.id), label: r.name })}
                  ariaLabel={w.regionId}
                  searchPlaceholder={w.regionIdPlaceholder}
                  errorText={w.loadFail}
                />
              </div>
            </>
          ) : (
            <>
              <label className="mb-1 block text-sm text-[var(--shell-group-title)]">{w.rejectNote}</label>
              <Textarea className="mb-3" rows={3} placeholder={w.rejectNotePlaceholder} value={rejectNote} onChange={(e) => setRejectNote(e.target.value)} />
            </>
          )}
          {formError && <Err msg={formError} />}
          <div className="flex justify-end gap-2">
            <Button variant="outline" size="sm" className={compactBtnCls} onClick={closeDialog}>{w.cancel}</Button>
            <Button variant={actMode === 'approve' ? 'default' : 'destructive'} size="sm" className={compactBtnCls} disabled={busy}
              onClick={actMode === 'approve' ? handleApprove : handleReject}>
              {busy ? t.pages.account.submitting : (actMode === 'approve' ? w.confirmApprove : w.confirmReject)}
            </Button>
          </div>
        </Shell>
      )}
    </div>
  )
}
