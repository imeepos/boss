// API key 管理页(组织与权限组,sysadmin):免登录密钥签发/吊销。
// 契约: GET /api-keys → {items:[apikey.APIKey]};POST → {plainKey} 仅返回一次;DELETE /:id 吊销。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead } from '../shared'
import { ApiKeyFormDrawer, type ApiKeyFormValues } from './KeyForm'
import { buildCreatePayload } from './payload'
import { formatTime } from '../../base/audit/logic'

export interface ApiKeyRow {
  id: number
  accountId: number
  accountName?: string
  name: string
  keyPrefix: string
  status: number
  lastUsedAt: string
  expiresAt: string
  createdAt: string
}

export default function ApiKeyPage() {
  const t = useT()
  const [rows, setRows] = useState<ApiKeyRow[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [form, setForm] = useState<ApiKeyFormValues | null>(null)
  const [formError, setFormError] = useState('')
  const [plainKey, setPlainKey] = useState('')
  const [busy, setBusy] = useState(false)
  const [revokeId, setRevokeId] = useState<number>(0)

  const load = () => {
    setError('')
    apiFetch<{ items: ApiKeyRow[] }>('/api-keys')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : t.pages.apikey.loadFail))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const submit = async () => {
    if (!form || busy) return
    setBusy(true)
    setFormError('')
    try {
      const res = await apiFetch<{ plainKey: string }>('/api-keys', {
        method: 'POST',
        body: buildCreatePayload(form.accountId, form.name),
      })
      setForm(null)
      setPlainKey(res?.plainKey ?? '')
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : t.pages.apikey.saveFail)
    } finally {
      setBusy(false)
    }
  }

  const revoke = async () => {
    if (!revokeId || busy) return
    setBusy(true)
    try {
      await apiFetch(`/api-keys/${revokeId}`, { method: 'DELETE' })
      setRevokeId(0)
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : t.pages.apikey.loadFail)
      setRevokeId(0)
    } finally {
      setBusy(false)
    }
  }

  const kw = keyword.trim()
  const shown = rows.filter((r) => !kw
    || r.name.includes(kw) || (r.accountName ?? '').includes(kw) || r.keyPrefix.includes(kw))
  const cols = t.pages.apikey.columns

  return (
    <div className="">
      <PageHead title={t.pages.apikey.title} desc={t.pages.apikey.desc} />
      <div className="flex flex-wrap items-center gap-2 p-4">
        <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={keyword}
          placeholder={t.pages.apikey.searchPlaceholder} onChange={(e) => setKeyword(e.target.value)} />
        <span className="spacer" />
        <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={load}>{t.pages.audit.refresh}</button>
        <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => { setPlainKey(''); setForm({ accountId: 0, name: '' }) }}>
          {t.pages.apikey.create}
        </button>
      </div>
      <div className="overflow-x-auto px-4 pb-4">
        <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
          <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{cols.map((c) => <th key={c} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{c}</th>)}</tr></thead>
          <tbody>
            {shown.map((r) => (
              <tr key={r.id}>
                <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.name}</td>
                <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.accountName || `#${r.accountId}`}</td>
                <td className="break-all rounded-sm bg-black/5 px-2 py-1.5 font-mono text-xs">{r.keyPrefix ? `${r.keyPrefix}…` : '—'}</td>
                <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.status === 1 ? t.pages.apikey.active : t.pages.apikey.revoked}</td>
                <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.lastUsedAt ? formatTime(r.lastUsedAt) : t.pages.apikey.neverUsed}</td>
                <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{formatTime(r.createdAt)}</td>
                <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                  {r.status === 1 && (
                    <span className="inline-flex items-center">
                      <button onClick={() => setRevokeId(r.id)}>{t.pages.apikey.revoke}</button>
                    </span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {shown.length === 0 && !error && <div className="py-8 text-center text-[13px] text-[var(--shell-group-title)]">{t.pages.company.empty}</div>}
        {error && <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div>}
      </div>

      <ApiKeyFormDrawer
        open={form !== null}
        values={form ?? { accountId: 0, name: '' }}
        onChange={setForm}
        onClose={() => setForm(null)}
        onSubmit={submit}
        busy={busy}
        submitError={formError}
      />

      {plainKey && (
        <div className="fixed bottom-6 right-6 z-[60] flex max-w-[420px] flex-col gap-2 rounded-md border border-[var(--shell-fab-bg)] bg-[var(--shell-card-bg)] p-4 text-[13px] text-[var(--shell-content-text)] shadow-[0_6px_24px_rgba(0,0,0,0.18)]">
          <div>{t.pages.apikey.plainOnce}</div>
          <code className="break-all rounded-sm bg-black/5 px-2 py-1.5 font-mono text-xs">{plainKey}</code>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={() => setPlainKey('')}>{t.pages.company.cancel}</button>
        </div>
      )}

      {revokeId > 0 && (
        <div className="fixed inset-0 z-[120] flex items-center justify-center bg-black/45">
          <div className="w-90 rounded-md bg-[var(--shell-card-bg)] p-5">
            <p>{t.pages.apikey.revokeConfirm}</p>
            <div className="flex justify-end gap-2">
              <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={() => setRevokeId(0)}>{t.pages.company.cancel}</button>
              <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy} onClick={revoke}>
                {busy ? t.pages.account.submitting : t.pages.apikey.revoke}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
