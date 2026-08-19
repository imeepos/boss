// 服务端信息配置页:表格 CRUD(逻辑复用 lib/serverConfig);空列表时弹框引导配置。
import { useEffect, useState } from 'react'
import { useT } from '../../../i18n'
import {
  activeServerId,
  listServers,
  removeServer,
  setActiveServerId,
  upsertServer,
  type ServerConfig,
  type ServerDraft,
} from '../../../lib/serverConfig'
import './servers.css'

export default function ServersPage() {
  const t = useT()
  const [items, setItems] = useState<ServerConfig[]>([])
  const [active, setActive] = useState<string | null>(null)
  const [editing, setEditing] = useState<ServerDraft | null>(null)
  const [errors, setErrors] = useState<{ name?: string; baseUrl?: string }>({})
  const [toast, setToast] = useState('')

  const reload = () => {
    setItems(listServers())
    setActive(activeServerId())
  }

  useEffect(reload, [])

  const flash = (msg: string) => {
    setToast(msg)
    window.setTimeout(() => setToast(''), 2500)
  }

  const openCreate = () => {
    setErrors({})
    setEditing({ name: '', baseUrl: '' })
  }

  const openEdit = (it: ServerConfig) => {
    setErrors({})
    setEditing({ id: it.id, name: it.name, baseUrl: it.baseUrl })
  }

  const submit = () => {
    if (!editing) return
    const r = upsertServer(editing)
    if (!r.ok) {
      setErrors({ [r.error === 'name' ? 'name' : 'baseUrl']: r.error === 'name' ? 'required' : 'invalid' })
      return
    }
    setEditing(null)
    setErrors({})
    reload()
    flash(t.pages.servers.saved)
  }

  const del = (it: ServerConfig) => {
    if (!window.confirm(t.pages.servers.deleteConfirm.replace('{name}', it.name))) return
    removeServer(it.id)
    reload()
    flash(t.pages.servers.deleted)
  }

  const use = (it: ServerConfig) => {
    if (!window.confirm(t.pages.servers.useConfirm.replace('{name}', it.name))) return
    setActiveServerId(it.id)
    window.location.reload()
  }

  const errText = (key: 'name' | 'baseUrl') => {
    if (!errors[key]) return ''
    if (key === 'name') return t.pages.servers.needName
    return errors[key] === 'required' ? t.pages.servers.needUrl : t.pages.servers.invalidUrl
  }

  return (
    <div className="servers-page">
      <div className="page-head">
        <h2>{t.pages.servers.title}</h2>
        <p className="page-desc">{t.pages.servers.desc}</p>
      </div>
      <div className="card">
        <div className="card-title">
          {t.pages.servers.cardTitle}
          <span className="extra">{t.pages.servers.localOnly}</span>
        </div>
        {!items.length ? (
          <>
            <p className="page-desc" style={{ marginBottom: 12 }}>{t.pages.servers.gateHint}</p>
            <div className="params-actions">
              <button className="btn btn-primary" onClick={openCreate}>{t.pages.servers.add}</button>
            </div>
          </>
        ) : (
          <table className="tbl">
            <thead>
              <tr>
                <th style={{ width: 200 }}>{t.pages.servers.colName}</th>
                <th>{t.pages.servers.colUrl}</th>
                <th style={{ width: 120 }}>{t.pages.servers.colStatus}</th>
                <th style={{ width: 200 }}>{t.pages.servers.colOp}</th>
              </tr>
            </thead>
            <tbody>
              {items.map((it) => (
                <tr key={it.id}>
                  <td>{it.name}</td>
                  <td>{it.baseUrl}</td>
                  <td>{active === it.id ? <span className="tag tag-blue">{t.pages.servers.current}</span> : <span className="tag">{t.pages.servers.ready}</span>}</td>
                  <td>
                    {active !== it.id && <a onClick={() => use(it)}>{t.pages.servers.use}</a>}
                    <a onClick={() => openEdit(it)}>{t.pages.servers.edit}</a>
                    <a className="danger" onClick={() => del(it)}>{t.pages.servers.delete}</a>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
        <div className="params-actions">
          {items.length > 0 && <button className="btn" onClick={openCreate}>{t.pages.servers.add}</button>}
          {toast && <span className="toast">{toast}</span>}
        </div>
      </div>
      {editing && (
        <div className="modal-mask" onClick={() => setEditing(null)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h3>{editing.id ? t.pages.servers.editTitle : t.pages.servers.addTitle}</h3>
            <dl>
              <dt>{t.pages.servers.colName}</dt>
              <dd>
                <input
                  className="ctl"
                  style={{ width: '100%' }}
                  value={editing.name}
                  onChange={(e) => setEditing({ ...editing, name: e.target.value })}
                  placeholder={t.pages.servers.namePlaceholder}
                />
                {errors.name && <span className="error-inline">{errText('name')}</span>}
              </dd>
              <dt>{t.pages.servers.colUrl}</dt>
              <dd>
                <input
                  className="ctl"
                  style={{ width: '100%' }}
                  value={editing.baseUrl}
                  onChange={(e) => setEditing({ ...editing, baseUrl: e.target.value })}
                  placeholder={t.pages.servers.urlPlaceholder}
                />
                {errors.baseUrl && <span className="error-inline">{errText('baseUrl')}</span>}
              </dd>
            </dl>
            <button className="btn btn-primary" onClick={submit}>{t.pages.servers.save}</button>
            <button className="btn" onClick={() => setEditing(null)}>{t.pages.servers.cancel}</button>
          </div>
        </div>
      )}
    </div>
  )
}
