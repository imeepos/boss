// 话单与认证日志页:契约 GET /cdrs?loid + GET /auth-logs?loid(双页签)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../api/client'
import { useT } from '../../i18n'
import { PageHead, pagerTexts } from '../org/shared'
import { Pagination } from '../../components/Pagination'
import { fmtTime } from '../../lib/format'
import { type AuthLogRow, type CdrRow } from '../quad/types'
import { TableStateRow } from '../../components/business'

export default function AaaLogPage() {
  const t = useT()
  const a = t.pages.aaaLogPage
  const [tab, setTab] = useState<'cdr' | 'auth'>('cdr')
  const [cdrs, setCdrs] = useState<CdrRow[]>([])
  const [auths, setAuths] = useState<AuthLogRow[]>([])
  const [total, setTotal] = useState(0)
  const [error, setError] = useState('')
  const [loid, setLoid] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = (key: string) => {
    setError('')
    setBusy(true)
    const req = key === 'cdr'
      ? apiFetch<{ items: CdrRow[]; total: number }>('/cdrs', { query: { loid: loid || undefined, page, pageSize } })
      : apiFetch<{ items: AuthLogRow[]; total: number }>('/auth-logs', { query: { loid: loid || undefined, page, pageSize } })
    req.then((x) => {
      const items = x?.items ?? []
      setTotal(x?.total ?? 0)
      if (key === 'cdr') setCdrs(items as CdrRow[])
      else setAuths(items as AuthLogRow[])
    })
      .catch((e) => setError(e instanceof Error ? e.message : a.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(() => { load(tab) }, [tab, page, pageSize])

  const slice = tab === 'cdr' ? cdrs : auths
  const fmtOct = (n: number) => (n >= 1024 * 1024 ? `${(n / 1024 / 1024).toFixed(1)}MB` : `${(n / 1024).toFixed(1)}KB`)

  return (
    <div>
      <PageHead title={a.title} desc={a.desc} />
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
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
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" style={{ width: 180 }} placeholder={a.filterLoid}
            value={loid} onChange={(e) => { setLoid(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={() => load(tab)}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : tab === 'cdr' ? (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{a.cdrColumns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {(slice as CdrRow[]).map((x) => (
                  <tr key={x.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.loid}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{a.acctStatus[x.acctStatus - 1] ?? x.acctStatus}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.sessionTime}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtOct(x.inputOctets)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtOct(x.outputOctets)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.billingStatus === 'BILLED' ? a.billed : a.unbilled}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtTime(x.startedAt)}</td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={7} loading={busy} text={a.empty} />}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{a.authColumns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {(slice as AuthLogRow[]).map((x) => (
                  <tr key={x.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.loid}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.result === 'SUCCESS' ? a.success : a.failed}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtTime(x.createdAt)}</td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={3} loading={busy} text={a.empty} />}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={total} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(a)} />
        </div>
      </div>
    </div>
  )
}
