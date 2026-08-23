// API key 管理页(组织与权限组,sysadmin):免登录密钥签发/吊销。
// 契约: GET /api-keys → {items:[apikey.APIKey]};POST → {plainKey} 仅返回一次;DELETE /:id 吊销。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead } from '../shared'
import { ApiKeyFormDrawer, type ApiKeyFormValues } from './KeyForm'
import { buildCreatePayload } from './payload'
import { formatTime } from '../../base/audit/logic'
import { useConfirm } from '../../../components/ConfirmDialog'
import { EmptyState, CopyButton } from '../../../components/business'

export interface ApiKeyRow {
  id: number
  subjectType: string
  subjectRef: number
  subjectName: string
  name: string
  keyPrefix: string
  status: number
  lastUsedAt: string
  expiresAt: string
  createdAt: string
}

// 主体类型枚举 -> 短标签(列表展示用,语言无关)。
function subjectLabel(t: string): string {
  if (t === 'account') return 'Account'
  if (t === 'worker') return 'Worker'
  if (t === 'customer') return 'Customer'
  return t || 'unknown'
}

// 绑定账号列渲染:有展示名用展示名 + 主体类型,无则退到 `type:#ref`,绝不渲染 `#undefined`。
export function subjectCell(row: ApiKeyRow): string {
  const tag = subjectLabel(row.subjectType)
  if (row.subjectName && row.subjectType) return `${row.subjectName} (${tag})`
  return `${tag}:#${row.subjectRef}`
}

export default function ApiKeyPage() {
  const t = useT()
  const confirmDialog = useConfirm()
  const [rows, setRows] = useState<ApiKeyRow[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [form, setForm] = useState<ApiKeyFormValues | null>(null)
  const [formError, setFormError] = useState('')
  const [plainKey, setPlainKey] = useState('')
  const [busy, setBusy] = useState(false)

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

  const revoke = async (id: number) => {
    if (busy) return
    if (!(await confirmDialog(t.pages.apikey.revokeConfirm, { danger: true }))) return
    setBusy(true)
    try {
      await apiFetch(`/api-keys/${id}`, { method: 'DELETE' })
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : t.pages.apikey.loadFail)
    } finally {
      setBusy(false)
    }
  }

  const kw = keyword.trim()
  const shown = rows.filter((r) => !kw
    || r.name.includes(kw) || r.subjectName.includes(kw) || r.keyPrefix.includes(kw))
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
                <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{subjectCell(r)}</td>
                <td className="break-all rounded-sm bg-black/5 px-2 py-1.5 font-mono text-xs">{r.keyPrefix ? `${r.keyPrefix}…` : '—'}</td>
                <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.status === 1 ? t.pages.apikey.active : t.pages.apikey.revoked}</td>
                <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.lastUsedAt ? formatTime(r.lastUsedAt) : t.pages.apikey.neverUsed}</td>
                <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{formatTime(r.createdAt)}</td>
                <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                  {r.status === 1 && (
                    <span className="inline-flex items-center">
                      <button disabled={busy} onClick={() => revoke(r.id)}>{t.pages.apikey.revoke}</button>
                    </span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {shown.length === 0 && !error && <EmptyState text={t.pages.company.empty} />}
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
          <div className="flex gap-2">
            <CopyButton text={plainKey} />
            <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={() => setPlainKey('')}>{t.pages.company.cancel}</button>
          </div>
        </div>
      )}

    </div>
  )
}
