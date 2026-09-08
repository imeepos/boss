// 配置模板页:列表 + 新建/编辑(PUT /:id)+ 状态切换(PUT /:id/status)+ 删除未被引用模板(DELETE /:id)。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { Dropdown } from '../../../components/Dropdown'
import { useConfirm } from '../../../components/ConfirmDialog'
import { pageSlice, type ProvisionTemplateRow } from '../types'
import type { LegalEntityRow } from '../../org/company/filter'
import { TemplateForm } from './TemplateForm'
import { ActionLink, ActionLinks, ActionSep, IdRef, TableStateRow, ToolbarButton } from '../../../components/business'
import { ErrorBanner } from '../../../components/business/page-head'
import { Card, CardContent, CardFooter } from '../../../components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'

export default function ProvisionTemplatePage() {
  const t = useT()
  const p = t.pages.templatePage
  const confirmDialog = useConfirm()
  const [rows, setRows] = useState<ProvisionTemplateRow[]>([])
  const [entities, setEntities] = useState<LegalEntityRow[]>([])
  const [error, setError] = useState('')
  const [creating, setCreating] = useState(false)
  const [editing, setEditing] = useState<ProvisionTemplateRow | null>(null)
  const [statusFilter, setStatusFilter] = useState('ALL')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: ProvisionTemplateRow[] }>('/provision-templates')
      .then((x) => setRows(x?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : p.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(() => {
    load()
    apiFetch<LegalEntityRow[]>('/legal-entities')
      .then((x) => setEntities(Array.isArray(x) ? x : []))
      .catch(() => setEntities([]))
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const toggleStatus = async (x: ProvisionTemplateRow) => {
    const next = x.status === 'ENABLED' ? 'DISABLED' : 'ENABLED'
    if (!(await confirmDialog(p.statusConfirm.replace('{status}', next), { danger: next === 'DISABLED' }))) return
    setBusy(true); setError('')
    try {
      await apiFetch(`/provision-templates/${x.id}/status`, { method: 'PUT', body: { status: next } })
      toast.success(p.statusOk)
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : p.loadFail)
    } finally {
      setBusy(false)
    }
  }

  const del = async (x: ProvisionTemplateRow) => {
    if (!(await confirmDialog(p.deleteConfirm.replace('{name}', x.name), { danger: true }))) return
    setBusy(true); setError('')
    try {
      await apiFetch(`/provision-templates/${x.id}`, { method: 'DELETE' })
      toast.success(p.deleteOk)
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : p.loadFail)
    } finally {
      setBusy(false)
    }
  }

  const entityName = (id: number) => entities.find((x) => x.id === id)?.name ?? `#${id}`
  const filtered = statusFilter === 'ALL' ? rows : rows.filter((x) => x.status === statusFilter)
  const slice = pageSlice(filtered, page, pageSize)
  const statusOptions = [
    { value: 'ALL', label: p.allStatus },
    { value: 'ENABLED', label: 'ENABLED' },
    { value: 'DISABLED', label: 'DISABLED' },
  ]

  return (
    <div>
      <PageHead title={p.title} desc={p.desc} />
      <Card>
        <CardContent>
          <div className="flex flex-wrap items-center gap-2">
            <div className="w-36">
              <Dropdown value={statusFilter} options={statusOptions}
                onChange={(v) => { setStatusFilter(v); setPage(1) }} ariaLabel={p.allStatus} />
            </div>
            <span className="spacer" />
            <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
            <ToolbarButton primary onClick={() => setCreating(true)}>{p.create}</ToolbarButton>
          </div>
        </CardContent>
        {error ? <ErrorBanner message={error} /> : (
          <div className="px-4 pb-4">
            <Table>
              <TableHeader>
                <TableRow>{p.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
              </TableHeader>
              <TableBody>
                {slice.map((x) => (
                  <TableRow key={x.id}>
                    <TableCell><IdRef value={x.id} /></TableCell>
                    <TableCell>{entityName(x.legalEntityId)}</TableCell>
                    <TableCell className="font-mono">{x.code}</TableCell>
                    <TableCell>{x.name}</TableCell>
                    <TableCell>
                      <span className={x.status === 'ENABLED' ? 'text-[var(--color-success)]' : 'text-[var(--shell-group-title)]'}>{x.status}</span>
                    </TableCell>
                    <TableCell>v{x.version}</TableCell>
                    <TableCell>{x.boundOffers > 0 ? x.boundOffers : <span className="text-[var(--shell-group-title)]">0</span>}</TableCell>
                    <TableCell>
                      <ActionLinks>
                        <ActionLink onClick={() => setEditing(x)} label={p.edit} testId={`tpl-edit-${x.id}`} />
                        <ActionSep />
                        <ActionLink onClick={() => toggleStatus(x)} label={x.status === 'ENABLED' ? 'DISABLE' : 'ENABLE'} testId={`tpl-status-${x.id}`} />
                        <ActionSep />
                        <ActionLink onClick={() => del(x)} label={p.delete} testId={`tpl-del-${x.id}`} />
                      </ActionLinks>
                    </TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={p.columns.length} loading={busy} text={p.empty} />}
              </TableBody>
            </Table>
          </div>
        )}
        <CardFooter>
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(p)} />
        </CardFooter>
      </Card>
      {creating && (
        <TemplateForm entities={entities}
          onDone={() => { setCreating(false); load() }}
          onCancel={() => setCreating(false)} />
      )}
      {editing && (
        <TemplateForm entities={entities} initial={editing}
          onDone={() => { setEditing(null); load() }}
          onCancel={() => setEditing(null)} />
      )}
    </div>
  )
}
