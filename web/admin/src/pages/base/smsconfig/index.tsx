// 短信配置页:通道配置(阿里云国际短信)/文案模板两卡片 + 试发自检。
// 契约:GET /sms-config(掩码)、PUT /sms-config/{channel|template}、POST /sms-config/channel/test。
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
import { CH_KEYS, TP_KEYS, draftFrom, smsPayloadFor, phoneError, type SmsFields } from './logic'

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

export default function SmsConfigPage() {
  const t = useT()
  const a = t.pages.smsconfig
  const [draft, setDraft] = useState<Draft>({})
  const [loaded, setLoaded] = useState<Draft>({})
  const [secretSet, setSecretSet] = useState<Record<string, boolean>>({})
  const [error, setError] = useState('')
  const [saving, setSaving] = useState('')
  const [testing, setTesting] = useState(false)
  const [testPhone, setTestPhone] = useState('')

  const set = (key: string, v: string) => setDraft((d) => ({ ...d, [key]: v }))

  const load = () => {
    setError('')
    apiFetch<{ fields: SmsFields }>('/sms-config')
      .then((d) => {
        const fields = d?.fields ?? {}
        const init = draftFrom(fields)
        setDraft(init)
        setLoaded(init)
        setSecretSet({ 'sms.accessKeySecret': !!fields['sms.accessKeySecret']?.hasValue })
      })
      .catch((e) => setError(e instanceof Error ? e.message : a.loadFail))
  }

  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const save = async (group: 'channel' | 'template', keys: readonly string[]) => {
    if (saving) return
    const values = smsPayloadFor(keys, draft, loaded)
    setSaving(group)
    try {
      await apiFetch(`/sms-config/${group}`, { method: 'PUT', body: { values } })
      setLoaded((l) => ({ ...l, ...values }))
      if (values['sms.accessKeySecret']) {
        setSecretSet((s) => ({ ...s, 'sms.accessKeySecret': true }))
      }
      setDraft((d) => ({ ...d, 'sms.accessKeySecret': '' }))
      toast.success(a.saved)
    } catch (e) {
      toast.error(e instanceof Error ? e.message : a.saveFail)
    } finally {
      setSaving('')
    }
  }

  const test = async () => {
    if (testing) return
    if (testPhone && phoneError(testPhone)) {
      toast.error(a.phoneInvalid)
      return
    }
    setTesting(true)
    try {
      const values = smsPayloadFor(CH_KEYS, draft, loaded)
      const d = await apiFetch<{ ok: boolean; message: string }>('/sms-config/channel/test', {
        method: 'POST',
        body: { values, phone: testPhone || undefined },
      })
      if (d?.ok) toast.success(d.message || a.testOk)
      else toast.error(d?.message || a.testFail)
    } catch (e) {
      toast.error(e instanceof Error ? e.message : a.testFail)
    } finally {
      setTesting(false)
    }
  }

  const enabled = draft['sms.enabled'] === 'true'
  const configured = !!secretSet['sms.accessKeySecret'] || draft['sms.accessKeyId'] !== ''

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
                onCheckedChange={(v) => set('sms.enabled', String(v))}
                aria-label={a.chTitle}
              />
            </span>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <FormField label={a.provider} hint={a.providerHint}>
              <Input className="w-72" value={draft['sms.provider'] ?? ''} readOnly />
            </FormField>
            <FormField label={a.senderId} hint={a.senderIdHint}>
              <Input className="w-72" value={draft['sms.from'] ?? ''} onChange={(e) => set('sms.from', e.target.value)} />
            </FormField>
            <FormField label={a.accessKeyId}>
              <Input className="w-72" value={draft['sms.accessKeyId'] ?? ''} onChange={(e) => set('sms.accessKeyId', e.target.value)} />
            </FormField>
            <FormField label={a.accessKeySecret}>
              <SecretInput
                value={draft['sms.accessKeySecret'] ?? ''}
                onChange={(v) => set('sms.accessKeySecret', v)}
                placeholder={a.secretSet}
                hasValue={!!secretSet['sms.accessKeySecret']}
              />
            </FormField>
          </div>
          <div className="mt-3.5 flex items-center justify-end gap-2">
            <span className="mr-auto flex items-center gap-2">
              <Input
                className="w-52"
                placeholder={a.testPhonePh}
                value={testPhone}
                onChange={(e) => setTestPhone(e.target.value)}
              />
              <ToolbarButton disabled={testing} onClick={test}>
                {testing ? a.testing : a.testBtn}
              </ToolbarButton>
            </span>
            <ToolbarButton primary disabled={saving === 'channel'} onClick={() => save('channel', CH_KEYS)}>
              {saving === 'channel' ? a.saving : a.save}
            </ToolbarButton>
          </div>
          <div className="mt-2 text-xs text-[var(--shell-crumb-text)]">ⓘ {a.envNote}</div>
        </Card>

        {/* 卡片二:文案模板 */}
        <Card className="p-4">
          <div className="mb-3 font-semibold text-[var(--shell-heading)]">{a.tpTitle}</div>
          <div className="grid grid-cols-2 gap-4">
            <FormField label={a.tpCn} hint={a.tpHint}>
              <Input className="w-72" value={draft['sms.template.cn'] ?? ''} onChange={(e) => set('sms.template.cn', e.target.value)} />
            </FormField>
            <FormField label={a.tpMy} hint={a.tpHint}>
              <Input className="w-72" value={draft['sms.template.my'] ?? ''} onChange={(e) => set('sms.template.my', e.target.value)} />
            </FormField>
          </div>
          <div className="mt-3.5 flex justify-end">
            <ToolbarButton primary disabled={saving === 'template'} onClick={() => save('template', TP_KEYS)}>
              {saving === 'template' ? a.saving : a.save}
            </ToolbarButton>
          </div>
        </Card>
      </div>
    </div>
  )
}
