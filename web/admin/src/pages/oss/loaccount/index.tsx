// 认证账号页(AAA 域,挂 oss 分组):契约 GET /lo-accounts。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { pageSlice, type LoAccountRow } from '../types'
import '../../org/org.css'

export default function LoAccountPage() {
  const t = useT()
  const l = t.pages.loAccountPage
  const [rows, setRows] = useState<LoAccountRow[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: LoAccountRow[] }>('/lo-accounts', {
      query: { keyword: keyword.trim() || undefined },
    })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : l.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const filtered = useMemo(() => {
    const k = keyword.trim().toLowerCase()
    if (!k) return rows
    return rows.filter((r) => r.loid.toLowerCase().includes(k))
  }, [rows, keyword])
  const slice = pageSlice(filtered, page, pageSize)

  return (
    <div>
      <PageHead title={l.title} desc={l.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <input className="org-input" placeholder={l.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{l.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.id}>
                    <td>{r.loid}</td>
                    <td>#{r.customerId}</td>
                    <td>{r.legalEntityName || `#${r.legalEntityId}`}</td>
                    <td>{r.regionName || r.regionPath || '—'}</td>
                    <td>{r.qosTemplateId ? `#${r.qosTemplateId}` : '—'}</td>
                    <td><StatusTag domain="loAccount" value={r.status} /></td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={6}><div className="org-empty">{l.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(l)} />
        </div>
      </div>
    </div>
  )
}
