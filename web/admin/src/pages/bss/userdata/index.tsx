// 用户端配置页:GET 列表族 + 行级开关/停用(userdata.yaml 配置管理面)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead } from '../../org/shared'
import { DataTable } from '../../../components/business/data-table'
import { fmtTime } from '../../../lib/format'
import { TABS, type Row, type TabDef } from './tabs'
import { useConfirm } from '../../../components/ConfirmDialog'

type Loader = { rows: Row[]; error: string; busy: boolean }

export default function UserDataPage() {
  const t = useT()
  const confirmDialog = useConfirm()
  const u = t.pages.userdataPage
  const [tab, setTab] = useState<string>(TABS[0].key)
  const [st, setSt] = useState<Record<string, Loader>>({})
  const [notice, setNotice] = useState('')

  const load = (def: TabDef) => {
    setSt((s) => ({ ...s, [def.key]: { rows: [], error: '', busy: true } }))
    apiFetch<{ items: Row[] }>(def.path)
      .then((d) => setSt((s) => ({ ...s, [def.key]: { rows: d?.items ?? [], error: '', busy: false } })))
      .catch((e) => setSt((s) => ({
        ...s, [def.key]: { rows: [], error: e instanceof Error ? e.message : u.loadFail, busy: false },
      })))
  }
  useEffect(() => { load(TABS[0]) }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const act = async (def: TabDef, row: Row) => {
    const id = row[def.idKey]
    if (id === undefined || id === null || id === '') {
      setNotice(u.actFail)
      return
    }
    if (!(await confirmDialog(u.actConfirm.replace('{id}', String(id)), { danger: true }))) return
    apiFetch(`${def.actionPath}/${encodeURIComponent(String(id))}/${def.action}`, { method: 'PUT' })
      .then(() => { setNotice(u.acted); load(def) })
      .catch(() => setNotice(u.actFail))
  }

  const cur = TABS.find((x) => x.key === tab) ?? TABS[0]
  const state = st[cur.key] ?? { rows: [], error: '', busy: false }

  // 双层防御:即使后端契约已 500 兜住,前端也 detect 主键 undefined,
  // 用 `—missing` 渲染而不是 `undefined` 字符串,console.warn 一次方便发现。
  const missingRows = state.rows.filter((r) => {
    const id = r[cur.idKey]
    return id === undefined || id === null || id === ''
  })
  if (missingRows.length > 0 && typeof console !== 'undefined') {
    console.warn(
      `[userdata] ${cur.key} 列表含 ${missingRows.length} 行缺主键列 ${cur.idKey}(后端 SQL AS 别名或断言失败)`,
    )
  }

  return (
    <div>
      <PageHead title={u.title} desc={u.desc} />
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          {TABS.map((d) => (
            <button key={d.key} className={'h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]' + (d.key === tab ? ' primary' : '')}
              onClick={() => { setTab(d.key); if (!st[d.key]) load(d) }}>{u.tabs[d.key]}</button>
          ))}
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={state.busy} onClick={() => load(cur)}>
            {t.pages.audit.refresh}
          </button>
        </div>
        {notice ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{notice}</div> : null}
        {state.error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{state.error}</div> : (
          <div className="px-4 pb-4">
            <DataTable
              emptyText={u.empty}
              rows={state.rows}
              columns={[
                { key: 'id', label: 'ID', render: (r) => {
                  const v = r[cur.idKey]
                  if (v === undefined || v === null || v === '') return <span title={`缺主键列 ${cur.idKey}`}>—missing</span>
                  return String(v)
                } },
                { key: 'name', label: u.nameCol, render: (r) => String(r.name ?? r.title ?? r.question ?? r.code ?? r.label ?? r.customerName ?? '—') },
                { key: 'status', label: u.statusCol, render: (r) => String(r.status ?? (r.enabled ? u.on : u.off) ?? (r.active ? u.on : u.off) ?? '—') },
                { key: 'op', label: u.opCol, render: (r) => cur.action ? (
                  <button
                    disabled={r[cur.idKey] === undefined || r[cur.idKey] === null || r[cur.idKey] === ''}
                    onClick={() => act(cur, r)}>{cur.action === 'disable' ? u.disable : u.toggle}</button>
                ) : <>—</> },
              ]}
            />
          </div>
        )}
        <p className="pl-3 text-xs text-[var(--shell-crumb-text)]">{u.total.replace('{count}', String(state.rows.length))} · {fmtTime(new Date().toISOString())}</p>
      </div>
    </div>
  )
}
