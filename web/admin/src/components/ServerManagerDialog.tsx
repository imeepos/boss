// 服务端配置管理弹框:增(新增表单)删(删除)改(编辑)查(列表) + 启用。
// 阻断模式(blocking)用于"无任何配置"时的强制门禁;非阻断用于登录页"管理服务端"入口。
import { useState } from 'react'
import { useT } from '../i18n'
import {
  activeServerId,
  listServers,
  removeServer,
  setActiveServerId,
  upsertServer,
  type ServerDraft,
} from '../lib/serverConfig'
import '../pages/base/servers/servers.css'

export interface ServerManagerDialogProps {
  /** 阻断模式:无关闭按钮,启用任一服务端后回调 onApply。 */
  blocking?: boolean
  onClose?: () => void
  /** 启用服务端后回调(阻断门禁用它放行;非阻断方自行刷新选择器状态)。 */
  onApply?: (activeId: string) => void
}

export function ServerManagerDialog({ blocking, onClose, onApply }: ServerManagerDialogProps) {
  const t = useT()
  const [items, setItems] = useState(listServers)
  const [active, setActive] = useState(activeServerId)
  const [editing, setEditing] = useState<ServerDraft | null>(null)
  const [errors, setErrors] = useState<{ name?: string; baseUrl?: string }>({})

  const reload = () => {
    setItems(listServers())
    setActive(activeServerId())
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
  }

  const del = (id: string) => {
    const name = items.find((it) => it.id === id)?.name ?? ''
    if (!window.confirm(t.pages.servers.deleteConfirm.replace('{name}', name))) return
    removeServer(id)
    reload()
  }

  const use = (id: string) => {
    const name = items.find((it) => it.id === id)?.name ?? ''
    if (!blocking && !window.confirm(t.pages.servers.useConfirm.replace('{name}', name))) return
    setActiveServerId(id)
    if (blocking) onApply?.(id)
    else window.location.reload()
  }

  const errText = (key: 'name' | 'baseUrl') => {
    if (!errors[key]) return ''
    if (key === 'name') return t.pages.servers.needName
    return errors[key] === 'required' ? t.pages.servers.needUrl : t.pages.servers.invalidUrl
  }

  return (
    <div className="modal-mask">
      <div className="modal server-manager" style={{ width: 520 }} onClick={(e) => e.stopPropagation()}>
        <h3>{t.pages.servers.manageTitle}</h3>
        {blocking && <p className="page-desc">{t.pages.servers.gateHint}</p>}
        <table className="tbl">
          <thead>
            <tr>
              <th>{t.pages.servers.colName}</th>
              <th>{t.pages.servers.colUrl}</th>
              <th style={{ width: 90 }}>{t.pages.servers.colStatus}</th>
              <th style={{ width: 170 }}>{t.pages.servers.colOp}</th>
            </tr>
          </thead>
          <tbody>
            {items.map((it) => (
              <tr key={it.id}>
                <td>{it.name}</td>
                <td>{it.baseUrl}</td>
                <td>{active === it.id ? <span className="tag tag-blue">{t.pages.servers.current}</span> : <span className="tag">{t.pages.servers.ready}</span>}</td>
                <td>
                  {active !== it.id && <a onClick={() => use(it.id)}>{t.pages.servers.use}</a>}
                  <a onClick={() => { setErrors({}); setEditing({ id: it.id, name: it.name, baseUrl: it.baseUrl }) }}>{t.pages.servers.edit}</a>
                  <a className="danger" onClick={() => del(it.id)}>{t.pages.servers.delete}</a>
                </td>
              </tr>
            ))}
            {!items.length && <tr><td colSpan={4} className="empty">{t.pages.servers.empty}</td></tr>}
          </tbody>
        </table>
        <div className="params-actions">
          <button className="btn btn-primary" onClick={() => { setErrors({}); setEditing({ name: '', baseUrl: '' }) }}>
            {t.pages.servers.add}
          </button>
          {!blocking && <button className="btn" onClick={onClose}>{t.pages.servers.cancel}</button>}
        </div>
        {editing && (
          <div style={{ marginTop: 12 }}>
            <h4 style={{ margin: '0 0 8px' }}>{editing.id ? t.pages.servers.editTitle : t.pages.servers.addTitle}</h4>
            <input
              className="ctl"
              style={{ width: '100%', marginBottom: 6 }}
              placeholder={t.pages.servers.namePlaceholder}
              value={editing.name}
              onChange={(e) => setEditing({ ...editing, name: e.target.value })}
            />
            {errors.name && <div className="error-inline">{errText('name')}</div>}
            <input
              className="ctl"
              style={{ width: '100%', marginBottom: 6 }}
              placeholder={t.pages.servers.urlPlaceholder}
              value={editing.baseUrl}
              onChange={(e) => setEditing({ ...editing, baseUrl: e.target.value })}
            />
            {errors.baseUrl && <div className="error-inline">{errText('baseUrl')}</div>}
            <div style={{ display: 'flex', gap: 8 }}>
              <button className="btn btn-primary" onClick={submit}>{t.pages.servers.save}</button>
              <button className="btn" onClick={() => { setEditing(null); setErrors({}) }}>{t.pages.servers.cancel}</button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
