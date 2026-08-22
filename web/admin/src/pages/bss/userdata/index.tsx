// 用户端配置页:GET 列表族 + 行级开关/停用(userdata.yaml 配置管理面)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead } from '../../org/shared'
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
    if (!(await confirmDialog(u.actConfirm.replace('{id}', String(id)), { danger: true }))) return
    apiFetch(`${def.actionPath}/${encodeURIComponent(String(id))}/${def.action}`, { method: 'PUT' })
      .then(() => { setNotice(u.acted); load(def) })
      .catch(() => setNotice(u.actFail))
  }

  const cur = TABS.find((x) => x.key === tab) ?? TABS[0]
  const state = st[cur.key] ?? { rows: [], error: '', busy: false }

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
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr><th className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">ID</th><th className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{u.nameCol}</th><th className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{u.statusCol}</th><th className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{u.opCol}</th></tr></thead>
              <tbody>
                {state.rows.map((r) => (
                  <tr key={String(r[cur.idKey])}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{String(r[cur.idKey])}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{String(r.name ?? r.title ?? r.question ?? r.code ?? r.label ?? r.customerName ?? '—')}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{String(r.status ?? (r.enabled ? u.on : u.off) ?? (r.active ? u.on : u.off) ?? '—')}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      {cur.action ? (
                        <span className="inline-flex items-center">
                          <button onClick={() => act(cur, r)}>{cur.action === "disable" ? u.disable : u.toggle}</button>
                        </span>
                      ) : '—'}
                    </td>
                  </tr>
                ))}
                {state.rows.length === 0 && <tr><td colSpan={4} className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{u.empty}</td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <p className="pl-3 text-xs text-[var(--shell-crumb-text)]">{u.total.replace('{count}', String(state.rows.length))} · {fmtTime(new Date().toISOString())}</p>
      </div>
    </div>
  )
}
