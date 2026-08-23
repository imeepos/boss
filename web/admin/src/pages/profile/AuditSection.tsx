// 操作审计分区:本人最近 20 条审计日志列表;403 单独提示。
import { useEffect, useState } from 'react'
import { useProfile } from '../../layouts/profile'
import { apiFetch } from '../../api/client'
import { ApiError } from '../../api/envelope'
import { toAuditLog, type AuditEntry, type AuditLog } from '../base/audit/logic'
import { useT } from '../../i18n'
import { LIST_BTN, PAGE, SectionTitle, TIP } from './shared'
import { EmptyState } from '../../components/business/feedback'

export function AuditSection() {
  const t = useT()
  const a = t.pages.profile.audit
  const profile = useProfile()
  const [rows, setRows] = useState<AuditLog[]>([])
  const [error, setError] = useState('')
  const [denied, setDenied] = useState(false)

  const load = () => {
    setError(''); setDenied(false)
    apiFetch<{ items: AuditEntry[] }>('/audit-logs', { query: { accountId: profile.accountId, limit: 20 } })
      .then((d) => setRows((d?.items ?? []).map(toAuditLog)))
      .catch((e) => {
        if (e instanceof ApiError && e.code === 403) setDenied(true)
        else setError(e instanceof Error ? e.message : a.loadFail)
      })
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className={PAGE}>
      <SectionTitle title={a.title} desc={a.desc} />
      {error && <div className={TIP}>{error}</div>}
      {denied && <div className={TIP}>{a.loadFail}</div>}
      {!error && !denied && (
        <div className="border-t border-[var(--shell-side-border)]">
          {rows.length === 0 && <div className="px-1 py-6"><EmptyState text={a.empty} /></div>}
          {rows.map((r) => (
            <button key={r.logId} className={LIST_BTN}>
              <span className="grid gap-1"><strong className="text-[13px] font-medium">{r.time}</strong><small className="text-xs text-[var(--shell-content-text)]">{r.type} · {r.action} · {r.ip}</small></span>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
