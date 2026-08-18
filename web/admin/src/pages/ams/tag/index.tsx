// 电子标签页:契约 GET /tags;绑定资产经 boundAssetId 反查展示。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { pageSlice, type TagRow } from '../types'
import '../../org/org.css'

export default function TagPage() {
  const t = useT()
  const g = t.pages.tagPage
  const [rows, setRows] = useState<TagRow[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: TagRow[] }>('/tags')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : g.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const filtered = useMemo(() => {
    const k = keyword.trim().toLowerCase()
    if (!k) return rows
    return rows.filter((r) => r.tagNo.toLowerCase().includes(k) || r.epcCode.toLowerCase().includes(k))
  }, [rows, keyword])
  const slice = pageSlice(filtered, page, pageSize)

  return (
    <div>
      <PageHead title={g.title} desc={g.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <input className="org-input" placeholder={g.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{g.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.tagId}>
                    <td>{r.tagNo}</td>
                    <td>{r.epcCode}</td>
                    <td>{r.band || '—'}</td>
                    <td>{r.boundAssetId ? `#${r.boundAssetId}` : '—'}</td>
                    <td>{r.battery || '—'}</td>
                    <td><StatusTag domain="tag" value={r.status} /></td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={6}><div className="org-empty">{g.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(g)} />
        </div>
      </div>
    </div>
  )
}
