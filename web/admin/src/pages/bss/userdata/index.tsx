// 用户端配置页:GET 列表族 + 行级开关/停用(userdata.yaml 配置管理面)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead } from '../../org/shared'
import { fmtTime } from '../../../lib/format'
import { TABS, type Row, type TabDef } from './tabs'
import '../user/user.css'

type Loader = { rows: Row[]; error: string; busy: boolean }

export default function UserDataPage() {
  const t = useT()
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

  const act = (def: TabDef, row: Row) => {
    if (!window.confirm(u.actConfirm.replace('{id}', String(row.id)))) return
    apiFetch(`${def.actionPath}/${encodeURIComponent(String(row.id))}/${def.action}`, { method: 'PUT' })
      .then(() => { setNotice(u.acted); load(def) })
      .catch(() => setNotice(u.actFail))
  }

  const cur = TABS.find((x) => x.key === tab) ?? TABS[0]
  const state = st[cur.key] ?? { rows: [], error: '', busy: false }

  return (
    <div>
      <PageHead title={u.title} desc={u.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          {TABS.map((d) => (
            <button key={d.key} className={'org-btn' + (d.key === tab ? ' primary' : '')}
              onClick={() => { setTab(d.key); if (!st[d.key]) load(d) }}>{u.tabs[d.key]}</button>
          ))}
          <span className="spacer" />
          <button className="org-btn" disabled={state.busy} onClick={() => load(cur)}>
            {t.pages.audit.refresh}
          </button>
        </div>
        {notice ? <div className="org-error">{notice}</div> : null}
        {state.error ? <div className="org-error">{state.error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr><th>ID</th><th>{u.nameCol}</th><th>{u.statusCol}</th><th>{u.opCol}</th></tr></thead>
              <tbody>
                {state.rows.map((r) => (
                  <tr key={String(r.id)}>
                    <td>{String(r.id)}</td>
                    <td>{String(r.name ?? r.title ?? r.question ?? r.code ?? r.label ?? '—')}</td>
                    <td>{String(r.status ?? (r.enabled ? u.on : u.off) ?? '—')}</td>
                    <td>
                      {cur.action ? (
                        <span className="org-act">
                          <button onClick={() => act(cur, r)}>{cur.action === "disable" ? u.disable : u.toggle}</button>
                        </span>
                      ) : '—'}
                    </td>
                  </tr>
                ))}
                {state.rows.length === 0 && <tr><td colSpan={4}>{u.empty}</td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <p className="user-muted">{u.total.replace('{count}', String(state.rows.length))} · {fmtTime(new Date().toISOString())}</p>
      </div>
    </div>
  )
}
