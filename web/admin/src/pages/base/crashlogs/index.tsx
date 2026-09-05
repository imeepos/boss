// 客户端崩溃日志:GET /crash-logs 列表 + 单条堆栈展开排查。
// 契约 admin/sys.yaml /crash-logs (menu:crash_logs);迁移 000140 授 sysadmin。
// 样式统一走 shell-* 令牌;堆栈面板主题中性底色,禁内联裸色值。
import { Fragment, useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { EmptyState } from '../../../components/business/feedback'

const TH = 'h-9 px-3 text-left text-xs font-semibold whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]'
const TD = 'px-3 py-2.5 align-top text-[13px] whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)]'
const EXPAND_BTN = 'cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 py-1 text-xs text-[var(--shell-content-text)] hover:border-[var(--shell-input-border-hover)] hover:text-[var(--shell-heading)]'

export default function CrashLogsPage() {
  const t = useT()
  const [logs, setLogs] = useState<CrashLog[]>([])
  const [error, setError] = useState('')
  const [openId, setOpenId] = useState<number | null>(null)

  const load = () => {
    setError('')
    apiFetch<CrashLog[]>('/crash-logs?limit=100')
      .then((d) => setLogs(Array.isArray(d) ? d : []))
      .catch((e) => setError(e instanceof Error ? e.message : t.pages.crashlogs.loadFail))
  }

  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div>
      <PageHead title={t.pages.crashlogs.title} desc={t.pages.crashlogs.desc} />
      <div className="mb-3">
        <ToolbarButton onClick={load}>{t.pages.crashlogs.refresh}</ToolbarButton>
      </div>
      {error ? (
        <ErrorBanner message={error} className="!mx-0" />
      ) : logs.length === 0 ? (
        <EmptyState text={t.pages.crashlogs.empty} />
      ) : (
        <div className="overflow-x-auto rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
          <table className="w-full border-collapse text-[13px]">
            <thead>
              <tr>
                <th className={TH}>{t.pages.crashlogs.colTime}</th>
                <th className={TH}>{t.pages.crashlogs.colApp}</th>
                <th className={TH}>{t.pages.crashlogs.colSubject}</th>
                <th className={TH}>{t.pages.crashlogs.colOp}</th>
              </tr>
            </thead>
            <tbody>
              {logs.map((l) => (
                <Fragment key={l.id}>
                  <tr className="border-b border-[var(--shell-side-border)] hover:bg-[var(--shell-menu-hover-bg)]">
                    <td className={TD}>{new Date(l.createdAt).toLocaleString()}</td>
                    <td className={TD}>{l.app || '—'}</td>
                    <td className={TD}>
                      {l.subjectType}/{l.subjectId || 0}
                    </td>
                    <td className={TD}>
                      <button className={EXPAND_BTN} onClick={() => setOpenId(openId === l.id ? null : l.id)}>
                        {openId === l.id ? t.pages.crashlogs.collapse : t.pages.crashlogs.expand}
                      </button>
                    </td>
                  </tr>
                  {openId === l.id ? (
                    <tr>
                      <td colSpan={4} className="whitespace-pre-wrap break-words border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] p-3 font-mono text-xs text-[var(--shell-content-text)]">
                        {l.log}
                      </td>
                    </tr>
                  ) : null}
                </Fragment>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

type CrashLog = {
  id: number
  subjectType: string
  subjectId: number
  app: string
  log: string
  createdAt: string
}
