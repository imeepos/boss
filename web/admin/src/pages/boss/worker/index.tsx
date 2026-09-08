// 师傅管理页(装维队视图,000141):左侧队伍卡片(操作下拉) + 右侧成员表;
// 头部添加装维队按钮;业绩统计走右侧抽屉;选择师傅添加到当前装维队。
import { useCallback, useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { useQueryState } from '../../../lib/useQueryState'
import { PageHead, pagerTexts } from '../../org/shared'
import { ErrorBanner, ToolbarButton } from '../../../components/business'
import { Pagination } from '../../../components/Pagination'
import { Dropdown } from '../../../components/Dropdown'
import { ResourcePicker } from '../../../components/ResourcePicker'
import { fmtTime } from '../../../lib/format'
import { pageSlice, type WorkerGroupRow, type WorkerRow } from '../types'
import { TableStateRow } from '../../../components/business'
import { Card } from '../../../components/ui/card'
import { Input } from '../../../components/ui/input'
import { Badge } from '../../../components/ui/badge'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { TeamDialogs, type DialogMode } from './TeamDialogs'
import { WorkerDialogs, loadRegionOptions, type RegionOption, type WorkerDialogMode } from './WorkerDialogs'
import { WorkerRegionsDialog } from './worker-regions-dialog'
import { WorkerDetailDrawer } from './worker-detail-drawer'

const smallBtn = 'h-7 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-3 text-[12px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]'

export default function WorkerPage() {
  const t = useT()
  const w = t.pages.workerPage
  const [groups, setGroups] = useState<WorkerGroupRow[]>([])
  const [rows, setRows] = useState<WorkerRow[]>([])
  const [error, setError] = useState('')
  const [urlKeyword, setUrlKeyword] = useQueryState('kw', '')
  const [keyword, setKeyword] = useState(urlKeyword)
  const [selGroup, setSelGroup] = useState(0) // 0=全部成员
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [dialog, setDialog] = useState<DialogMode>(null)
  const [workerDialog, setWorkerDialog] = useState<WorkerDialogMode>(null)
  const [pickWorker, setPickWorker] = useState('')
  const [detailId, setDetailId] = useState<number | null>(null)
  const [regions, setRegions] = useState<RegionOption[]>([])
  const [regionsWorker, setRegionsWorker] = useState<WorkerRow | null>(null)

  const load = useCallback(() => {
    setError('')
    setBusy(true)
    Promise.all([
      apiFetch<{ items: WorkerGroupRow[] }>('/worker-groups'),
      apiFetch<{ items: WorkerRow[] }>('/workers'),
      loadRegionOptions(),
    ])
      .then(([g, wk, rg]) => { setGroups(g?.items ?? []); setRows(wk?.items ?? []); setRegions(rg) })
      .catch((e) => setError(e instanceof Error ? e.message : w.loadFail))
      .finally(() => setBusy(false))
  }, []) // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => { load() }, [load])

  const groupName = (id: number) => groups.find((g) => g.id === id)?.name ?? `#${id}`
  const regionNames = (r: WorkerRow) =>
    (r.regionIds ?? []).map((id) => regions.find((x) => x.id === id)?.name ?? `#${id}`).join('、') || '—'
  const filtered = useMemo(() => {
    const k = keyword.trim().toLowerCase()
    return rows.filter((r) => (!selGroup || r.groupId === selGroup)
      && (!k || r.name.toLowerCase().includes(k) || r.staffNo.toLowerCase().includes(k) || (r.phone || '').toLowerCase().includes(k)))
  }, [rows, keyword, selGroup])
  const slice = pageSlice(filtered, page, pageSize)

  // setCaptain 行内快捷:指定队长 = PUT 队伍(带当前名 + 新 leaderId)。
  const setCaptain = async (r: WorkerRow) => {
    try {
      await apiFetch(`/worker-groups/${r.groupId}`, { method: 'PUT', body: { name: groupName(r.groupId), leaderId: r.id } })
      toast.success(w.captainSet)
      load()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : w.actionFail)
    }
  }

  // 选择师傅后添加到当前装维队(目标 = 左侧选中的队伍)。
  const addWorkerToGroup = async (wid: string) => {
    if (!selGroup || !wid) return
    try {
      await apiFetch(`/workers/${wid}/transfer`, { method: 'POST', body: { groupId: selGroup, reason: w.addToGroupHint } })
      toast.success(w.addedToGroup)
      setPickWorker('')
      load()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : w.actionFail)
      setPickWorker('')
    }
  }

  const cardCls = (id: number) =>
    `cursor-pointer rounded-md border p-3 text-left transition-colors ${selGroup === id
      ? 'border-[var(--color-brand-bg)] bg-[color-mix(in_srgb,var(--color-brand-bg)_8%,transparent)]'
      : 'border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] hover:border-[var(--color-border-hover)]'}`

  return (
    <div>
      <PageHead title={w.title} desc={w.desc} />
      {/* 页面级操作栏:新增师傅(录入主档+登录密码) / 添加装维队 */}
      <div className="mb-4 flex items-center justify-end gap-2">
        <ToolbarButton primary onClick={() => setWorkerDialog({ type: 'create' })}>{w.newWorker}</ToolbarButton>
        <ToolbarButton primary onClick={() => setDialog({ type: 'create' })}>{w.newTeam}</ToolbarButton>
      </div>
      <div className="mb-4 flex flex-col gap-4 lg:flex-row">
        {/* 装维队卡片列 */}
        <Card className="w-full shrink-0 p-4 lg:w-72">
          <div className="mb-3">
            <span className="text-sm font-medium text-[var(--shell-heading)]">{w.teamTitle}</span>
          </div>
          <div className="flex flex-col gap-2">
            <button className={cardCls(0)} onClick={() => { setSelGroup(0); setPage(1) }}>
              <div className="text-[13px] font-medium text-[var(--shell-heading)]">{w.allMembers}</div>
              <div className="mt-1 text-xs text-[var(--shell-group-title)]">{w.memberCount}: {rows.filter((r) => r.status === 1).length}</div>
            </button>
            {groups.map((g) => (
              <div key={g.id} className={cardCls(g.id)} role="button" tabIndex={0}
                onClick={() => { setSelGroup(g.id); setPage(1) }}
                onKeyDown={(e) => { if (e.key === 'Enter') { setSelGroup(g.id); setPage(1) } }}>
                <div className="flex items-center justify-between gap-2">
                  <span className="truncate text-[13px] font-medium text-[var(--shell-heading)]">{g.name}</span>
                  <span className="shrink-0 text-xs text-[var(--shell-group-title)]">{g.code}</span>
                </div>
                <div className="mt-1 flex items-center gap-2 text-xs text-[var(--shell-group-title)]">
                  <span>{w.captain}: {g.leaderName || w.captainEmpty}</span>
                  <span>{w.memberCount}: {g.memberCount}</span>
                </div>
                {/* 卡片操作下拉:编辑 / 业绩统计 / 解散 */}
                <div className="mt-2" onClick={(e) => e.stopPropagation()}>
                  <Dropdown
                    value=""
                    options={[
                      { value: 'edit', label: w.editTeam },
                      { value: 'perf', label: w.perfBtn },
                      { value: 'disband', label: w.disband },
                    ]}
                    onChange={(op) => {
                      if (op === 'edit') setDialog({ type: 'edit', group: g })
                      else if (op === 'perf') setDialog({ type: 'perf', group: g })
                      else if (op === 'disband') setDialog({ type: 'disband', group: g })
                    }}
                    ariaLabel={w.teamOps}
                  />
                </div>
              </div>
            ))}
          </div>
        </Card>

        {/* 成员表 */}
        <Card className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2 p-4">
            <Input className="w-60" placeholder={w.searchPlaceholder}
              value={keyword} onChange={(e) => { setKeyword(e.target.value); setUrlKeyword(e.target.value); setPage(1) }} />
            <span className="text-sm text-[var(--shell-group-title)]">{selGroup ? groupName(selGroup) : w.allMembers}</span>
            <span className="spacer" />
            {!selGroup ? (
              <span className="text-xs text-[var(--shell-group-title)]">{w.needSelectGroup}</span>
            ) : (
              <ResourcePicker
                key={selGroup}
                value={pickWorker}
                onChange={(v) => { setPickWorker(v); if (v) addWorkerToGroup(v) }}
                load={() => Promise.resolve(rows.filter((r) => r.status === 1 && r.groupId !== selGroup))}
                toOption={(r) => ({ value: String(r.id), label: `${r.name}(${r.staffNo})` })}
                ariaLabel={w.pickWorker}
                searchPlaceholder={w.pickWorkerPlaceholder}
                errorText={w.loadFail}
                minWidth={180}
                pinnedOptions={(() => {
                  // 已选钉选:rows 即选项源,直接给人类可读回显(W0 契约 6)。
                  const r = rows.find((x) => String(x.id) === pickWorker && x.status === 1 && x.groupId !== selGroup)
                  return pickWorker && r ? [{ value: pickWorker, label: `${r.name}(${r.staffNo})` }] : undefined
                })()}
              />
            )}
            <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
          </div>
          {error && <ErrorBanner message={error} />}
          <div className="px-4 pb-4">
            <Table>
              <TableHeader>
                <TableRow>{w.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
              </TableHeader>
              <TableBody>
                {slice.map((r) => (
                  <TableRow key={r.id}>
                    <TableCell>{r.staffNo}</TableCell>
                    <TableCell>{r.name}</TableCell>
                    <TableCell>{groupName(r.groupId)}</TableCell>
                    <TableCell className="whitespace-normal">{regionNames(r)}</TableCell>
                    <TableCell>{r.phone || '—'}</TableCell>
                    <TableCell><Badge variant={r.status === 1 ? 'success' : 'default'}>{r.status === 1 ? w.active : w.left}</Badge></TableCell>
                    <TableCell>{fmtTime(r.joinedAt)}</TableCell>
                    <TableCell>
                      <div className="flex gap-2">
                        <button className={smallBtn} onClick={() => setDetailId(r.id)}>{w.detail}</button>
                        {r.status === 1 && (
                          <>
                            <button className={smallBtn} onClick={() => setRegionsWorker(r)}>{w.editRegions}</button>
                            <button className={smallBtn} onClick={() => setCaptain(r)}>{w.setCaptain}</button>
                            <button className={smallBtn} onClick={() => setDialog({ type: 'transfer', worker: r })}>{w.transfer}</button>
                          </>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={w.columns.length} loading={busy} text={w.empty} />}
              </TableBody>
            </Table>
          </div>
          <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
            <Pagination total={filtered.length} page={page} pageSize={pageSize}
              onPage={setPage} onSize={(s) => { setPageSize(s); setPage(1) }} {...pagerTexts(w)} />
          </div>
        </Card>
      </div>

      <TeamDialogs mode={dialog} groups={groups} workers={rows} onClose={() => setDialog(null)} onDone={load} />
      <WorkerDialogs mode={workerDialog} groups={groups} onClose={() => setWorkerDialog(null)} onDone={load} />
      {regionsWorker && (
        <WorkerRegionsDialog worker={regionsWorker} onClose={() => setRegionsWorker(null)} onDone={load} />
      )}
      {detailId !== null && (
        <WorkerDetailDrawer id={detailId} groupName={detailId ? groupName(rows.find((r) => r.id === detailId)?.groupId ?? 0) : ''}
          onResetPwd={(wid, wname) => { setDetailId(null); setWorkerDialog({ type: 'resetPwd', workerId: wid, name: wname }) }}
          onClose={() => setDetailId(null)} />
      )}
    </div>
  )
}
