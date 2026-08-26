// Stripe 支付配置页:通道/回调两卡片摘要 + 抽屉式编辑 + 通道自检(余额探活)。
// 契约:GET /stripe-config(掩码)、PUT /stripe-config/{channel|webhook}、POST /stripe-config/channel/test。
import { useEffect, useState, type ReactNode } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { PageHead, ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { FormField } from '../../../components/business/form-field'
import { Card } from '../../../components/ui/card'
import { Input } from '../../../components/ui/input'
import { Badge } from '../../../components/ui/badge'
import { Switch } from '../../../components/ui/switch'
import { CH_KEYS, WH_KEYS, STRIPE_SECRET_KEYS, draftFrom, stripePayloadFor, type StripeFields } from './logic'

type Draft = Record<string, string>

/** 密码输入 + 眼睛切换;占位符提示"已配置(不回显)"而非明文。 */
function SecretInput({ value, onChange, placeholder, hasValue }: {
  value: string
  onChange: (v: string) => void
  placeholder: string
  hasValue: boolean
}) {
  const [show, setShow] = useState(false)
  return (
    <span className="flex w-72 items-center gap-1">
      <Input
        type={show ? 'text' : 'password'}
        className="w-64"
        value={value}
        placeholder={hasValue ? placeholder : ''}
        onChange={(e) => onChange(e.target.value)}
      />
      <button
        type="button"
        className="cursor-pointer border-none bg-none px-1 py-0.5 text-xs text-[var(--shell-group-title)] hover:text-[var(--shell-content-text)]"
        onClick={() => setShow((v) => !v)}
        aria-label="toggle visibility"
      >
        {show ? '隐藏' : '显示'}
      </button>
    </span>
  )
}

export default function StripeConfigPage() {
  const t = useT()
  const a = t.pages.stripeconfig
  const [draft, setDraft] = useState<Draft>({})
  const [loaded, setLoaded] = useState<Draft>({})
  const [secretSet, setSecretSet] = useState<Record<string, boolean>>({})
  const [error, setError] = useState('')
  const [saving, setSaving] = useState('')
  const [testing, setTesting] = useState(false)
  const [editing, setEditing] = useState<'channel' | 'webhook' | null>(null)

  const set = (key: string, v: string) => setDraft((d) => ({ ...d, [key]: v }))

  const load = () => {
    setError('')
    apiFetch<{ fields: StripeFields }>('/stripe-config')
      .then((d) => {
        const fields = d?.fields ?? {}
        const init = draftFrom(fields)
        setDraft(init)
        setLoaded(init)
        setSecretSet({
          'stripe.apiKey': !!fields['stripe.apiKey']?.hasValue,
          'stripe.webhookSecret': !!fields['stripe.webhookSecret']?.hasValue,
        })
      })
      .catch((e) => setError(e instanceof Error ? e.message : a.loadFail))
  }

  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const save = async (group: 'channel' | 'webhook', keys: readonly string[]) => {
    if (saving) return
    const values = stripePayloadFor(keys, draft, loaded)
    setSaving(group)
    try {
      await apiFetch(`/stripe-config/${group}`, { method: 'PUT', body: { values } })
      setLoaded((l) => ({ ...l, ...values }))
      const secretKeys = keys.filter((k) => STRIPE_SECRET_KEYS.has(k))
      for (const k of secretKeys) {
        if (values[k]) setSecretSet((s) => ({ ...s, [k]: true }))
        setDraft((d) => ({ ...d, [k]: '' }))
      }
      setEditing(null)
      toast.success(a.saved)
    } catch (e) {
      toast.error(e instanceof Error ? e.message : a.saveFail)
    } finally {
      setSaving('')
    }
  }

  const test = async () => {
    if (testing) return
    setTesting(true)
    try {
      const values = stripePayloadFor(CH_KEYS, draft, loaded)
      const d = await apiFetch<{ ok: boolean; message: string }>('/stripe-config/channel/test', {
        method: 'POST',
        body: { values },
      })
      if (d?.ok) toast.success(d.message || a.testOk)
      else toast.error(d?.message || a.testFail)
    } catch (e) {
      toast.error(e instanceof Error ? e.message : a.testFail)
    } finally {
      setTesting(false)
    }
  }

  const enabled = draft['stripe.enabled'] === 'true'
  const configured = !!secretSet['stripe.apiKey'] || draft['stripe.apiKey'] !== ''

  const summaryRow = (label: string, value: ReactNode) => (
    <div className="flex gap-2 text-[13px] text-[var(--shell-content-text)]">
      <span className="w-32 shrink-0 text-[var(--shell-crumb-text)]">{label}</span>
      <span className="break-all">{value || '—'}</span>
    </div>
  )
  const footerBtns = (group: 'channel' | 'webhook', onSaved: () => void) => (
    <>
      <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={() => setEditing(null)}>
        {t.common.confirmDialog.cancel}
      </button>
      <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={saving === group} onClick={onSaved}>
        {saving === group ? a.saving : a.save}
      </button>
    </>
  )

  if (error) return (
    <div>
      <PageHead title={a.title} desc={a.desc} />
      <ErrorBanner message={error} />
      <div className="mt-3"><ToolbarButton onClick={load}>{a.retry}</ToolbarButton></div>
    </div>
  )

  return (
    <div>
      <PageHead title={a.title} desc={a.desc} />
      <div className="flex flex-col gap-4">
        {/* 卡片一:通道配置 */}
        <Card className="flex flex-col gap-2 p-4">
          <div className="mb-3 flex items-center justify-between">
            <span className="flex items-center gap-2 font-semibold text-[var(--shell-heading)]">
              {a.chTitle}
              {enabled
                ? (configured
                    ? <Badge variant="success">{a.enabledReady}</Badge>
                    : <Badge variant="warning">{a.pending}</Badge>)
                : <Badge>{a.disabled}</Badge>}
            </span>
            <ToolbarButton primary onClick={() => setEditing('channel')}>{a.edit}</ToolbarButton>
          </div>
          {summaryRow(a.apiKey, secretSet['stripe.apiKey'] ? a.secretSet : '')}
          {summaryRow(a.publishableKey, draft['stripe.publishableKey'])}
          {summaryRow(a.currency, draft['stripe.currency'])}
          <div className="mt-1 text-xs text-[var(--shell-crumb-text)]">ⓘ {a.envNote}</div>
        </Card>

        {/* 卡片二:回调配置(webhook) */}
        <Card className="flex flex-col gap-2 p-4">
          <div className="mb-3 flex items-center justify-between">
            <span className="font-semibold text-[var(--shell-heading)]">{a.whTitle}</span>
            <ToolbarButton primary onClick={() => setEditing('webhook')}>{a.edit}</ToolbarButton>
          </div>
          {summaryRow(a.webhookSecret, secretSet['stripe.webhookSecret'] ? a.secretSet : '')}
          <div className="mt-1 text-xs text-[var(--shell-crumb-text)]">ⓘ {a.webhookSecretHint}</div>
        </Card>
      </div>

      {editing === 'channel' && (
        <Drawer title={a.chTitle} onClose={() => setEditing(null)} footer={footerBtns('channel', () => save('channel', CH_KEYS))}>
          <div className="mb-4 flex items-center gap-2">
            <Switch checked={enabled} onCheckedChange={(v) => set('stripe.enabled', String(v))} aria-label={a.chTitle} />
            {enabled ? a.enabledReady : a.disabled}
          </div>
          <div className="grid grid-cols-2 gap-4">
            <FormField label={a.apiKey} hint={a.apiKeyHint}>
              <SecretInput
                value={draft['stripe.apiKey'] ?? ''}
                onChange={(v) => set('stripe.apiKey', v)}
                placeholder={a.secretSet}
                hasValue={!!secretSet['stripe.apiKey']}
              />
            </FormField>
            <FormField label={a.publishableKey} hint={a.publishableKeyHint}>
              <Input className="w-72" value={draft['stripe.publishableKey'] ?? ''} onChange={(e) => set('stripe.publishableKey', e.target.value)} />
            </FormField>
            <FormField label={a.currency} hint={a.currencyHint}>
              <Input className="w-72" value={draft['stripe.currency'] ?? ''} onChange={(e) => set('stripe.currency', e.target.value)} />
            </FormField>
            <FormField label={a.apiBaseUrl} hint={a.apiBaseUrlHint}>
              <Input className="w-72" value={draft['stripe.apiBaseUrl'] ?? ''} onChange={(e) => set('stripe.apiBaseUrl', e.target.value)} />
            </FormField>
          </div>
          <div className="mt-4 flex items-center gap-2 border-t border-[var(--shell-side-border)] pt-3">
            <ToolbarButton disabled={testing} onClick={test}>
              {testing ? a.testing : a.testBtn}
            </ToolbarButton>
            <span className="text-xs text-[var(--shell-crumb-text)]">{a.testHint}</span>
          </div>
        </Drawer>
      )}

      {editing === 'webhook' && (
        <Drawer title={a.whTitle} onClose={() => setEditing(null)} footer={footerBtns('webhook', () => save('webhook', WH_KEYS))}>
          <div className="grid grid-cols-2 gap-4">
            <FormField label={a.webhookSecret} hint={a.webhookSecretHint}>
              <SecretInput
                value={draft['stripe.webhookSecret'] ?? ''}
                onChange={(v) => set('stripe.webhookSecret', v)}
                placeholder={a.secretSet}
                hasValue={!!secretSet['stripe.webhookSecret']}
              />
            </FormField>
          </div>
        </Drawer>
      )}
    </div>
  )
}