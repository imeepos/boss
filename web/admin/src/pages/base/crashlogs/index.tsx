// 客户端崩溃日志:GET /crash-logs 列表 + 单条堆栈展开排查。
// 契约 admin/sys.yaml /crash-logs (menu:crash_logs);迁移 000140 授 sysadmin。
// 样式统一走 shell-* 令牌;堆栈面板主题中性底色,禁内联裸色值。
import { Fragment, useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, ErrorBanner, ToolbarButton, pagerTexts } from '../../../components/business/page-head'
import { EmptyState } from '../../../components/business/feedback'
import { Pagination } from '../../../components/Pagination'
import { Card } from '../../../components/ui/card'
import { Input } from '../../../components/ui/input'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'

const EXPAND_BTN = 'cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 py-1 text-xs text-[var(--shell-content-text)] hover:border-[var(--shell-input-border-hover)] hover:text-[var(--shell-heading)]'

type CrashLog = {
  id: number
  subjectType: string
  subjectId: number
  app: string
  log: string
  createdAt: string
}

export default function CrashLogsPage() {
  const t = useT()
  const [logs, setLogs] = useState<CrashLog[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [openId, setOpenId] = useState<number | null>(null)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)

  const load = () => {
    setError('')
    apiFetch<CrashLog[]>('/crash-logs?limit=100')
      .then((d) => setLogs(Array.isArray(d) ? d : []))
      .catch((e) => setError(e instanceof Error ? e.message : t.pages.crashlogs.loadFail))
  }

  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const filtered = useMemo(() => {
    const kw = keyword.trim().toLowerCase()
    if (!kw) return logs
    return logs.filter((l) =>
      String(l.id).includes(kw) || (l.app || '').toLowerCase().includes(kw)
      || l.subjectType.toLowerCase().includes(kw) || String(l.subjectId).includes(kw))
  }, [logs, keyword])

  const slice = filtered.slice((page - 1) * pageSize, page * pageSize)

  return (
    <div>
      <PageHead title={t.pages.crashlogs.title} desc={t.pages.crashlogs.desc} />
      {error && <div className="mb-3"><ErrorBanner message={error} /></div>}
      <Card>
        <div className="flex flex-wrap items-center gap-2 p-4">
          <Input
            className="w-64"
            placeholder={t.pages.crashlogs.searchPlaceholder}
            value={keyword}
            onChange={(e) => { setKeyword(e.target.value); setPage(1) }}
          />
          <span className="flex-1" />
          <ToolbarButton onClick={load}>{t.pages.crashlogs.refresh}</ToolbarButton>
        </div>
        <div className="px-4 pb-4">
          {filtered.length === 0 ? (
            <EmptyState text={keyword ? t.pages.crashlogs.noMatch : t.pages.crashlogs.empty} />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t.pages.crashlogs.colTime}</TableHead>
                  <TableHead>{t.pages.crashlogs.colApp}</TableHead>
                  <TableHead>{t.pages.crashlogs.colSubject}</TableHead>
                  <TableHead className="w-24">{t.pages.crashlogs.colOp}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {slice.map((l) => (
                  <Fragment key={l.id}>
                    <TableRow>
                      <TableCell>{new Date(l.createdAt).toLocaleString()}</TableCell>
                      <TableCell>{l.app || '—'}</TableCell>
                      <TableCell>{l.subjectType}/{l.subjectId || 0}</TableCell>
                      <TableCell>
                        <button className={EXPAND_BTN} onClick={() => setOpenId(openId === l.id ? null : l.id)}>
                          {openId === l.id ? t.pages.crashlogs.collapse : t.pages.crashlogs.expand}
                        </button>
                      </TableCell>
                    </TableRow>
                    {openId === l.id ? (
                      <TableRow>
                        <TableCell colSpan={4} className="whitespace-pre-wrap break-words bg-[var(--shell-menu-hover-bg)] p-3 font-mono text-xs">
                          {l.log}
                        </TableCell>
                      </TableRow>
                    ) : null}
                  </Fragment>
                ))}
              </TableBody>
            </Table>
          )}
        </div>
      </Card>
      {filtered.length > 0 && (
        <div className="mt-3 flex justify-end px-1 text-xs text-[var(--shell-group-title)]">
          <Pagination
            total={filtered.length}
            page={page}
            pageSize={pageSize}
            onPage={setPage}
            onSize={(n) => { setPageSize(n); setPage(1) }}
            {...pagerTexts(t.pages.company)}
          />
        </div>
      )}
    </div>
  )
}
