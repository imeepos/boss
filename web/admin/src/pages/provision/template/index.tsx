// 配置模板页:契约 GET /provision-templates(裸 items)+ POST /provision-templates(创建)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { pageSlice, type ProvisionTemplateRow } from '../types'
import type { LegalEntityRow } from '../../org/company/filter'
import { TemplateForm } from './TemplateForm'
import '../../org/org.css'

export default function ProvisionTemplatePage() {
  const t = useT()
  const p = t.pages.templatePage
  const [rows, setRows] = useState<ProvisionTemplateRow[]>([])
  const [entities, setEntities] = useState<LegalEntityRow[]>([])
  const [error, setError] = useState('')
  const [showForm, setShowForm] = useState(false)
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

  const entityName = (id: number) => entities.find((x) => x.id === id)?.name ?? `#${id}`
  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={p.title} desc={p.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
          <button className="org-btn org-btn-primary" onClick={() => setShowForm(true)}>{p.create}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{p.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((x) => (
                  <tr key={x.id}>
                    <td>#{x.id}</td>
                    <td>{entityName(x.legalEntityId)}</td>
                    <td>{x.code}</td>
                    <td>{x.name}</td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={4}><div className="org-empty">{p.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(p)} />
        </div>
      </div>
      {showForm && (
        <TemplateForm entities={entities}
          onDone={() => { setShowForm(false); load() }}
          onCancel={() => setShowForm(false)} />
      )}
    </div>
  )
}
