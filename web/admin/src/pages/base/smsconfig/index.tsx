// 短信配置页:通道配置/文案模板两卡片摘要 + 抽屉式编辑 + 试发自检。
// 契约:GET /sms-config(掩码)、PUT /sms-config/{channel|template}、POST /sms-config/channel/test。
import { useEffect, useState, type ReactNode } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { PageHead, ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { SubmitButton, type SubmitState } from '../../../components/business/submit-button'
import { FormField } from '../../../components/business/form-field'
import { Card } from '../../../components/ui/card'
import { Input } from '../../../components/ui/input'
import { Badge } from '../../../components/ui/badge'
import { Switch } from '../../../components/ui/switch'
import { CH_KEYS, TP_KEYS, draftFrom, smsPayloadFor, phoneError, type SmsFields } from './logic'

type Draft = Record<string, string>
type Group = 'channel' | 'template'

function InfoIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden className="shrink-0 text-[var(--shell-crumb-text)]">
      <circle cx="12" cy="12" r="9" />
      <path d="M12 8h.01M11 12h1v5h1" />
    </svg>
  )
}

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
  const [saveState, setSaveState] = useState<Record<Group, SubmitState>>({ channel: 'idle', template: 'idle' })
  const [saveError, setSaveError] = useState<Record<Group, string>>({ channel: '', template: '' })
  const [testState, setTestState] = useState<SubmitState>('idle')
  const [testError, setTestError] = useState('')
  const [testing, setTesting] = useState(false)
  const [testPhone, setTestPhone] = useState('')
  const [editing, setEditing] = useState<Group | null>(null)

  const set = (key: string, v: string) => setDraft((d) => ({ ...d, [key]: v }))
  const setGroupState = (g: Group, s: SubmitState) => setSaveState((p) => ({ ...p, [g]: s }))
  const setGroupError = (g: Group, msg: string) => setSaveError((p) => ({ ...p, [g]: msg }))

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

  const save = async (group: Group, keys: readonly string[]) => {
    if (saveState[group] === 'loading') return
    const values = smsPayloadFor(keys, draft, loaded)
    setGroupState(group, 'loading')
    setGroupError(group, '')
    try {
      await apiFetch(`/sms-config/${group}`, { method: 'PUT', body: { values } })
      setLoaded((l) => ({ ...l, ...values }))
      if (values['sms.accessKeySecret']) {
        setSecretSet((s) => ({ ...s, 'sms.accessKeySecret': true }))
      }
      setDraft((d) => ({ ...d, 'sms.accessKeySecret': '' }))
      setEditing(null)
      toast.success(a.saved)
      setGroupState(group, 'success')
      setTimeout(() => setGroupState(group, 'idle'), 1500)
    } catch (e) {
      const msg = e instanceof Error ? e.message : a.saveFail
      setGroupError(group, msg)
      toast.error(msg)
      setGroupState(group, 'failed')
      setTimeout(() => setGroupState(group, 'idle'), 2500)
    }
  }

  const test = async () => {
    if (testing) return
    if (testPhone && phoneError(testPhone)) {
      setTestError(a.phoneInvalid)
      toast.error(a.phoneInvalid)
      return
    }
    setTesting(true)
    setTestError('')
    setTestState('loading')
    try {
      const values = smsPayloadFor(CH_KEYS, draft, loaded)
      const d = await apiFetch<{ ok: boolean; message: string }>('/sms-config/channel/test', {
        method: 'POST',
        body: { values, phone: testPhone || undefined },
      })
      if (d?.ok) {
        toast.success(d.message || a.testOk)
        setTestState('success')
        setTimeout(() => setTestState((s) => (s === 'success' ? 'idle' : s)), 1500)
      } else {
        const msg = d?.message || a.testFail
        setTestError(msg)
        toast.error(msg)
        setTestState('failed')
        setTimeout(() => setTestState((s) => (s === 'failed' ? 'idle' : s)), 2500)
      }
    } catch (e) {
      const msg = e instanceof Error ? e.message : a.testFail
      setTestError(msg)
      toast.error(msg)
      setTestState('failed')
      setTimeout(() => setTestState((s) => (s === 'failed' ? 'idle' : s)), 2500)
    } finally {
      setTesting(false)
    }
  }

  const enabled = draft['sms.enabled'] === 'true'
  const configured = !!secretSet['sms.accessKeySecret'] || draft['sms.accessKeyId'] !== ''

  const summaryRow = (label: string, value: ReactNode) => (
    <div className="flex gap-2 text-[13px] text-[var(--shell-content-text)]">
      <span className="w-32 shrink-0 text-[var(--shell-crumb-text)]">{label}</span>
      <span className="break-all">{value || '—'}</span>
    </div>
  )
  const footerBtns = (group: Group, onSaved: () => void) => (
    <>
      <ToolbarButton onClick={() => setEditing(null)} disabled={saveState[group] === 'loading'}>
        {t.common.confirmDialog.cancel}
      </ToolbarButton>
      <SubmitButton
        state={saveState[group]}
        labels={{ idle: a.save, loading: a.saving, success: a.saved, failed: a.saveFail }}
        disabled={saveState[group] === 'loading'}
        onClick={onSaved}
      />
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
          {summaryRow(a.provider, draft['sms.provider'])}
          {summaryRow(a.accessKeyId, draft['sms.accessKeyId'])}
          {summaryRow(a.accessKeySecret, secretSet['sms.accessKeySecret'] ? a.secretSet : '')}
          <div className="mt-1 flex items-center gap-1 text-xs text-[var(--shell-crumb-text)]"><InfoIcon /><span>{a.envNote}</span></div>
        </Card>

        {/* 卡片二:报备模板 ContentCode */}
        <Card className="flex flex-col gap-2 p-4">
          <div className="mb-3 flex items-center justify-between">
            <span className="font-semibold text-[var(--shell-heading)]">{a.tpTitle}</span>
            <ToolbarButton primary onClick={() => setEditing('template')}>{a.edit}</ToolbarButton>
          </div>
          {summaryRow(a.tpCn, draft['sms.contentCode.cn'])}
          {summaryRow(a.tpMy, draft['sms.contentCode.my'])}
        </Card>
      </div>

      {editing === 'channel' && (
        <Drawer title={a.chTitle} onClose={() => setEditing(null)} footer={footerBtns('channel', () => save('channel', CH_KEYS))}>
          {saveError.channel && <div className="mb-3"><ErrorBanner message={saveError.channel} /></div>}
          <div className="mb-4 flex items-center gap-2">
            <Switch checked={enabled} onCheckedChange={(v) => set('sms.enabled', String(v))} aria-label={a.chTitle} />
            {enabled ? a.enabledReady : a.disabled}
          </div>
          <div className="grid grid-cols-2 gap-4">
            <FormField label={a.provider} hint={a.providerHint}>
              <Input className="w-72" value={draft['sms.provider'] ?? ''} readOnly />
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
          {testError && <div className="mt-3"><ErrorBanner message={testError} /></div>}
          <div className="mt-4 flex items-center gap-2 border-t border-[var(--shell-side-border)] pt-3">
            <Input
              className="w-52"
              placeholder={a.testPhonePh}
              value={testPhone}
              onChange={(e) => setTestPhone(e.target.value)}
            />
            <SubmitButton
              state={testState}
              labels={{ idle: a.testBtn, loading: a.testing, success: a.testOk, failed: a.testFail }}
              disabled={testing}
              onClick={test}
            />
          </div>
        </Drawer>
      )}

      {editing === 'template' && (
        <Drawer title={a.tpTitle} onClose={() => setEditing(null)} footer={footerBtns('template', () => save('template', TP_KEYS))}>
          {saveError.template && <div className="mb-3"><ErrorBanner message={saveError.template} /></div>}
          <div className="grid grid-cols-2 gap-4">
            <FormField label={a.tpCn} hint={a.tpHint}>
              <Input className="w-72" value={draft['sms.contentCode.cn'] ?? ''} onChange={(e) => set('sms.contentCode.cn', e.target.value)} />
            </FormField>
            <FormField label={a.tpMy} hint={a.tpHint}>
              <Input className="w-72" value={draft['sms.contentCode.my'] ?? ''} onChange={(e) => set('sms.contentCode.my', e.target.value)} />
            </FormField>
          </div>
        </Drawer>
      )}
    </div>
  )
}
