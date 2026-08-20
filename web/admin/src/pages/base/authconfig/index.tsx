// 认证配置页(auth-config-v1.spec.md):中国区一键登录 / 海外号码认证 / 降级与合规三卡片。
// 契约:GET /auth-config(掩码)、PUT /auth-config/{cn|my|fallback}、POST /auth-config/{group}/test。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Dropdown } from '../../../components/Dropdown'
import { PageHead, ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { FormField } from '../../../components/business/form-field'
import { Card } from '../../../components/ui/card'
import { Input } from '../../../components/ui/input'
import { Badge } from '../../../components/ui/badge'
import { Switch } from '../../../components/ui/switch'
import { CN_KEYS, MY_KEYS, FB_KEYS, initDraft, payloadFor, timeoutError, type AuthFields } from './logic'

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

export default function AuthConfigPage() {
  const t = useT()
  const a = t.pages.authconfig
  const [draft, setDraft] = useState<Draft>({})
  const [loaded, setLoaded] = useState<Draft>({})
  const [secretSet, setSecretSet] = useState<Record<string, boolean>>({})
  const [error, setError] = useState('')
  const [saving, setSaving] = useState('')
  const [testing, setTesting] = useState('')

  const set = (key: string, v: string) => setDraft((d) => ({ ...d, [key]: v }))

  const load = () => {
    setError('')
    apiFetch<{ fields: AuthFields }>('/auth-config')
      .then((d) => {
        const fields = d?.fields ?? {}
        const init = initDraft(fields)
        setDraft(init)
        setLoaded(init)
        setSecretSet({
          'auth.cn.appSecret': !!fields['auth.cn.appSecret']?.hasValue,
          'auth.my.apiKey': !!fields['auth.my.apiKey']?.hasValue,
        })
      })
      .catch((e) => setError(e instanceof Error ? e.message : a.loadFail))
  }

  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const save = async (group: 'cn' | 'my' | 'fallback', keys: readonly string[]) => {
    if (saving) return
    if (group === 'cn' && timeoutError(draft['auth.cn.preloadTimeoutMs'] ?? '')) {
      toast.error(a.timeoutInvalid)
      return
    }
    const values = payloadFor(keys, draft, loaded)
    setSaving(group)
    try {
      await apiFetch(`/auth-config/${group}`, { method: 'PUT', body: { values } })
      setLoaded((l) => ({ ...l, ...values }))
      for (const k of ['auth.cn.appSecret', 'auth.my.apiKey']) {
        if (values[k]) setSecretSet((s) => ({ ...s, [k]: true }))
      }
      setDraft((d) => ({ ...d, 'auth.cn.appSecret': '', 'auth.my.apiKey': '' }))
      toast.success(a.saved)
    } catch (e) {
      toast.error(e instanceof Error ? e.message : a.saveFail)
    } finally {
      setSaving('')
    }
  }

  const test = async (group: 'cn' | 'my' | 'fallback', keys: readonly string[]) => {
    if (testing) return
    setTesting(group)
    try {
      const values = payloadFor(keys, draft, loaded)
      const d = await apiFetch<{ ok: boolean; message: string }>(`/auth-config/${group}/test`, {
        method: 'POST', body: { values },
      })
      if (d?.ok) toast.success(d.message || a.testOk)
      else toast.error(d?.message || a.testFail)
    } catch (e) {
      toast.error(e instanceof Error ? e.message : a.testFail)
    } finally {
      setTesting('')
    }
  }

  const cnOn = draft['auth.cn.enabled'] === 'true'
  const myOn = draft['auth.my.enabled'] === 'true'
  const cardActions = (group: 'cn' | 'my' | 'fallback', keys: readonly string[]) => (
    <div className="mt-3.5 flex justify-end gap-2">
      <ToolbarButton disabled={testing === group} onClick={() => test(group, keys)}>
        {testing === group ? a.testing : a.testBtn}
      </ToolbarButton>
      <ToolbarButton primary disabled={saving === group} onClick={() => save(group, keys)}>
        {saving === group ? a.saving : a.save}
      </ToolbarButton>
    </div>
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
        {/* 卡片一:中国区一键登录 */}
        <Card className="p-4">
          <div className="mb-3 flex items-center justify-between">
            <span className="font-semibold text-[var(--shell-heading)]">{a.cnTitle}</span>
            <span className="flex items-center gap-2">
              {cnOn
                ? <Badge variant="success">{a.enabled}</Badge>
                : <Badge>{a.disabled}</Badge>}
              <Switch
                checked={cnOn}
                onCheckedChange={(v) => set('auth.cn.enabled', String(v))}
                aria-label={a.cnTitle}
              />
            </span>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <FormField label={a.cnAppKey}>
              <Input className="w-72" value={draft['auth.cn.appKey'] ?? ''} onChange={(e) => set('auth.cn.appKey', e.target.value)} />
            </FormField>
            <FormField label={a.cnAppSecret}>
              <SecretInput
                value={draft['auth.cn.appSecret'] ?? ''}
                onChange={(v) => set('auth.cn.appSecret', v)}
                placeholder={a.secretSet}
                hasValue={!!secretSet['auth.cn.appSecret']}
              />
            </FormField>
            <FormField label={a.cnPackage}>
              <Input className="w-72" value={draft['auth.cn.packageName'] ?? ''} onChange={(e) => set('auth.cn.packageName', e.target.value)} />
            </FormField>
            <FormField label={a.cnTimeout} hint={a.cnTimeoutHint}>
              <Input className="w-72" inputMode="numeric" value={draft['auth.cn.preloadTimeoutMs'] ?? ''} onChange={(e) => set('auth.cn.preloadTimeoutMs', e.target.value)} />
            </FormField>
          </div>
          {cardActions('cn', CN_KEYS)}
        </Card>

        {/* 卡片二:海外号码认证 */}
        <Card className="p-4">
          <div className="mb-3 flex items-center justify-between">
            <span className="font-semibold text-[var(--shell-heading)]">{a.myTitle}</span>
            <span className="flex items-center gap-2">
              {myOn
                ? <Badge variant="success">{a.enabled}</Badge>
                : <Badge>{a.pending}</Badge>}
              <Switch
                checked={myOn}
                onCheckedChange={(v) => set('auth.my.enabled', String(v))}
                aria-label={a.myTitle}
              />
            </span>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <FormField label={a.myProvider}>
              <Dropdown
                value={draft['auth.my.provider'] ?? 'none'}
                options={[
                  { value: 'opengateway', label: a.providerOg },
                  { value: 'none', label: a.providerNone },
                ]}
                onChange={(v) => set('auth.my.provider', v)}
                ariaLabel={a.myProvider}
              />
            </FormField>
            <FormField label={a.mySmsProvider}>
              <Dropdown
                value={draft['auth.my.smsProvider'] ?? 'engagelab'}
                options={[
                  { value: 'engagelab', label: 'EngageLab' },
                  { value: 'twilio', label: 'Twilio' },
                  { value: 'vonage', label: 'Vonage' },
                ]}
                onChange={(v) => set('auth.my.smsProvider', v)}
                ariaLabel={a.mySmsProvider}
              />
            </FormField>
            <FormField label={a.myApiKey}>
              <SecretInput
                value={draft['auth.my.apiKey'] ?? ''}
                onChange={(v) => set('auth.my.apiKey', v)}
                placeholder={a.secretSet}
                hasValue={!!secretSet['auth.my.apiKey']}
              />
            </FormField>
            <FormField label={a.myCountryCode}>
              <Input className="w-72" value={draft['auth.my.countryCode'] ?? ''} onChange={(e) => set('auth.my.countryCode', e.target.value)} />
            </FormField>
            <FormField label={a.mySmsSign}>
              <Input className="w-72" value={draft['auth.my.smsSign'] ?? ''} onChange={(e) => set('auth.my.smsSign', e.target.value)} />
            </FormField>
          </div>
          {cardActions('my', MY_KEYS)}
        </Card>

        {/* 卡片三:降级与合规 */}
        <Card className="p-4">
          <div className="mb-3 font-semibold text-[var(--shell-heading)]">{a.fbTitle}</div>
          <div className="mb-4 flex items-center gap-8">
            {([
              ['auth.fallback.smsOnFail', a.fbSmsOnFail],
              ['auth.fallback.billingAlert', a.fbBillingAlert],
              ['auth.fallback.autoRegister', a.fbAutoRegister],
            ] as const).map(([key, label]) => (
              <label key={key} className="flex cursor-pointer items-center gap-2 text-[13px] text-[var(--shell-content-text)]">
                <Switch
                  checked={draft[key] === 'true'}
                  onCheckedChange={(v) => set(key, String(v))}
                  aria-label={label}
                />
                {label}
              </label>
            ))}
          </div>
          <div className="grid grid-cols-2 gap-4">
            <FormField label={a.fbPrivacyVersion}>
              <Input className="w-72" value={draft['auth.compliance.privacyVersion'] ?? ''} onChange={(e) => set('auth.compliance.privacyVersion', e.target.value)} />
            </FormField>
            <FormField label={a.fbAgreementUrl}>
              <Input className="w-72" value={draft['auth.compliance.agreementUrl'] ?? ''} onChange={(e) => set('auth.compliance.agreementUrl', e.target.value)} />
            </FormField>
          </div>
          {cardActions('fallback', FB_KEYS)}
          <div className="mt-3 text-xs text-[var(--shell-crumb-text)]">ⓘ {a.complianceNote}</div>
        </Card>
      </div>
    </div>
  )
}
