// 服务端配置管理弹框:增(新增表单)删(删除)改(编辑)查(列表) + 启用。
// 阻断模式(blocking)用于"无任何配置"时的强制门禁;非阻断用于登录页"管理服务端"入口。
import { useState } from 'react'
import { useT } from '../i18n'
import { Badge } from './ui/badge'
import { ToolbarButton } from './business/page-head'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from './ui/table'
import {
  activeServerId,
  listServers,
  removeServer,
  setActiveServerId,
  upsertServer,
  type ServerDraft,
} from '../lib/serverConfig'
import { useConfirm } from './ConfirmDialog'

export interface ServerManagerDialogProps {
  /** 阻断模式:无关闭按钮,启用任一服务端后回调 onApply。 */
  blocking?: boolean
  onClose?: () => void
  /** 启用服务端后回调(阻断门禁用它放行;非阻断方自行刷新选择器状态)。 */
  onApply?: (activeId: string) => void
}

const INPUT_CLS = 'h-8 w-full rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-xs text-[var(--shell-input-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--shell-input-border-focus)]'
const ACT_CLS = 'mr-2.5 border-none bg-none px-0 text-xs text-[var(--color-text-link)] cursor-pointer hover:underline'

export function ServerManagerDialog({ blocking, onClose, onApply }: ServerManagerDialogProps) {
  const t = useT()
  const confirmDialog = useConfirm()
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

  const del = async (id: string) => {
    const name = items.find((it) => it.id === id)?.name ?? ''
    if (!(await confirmDialog(t.pages.servers.deleteConfirm.replace('{name}', name), { danger: true }))) return
    removeServer(id)
    reload()
  }

  const use = async (id: string) => {
    const name = items.find((it) => it.id === id)?.name ?? ''
    if (!blocking && !(await confirmDialog(t.pages.servers.useConfirm.replace('{name}', name)))) return
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
    <div className="fixed inset-0 z-page-modal flex items-center justify-center bg-black/45">
      <div className="max-h-[80vh] w-130 overflow-auto rounded-md bg-[var(--shell-card-bg)] p-5 shadow-[var(--shadow-panel)]" onClick={(e) => e.stopPropagation()}>
        <h3 className="mb-3 text-base font-semibold text-[var(--shell-heading)]">{t.pages.servers.manageTitle}</h3>
        {blocking && <p className="mb-2.5 text-xs text-[var(--shell-crumb-text)]">{t.pages.servers.gateHint}</p>}
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t.pages.servers.colName}</TableHead>
              <TableHead>{t.pages.servers.colUrl}</TableHead>
              <TableHead className="w-22">{t.pages.servers.colStatus}</TableHead>
              <TableHead className="w-42">{t.pages.servers.colOp}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((it) => (
              <TableRow key={it.id}>
                <TableCell>{it.name}</TableCell>
                <TableCell>{it.baseUrl}</TableCell>
                <TableCell>
                  {active === it.id
                    ? <Badge variant="info">{t.pages.servers.current}</Badge>
                    : <Badge>{t.pages.servers.ready}</Badge>}
                </TableCell>
                <TableCell>
                  {active !== it.id && <button className={ACT_CLS} onClick={() => use(it.id)}>{t.pages.servers.use}</button>}
                  <button className={ACT_CLS} onClick={() => { setErrors({}); setEditing({ id: it.id, name: it.name, baseUrl: it.baseUrl }) }}>{t.pages.servers.edit}</button>
                  <button className={ACT_CLS + ' text-[var(--color-danger)]'} onClick={() => del(it.id)}>{t.pages.servers.delete}</button>
                </TableCell>
              </TableRow>
            ))}
            {!items.length && <TableRow><TableCell colSpan={4} className="py-6 text-center text-xs text-[var(--shell-group-title)]">{t.pages.servers.empty}</TableCell></TableRow>}
          </TableBody>
        </Table>
        <div className="mt-3.5 flex items-center gap-2">
          <ToolbarButton primary onClick={() => { setErrors({}); setEditing({ name: '', baseUrl: '' }) }}>
            {t.pages.servers.add}
          </ToolbarButton>
          {!blocking && <ToolbarButton onClick={() => onClose?.()}>{t.pages.servers.cancel}</ToolbarButton>}
        </div>
        {editing && (
          <div className="mt-3">
            <h4 className="mb-2 m-0 font-semibold text-[var(--shell-heading)]">{editing.id ? t.pages.servers.editTitle : t.pages.servers.addTitle}</h4>
            <input
              className={INPUT_CLS + ' mb-1.5'}
              placeholder={t.pages.servers.namePlaceholder}
              value={editing.name}
              onChange={(e) => setEditing({ ...editing, name: e.target.value })}
            />
            {errors.name && <div className="mb-1 block text-xs text-[var(--color-danger)]">{errText('name')}</div>}
            <input
              className={INPUT_CLS + ' mb-1.5'}
              placeholder={t.pages.servers.urlPlaceholder}
              value={editing.baseUrl}
              onChange={(e) => setEditing({ ...editing, baseUrl: e.target.value })}
            />
            {errors.baseUrl && <div className="mb-1 block text-xs text-[var(--color-danger)]">{errText('baseUrl')}</div>}
            <div className="flex gap-2">
              <ToolbarButton primary onClick={submit}>{t.pages.servers.save}</ToolbarButton>
              <ToolbarButton onClick={() => { setEditing(null); setErrors({}) }}>{t.pages.servers.cancel}</ToolbarButton>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
