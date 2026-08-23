// 配置模板页:列表 + 新建/编辑(PUT /:id)+ 状态切换(PUT /:id/status)+ 删除未被引用模板(DELETE /:id)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { Dropdown } from '../../../components/Dropdown'
import { useConfirm } from '../../../components/ConfirmDialog'
import { pageSlice, type ProvisionTemplateRow } from '../types'
import type { LegalEntityRow } from '../../org/company/filter'
import { TemplateForm } from './TemplateForm'
import { TableStateRow } from '../../../components/business'

const TD = 'h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]'
const TH = 'h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]'
const ACT = 'mr-2.5 cursor-pointer border-none bg-none px-0 text-xs text-[var(--color-text-link)] hover:underline'
const DANGER = ACT + ' text-[var(--color-danger)]'

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
    setBusy(true); setError('')
    try {
      await apiFetch(`/provision-templates/${x.id}/status`, { method: 'PUT', body: { status: next } })
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
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <div className="w-36">
            <Dropdown value={statusFilter} options={statusOptions}
              onChange={(v) => { setStatusFilter(v); setPage(1) }} ariaLabel={p.allStatus} />
          </div>
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => setCreating(true)}>{p.create}</button>
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead><tr>{p.columns.map((x) => <th key={x} className={TH}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((x) => (
                  <tr key={x.id}>
                    <td className={TD}>#{x.id}</td>
                    <td className={TD}>{entityName(x.legalEntityId)}</td>
                    <td className={TD}>{x.code}</td>
                    <td className={TD}>{x.name}</td>
                    <td className={TD}>
                      <span className={x.status === 'ENABLED' ? 'text-[var(--color-success)]' : 'text-[var(--shell-group-title)]'}>{x.status}</span>
                    </td>
                    <td className={TD}>v{x.version}</td>
                    <td className={TD}>
                      <button className={ACT} disabled={busy} onClick={() => setEditing(x)}>{p.edit}</button>
                      <button className={ACT} disabled={busy} onClick={() => toggleStatus(x)}>{x.status === 'ENABLED' ? 'DISABLE' : 'ENABLE'}</button>
                      <button className={DANGER} disabled={busy} onClick={() => del(x)}>{p.delete}</button>
                    </td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={p.columns.length} loading={busy} text={p.empty} />}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(p)} />
        </div>
      </div>
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
