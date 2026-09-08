// 认证配置编辑抽屉集(cn/my/fallback)+ 密钥输入 + 抽屉按钮排;自 index.tsx 拆出(单文件行数红线)。
import { useState } from 'react'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { FormField } from '../../../components/business/form-field'
import { Input } from '../../../components/ui/input'
import { Switch } from '../../../components/ui/switch'
import { ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { SubmitButton, type SubmitState } from '../../../components/business/submit-button'
import type { Translations } from '../../../i18n/types'

type Draft = Record<string, string>
type Group = 'cn' | 'my' | 'fallback'
type Texts = Translations['pages']['authconfig']

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

/** 配置抽屉 footer:测试 / 取消 / 保存(SubmitButton 状态机)。 */
function ConfigFooter({ busy, testing, testLabel, testingLabel, saveLabel, savingLabel, savedLabel, saveFailLabel, cancelLabel, onCancel, onTest, onSave, saveState, testState }: {
  busy: boolean
  testing: boolean
  testLabel: string
  testingLabel: string
  saveLabel: string
  savingLabel: string
  savedLabel: string
  saveFailLabel: string
  cancelLabel: string
  onCancel: () => void
  onTest: () => void
  onSave: () => void
  saveState: SubmitState
  testState: SubmitState
}) {
  return (
    <>
      <SubmitButton
        state={testState}
        labels={{ idle: testLabel, loading: testingLabel, success: testLabel, failed: testLabel }}
        disabled={testing}
        onClick={onTest}
      />
      <span className="flex-1" />
      <ToolbarButton onClick={onCancel} disabled={busy}>{cancelLabel}</ToolbarButton>
      <SubmitButton
        state={saveState}
        labels={{ idle: saveLabel, loading: savingLabel, success: savedLabel, failed: saveFailLabel }}
        disabled={busy}
        onClick={onSave}
      />
    </>
  )
}

/** 按 editing 渲染对应配置抽屉;加载/保存/测试状态由页面持有。 */
export function AuthConfigDrawers({ editing, draft, set, secretSet, a, t, setEditing, save, test, saveState, testState, saveError, testError }: {
  editing: Group | null
  draft: Draft
  set: (key: string, v: string) => void
  secretSet: Record<string, boolean>
  a: Texts
  t: Translations
  setEditing: (g: Group | null) => void
  save: (g: Group) => void
  test: (g: Group) => void
  saveState: SubmitState
  testState: SubmitState
  saveError: string
  testError: string
}) {
  const cnOn = draft['auth.cn.enabled'] === 'true'
  const myOn = draft['auth.my.enabled'] === 'true'
  return (
    <>
      {editing === 'cn' && (
        <Drawer title={a.cnTitle} onClose={() => setEditing(null)}
          footer={<ConfigFooter busy={saveState === 'loading'} testing={testState === 'loading'}
            testLabel={a.testBtn} testingLabel={a.testing}
            saveLabel={a.save} savingLabel={a.saving} savedLabel={a.saved} saveFailLabel={a.saveFail}
            cancelLabel={t.common.confirmDialog.cancel}
            onCancel={() => setEditing(null)} onTest={() => test('cn')} onSave={() => save('cn')}
            saveState={saveState} testState={testState} />}>
          {saveError && <div className="mb-3"><ErrorBanner message={saveError} /></div>}
          {testError && <div className="mb-3"><ErrorBanner message={testError} /></div>}
          <div className="mb-4 flex items-center gap-2">
            <Switch checked={cnOn} onCheckedChange={(v) => set('auth.cn.enabled', String(v))} aria-label={a.cnTitle} />
            {cnOn ? a.enabled : a.disabled}
          </div>
          <div className="grid grid-cols-2 gap-4">
            <FormField label={a.cnAppKey}>
              <Input className="w-72" value={draft['auth.cn.appKey'] ?? ''} onChange={(e) => set('auth.cn.appKey', e.target.value)} />
            </FormField>
            <FormField label={a.cnAppSecret}>
              <SecretInput value={draft['auth.cn.appSecret'] ?? ''} onChange={(v) => set('auth.cn.appSecret', v)}
                placeholder={a.secretSet} hasValue={!!secretSet['auth.cn.appSecret']} />
            </FormField>
            <FormField label={a.cnPackage}>
              <Input className="w-72" value={draft['auth.cn.packageName'] ?? ''} onChange={(e) => set('auth.cn.packageName', e.target.value)} />
            </FormField>
            <FormField label={a.cnTimeout} hint={a.cnTimeoutHint}>
              <Input className="w-72" inputMode="numeric" value={draft['auth.cn.preloadTimeoutMs'] ?? ''} onChange={(e) => set('auth.cn.preloadTimeoutMs', e.target.value)} />
            </FormField>
          </div>
        </Drawer>
      )}

      {editing === 'my' && (
        <Drawer title={a.myTitle} onClose={() => setEditing(null)}
          footer={<ConfigFooter busy={saveState === 'loading'} testing={testState === 'loading'}
            testLabel={a.testBtn} testingLabel={a.testing}
            saveLabel={a.save} savingLabel={a.saving} savedLabel={a.saved} saveFailLabel={a.saveFail}
            cancelLabel={t.common.confirmDialog.cancel}
            onCancel={() => setEditing(null)} onTest={() => test('my')} onSave={() => save('my')}
            saveState={saveState} testState={testState} />}>
          {saveError && <div className="mb-3"><ErrorBanner message={saveError} /></div>}
          {testError && <div className="mb-3"><ErrorBanner message={testError} /></div>}
          <div className="mb-4 flex items-center gap-2">
            <Switch checked={myOn} onCheckedChange={(v) => set('auth.my.enabled', String(v))} aria-label={a.myTitle} />
            {myOn ? a.enabled : a.pending}
          </div>
          <div className="grid grid-cols-2 gap-4">
            <FormField label={a.myProvider}>
              <Dropdown value={draft['auth.my.provider'] ?? 'none'}
                options={[{ value: 'opengateway', label: a.providerOg }, { value: 'none', label: a.providerNone }]}
                onChange={(v) => set('auth.my.provider', v)} ariaLabel={a.myProvider} />
            </FormField>
            <FormField label={a.mySmsProvider}>
              <Dropdown value={draft['auth.my.smsProvider'] ?? 'engagelab'}
                options={[{ value: 'engagelab', label: 'EngageLab' }, { value: 'twilio', label: 'Twilio' }, { value: 'vonage', label: 'Vonage' }]}
                onChange={(v) => set('auth.my.smsProvider', v)} ariaLabel={a.mySmsProvider} />
            </FormField>
            <FormField label={a.myApiKey}>
              <SecretInput value={draft['auth.my.apiKey'] ?? ''} onChange={(v) => set('auth.my.apiKey', v)}
                placeholder={a.secretSet} hasValue={!!secretSet['auth.my.apiKey']} />
            </FormField>
            <FormField label={a.myCountryCode}>
              <Input className="w-72" value={draft['auth.my.countryCode'] ?? ''} onChange={(e) => set('auth.my.countryCode', e.target.value)} />
            </FormField>
            <FormField label={a.mySmsSign}>
              <Input className="w-72" value={draft['auth.my.smsSign'] ?? ''} onChange={(e) => set('auth.my.smsSign', e.target.value)} />
            </FormField>
          </div>
        </Drawer>
      )}

      {editing === 'fallback' && (
        <Drawer title={a.fbTitle} onClose={() => setEditing(null)}
          footer={<ConfigFooter busy={saveState === 'loading'} testing={testState === 'loading'}
            testLabel={a.testBtn} testingLabel={a.testing}
            saveLabel={a.save} savingLabel={a.saving} savedLabel={a.saved} saveFailLabel={a.saveFail}
            cancelLabel={t.common.confirmDialog.cancel}
            onCancel={() => setEditing(null)} onTest={() => test('fallback')} onSave={() => save('fallback')}
            saveState={saveState} testState={testState} />}>
          {saveError && <div className="mb-3"><ErrorBanner message={saveError} /></div>}
          {testError && <div className="mb-3"><ErrorBanner message={testError} /></div>}
          <div className="mb-4 flex items-center gap-8">
            {([
              ['auth.fallback.smsOnFail', a.fbSmsOnFail],
              ['auth.fallback.billingAlert', a.fbBillingAlert],
              ['auth.fallback.autoRegister', a.fbAutoRegister],
            ] as const).map(([key, label]) => (
              <label key={key} className="flex cursor-pointer items-center gap-2 text-[13px] text-[var(--shell-content-text)]">
                <Switch checked={draft[key] === 'true'} onCheckedChange={(v) => set(key, String(v))} aria-label={label} />
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
          <div className="mt-3 flex items-center gap-1 text-xs text-[var(--shell-crumb-text)]"><InfoIcon /><span>{a.complianceNote}</span></div>
        </Drawer>
      )}
    </>
  )
}
