// 新增订阅抽屉:事件选择器(MultiSelect + 事件目录)+ 回调端点,一次提交多事件。
// 契约:GET /openplat/event-types(目录)、POST /openplat/apps/{id}/subscriptions(eventTypes[])。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { MultiSelect, type MultiOption } from '../../../components/MultiSelect'
import { Input } from '../../../components/ui/input'
import { ErrorBanner } from '../../../components/business/page-head'

interface SubFormProps {
  appId: number
  onClose: () => void
  onCreated: () => void
}

export function SubForm({ appId, onClose, onCreated }: SubFormProps) {
  const t = useT()
  const [events, setEvents] = useState<string[]>([])
  const [options, setOptions] = useState<MultiOption[]>([])
  const [loadError, setLoadError] = useState(false)
  const [endpoint, setEndpoint] = useState('')
  const [formError, setFormError] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    apiFetch<{ items: { type: string; description: string }[] }>('/openplat/event-types')
      .then((d) => setOptions((d?.items ?? []).map((e) => ({ value: e.type, label: e.type, description: e.description }))))
      .catch(() => setLoadError(true))
  }, [])

  const submit = async () => {
    if (busy) return
    if (events.length === 0) { setFormError(t.pages.openplat.evRequired); return }
    if (!endpoint.trim()) { setFormError(t.pages.openplat.epRequired); return }
    setBusy(true)
    setFormError('')
    try {
      await apiFetch(`/openplat/apps/${appId}/subscriptions`, {
        method: 'POST',
        body: { eventTypes: events, endpointUrl: endpoint.trim() },
      })
      onCreated()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : t.pages.openplat.saveFail)
      setBusy(false)
    }
  }

  return (
    <Drawer title={t.pages.openplat.addSub} onClose={onClose}
      footer={
        <>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onClose}>
            {t.common.confirmDialog.cancel}
          </button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy} onClick={submit}>
            {t.pages.openplat.addSub}
          </button>
        </>
      }>
      <div className="grid gap-3">
        <label className="text-xs text-[var(--shell-crumb-text)]">
          {t.pages.openplat.fEvents}
          <span className="mt-1 block">
            <MultiSelect values={events} options={options} onChange={setEvents}
              ariaLabel={t.pages.openplat.fEvents} placeholder={t.pages.openplat.evPlaceholder}
              searchPlaceholder={t.pages.openplat.evSearch} emptyText={t.pages.openplat.evNone} />
          </span>
          <span className="mt-1 block text-[11px] text-[var(--shell-input-placeholder)]">{loadError ? t.pages.openplat.evLoadFail : t.pages.openplat.evHint}</span>
        </label>
        <label className="text-xs text-[var(--shell-crumb-text)]">
          {t.pages.openplat.pEndpoint}
          <span className="mt-1 block">
            <Input value={endpoint} placeholder={t.pages.openplat.pEndpoint} onChange={(e) => setEndpoint(e.target.value)} />
          </span>
        </label>
        {formError && <ErrorBanner message={formError} />}
      </div>
    </Drawer>
  )
}
