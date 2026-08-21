// 推送配置页:JPush 聚合通道卡片 + 自检(完整性 / 真实试发)。
// 契约:GET /push-config(掩码)、PUT /push-config/channel、POST /push-config/channel/test。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { FormField } from '../../../components/business/form-field'
import { Card } from '../../../components/ui/card'
import { Input } from '../../../components/ui/input'
import { Badge } from '../../../components/ui/badge'
import { Switch } from '../../../components/ui/switch'
import { Dropdown } from '../../../components/Dropdown'
import { CH_KEYS, draftFrom, pushPayloadFor, targetError, type PushFields } from './logic'

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

export default function PushConfigPage() {
  const t = useT()
  const a = t.pages.pushconfig
  const [draft, setDraft] = useState<Draft>({})
  const [loaded, setLoaded] = useState<Draft>({})
  const [secretSet, setSecretSet] = useState<Record<string, boolean>>({})
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [testTarget, setTestTarget] = useState('')
  const [targetKind, setTargetKind] = useState<'registration_id' | 'alias'>('registration_id')

  const set = (key: string, v: string) => setDraft((d) => ({ ...d, [key]: v }))

  const load = () => {
    setError('')
    apiFetch<{ fields: PushFields }>('/push-config')
      .then((d) => {
        const fields = d?.fields ?? {}
        const init = draftFrom(fields)
        setDraft(init)
        setLoaded(init)
        setSecretSet({ 'push.jpush.masterSecret': !!fields['push.jpush.masterSecret']?.hasValue })
      })
      .catch((e) => setError(e instanceof Error ? e.message : a.loadFail))
  }

  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const save = async () => {
    if (saving) return
    const values = pushPayloadFor(draft, loaded)
    setSaving(true)
    try {
      await apiFetch('/push-config/channel', { method: 'PUT', body: { values } })
      setLoaded((l) => ({ ...l, ...values }))
      if (values['push.jpush.masterSecret']) {
        setSecretSet((s) => ({ ...s, 'push.jpush.masterSecret': true }))
      }
      setDraft((d) => ({ ...d, 'push.jpush.masterSecret': '' }))
      toast.success(a.saved)
    } catch (e) {
      toast.error(e instanceof Error ? e.message : a.saveFail)
    } finally {
      setSaving(false)
    }
  }

  const test = async () => {
    if (testing) return
    if (testTarget && targetError(testTarget, targetKind)) {
      toast.error(a.targetInvalid)
      return
    }
    setTesting(true)
    try {
      const values = pushPayloadFor(draft, loaded)
      const d = await apiFetch<{ ok: boolean; message: string }>('/push-config/channel/test', {
        method: 'POST',
        body: {
          values,
          target: testTarget || undefined,
          targetKind: testTarget ? targetKind : undefined,
        },
      })
      if (d?.ok) toast.success(d.message || a.testOk)
      else toast.error(d?.message || a.testFail)
    } catch (e) {
      toast.error(e instanceof Error ? e.message : a.testFail)
    } finally {
      setTesting(false)
    }
  }

  const enabled = draft['push.enabled'] === 'true'
  const configured =
    !!secretSet['push.jpush.masterSecret'] || draft['push.jpush.appKey'] !== ''

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
      <Card className="p-4">
        <div className="mb-3 flex items-center justify-between">
          <span className="font-semibold text-[var(--shell-heading)]">{a.chTitle}</span>
          <span className="flex items-center gap-2">
            {enabled
              ? (configured
                  ? <Badge variant="success">{a.enabledReady}</Badge>
                  : <Badge variant="warning">{a.pending}</Badge>)
              : <Badge>{a.disabled}</Badge>}
            <Switch
              checked={enabled}
              onCheckedChange={(v) => set('push.enabled', String(v))}
              aria-label={a.chTitle}
            />
          </span>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <FormField label={a.provider} hint={a.providerHint}>
            <Input className="w-72" value={draft['push.provider'] ?? ''} readOnly />
          </FormField>
          <FormField label={a.appKey} hint={a.appKeyHint}>
            <Input className="w-72" value={draft['push.jpush.appKey'] ?? ''} onChange={(e) => set('push.jpush.appKey', e.target.value)} />
          </FormField>
          <FormField label={a.masterSecret}>
            <SecretInput
              value={draft['push.jpush.masterSecret'] ?? ''}
              onChange={(v) => set('push.jpush.masterSecret', v)}
              placeholder={a.secretSet}
              hasValue={!!secretSet['push.jpush.masterSecret']}
            />
          </FormField>
          <FormField label={a.apiUrl} hint={a.apiUrlHint}>
            <Input className="w-72" value={draft['push.jpush.apiUrl'] ?? ''} onChange={(e) => set('push.jpush.apiUrl', e.target.value)} />
          </FormField>
          <FormField label={a.apnsProduction} hint={a.apnsProductionHint}>
            <span className="flex w-72 items-center gap-2">
              <Switch
                checked={draft['push.jpush.apnsProduction'] !== 'false'}
                onCheckedChange={(v) => set('push.jpush.apnsProduction', String(v))}
                aria-label={a.apnsProduction}
              />
              <span className="text-xs text-[var(--shell-crumb-text)]">
                {draft['push.jpush.apnsProduction'] !== 'false' ? a.prod : a.dev}
              </span>
            </span>
          </FormField>
          <FormField label={a.liveTime} hint={a.liveTimeHint}>
            <Input className="w-72" value={draft['push.jpush.liveTime'] ?? ''} onChange={(e) => set('push.jpush.liveTime', e.target.value)} />
          </FormField>
        </div>
        <div className="mt-3.5 flex items-center justify-end gap-2">
          <span className="mr-auto flex items-center gap-2">
            <Dropdown
              value={targetKind}
              options={[
                { value: 'registration_id', label: a.targetRegId },
                { value: 'alias', label: a.targetAlias },
              ]}
              onChange={(v) => setTargetKind(v as 'registration_id' | 'alias')}
              ariaLabel={a.targetKindLabel}
              triggerStyle={{ width: 132 }}
            />
            <Input
              className="w-52"
              placeholder={a.testTargetPh}
              value={testTarget}
              onChange={(e) => setTestTarget(e.target.value)}
            />
            <ToolbarButton disabled={testing} onClick={test}>
              {testing ? a.testing : a.testBtn}
            </ToolbarButton>
          </span>
          <ToolbarButton primary disabled={saving} onClick={save}>
            {saving ? a.saving : a.save}
          </ToolbarButton>
        </div>
        <div className="mt-2 text-xs text-[var(--shell-crumb-text)]">ⓘ {a.envNote}</div>
      </Card>
    </div>
  )
}
