// 话单与认证日志页:契约 GET /cdrs?loid + GET /auth-logs?loid(双页签)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../api/client'
import { useT } from '../../i18n'
import { PageHead, pagerTexts } from '../org/shared'
import { Pagination } from '../../components/Pagination'
import { fmtTime } from '../../lib/format'
import { pageSlice, type AuthLogRow, type CdrRow } from '../quad/types'
import '../org/org.css'

export default function AaaLogPage() {
  const t = useT()
  const a = t.pages.aaaLogPage
  const [tab, setTab] = useState<'cdr' | 'auth'>('cdr')
  const [cdrs, setCdrs] = useState<CdrRow[]>([])
  const [auths, setAuths] = useState<AuthLogRow[]>([])
  const [error, setError] = useState('')
  const [loid, setLoid] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = (key: string) => {
    setError('')
    setBusy(true)
    const req = key === 'cdr'
      ? apiFetch<{ items: CdrRow[] }>('/cdrs', { query: { loid: loid || undefined } })
      : apiFetch<{ items: AuthLogRow[] }>('/auth-logs', { query: { loid: loid || undefined } })
    req.then((x) => {
      const items = (x as { items?: CdrRow[] & AuthLogRow[] } | null)?.items ?? []
      if (key === 'cdr') setCdrs(items as CdrRow[])
      else setAuths(items as AuthLogRow[])
    })
      .catch((e) => setError(e instanceof Error ? e.message : a.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(() => { load(tab) }, [tab]) // eslint-disable-line react-hooks/exhaustive-deps

  const slice = pageSlice<CdrRow | AuthLogRow>(tab === 'cdr' ? cdrs : auths, page, pageSize)
  const count = tab === 'cdr' ? cdrs.length : auths.length
  const fmtOct = (n: number) => (n >= 1024 * 1024 ? `${(n / 1024 / 1024).toFixed(1)}MB` : `${(n / 1024).toFixed(1)}KB`)

  return (
    <div>
      <PageHead title={a.title} desc={a.desc} />
      <div className="org-card">
        <div style={{ display: 'flex', gap: 4, marginBottom: 12, borderBottom: '1px solid #f0f0f0', alignItems: 'center' }}>
          {(['cdr', 'auth'] as const).map((key) => (
            <button key={key} onClick={() => { setTab(key); setPage(1) }}
              style={{
                padding: '8px 16px', fontSize: 14, cursor: 'pointer', background: 'none', border: 'none',
                borderBottom: tab === key ? '2px solid #1677ff' : '2px solid transparent',
                color: tab === key ? '#1677ff' : '#666', fontWeight: tab === key ? 600 : 400,
              }}>
              {key === 'cdr' ? a.tabCdr : a.tabAuth}
            </button>
          ))}
          <input className="org-input" style={{ width: 180 }} placeholder={a.filterLoid}
            value={loid} onChange={(e) => { setLoid(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={() => load(tab)}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : tab === 'cdr' ? (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{a.cdrColumns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {(slice as CdrRow[]).map((x) => (
                  <tr key={x.id}>
                    <td>{x.loid}</td>
                    <td>{a.acctStatus[x.acctStatus - 1] ?? x.acctStatus}</td>
                    <td>{x.sessionTime}</td>
                    <td>{fmtOct(x.inputOctets)}</td>
                    <td>{fmtOct(x.outputOctets)}</td>
                    <td>{x.billingStatus}</td>
                    <td>{fmtTime(x.startedAt)}</td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={7}><div className="org-empty">{a.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{a.authColumns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {(slice as AuthLogRow[]).map((x) => (
                  <tr key={x.id}>
                    <td>{x.loid}</td>
                    <td>{x.result === 'SUCCESS' ? 'SUCCESS' : 'FAILED'}</td>
                    <td>{fmtTime(x.createdAt)}</td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={3}><div className="org-empty">{a.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={count} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(a)} />
        </div>
      </div>
    </div>
  )
}
