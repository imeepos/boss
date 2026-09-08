// API 密钥分区:本人密钥列表/创建(弹窗+一次性明钥展示)/吊销。
// 403 视为无权限单独提示;文案走 i18n profile.apiKey 块。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { useProfile } from '../../layouts/profile'
import { apiFetch } from '../../api/client'
import { ApiError } from '../../api/envelope'
import { useT } from '../../i18n'
import { ToolbarButton } from '../../components/business/page-head'
import { Badge } from '../../components/ui/badge'
import { Input } from '../../components/ui/input'
import { useConfirm } from '../../components/ConfirmDialog'
import { FORM_LABEL, PAGE, TIP } from './shared'

interface OwnKeyRow {
  id: number
  subjectType: string
  subjectRef: number
  name: string
  keyPrefix: string
  status: number
  lastUsedAt: string
  createdAt: string
}

export function ApiKeySection() {
  const t = useT()
  const k = t.pages.profile.apiKey
  const confirmDialog = useConfirm()
  const profile = useProfile()
  const [rows, setRows] = useState<OwnKeyRow[]>([])
  const [error, setError] = useState('')
  const [denied, setDenied] = useState(false)
  const [keyName, setKeyName] = useState('')
  const [modalOpen, setModalOpen] = useState(false)
  const [plainKey, setPlainKey] = useState('')
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError(''); setDenied(false)
    apiFetch<{ items: OwnKeyRow[] }>('/api-keys')
      .then((d) => setRows((d?.items ?? []).filter((r) => r.subjectType === 'account' && r.subjectRef === profile.accountId)))
      .catch((e) => {
        if (e instanceof ApiError && e.code === 403) setDenied(true)
        else setError(e instanceof Error ? e.message : k.loadFail)
      })
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const create = () => {
    if (busy || !keyName.trim()) return
    setBusy(true)
    apiFetch<{ plainKey: string }>('/api-keys', {
      method: 'POST',
      body: { subjectType: 'account', subjectRef: profile.accountId, name: keyName.trim() },
    })
      .then((res) => { setModalOpen(false); setKeyName(''); setPlainKey(res?.plainKey ?? ''); load() })
      .catch((e) => toast.error(k.loadFail, { description: e instanceof Error ? e.message : undefined }))
      .finally(() => setBusy(false))
  }

  const revoke = async (id: number) => {
    if (busy) return
    if (!(await confirmDialog(t.pages.profile.apiKey.revokeConfirm, { danger: true }))) return
    setBusy(true)
    apiFetch(`/api-keys/${id}`, { method: 'DELETE' })
      .then(() => { load(); toast.success(t.pages.profile.apiKey.revoke) })
      .catch((e) => toast.error(t.pages.profile.apiKey.revoke, { description: e instanceof Error ? e.message : undefined }))
      .finally(() => setBusy(false))
  }

  const headRow = 'grid min-w-[620px] grid-cols-[1.2fr_1fr_1fr_.7fr_70px] items-center gap-4 px-4 py-3 text-[13px]'
  return (
    <div className={PAGE}>
      <div className="flex items-start justify-between gap-5 border-b border-[var(--shell-side-border)] pb-[18px]">
        <div><h1 className="m-0 mb-1.5 text-lg font-semibold text-[var(--shell-heading)]">{k.title}</h1><p className="m-0 text-[13px] text-[var(--shell-content-text)]">{k.desc}</p></div>
        <ToolbarButton primary onClick={() => { setPlainKey(''); setModalOpen(true) }}>{k.create}</ToolbarButton>
      </div>
      {denied ? <div className={TIP}>{k.denied}</div> : error ? <div className={TIP}>{error}</div> : (
        <div className="mt-6 overflow-x-auto border border-[var(--shell-side-border)]">
          <div className={headRow + ' bg-[var(--shell-menu-hover-bg)] text-xs text-[var(--shell-crumb-text)]'}>
            <span>{k.name}</span><span>{k.key}</span><span>{k.lastUsed}</span><span>{k.status}</span><span />
          </div>
          {rows.length === 0 && <div className={headRow + ' min-h-14 text-[var(--shell-heading)]'}><strong className="font-medium">{k.empty}</strong><span /><span /><span /><span /></div>}
          {rows.map((r) => (
            <div key={r.id} className={headRow + ' min-h-14 border-t border-[var(--shell-side-border)] text-[var(--shell-heading)]'}>
              <strong className="text-[13px] font-medium">{r.name}</strong>
              <span className="text-[var(--shell-content-text)]">{r.keyPrefix ? `${r.keyPrefix}…` : '—'}</span>
              <span className="text-[var(--shell-content-text)]">{r.lastUsedAt ? r.lastUsedAt : k.neverUsed}</span>
              {r.status === 1 ? <Badge variant="success">{k.active}</Badge> : <span className="text-[var(--shell-content-text)]">{t.pages.apikey.revoked}</span>}
              {r.status === 1 ? <button className="cursor-pointer border-0 bg-transparent p-0 text-xs text-[var(--color-danger)]" disabled={busy} onClick={() => revoke(r.id)}>{k.revoke}</button> : <span />}
            </div>
          ))}
        </div>)}
      <div className={TIP}>{k.securityTip}</div>
      {plainKey && (
        <div className="mt-4 flex items-center gap-3 border border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] px-4 py-3">
          <div className="text-xs text-[var(--shell-content-text)]">{t.pages.apikey.plainOnce}</div>
          <code className="min-w-0 flex-1 break-all rounded-sm bg-black/5 px-2 py-1.5 text-xs">{plainKey}</code>
          <ToolbarButton onClick={() => setPlainKey('')}>{t.pages.company.cancel}</ToolbarButton>
        </div>
      )}
      {modalOpen && (
        <div className="fixed inset-0 z-page-modal grid place-items-center bg-black/45 p-5">
          <div className="w-[min(440px,100%)] rounded-md bg-[var(--shell-card-bg)] p-6 shadow-[var(--shadow-panel)]" role="dialog" aria-modal="true">
            <div className="mb-5 flex items-center justify-between">
              <h2 className="m-0 text-[17px] text-[var(--shell-heading)]">{k.create}</h2>
              <button aria-label={t.pages.profile.cancel} className="cursor-pointer border-0 bg-transparent p-0 text-sm text-[var(--shell-crumb-text)]" onClick={() => setModalOpen(false)}>×</button>
            </div>
            <label className={FORM_LABEL}>{k.name}<Input value={keyName} onChange={(event) => setKeyName(event.target.value)} placeholder={k.namePlaceholder} autoFocus /></label>
            <div className="mt-6 flex justify-end gap-2.5">
              <ToolbarButton onClick={() => setModalOpen(false)}>{t.pages.profile.cancel}</ToolbarButton>
              <ToolbarButton primary disabled={busy || !keyName.trim()} onClick={create}>{k.create}</ToolbarButton>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
