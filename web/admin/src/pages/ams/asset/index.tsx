// 资产台账页:列名以 fields.md §4.1 为准;契约 GET /assets(+ /tags 联表标签编号/EPC)。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { pageSlice, type AssetRow, type TagRow } from '../types'
import { AssetTrailDrawer } from './TrailDrawer'
import '../../org/org.css'

export default function AssetPage() {
  const t = useT()
  const a = t.pages.assetPage
  const [rows, setRows] = useState<AssetRow[]>([])
  const [tags, setTags] = useState<TagRow[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [trail, setTrail] = useState<AssetRow | null>(null)

  const load = () => {
    setError('')
    setBusy(true)
    Promise.all([
      apiFetch<{ items: AssetRow[] }>('/assets'),
      apiFetch<{ items: TagRow[] }>('/tags').catch(() => ({ items: [] as TagRow[] })),
    ])
      .then(([d, tg]) => { setRows(d?.items ?? []); setTags(tg?.items ?? []) })
      .catch((e) => setError(e instanceof Error ? e.message : a.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const tagOf = (tagId: number) => tags.find((x) => x.tagId === tagId)
  const filtered = useMemo(
    () => rows.filter((r) => r.assetCode.toLowerCase().includes(keyword.trim().toLowerCase())),
    [rows, keyword],
  )
  const slice = pageSlice(filtered, page, pageSize)

  return (
    <div>
      <PageHead title={a.title} desc={a.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <input className="org-input" placeholder={a.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{a.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.assetId}>
                    <td>{r.assetCode}</td>
                    <td>{tagOf(r.tagId)?.tagNo ?? '—'}</td>
                    <td>{tagOf(r.tagId)?.epcCode ?? '—'}</td>
                    <td>{r.type || '—'}</td>
                    <td>#{r.batchId}</td>
                    <td>{r.addressId ? `#${r.addressId}` : '—'}</td>
                    <td><StatusTag domain="asset" value={r.status} /></td>
                    <td>
                      <span className="org-act">
                        <button onClick={() => setTrail(r)}>{a.lifecycle}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={8}><div className="org-empty">{a.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(a)} />
        </div>
      </div>
      {trail && <AssetTrailDrawer asset={trail} onClose={() => setTrail(null)} />}
    </div>
  )
}
