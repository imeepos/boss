// 数据备份迁移页:任务清单(备份/导入) + 新建备份抽屉 + 导入恢复抽屉。
// 两个抽屉在 BackupDrawers.tsx,纯逻辑在 logic.ts,样式常量在 styles.ts。
// 契约:GET/POST /backup/*(menu:backup 门禁);字段口径 docs/contract/fields.md 1.5.6。
// 筛选/分页状态经 URL search 持久;有 running 任务时 3s 轮询刷新。
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { apiFetch } from '../../api/client'
import { useT } from '../../i18n'
import { useQueryInt, useQueryState } from '../../lib/useQueryState'
import { Dropdown } from '../../components/Dropdown'
import { Pagination } from '../../components/Pagination'
import { StatusTag } from '../../components/StatusTag'
import { useConfirm } from '../../components/ConfirmDialog'
import { TableStateRow } from '../../components/business'
import { BackupCreateDrawer, RestoreDrawer } from './BackupDrawers'
import { downloadArchive, formatBytes, formatTime, toJob, type BackupJobEntry } from './logic'
import { BTN, BTN_PRIMARY, BTN_LINK, TH, TD } from './styles'

export default function BackupPage() {
  const t = useT()
  const confirmDialog = useConfirm()
  const [kind, setKind] = useQueryState('kind', '')
  const [status, setStatus] = useQueryState('status', '')
  const [page, setPage] = useQueryInt('page', 1)
  const [pageSize, setPageSize] = useState(10)
  const [rows, setRows] = useState<BackupJobEntry[]>([])
  const [total, setTotal] = useState(0)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [showCreate, setShowCreate] = useState(false)
  const [showRestore, setShowRestore] = useState(false)
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const load = useCallback(() => {
    setError('')
    apiFetch<{ items: BackupJobEntry[]; total: number }>('/backup/jobs', {
      query: { kind: kind || undefined, status: status || undefined, pageSize, offset: (page - 1) * pageSize },
    })
      .then((d) => {
        setRows((d?.items ?? []).map(toJob))
        setTotal(d?.total ?? 0)
      })
      .catch((e) => setError(e instanceof Error ? e.message : t.pages.backup.loadFail))
  }, [kind, status, page, pageSize, t.pages.backup.loadFail])

  useEffect(load, [load]) // eslint-disable-line react-hooks/exhaustive-deps

  // 有 running 任务时轮询,全部落定即停。
  const anyRunning = rows.some((r) => r.status === 'running')
  useEffect(() => {
    if (pollRef.current) { clearInterval(pollRef.current); pollRef.current = null }
    if (!anyRunning) return
    pollRef.current = setInterval(load, 3000)
    return () => { if (pollRef.current) clearInterval(pollRef.current) }
  }, [anyRunning, load])

  const rangeText = useMemo(
    () => t.pages.backup.rangeText
      .replace('{from}', String(total === 0 ? 0 : (page - 1) * pageSize + 1))
      .replace('{to}', String(Math.min(total, page * pageSize)))
      .replace('{count}', String(total)),
    [t.pages.backup.rangeText, total, page, pageSize],
  )

  const handleDelete = async (row: BackupJobEntry) => {
    if (!(await confirmDialog(t.pages.backup.deleteConfirm, { danger: true }))) return
    apiFetch(`/backup/jobs/${row.id}`, { method: 'DELETE' })
      .then(load)
      .catch((e) => setError(e instanceof Error ? e.message : t.pages.backup.loadFail))
  }

  const handleDownload = (row: BackupJobEntry) => {
    downloadArchive(row.id, row.fileName).catch((e) => setError(e instanceof Error ? e.message : t.pages.backup.loadFail))
  }

  const b = t.pages.backup
  return (
    <div>
      <div className="mb-4">
        <h2 className="m-0 text-xl font-bold text-[var(--shell-heading)]">{b.title}</h2>
        <p className="mt-1 text-xs text-[var(--shell-crumb-text)]">{b.desc}</p>
      </div>
      <div className="rounded-lg border border-[var(--shell-side-border)] bg-[var(--shell-card-bg)] p-4 shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2">
          <Dropdown
            value={kind}
            options={[
              { value: '', label: b.columns[1] },
              { value: 'backup', label: b.kindBackup },
              { value: 'restore', label: b.kindRestore },
            ]}
            onChange={(v) => { setKind(v); setPage(1) }}
            ariaLabel={b.columns[1]}
          />
          <Dropdown
            value={status}
            options={[
              { value: '', label: b.columns[3] },
              { value: 'running', label: b.statusRunning },
              { value: 'succeeded', label: b.statusSucceeded },
              { value: 'failed', label: b.statusFailed },
            ]}
            onChange={(v) => { setStatus(v); setPage(1) }}
            ariaLabel={b.columns[3]}
          />
          <span className="flex-1" />
          {notice && <span className="text-xs text-[var(--shell-crumb-text)]">{notice}</span>}
          <button className={BTN} onClick={load}>{b.refresh}</button>
          <button className={BTN_PRIMARY} onClick={() => setShowCreate(true)}>{b.newBackup}</button>
          <button className={BTN} onClick={() => setShowRestore(true)}>{b.restore}</button>
        </div>
        {error && <p className="m-0 mt-3 text-[13px] text-[var(--color-danger)]">{error}</p>}
        <div className="mt-3 overflow-x-auto">
          <table className="w-full border-collapse">
            <thead>
              <tr className="border-b border-[var(--shell-side-border)]">
                {b.columns.map((c) => <th key={c} className={TH}>{c}</th>)}
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.id} className="border-b border-[var(--shell-side-border)] last:border-none hover:bg-[var(--shell-menu-hover-bg)]">
                  <td className={TD}>{r.id}</td>
                  <td className={TD}>{r.kind === 'backup' ? b.kindBackup : b.kindRestore}</td>
                  <td className={TD}>{r.scope === 'all' ? b.scopeAll : `${b.scopeTables}(${r.tables.length})`}</td>
                  <td className={TD}><StatusTag domain="backupStatus" value={r.status} /></td>
                  <td className={TD} title={r.fileName}>{r.fileName ? (r.fileName.length > 28 ? `${r.fileName.slice(0, 25)}...` : r.fileName) : '-'}</td>
                  <td className={TD}>{formatBytes(r.sizeBytes)}</td>
                  <td className={TD}>{r.tableCount}</td>
                  <td className={TD}>{r.rowCount}</td>
                  <td className={TD}>{r.operator || '-'}</td>
                  <td className={TD}>{formatTime(r.createdAt)}</td>
                  <td className={TD}>{formatTime(r.finishedAt ?? '') || '-'}</td>
                  <td className={TD}>
                    <span className="flex items-center gap-1">
                      {r.kind === 'backup' && r.status === 'succeeded' && (
                        <button className={BTN_LINK} onClick={() => handleDownload(r)}>{b.download}</button>
                      )}
                      {r.status !== 'running' && (
                        <button className={BTN_LINK} onClick={() => handleDelete(r)}>{b.del}</button>
                      )}
                      {r.error && <span className="cursor-help text-[var(--shell-crumb-text)]" title={r.error}>?</span>}
                    </span>
                  </td>
                </tr>
              ))}
              {rows.length === 0 && !error && (
                <TableStateRow colSpan={b.columns.length} text={b.empty} />
              )}
            </tbody>
          </table>
        </div>
        <Pagination
          page={page} pageSize={pageSize} total={total}
          onPage={setPage} onSize={(s) => { setPageSize(s); setPage(1) }}
          rangeText={rangeText} prevText={b.prev} nextText={b.next}
          perPageText={b.perPage} jumpText={b.jump} pageUnitText={b.pageUnit}
        />
      </div>
      {showCreate && (
        <BackupCreateDrawer
          onClose={() => setShowCreate(false)}
          onChanged={(msg) => { setNotice(msg); load() }}
          onBusy={() => setNotice(b.busy)}
        />
      )}
      {showRestore && (
        <RestoreDrawer
          onClose={() => setShowRestore(false)}
          onChanged={(msg) => { setNotice(msg); load() }}
          onBusy={() => setNotice(b.busy)}
        />
      )}
    </div>
  )
}
