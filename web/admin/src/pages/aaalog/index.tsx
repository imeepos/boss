// 话单与认证日志页:契约 GET /cdrs?loid + GET /auth-logs?loid(双页签)。
// W3 收尾:裸卡片壳/裸 table 收口为 Card/ui-table,工具钮/错误横幅走标准组件。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../api/client'
import { useT } from '../../i18n'
import { PageHead, pagerTexts } from '../org/shared'
import { Pagination } from '../../components/Pagination'
import { fmtTime } from '../../lib/format'
import { useDebouncedValue } from '../../lib/useDebouncedValue'
import { type AuthLogRow, type CdrRow } from '../quad/types'
import { failReasonText } from './failReason'
import { ErrorBanner, TableStateRow, TabBar, ToolbarButton } from '../../components/business'
import { Card, CardFooter } from '../../components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../components/ui/table'

export default function AaaLogPage() {
  const t = useT()
  const a = t.pages.aaaLogPage
  const [tab, setTab] = useState<'cdr' | 'auth'>('cdr')
  const [cdrs, setCdrs] = useState<CdrRow[]>([])
  const [auths, setAuths] = useState<AuthLogRow[]>([])
  const [total, setTotal] = useState(0)
  const [error, setError] = useState('')
  const [loid, setLoid] = useState('')
  const debouncedLoid = useDebouncedValue(loid)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = (key: string) => {
    setError('')
    setBusy(true)
    const req = key === 'cdr'
      ? apiFetch<{ items: CdrRow[]; total: number }>('/cdrs', { query: { loid: debouncedLoid || undefined, page, pageSize } })
      : apiFetch<{ items: AuthLogRow[]; total: number }>('/auth-logs', { query: { loid: debouncedLoid || undefined, page, pageSize } })
    req.then((x) => {
      const items = x?.items ?? []
      setTotal(x?.total ?? 0)
      if (key === 'cdr') setCdrs(items as CdrRow[])
      else setAuths(items as AuthLogRow[])
    })
      .catch((e) => setError(e instanceof Error ? e.message : a.loadFail))
      .finally(() => setBusy(false))
  }
  // loid 过滤防抖:输入停顿 300ms 才触发检索(修复 loid 不在依赖导致过滤不生效)。
  useEffect(() => { load(tab) }, [tab, page, pageSize, debouncedLoid])

  const slice = tab === 'cdr' ? cdrs : auths
  const fmtOct = (n: number) => (n >= 1024 * 1024 ? `${(n / 1024 / 1024).toFixed(1)}MB` : `${(n / 1024).toFixed(1)}KB`)

  return (
    <div>
      <PageHead title={a.title} desc={a.desc} />
      <Card>
        <div className="px-4 pt-3">
          <TabBar
            tabs={[{ key: 'cdr' as const, label: a.tabCdr }, { key: 'auth' as const, label: a.tabAuth }]}
            value={tab}
            onChange={(key) => { setTab(key); setPage(1) }}
          />
        </div>
        <div className="mb-3 flex items-center gap-1 px-4">
          <input className="h-8 w-[180px] rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" placeholder={a.filterLoid}
            value={loid} onChange={(e) => { setLoid(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <ToolbarButton disabled={busy} onClick={() => load(tab)}>{t.pages.audit.refresh}</ToolbarButton>
        </div>
        {error && <ErrorBanner message={error} />}
        {!error && tab === 'cdr' && (
          <div className="px-4 pb-4">
            <Table>
              <TableHeader>
                <TableRow>
                  {a.cdrColumns.map((x) => <TableHead key={x}>{x}</TableHead>)}
                </TableRow>
              </TableHeader>
              <TableBody>
                {(slice as CdrRow[]).map((x) => (
                  <TableRow key={x.id}>
                    <TableCell>{x.loid}</TableCell>
                    <TableCell>{a.acctStatus[x.acctStatus - 1] ?? x.acctStatus}</TableCell>
                    <TableCell>{x.sessionTime}</TableCell>
                    <TableCell>{fmtOct(x.inputOctets)}</TableCell>
                    <TableCell>{fmtOct(x.outputOctets)}</TableCell>
                    <TableCell>{x.billingStatus === 'BILLED' ? a.billed : a.unbilled}</TableCell>
                    <TableCell>{fmtTime(x.startedAt)}</TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={7} loading={busy} text={a.empty} />}
              </TableBody>
            </Table>
          </div>
        )}
        {!error && tab === 'auth' && (
          <div className="px-4 pb-4">
            <Table>
              <TableHeader>
                <TableRow>
                  {a.authColumns.map((x) => <TableHead key={x}>{x}</TableHead>)}
                </TableRow>
              </TableHeader>
              <TableBody>
                {(slice as AuthLogRow[]).map((x) => (
                  <TableRow key={x.id}>
                    <TableCell>{x.loid}</TableCell>
                    <TableCell>{x.result === 'SUCCESS' ? a.success : a.failed}</TableCell>
                    <TableCell data-testid="auth-fail-reason">{failReasonText(x, a)}</TableCell>
                    <TableCell>{fmtTime(x.createdAt)}</TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={4} loading={busy} text={a.empty} />}
              </TableBody>
            </Table>
          </div>
        )}
        <CardFooter>
          <Pagination total={total} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(a)} />
        </CardFooter>
      </Card>
    </div>
  )
}
