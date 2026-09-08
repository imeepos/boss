// 在线会话页:契约 GET /aaa/sessions(loid/nasIp/status 过滤,menu:loaccount)
// + POST /aaa/sessions/:id/disconnect(RFC 5176,异步受理语义)。
// W3 收尾:裸卡片壳/裸 table 收口为 Card/ui-table,工具钮/错误横幅走标准组件。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../api/client'
import { useT } from '../../i18n'
import { PageHead, pagerTexts } from '../org/shared'
import { StatusTag } from '../../components/StatusTag'
import { Dropdown } from '../../components/Dropdown'
import { Pagination } from '../../components/Pagination'
import { ErrorBanner, TableStateRow, ToolbarButton } from '../../components/business'
import { useConfirm } from '../../components/ConfirmDialog'
import { fmtTime } from '../../lib/format'
import { type SessionRow } from './types'
import { Card, CardContent, CardFooter } from '../../components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../components/ui/table'

const fmtOct = (n: number) => (n >= 1024 * 1024 ? (n / 1024 / 1024).toFixed(1) + 'MB' : (n / 1024).toFixed(1) + 'KB')

/** 会话表格:列渲染独立成组件,页面主体保持薄。 */
function SessionsTable({ rows, busy, empty, actionLabel, onDisconnect, busyId }: {
  rows: SessionRow[]
  busy: boolean
  empty: string
  actionLabel: string
  onDisconnect: (row: SessionRow) => void
  busyId: number
}) {
  const s = useT().pages.aaaSessionPage
  return (
    <div className="px-4 pb-4">
      <Table>
        <TableHeader>
          <TableRow>
            {s.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((r) => (
            <TableRow key={r.id}>
              <TableCell>{r.loid}</TableCell>
              <TableCell>{r.nasIp}</TableCell>
              <TableCell><StatusTag domain="aaaSession" value={r.status} /></TableCell>
              <TableCell>{fmtOct(r.inputOctets)}</TableCell>
              <TableCell>{fmtOct(r.outputOctets)}</TableCell>
              <TableCell>{fmtTime(r.startedAt)}</TableCell>
              <TableCell>{fmtTime(r.lastUpdate)}</TableCell>
              <TableCell>
                <button type="button" data-testid={'disconnect-' + r.id}
                  className="h-7 cursor-pointer rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_35%,transparent)] bg-transparent px-2.5 text-[12px] text-[var(--color-danger)] hover:border-[var(--color-danger)] disabled:cursor-not-allowed disabled:opacity-60"
                  disabled={busyId === r.id}
                  onClick={() => onDisconnect(r)}>{actionLabel}</button>
              </TableCell>
            </TableRow>
          ))}
          {!rows.length && <TableStateRow colSpan={s.columns.length} loading={busy} text={empty} />}
        </TableBody>
      </Table>
    </div>
  )
}

export default function AaaSessionPage() {
  const t = useT()
  const s = t.pages.aaaSessionPage
  const [rows, setRows] = useState<SessionRow[]>([])
  const [total, setTotal] = useState(0)
  const [error, setError] = useState('')
  const [accepted, setAccepted] = useState('')
  const [loid, setLoid] = useState('')
  const [nasIp, setNasIp] = useState('')
  const [status, setStatus] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [busyId, setBusyId] = useState(0)
  const confirmDialog = useConfirm()

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: SessionRow[]; total: number }>('/aaa/sessions', {
      query: { loid: loid.trim() || undefined, nasIp: nasIp.trim() || undefined, status: status || undefined, page, pageSize },
    })
      .then((d) => { setRows(d?.items ?? []); setTotal(d?.total ?? 0) })
      .catch((e) => setError(e instanceof Error ? e.message : s.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, [loid, nasIp, status, page, pageSize])

  // 强制下线:二次确认 → POST disconnect(数字会话 id);后端向 NAS 发 Disconnect,
  // NAS 不可达转 PENDING_OFFLINE 由后台重试,故受理成功仅提示"异步生效"。
  const handleDisconnect = async (row: SessionRow) => {
    if (!(await confirmDialog(s.disconnectConfirm, { title: s.forceOffline, danger: true }))) return
    setBusyId(row.id)
    setAccepted('')
    try {
      await apiFetch('/aaa/sessions/' + row.id + '/disconnect', { method: 'POST' })
      setAccepted(row.loid + ' · ' + row.sessionId)
    } catch (e) {
      setError(e instanceof Error ? e.message : s.disconnectFail)
    } finally {
      setBusyId(0)
      load()
    }
  }

  return (
    <div>
      <PageHead title={s.title} desc={s.desc} />
      <Card>
        <CardContent>
          <div className="flex flex-wrap items-center gap-2">
            <input className="h-8 w-40 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]"
              placeholder={s.filterLoid} aria-label={s.filterLoid}
              value={loid} onChange={(e) => { setLoid(e.target.value); setPage(1) }} />
            <input className="h-8 w-[150px] rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]"
              placeholder={s.filterNasIp} aria-label={s.filterNasIp}
              value={nasIp} onChange={(e) => { setNasIp(e.target.value); setPage(1) }} />
            <Dropdown
              value={status}
              options={[
                { value: '', label: s.allStatus },
                { value: 'ONLINE', label: s.statusOnline },
                { value: 'PENDING_OFFLINE', label: s.statusPendingOffline },
                { value: 'OFFLINE', label: s.statusOffline },
                { value: 'OFFLINE_FAILED', label: s.statusOfflineFailed },
              ]}
              onChange={(value) => { setStatus(value); setPage(1) }}
              ariaLabel={s.allStatus}
            />
            <span className="spacer" />
            <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
          </div>
        </CardContent>
        {accepted && (
          <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-success)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-success)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-success)]" data-testid="disconnect-accepted">
            <span className="font-medium">{s.accepted}</span>
            <span className="mx-1">·</span>
            <span>{accepted}</span>
            <span className="mx-1">·</span>
            <span>{s.acceptedDesc}</span>
          </div>
        )}
        {error && <ErrorBanner message={error} />}
        {!error && (
          <SessionsTable rows={rows} busy={busy} empty={s.empty} actionLabel={s.forceOffline}
            onDisconnect={handleDisconnect} busyId={busyId} />
        )}
        <CardFooter>
          <Pagination total={total} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(s)} />
        </CardFooter>
      </Card>
    </div>
  )
}
