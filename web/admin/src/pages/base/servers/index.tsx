// 服务端信息配置页:表格 CRUD(逻辑复用 lib/serverConfig);空列表时弹框引导配置。
import { useEffect, useState } from 'react'
import { useT } from '../../../i18n'
import { PageHead, ToolbarButton } from '../../../components/business/page-head'
import { Card } from '../../../components/ui/card'
import { Badge } from '../../../components/ui/badge'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import {
  activeServerId,
  listServers,
  removeServer,
  setActiveServerId,
  upsertServer,
  type ServerConfig,
  type ServerDraft,
} from '../../../lib/serverConfig'
import { useConfirm } from '../../../components/ConfirmDialog'

const ACT_CLS = 'mr-2.5 border-none bg-none px-0 text-xs text-[var(--color-text-link)] cursor-pointer hover:underline'
const DANGER_CLS = ACT_CLS + ' text-[var(--color-danger)]'

export default function ServersPage() {
  const t = useT()
  const confirmDialog = useConfirm()
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

  const del = async (it: ServerConfig) => {
    if (!(await confirmDialog(t.pages.servers.deleteConfirm.replace('{name}', it.name), { danger: true }))) return
    removeServer(it.id)
    reload()
    flash(t.pages.servers.deleted)
  }

  const use = async (it: ServerConfig) => {
    if (!(await confirmDialog(t.pages.servers.useConfirm.replace('{name}', it.name)))) return
    setActiveServerId(it.id)
    window.location.reload()
  }

  const errText = (key: 'name' | 'baseUrl') => {
    if (!errors[key]) return ''
    if (key === 'name') return t.pages.servers.needName
    return errors[key] === 'required' ? t.pages.servers.needUrl : t.pages.servers.invalidUrl
  }

  return (
    <div>
      <PageHead title={t.pages.servers.title} desc={t.pages.servers.desc} />
      <Card className="p-4">
        <div className="mb-3 font-semibold text-[var(--shell-heading)]">
          {t.pages.servers.cardTitle}
          <span className="ml-2 text-xs font-normal text-[var(--shell-crumb-text)]">{t.pages.servers.localOnly}</span>
        </div>
        {!items.length ? (
          <>
            <p className="mb-3 text-xs text-[var(--shell-crumb-text)]">{t.pages.servers.gateHint}</p>
            <div className="mt-3.5 flex items-center gap-2">
              <ToolbarButton primary onClick={openCreate}>{t.pages.servers.add}</ToolbarButton>
            </div>
          </>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-50">{t.pages.servers.colName}</TableHead>
                <TableHead>{t.pages.servers.colUrl}</TableHead>
                <TableHead className="w-30">{t.pages.servers.colStatus}</TableHead>
                <TableHead className="w-50">{t.pages.servers.colOp}</TableHead>
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
                    {active !== it.id && <button className={ACT_CLS} onClick={() => use(it)}>{t.pages.servers.use}</button>}
                    <button className={ACT_CLS} onClick={() => openEdit(it)}>{t.pages.servers.edit}</button>
                    <button className={DANGER_CLS} onClick={() => del(it)}>{t.pages.servers.delete}</button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
        <div className="mt-3.5 flex items-center gap-2">
          {items.length > 0 && <ToolbarButton onClick={openCreate}>{t.pages.servers.add}</ToolbarButton>}
          {toast && <span className="text-xs text-[var(--color-success)]">{toast}</span>}
        </div>
      </Card>
      {editing && (
        <div className="fixed inset-0 z-page-modal flex items-center justify-center bg-black/45" onClick={() => setEditing(null)}>
          <div className="w-95 rounded-md bg-[var(--shell-card-bg)] p-5 shadow-[var(--shadow-panel)]" onClick={(e) => e.stopPropagation()}>
            <h3 className="mb-3 text-base font-semibold text-[var(--shell-heading)]">{editing.id ? t.pages.servers.editTitle : t.pages.servers.addTitle}</h3>
            <dl className="mb-4 grid grid-cols-[80px_1fr] gap-x-3 gap-y-2 text-[13px]">
              <dt className="text-[var(--shell-crumb-text)]">{t.pages.servers.colName}</dt>
              <dd className="m-0">
                <input
                  className="h-8 w-full rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-xs text-[var(--shell-input-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--shell-input-border-focus)]"
                  value={editing.name}
                  onChange={(e) => setEditing({ ...editing, name: e.target.value })}
                  placeholder={t.pages.servers.namePlaceholder}
                />
                {errors.name && <span className="mt-1 block text-xs text-[var(--color-danger)]">{errText('name')}</span>}
              </dd>
              <dt className="text-[var(--shell-crumb-text)]">{t.pages.servers.colUrl}</dt>
              <dd className="m-0">
                <input
                  className="h-8 w-full rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-xs text-[var(--shell-input-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--shell-input-border-focus)]"
                  value={editing.baseUrl}
                  onChange={(e) => setEditing({ ...editing, baseUrl: e.target.value })}
                  placeholder={t.pages.servers.urlPlaceholder}
                />
                {errors.baseUrl && <span className="mt-1 block text-xs text-[var(--color-danger)]">{errText('baseUrl')}</span>}
              </dd>
            </dl>
            <div className="flex gap-2">
              <ToolbarButton primary onClick={submit}>{t.pages.servers.save}</ToolbarButton>
              <ToolbarButton onClick={() => setEditing(null)}>{t.pages.servers.cancel}</ToolbarButton>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
