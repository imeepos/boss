// API key 管理页(组织与权限组,sysadmin):免登录密钥签发/吊销。
// 契约: GET /api-keys → {items:[apikey.APIKey]};POST → {plainKey} 仅返回一次;DELETE /:id 吊销。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead } from '../shared'
import { ApiKeyFormDrawer, type ApiKeyFormValues } from './KeyForm'
import { formatTime } from '../../base/audit/logic'
import '../../base/account/account.css'
import '../org.css'

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
        body: { accountId: form.accountId, name: form.name.trim() },
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
    <div className="org-page">
      <PageHead title={t.pages.apikey.title} desc={t.pages.apikey.desc} />
      <div className="org-toolbar">
        <input className="org-input org-search" value={keyword}
          placeholder={t.pages.apikey.searchPlaceholder} onChange={(e) => setKeyword(e.target.value)} />
        <span className="spacer" />
        <button className="org-btn" onClick={load}>{t.pages.audit.refresh}</button>
        <button className="org-btn org-btn-primary" onClick={() => { setPlainKey(''); setForm({ accountId: 0, name: '' }) }}>
          {t.pages.apikey.create}
        </button>
      </div>
      <div className="org-table-wrap">
        <table className="org-table">
          <thead><tr>{cols.map((c) => <th key={c}>{c}</th>)}</tr></thead>
          <tbody>
            {shown.map((r) => (
              <tr key={r.id}>
                <td>{r.name}</td>
                <td>{r.accountName || `#${r.accountId}`}</td>
                <td className="mono">{r.keyPrefix ? `${r.keyPrefix}…` : '—'}</td>
                <td>{r.status === 1 ? t.pages.apikey.active : t.pages.apikey.revoked}</td>
                <td>{r.lastUsedAt ? formatTime(r.lastUsedAt) : t.pages.apikey.neverUsed}</td>
                <td>{formatTime(r.createdAt)}</td>
                <td>
                  {r.status === 1 && (
                    <span className="org-act">
                      <button onClick={() => setRevokeId(r.id)}>{t.pages.apikey.revoke}</button>
                    </span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {shown.length === 0 && !error && <div className="org-empty">{t.pages.company.empty}</div>}
        {error && <div className="org-error">{error}</div>}
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
        <div className="apikey-plain-banner">
          <div>{t.pages.apikey.plainOnce}</div>
          <code className="mono">{plainKey}</code>
          <button className="org-btn" onClick={() => setPlainKey('')}>{t.pages.company.cancel}</button>
        </div>
      )}

      {revokeId > 0 && (
        <div className="acc-confirm-mask">
          <div className="acc-confirm">
            <p>{t.pages.apikey.revokeConfirm}</p>
            <div className="acc-confirm-actions">
              <button className="org-btn" onClick={() => setRevokeId(0)}>{t.pages.company.cancel}</button>
              <button className="org-btn org-btn-primary" disabled={busy} onClick={revoke}>
                {busy ? t.pages.account.submitting : t.pages.apikey.revoke}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
