// 认证配置页(auth-config-v1.spec.md):三卡片摘要 + 抽屉式编辑。
// 中国区一键登录 / 海外号码认证 / 降级与合规;契约:GET /auth-config(掩码)、
// PUT /auth-config/{cn|my|fallback}、POST /auth-config/{group}/test。
import { useEffect, useState, type ReactNode } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { Card } from '../../../components/ui/card'
import { Badge } from '../../../components/ui/badge'
import { AuthConfigDrawers } from './Drawers'
import { CN_KEYS, MY_KEYS, FB_KEYS, initDraft, payloadFor, timeoutError, type AuthFields } from './logic'

type Draft = Record<string, string>
type Group = 'cn' | 'my' | 'fallback'

interface GroupState { state: 'idle' | 'loading' | 'success' | 'failed'; error: string }

function InfoIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden className="shrink-0 text-[var(--shell-crumb-text)]">
      <circle cx="12" cy="12" r="9" />
      <path d="M12 8h.01M11 12h1v5h1" />
    </svg>
  )
}

export default function AuthConfigPage() {
  const t = useT()
  const a = t.pages.authconfig
  const [draft, setDraft] = useState<Draft>({})
  const [loaded, setLoaded] = useState<Draft>({})
  const [secretSet, setSecretSet] = useState<Record<string, boolean>>({})
  const [error, setError] = useState('')
  const [saveState, setSaveState] = useState<Record<Group, GroupState>>({
    cn: { state: 'idle', error: '' },
    my: { state: 'idle', error: '' },
    fallback: { state: 'idle', error: '' },
  })
  const [testState, setTestState] = useState<Record<Group, GroupState>>({
    cn: { state: 'idle', error: '' },
    my: { state: 'idle', error: '' },
    fallback: { state: 'idle', error: '' },
  })
  const [editing, setEditing] = useState<Group | null>(null)

  const set = (key: string, v: string) => setDraft((d) => ({ ...d, [key]: v }))
  const setSaveGroup = (g: Group, patch: Partial<GroupState>) =>
    setSaveState((s) => ({ ...s, [g]: { ...s[g], ...patch } }))
  const setTestGroup = (g: Group, patch: Partial<GroupState>) =>
    setTestState((s) => ({ ...s, [g]: { ...s[g], ...patch } }))

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

  const groupKeys = (g: Group) => (g === 'cn' ? CN_KEYS : g === 'my' ? MY_KEYS : FB_KEYS)

  const save = async (group: Group) => {
    const cur = saveState[group]
    if (cur.state === 'loading') return
    if (group === 'cn' && timeoutError(draft['auth.cn.preloadTimeoutMs'] ?? '')) {
      const msg = a.timeoutInvalid
      setSaveGroup(group, { state: 'failed', error: msg })
      toast.error(msg)
      setTimeout(() => setSaveGroup(group, { state: cur.state === 'failed' ? 'idle' : cur.state }), 2500)
      return
    }
    const keys = groupKeys(group)
    const values = payloadFor(keys, draft, loaded)
    setSaveGroup(group, { state: 'loading', error: '' })
    try {
      await apiFetch(`/auth-config/${group}`, { method: 'PUT', body: { values } })
      setLoaded((l) => ({ ...l, ...values }))
      for (const k of ['auth.cn.appSecret', 'auth.my.apiKey']) {
        if (values[k]) setSecretSet((s) => ({ ...s, [k]: true }))
      }
      setDraft((d) => ({ ...d, 'auth.cn.appSecret': '', 'auth.my.apiKey': '' }))
      setEditing(null)
      toast.success(a.saved)
      setSaveGroup(group, { state: 'success', error: '' })
      setTimeout(() => setSaveGroup(group, { state: 'idle' }), 1500)
    } catch (e) {
      const msg = e instanceof Error ? e.message : a.saveFail
      setSaveGroup(group, { state: 'failed', error: msg })
      toast.error(msg)
      setTimeout(() => setSaveGroup(group, { state: 'idle' }), 2500)
    }
  }

  const test = async (group: Group) => {
    const cur = testState[group]
    if (cur.state === 'loading') return
    setTestGroup(group, { state: 'loading', error: '' })
    try {
      const values = payloadFor(groupKeys(group), draft, loaded)
      const d = await apiFetch<{ ok: boolean; message: string }>(`/auth-config/${group}/test`, {
        method: 'POST', body: { values },
      })
      if (d?.ok) {
        toast.success(d.message || a.testOk)
        setTestGroup(group, { state: 'success', error: '' })
        setTimeout(() => setTestGroup(group, { state: 'idle' }), 1500)
      } else {
        const msg = d?.message || a.testFail
        setTestGroup(group, { state: 'failed', error: msg })
        toast.error(msg)
        setTimeout(() => setTestGroup(group, { state: 'idle' }), 2500)
      }
    } catch (e) {
      const msg = e instanceof Error ? e.message : a.testFail
      setTestGroup(group, { state: 'failed', error: msg })
      toast.error(msg)
      setTimeout(() => setTestGroup(group, { state: 'idle' }), 2500)
    }
  }

  const cnOn = draft['auth.cn.enabled'] === 'true'
  const myOn = draft['auth.my.enabled'] === 'true'

  const summaryRow = (label: string, value: ReactNode) => (
    <div className="flex gap-2 text-[13px] text-[var(--shell-content-text)]">
      <span className="w-32 shrink-0 text-[var(--shell-crumb-text)]">{label}</span>
      <span className="break-all">{value || '—'}</span>
    </div>
  )
  const cardHead = (title: string, badge: ReactNode, group: Group) => (
    <div className="mb-3 flex items-center justify-between">
      <span className="flex items-center gap-2 font-semibold text-[var(--shell-heading)]">
        {title}{badge}
      </span>
      <ToolbarButton primary onClick={() => setEditing(group)}>{a.edit}</ToolbarButton>
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
        <Card className="flex flex-col gap-2 p-4">
          {cardHead(a.cnTitle, cnOn ? <Badge variant="success">{a.enabled}</Badge> : <Badge>{a.disabled}</Badge>, 'cn')}
          {summaryRow(a.cnAppKey, draft['auth.cn.appKey'])}
          {summaryRow(a.cnAppSecret, secretSet['auth.cn.appSecret'] ? a.secretSet : '')}
          {summaryRow(a.cnPackage, draft['auth.cn.packageName'])}
          {summaryRow(a.cnTimeout, draft['auth.cn.preloadTimeoutMs'])}
        </Card>

        {/* 卡片二:海外号码认证 */}
        <Card className="flex flex-col gap-2 p-4">
          {cardHead(a.myTitle, myOn ? <Badge variant="success">{a.enabled}</Badge> : <Badge>{a.pending}</Badge>, 'my')}
          {summaryRow(a.myProvider, draft['auth.my.provider'])}
          {summaryRow(a.mySmsProvider, draft['auth.my.smsProvider'])}
          {summaryRow(a.myApiKey, secretSet['auth.my.apiKey'] ? a.secretSet : '')}
          {summaryRow(a.myCountryCode, draft['auth.my.countryCode'])}
          {summaryRow(a.mySmsSign, draft['auth.my.smsSign'])}
        </Card>

        {/* 卡片三:降级与合规 */}
        <Card className="flex flex-col gap-2 p-4">
          {cardHead(a.fbTitle, null, 'fallback')}
          {summaryRow(a.fbSmsOnFail, draft['auth.fallback.smsOnFail'] === 'true' ? a.enabled : a.disabled)}
          {summaryRow(a.fbBillingAlert, draft['auth.fallback.billingAlert'] === 'true' ? a.enabled : a.disabled)}
          {summaryRow(a.fbAutoRegister, draft['auth.fallback.autoRegister'] === 'true' ? a.enabled : a.disabled)}
          {summaryRow(a.fbPrivacyVersion, draft['auth.compliance.privacyVersion'])}
          {summaryRow(a.fbAgreementUrl, draft['auth.compliance.agreementUrl'])}
          <div className="mt-1 flex items-center gap-1 text-xs text-[var(--shell-crumb-text)]"><InfoIcon /><span>{a.complianceNote}</span></div>
        </Card>
      </div>

      <AuthConfigDrawers editing={editing} draft={draft} set={set} secretSet={secretSet} a={a} t={t}
        setEditing={setEditing} save={save} test={test}
        saveState={saveState[editing ?? 'cn']?.state ?? 'idle'}
        testState={testState[editing ?? 'cn']?.state ?? 'idle'}
        saveError={saveState[editing ?? 'cn']?.error ?? ''}
        testError={testState[editing ?? 'cn']?.error ?? ''} />
    </div>
  )
}

