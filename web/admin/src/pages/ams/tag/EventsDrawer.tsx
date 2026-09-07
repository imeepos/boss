// 标签事件流抽屉:契约 GET /tags/{id}/events;时间线展示 BIND/UNBIND/RECYCLE 轨迹。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { EmptyState } from '../../../components/business'
import type { TagEventRow } from '../types'

const dotOf: Record<string, string> = {
  BIND: 'var(--color-success)',
  UNBIND: 'var(--color-warning)',
  RECYCLE: 'var(--color-danger)',
}

function fmt(at: string): string {
  const d = new Date(at)
  return Number.isNaN(d.getTime()) ? at : d.toLocaleString('sv-SE')
}

export function TagEventsDrawer({ tag, onClose }: { tag: { tagId: number; tagNo: string }; onClose: () => void }) {
  const t = useT()
  const g = t.pages.tagPage
  const [rows, setRows] = useState<TagEventRow[]>([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(true)

  useEffect(() => {
    setBusy(true)
    apiFetch<{ items: TagEventRow[] }>('/tags/' + tag.tagId + '/events')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : g.loadFail))
      .finally(() => setBusy(false))
  }, [tag.tagId]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <Drawer title={g.eventsTitle.replace('{no}', tag.tagNo || '#' + tag.tagId)} onClose={onClose} width={480}>
      {error ? (
        <div className="rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div>
      ) : busy ? (
        <div className="py-8 text-center text-[13px] text-[var(--shell-group-title)]">{t.common.loading}</div>
      ) : !rows.length ? (
        <EmptyState text={g.empty} />
      ) : (
        <ol className="m-0 flex list-none flex-col gap-0 p-0">
          {rows.map((ev, i) => (
            <li key={ev.id ?? i} className="relative flex gap-3 pb-4 pl-1 last:pb-0">
              {i < rows.length - 1 && <span className="absolute top-4 bottom-0 left-[6px] w-px bg-[var(--shell-side-border)]" aria-hidden />}
              <span className="mt-1.5 h-3 w-3 shrink-0 rounded-full border-2 border-[var(--shell-card-bg)]"
                style={{ background: dotOf[ev.action] ?? 'var(--shell-group-title)' }} aria-hidden />
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="font-mono text-[12px] font-medium text-[var(--shell-heading)]">{ev.action}</span>
                  <span className="text-[12px] text-[var(--shell-group-title)]">{fmt(ev.createdAt)}</span>
                </div>
                {ev.detail && <div className="mt-0.5 text-[12px] break-all text-[var(--shell-group-title)]">{ev.detail}</div>}
              </div>
            </li>
          ))}
        </ol>
      )}
    </Drawer>
  )
}
