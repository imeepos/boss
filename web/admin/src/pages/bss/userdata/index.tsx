// 用户端配置页(列表页模式 × TabBar 页签,antd pro Card+Tabs 布局):
// 页签行(含计数徽标 + 右侧刷新)→ 错误横幅 → DataTable(per-tab 列)→ 合计/更新时间页脚。
// 数据面不变:GET 列表族 + 行级开关/停用(userdata.yaml 配置管理面)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead } from '../../org/shared'
import { ErrorBanner, EmptyState } from '../../../components/business/page-head'
import { DataTable } from '../../../components/business/data-table'
import { TabBar } from '../../../components/business/tab-bar'
import { fmtTime } from '../../../lib/format'
import { TABS, type Row, type TabDef } from './tabs'
import { columnsFor } from './columns'
import { useConfirm } from '../../../components/ConfirmDialog'

interface Loader {
  rows: Row[]
  error: string
  busy: boolean
  at?: string
}

const EMPTY: Loader = { rows: [], error: '', busy: false }

export default function UserDataPage() {
  const t = useT()
  const confirmDialog = useConfirm()
  const u = t.pages.userdataPage
  const [tab, setTab] = useState<string>(TABS[0].key)
  const [st, setSt] = useState<Record<string, Loader>>({})
  const [notice, setNotice] = useState<{ text: string; ok: boolean } | null>(null)

  const load = (def: TabDef) => {
    const at = new Date().toISOString()
    setSt((s) => ({ ...s, [def.key]: { rows: [], error: '', busy: true } }))
    apiFetch<{ items: Row[] }>(def.path)
      .then((d) => setSt((s) => ({ ...s, [def.key]: { rows: d?.items ?? [], error: '', busy: false, at } })))
      .catch((e) => setSt((s) => ({
        ...s, [def.key]: { rows: [], error: e instanceof Error ? e.message : u.loadFail, busy: false, at },
      })))
  }
  useEffect(() => { load(TABS[0]) }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const act = async (def: TabDef, row: Row) => {
    const id = row[def.idKey]
    if (id === undefined || id === null || id === '') {
      setNotice({ text: u.actFail, ok: false })
      return
    }
    if (!(await confirmDialog(u.actConfirm.replace('{id}', String(id)), { danger: true }))) return
    apiFetch(`${def.actionPath}/${encodeURIComponent(String(id))}/${def.action}`, { method: 'PUT' })
      .then(() => { setNotice({ text: u.acted, ok: true }); load(def) })
      .catch(() => setNotice({ text: u.actFail, ok: false }))
  }

  const cur = TABS.find((x) => x.key === tab) ?? TABS[0]
  const state = st[cur.key] ?? EMPTY

  // 双层防御:即使后端契约已 500 兜住,前端也 detect 主键 undefined,console.warn 一次方便发现。
  const missingRows = state.rows.filter((r) => {
    const id = r[cur.idKey]
    return id === undefined || id === null || id === ''
  })
  if (missingRows.length > 0 && typeof console !== 'undefined') {
    console.warn(
      `[userdata] ${cur.key} 列表含 ${missingRows.length} 行缺主键列 ${cur.idKey}(后端 SQL AS 别名或断言失败)`,
    )
  }

  const tabLabel = (d: TabDef) => {
    const n = st[d.key]?.rows.length
    return n === undefined ? u.tabs[d.key] : `${u.tabs[d.key]} (${n})`
  }

  return (
    <div>
      <PageHead title={u.title} desc={u.desc} />
      <div className="rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="px-4 pt-3">
          <TabBar
            tabs={TABS.map((d) => ({ key: d.key, label: tabLabel(d) }))}
            value={tab}
            onChange={(key) => {
              setNotice(null)
              setTab(key)
              const def = TABS.find((x) => x.key === key)
              if (def && !st[key]) load(def)
            }}
            extra={
              <button
                className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]"
                disabled={state.busy}
                onClick={() => load(cur)}
              >{t.pages.audit.refresh}</button>
            }
          />
        </div>
        {notice ? (
          <div
            className="mx-4 mb-3 rounded-sm border px-3 py-2 text-[13px]"
            style={notice.ok ? {
              borderColor: 'color-mix(in srgb, var(--color-success) 25%, transparent)',
              background: 'color-mix(in srgb, var(--color-success) 8%, transparent)',
              color: 'var(--color-success)',
            } : {
              borderColor: 'color-mix(in srgb, var(--color-danger) 25%, transparent)',
              background: 'color-mix(in srgb, var(--color-danger) 8%, transparent)',
              color: 'var(--color-danger)',
            }}
          >{notice.text}</div>
        ) : null}
        {state.error ? <ErrorBanner message={state.error} /> : (
          <>
            <div className="px-4 pb-4">
              {state.rows.length ? (
                <DataTable
                  emptyText={u.empty}
                  rows={state.rows}
                  columns={columnsFor(cur, {
                    cols: u.cols, channels: u.channels, nameCol: u.nameCol, statusCol: u.statusCol,
                    opCol: u.opCol, disable: u.disable, on: u.on, off: u.off, list: u.list, unlist: u.unlist,
                  }, (row) => act(cur, row))}
                />
              ) : (
                <div className="rounded-sm border border-[var(--shell-side-border)] px-4 py-8">
                  <EmptyState text={u.empty} />
                </div>
              )}
            </div>
            <p className="border-t border-[var(--shell-side-border)] px-4 py-3 text-xs text-[var(--shell-crumb-text)]">
              {u.total.replace('{count}', String(state.rows.length))}
              {state.at ? ` · ${u.updated.replace('{time}', fmtTime(state.at))}` : ''}
            </p>
          </>
        )}
      </div>
    </div>
  )
}
