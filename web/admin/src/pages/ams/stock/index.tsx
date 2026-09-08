// 盘点管理页:契约 GET /stocktakes、POST /stocktakes、POST /stocktakes/:taskId/diff-handle。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { pageSlice, type StocktakeRow } from '../types'
import { useConfirm } from '../../../components/ConfirmDialog'
import { ActionLink, ActionLinks, ActionSep, TableStateRow, ToolbarButton } from '../../../components/business'
import { FormField } from '../../../components/business/form-field'
import { ErrorBanner } from '../../../components/business/page-head'
import { SubmitButton } from '../../../components/business/submit-button'
import { Card, CardContent, CardFooter } from '../../../components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { toast } from 'sonner'
import { ItemsDrawer } from './ItemsDrawer'

export default function StockPage() {
  const t = useT()
  const confirmDialog = useConfirm()
  const s = t.pages.stock
  const [rows, setRows] = useState<StocktakeRow[]>([])
  const [companies, setCompanies] = useState<{ id: number; name: string }[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [open, setOpen] = useState(false)
  const [legalEntityId, setLegalEntityId] = useState(0)
  const [scope, setScope] = useState('')
  const [formError, setFormError] = useState('')
  const [detailTask, setDetailTask] = useState<StocktakeRow | null>(null)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: StocktakeRow[] }>('/stocktakes')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : s.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => {
    apiFetch<{ id: number; name: string }[]>('/legal-entities')
      .then((d) => setCompanies(d ?? []))
      .catch(() => setCompanies([]))
  }, [])

  const submit = async () => {
    if (busy) return
    setBusy(true)
    setFormError('')
    try {
      await apiFetch('/stocktakes', {
        method: 'POST',
        body: { legalEntityId, scope: scope.trim() },
      })
      setOpen(false)
      setScope('')
      setLegalEntityId(0)
      toast.success(s.createOk)
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : s.saveFail)
      setBusy(false)
    }
  }

  const diffHandle = async (taskId: number) => {
    if (busy || !(await confirmDialog(s.diffConfirm))) return
    setBusy(true)
    try {
      await apiFetch(`/stocktakes/${taskId}/diff-handle`, { method: 'POST' })
      toast.success(s.diffHandleOk)
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : s.actionFail)
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, page, pageSize)
  const scopeOk = scope.trim().length > 0 && scope.trim().length <= 64

  const createState = busy ? 'loading' : (formError ? 'failed' : 'idle')
  const createLabels = { idle: t.pages.company.save, loading: t.pages.account.submitting, success: s.createOk, failed: s.saveFail }

  return (
    <div>
      <PageHead title={s.title} desc={s.desc} />
      <Card>
        <CardContent>
          <div className="flex flex-wrap items-center gap-2">
            <span className="spacer" />
            <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
            <ToolbarButton primary onClick={() => setOpen(true)}>{s.create}</ToolbarButton>
          </div>
        </CardContent>
        {error && <ErrorBanner message={error} />}
        <div className="px-4 pb-4">
          <Table>
            <TableHeader>
              <TableRow>{s.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
            </TableHeader>
            <TableBody>
              {slice.map((r) => (
                <TableRow key={r.id}>
                  <TableCell className="font-mono">#{r.id}</TableCell>
                  <TableCell>{r.scope}</TableCell>
                  <TableCell>{s.progress.replace('{n}', String(r.progress))}</TableCell>
                  <TableCell>{r.diffCount}</TableCell>
                  <TableCell><StatusTag domain="task" value={r.status} /></TableCell>
                  <TableCell>
                    <ActionLinks>
                      <ActionLink onClick={() => setDetailTask(r)} label={s.detail} testId={`stock-detail-${r.id}`} />
                      {r.status === 'DOING' && (
                        <>
                          <ActionSep />
                          <ActionLink onClick={() => diffHandle(r.id)} label={s.diffHandle} testId={`stock-diff-${r.id}`} />
                        </>
                      )}
                    </ActionLinks>
                  </TableCell>
                </TableRow>
              ))}
              {!slice.length && <TableStateRow colSpan={6} loading={busy} text={s.empty} />}
            </TableBody>
          </Table>
        </div>
        <CardFooter>
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(s)} />
        </CardFooter>
      </Card>
      {open && (
        <Drawer title={s.createTitle} onClose={() => setOpen(false)}
          footer={
            <>
              <ToolbarButton onClick={() => setOpen(false)} disabled={busy}>{t.pages.company.cancel}</ToolbarButton>
              <SubmitButton state={createState} labels={createLabels} disabled={busy || !legalEntityId || !scopeOk} onClick={submit} />
            </>
          }>
          <div className="flex flex-col gap-3.5">
            <FormField label={s.fCompany} required>
              <Dropdown
                value={legalEntityId ? String(legalEntityId) : ''}
                options={[{ value: '', label: s.pCompany }, ...companies.map((c) => ({ value: String(c.id), label: c.name }))]}
                onChange={(v) => setLegalEntityId(Number(v) || 0)}
                ariaLabel={s.pCompany}
              />
            </FormField>
            <FormField label={s.fScope} required error={!scopeOk && scope !== '' ? s.eScope : undefined}>
              <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={scope} placeholder={s.pScope}
                onChange={(e) => setScope(e.target.value)} />
              <span className="text-[11px] text-[var(--shell-crumb-text)]">保留值 ODN = 网络资产专项盘点(仅盘有资产化凭证的在网资产,W8)</span>
            </FormField>
            {formError && <ErrorBanner message={formError} />}
          </div>
        </Drawer>
      )}
      {detailTask && (
        <ItemsDrawer taskId={detailTask.id} canEdit={detailTask.status === 'DOING'}
          onClose={() => setDetailTask(null)} onChanged={load} />
      )}
    </div>
  )
}
