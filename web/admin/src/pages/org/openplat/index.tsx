// 开放平台开发者门户(M6):应用凭证 / Webhook 订阅 / 投递 outbox 管理。
// 契约: /openplat/apps(系列)、/openplat/deliveries、/openplat/subscriptions/{id};
// Secret 仅创建响应返回一次(与 apikey 页同安全约定)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead } from '../shared'
import { formatTime } from '../../base/audit/logic'
import { EmptyState, CopyButton } from '../../../components/business'
import { AppFormDrawer, type AppFormValues } from './AppForm'
import { AppDetail } from './AppDetail'

export interface OpenAppRow {
  id: number
  appId: string
  name: string
  status: number
  rateLimitRpm: number
  dailyQuota: number
  sandbox: boolean
  lastUsedAt: string
  createdAt: string
}

export default function OpenPlatPage() {
  const t = useT()
  const [rows, setRows] = useState<OpenAppRow[]>([])
  const [error, setError] = useState('')
  const [form, setForm] = useState<AppFormValues | null>(null)
  const [formError, setFormError] = useState('')
  const [secret, setSecret] = useState('')
  const [busy, setBusy] = useState(false)
  const [selected, setSelected] = useState<number>(0)

  const load = () => {
    setError('')
    apiFetch<{ items: OpenAppRow[] }>('/openplat/apps')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : t.pages.openplat.loadFail))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const submit = async () => {
    if (!form || busy) return
    const name = form.name.trim()
    if (!name) {
      setFormError(t.pages.openplat.nameRequired)
      return
    }
    setBusy(true)
    setFormError('')
    try {
      const res = await apiFetch<{ secret: string }>('/openplat/apps', {
        method: 'POST',
        body: {
          name,
          rateLimitRpm: form.rateLimitRpm,
          dailyQuota: form.dailyQuota,
          sandbox: form.sandbox,
        },
      })
      setForm(null)
      setSecret(res?.secret ?? '')
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : t.pages.openplat.saveFail)
    } finally {
      setBusy(false)
    }
  }

  const toggleStatus = async (row: OpenAppRow) => {
    if (busy) return
    setBusy(true)
    try {
      await apiFetch(`/openplat/apps/${row.id}/status`, {
        method: 'PUT',
        body: { status: row.status === 1 ? 0 : 1 },
      })
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : t.pages.openplat.loadFail)
    } finally {
      setBusy(false)
    }
  }

  const cols = t.pages.openplat.columns
  const sel = rows.find((r) => r.id === selected)

  return (
    <div>
      <PageHead title={t.pages.openplat.title} desc={t.pages.openplat.desc} />
      <div className="flex flex-wrap items-center gap-2 p-4">
        <span className="text-[13px] text-[var(--shell-group-title)]">{t.pages.openplat.count.replace('{n}', String(rows.length))}</span>
        <span className="spacer" />
        <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={load}>{t.pages.audit.refresh}</button>
        <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => { setSecret(''); setForm({ name: '', rateLimitRpm: 60, dailyQuota: 10000, sandbox: false }) }}>
          {t.pages.openplat.create}
        </button>
      </div>
      <div className="overflow-x-auto px-4 pb-4">
        <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
          <thead><tr>{cols.map((c) => (
            <th key={c} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{c}</th>
          ))}</tr></thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.id} className="hover:bg-[var(--shell-menu-hover-bg)]">
                <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)]">
                  <button className="cursor-pointer text-[var(--shell-fab-bg)] underline-offset-2 hover:underline" onClick={() => setSelected(r.id)}>{r.name}</button>
                </td>
                <td className="break-all px-3 py-2 border-b border-[var(--shell-side-border)] font-mono text-xs">{r.appId}</td>
                <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)]">{r.status === 1 ? t.pages.openplat.active : t.pages.openplat.disabled}</td>
                <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)]">{r.sandbox ? t.pages.openplat.sandboxOn : '—'}</td>
                <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)]">{r.rateLimitRpm} / {r.dailyQuota}</td>
                <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)]">{r.lastUsedAt ? formatTime(r.lastUsedAt) : t.pages.openplat.neverUsed}</td>
                <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)]">
                  <span className="inline-flex items-center gap-3">
                    <button className="cursor-pointer text-[var(--color-text-link)] hover:underline disabled:cursor-not-allowed disabled:opacity-50" disabled={busy} onClick={() => setSelected(r.id)}>{t.pages.openplat.view}</button>
                    <span className="text-[var(--shell-side-border)]">|</span>
                    <button className="cursor-pointer hover:underline disabled:cursor-not-allowed disabled:opacity-50" disabled={busy} onClick={() => toggleStatus(r)}>{r.status === 1 ? t.pages.openplat.disable : t.pages.openplat.enable}</button>
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {rows.length === 0 && !error && <EmptyState text={t.pages.openplat.empty} />}
        {error && <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div>}
      </div>

      {sel && <AppDetail app={sel} onRefreshApps={load} onClose={() => setSelected(0)} />}

      <AppFormDrawer
        open={form !== null}
        values={form ?? { name: '', rateLimitRpm: 60, dailyQuota: 10000, sandbox: false }}
        onChange={setForm}
        onClose={() => setForm(null)}
        onSubmit={submit}
        busy={busy}
        submitError={formError}
      />

      {secret && (
        <div className="fixed bottom-6 right-6 z-[60] flex max-w-[460px] flex-col gap-2 rounded-md border border-[var(--shell-fab-bg)] bg-[var(--shell-card-bg)] p-4 text-[13px] text-[var(--shell-content-text)] shadow-[0_6px_24px_rgba(0,0,0,0.18)]">
          <div>{t.pages.openplat.secretOnce}</div>
          <code className="break-all rounded-sm bg-black/5 px-2 py-1.5 font-mono text-xs">{secret}</code>
          <div className="flex gap-2">
            <CopyButton text={secret} />
            <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={() => setSecret('')}>{t.pages.company.cancel}</button>
          </div>
        </div>
      )}
    </div>
  )
}
