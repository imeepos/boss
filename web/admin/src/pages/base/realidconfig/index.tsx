// 实名核验配置页:自动核验通道(阿里云实人认证·身份二要素)卡片摘要 + 抽屉式编辑 + 试核自检。
// 契约:GET /realid-config(掩码)、PUT /realid-config/channel、POST /realid-config/channel/test。
// 未启用/未配置时实名提交保持 PENDING 人工核验,人工审核入口(客户档案)不受影响。
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
import { CH_KEYS, draftFrom, payloadFor, idPairError, type RealIDFields } from './logic'

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

function InfoIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden className="shrink-0 text-[var(--shell-crumb-text)]">
      <circle cx="12" cy="12" r="9" />
      <path d="M12 8h.01M11 12h1v5h1" />
    </svg>
  )
}

export default function RealIDConfigPage() {
  const t = useT()
  const a = t.pages.realidconfig
  const [draft, setDraft] = useState<Draft>({})
  const [loaded, setLoaded] = useState<Draft>({})
  const [secretSet, setSecretSet] = useState(false)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [saveState, setSaveState] = useState<SubmitState>('idle')
  const [saveError, setSaveError] = useState('')
  const [testing, setTesting] = useState(false)
  const [testState, setTestState] = useState<SubmitState>('idle')
  const [testError, setTestError] = useState('')
  const [testName, setTestName] = useState('')
  const [testIdNo, setTestIdNo] = useState('')
  const [editing, setEditing] = useState(false)

  const set = (key: string, v: string) => setDraft((d) => ({ ...d, [key]: v }))

  const load = () => {
    setError('')
    apiFetch<{ fields: RealIDFields }>('/realid-config')
      .then((d) => {
        const fields = d?.fields ?? {}
        const init = draftFrom(fields)
        setDraft(init)
        setLoaded(init)
        setSecretSet(!!fields['realid.accessKeySecret']?.hasValue)
      })
      .catch((e) => setError(e instanceof Error ? e.message : a.loadFail))
  }

  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const save = async () => {
    if (saving) return
    const values = payloadFor(CH_KEYS, draft, loaded)
    setSaving(true)
    setSaveError('')
    setSaveState('loading')
    try {
      await apiFetch('/realid-config/channel', { method: 'PUT', body: { values } })
      setLoaded((l) => ({ ...l, ...values }))
      if (values['realid.accessKeySecret']) setSecretSet(true)
      setDraft((d) => ({ ...d, 'realid.accessKeySecret': '' }))
      setEditing(false)
      toast.success(a.saved)
      setSaveState('success')
      setTimeout(() => setSaveState((s) => (s === 'success' ? 'idle' : s)), 1500)
    } catch (e) {
      const msg = e instanceof Error ? e.message : a.saveFail
      setSaveError(msg)
      toast.error(msg)
      setSaveState('failed')
      setTimeout(() => setSaveState((s) => (s === 'failed' ? 'idle' : s)), 2500)
    } finally {
      setSaving(false)
    }
  }

  const test = async () => {
    if (testing) return
    const pairErr = idPairError(testName, testIdNo)
    if (pairErr) {
      const msg = pairErr === 'bad-name' ? a.nameInvalid : a.idNoInvalid
      setTestError(msg)
      toast.error(msg)
      return
    }
    setTesting(true)
    setTestError('')
    setTestState('loading')
    try {
      const values = payloadFor(CH_KEYS, draft, loaded)
      const d = await apiFetch<{ ok: boolean; message: string }>('/realid-config/channel/test', {
        method: 'POST',
        body: { values, name: testName || undefined, idNo: testIdNo || undefined },
      })
      if (d?.ok) {
        toast.success(d.message || a.testOk)
        setTestState('success')
        setTestError('')
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

  const enabled = draft['realid.enabled'] === 'true'
  const configured = secretSet || draft['realid.accessKeyId'] !== ''

  const summaryRow = (label: string, value: ReactNode) => (
    <div className="flex gap-2 text-[13px] text-[var(--shell-content-text)]">
      <span className="w-32 shrink-0 text-[var(--shell-crumb-text)]">{label}</span>
      <span className="break-all">{value || '—'}</span>
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
            <ToolbarButton primary onClick={() => setEditing(true)}>{a.edit}</ToolbarButton>
          </div>
          {summaryRow(a.provider, draft['realid.provider'])}
          {summaryRow(a.endpoint, draft['realid.endpoint'])}
          {summaryRow(a.accessKeyId, draft['realid.accessKeyId'])}
          {summaryRow(a.accessKeySecret, secretSet ? a.secretSet : '')}
          <div className="mt-1 flex items-center gap-1 text-xs text-[var(--shell-crumb-text)]"><InfoIcon /><span>{a.envNote}</span></div>
        </Card>
      </div>

      {editing && (
        <Drawer title={a.chTitle} onClose={() => setEditing(false)}
          footer={
            <>
              <ToolbarButton onClick={() => setEditing(false)} disabled={saving}>{t.common.confirmDialog.cancel}</ToolbarButton>
              <SubmitButton
                state={saveState}
                labels={{ idle: a.save, loading: a.saving, success: a.saved, failed: a.saveFail }}
                disabled={saving}
                onClick={save}
              />
            </>
          }>
          {saveError && <div className="mb-3"><ErrorBanner message={saveError} /></div>}
          <div className="mb-4 flex items-center gap-2">
            <Switch checked={enabled} onCheckedChange={(v) => set('realid.enabled', String(v))} aria-label={a.chTitle} />
            {enabled ? a.enabledReady : a.disabled}
          </div>
          <div className="grid grid-cols-2 gap-4">
            <FormField label={a.provider} hint={a.providerHint}>
              <Input className="w-72" value={draft['realid.provider'] ?? ''} readOnly />
            </FormField>
            <FormField label={a.endpoint} hint={a.endpointHint}>
              <Input
                className="w-72"
                placeholder={a.endpointPh}
                value={draft['realid.endpoint'] ?? ''}
                onChange={(e) => set('realid.endpoint', e.target.value)}
              />
            </FormField>
            <FormField label={a.accessKeyId}>
              <Input className="w-72" value={draft['realid.accessKeyId'] ?? ''} onChange={(e) => set('realid.accessKeyId', e.target.value)} />
            </FormField>
            <FormField label={a.accessKeySecret}>
              <SecretInput
                value={draft['realid.accessKeySecret'] ?? ''}
                onChange={(v) => set('realid.accessKeySecret', v)}
                placeholder={a.secretSet}
                hasValue={secretSet}
              />
            </FormField>
          </div>
          <div className="mt-4 flex items-center gap-2 border-t border-[var(--shell-side-border)] pt-3">
            <Input
              className="w-36"
              placeholder={a.testNamePh}
              value={testName}
              onChange={(e) => setTestName(e.target.value)}
            />
            <Input
              className="w-52"
              placeholder={a.testIdNoPh}
              value={testIdNo}
              onChange={(e) => setTestIdNo(e.target.value)}
            />
            <SubmitButton
              state={testState}
              labels={{ idle: a.testBtn, loading: a.testing, success: a.testOk, failed: a.testFail }}
              disabled={testing}
              onClick={test}
            />
          </div>
          {testError && <div className="mt-3"><ErrorBanner message={testError} /></div>}
        </Drawer>
      )}
    </div>
  )
}
