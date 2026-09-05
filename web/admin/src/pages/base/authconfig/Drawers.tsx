// 认证配置编辑抽屉集(cn/my/fallback)+ 密钥输入 + 抽屉按钮排;自 index.tsx 拆出(单文件行数红线)。
import { useState } from 'react'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { FormField } from '../../../components/business/form-field'
import { Input } from '../../../components/ui/input'
import { Switch } from '../../../components/ui/switch'
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

/** 配置抽屉 footer:测试 / 取消 / 保存。 */
function ConfigFooter({ busy, testing, testLabel, testingLabel, saveLabel, savingLabel, cancelLabel, onCancel, onTest, onSave }: {
  busy: boolean
  testing: boolean
  testLabel: string
  testingLabel: string
  saveLabel: string
  savingLabel: string
  cancelLabel: string
  onCancel: () => void
  onTest: () => void
  onSave: () => void
}) {
  return (
    <>
      <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]"
        disabled={testing} onClick={onTest}>
        {testing ? testingLabel : testLabel}
      </button>
      <span className="flex-1" />
      <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onCancel}>
        {cancelLabel}
      </button>
      <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy} onClick={onSave}>
        {busy ? savingLabel : saveLabel}
      </button>
    </>
  )
}

/** 按 editing 渲染对应配置抽屉;加载/保存/测试状态由页面持有。 */
export function AuthConfigDrawers({ editing, draft, set, secretSet, saving, testing, a, t, setEditing, save, test }: {
  editing: Group | null
  draft: Draft
  set: (key: string, v: string) => void
  secretSet: Record<string, boolean>
  saving: string
  testing: string
  a: Texts
  t: Translations
  setEditing: (g: Group | null) => void
  save: (g: Group) => void
  test: (g: Group) => void
}) {
  const cnOn = draft['auth.cn.enabled'] === 'true'
  const myOn = draft['auth.my.enabled'] === 'true'
  return (
    <>
      {editing === 'cn' && (
        <Drawer title={a.cnTitle} onClose={() => setEditing(null)}
          footer={<ConfigFooter busy={saving === 'cn'} testing={testing === 'cn'} testLabel={a.testBtn} testingLabel={a.testing}
            saveLabel={a.save} savingLabel={a.saving} cancelLabel={t.common.confirmDialog.cancel}
            onCancel={() => setEditing(null)} onTest={() => test('cn')} onSave={() => save('cn')} />}>
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
          footer={<ConfigFooter busy={saving === 'my'} testing={testing === 'my'} testLabel={a.testBtn} testingLabel={a.testing}
            saveLabel={a.save} savingLabel={a.saving} cancelLabel={t.common.confirmDialog.cancel}
            onCancel={() => setEditing(null)} onTest={() => test('my')} onSave={() => save('my')} />}>
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
          footer={<ConfigFooter busy={saving === 'fallback'} testing={testing === 'fallback'} testLabel={a.testBtn} testingLabel={a.testing}
            saveLabel={a.save} savingLabel={a.saving} cancelLabel={t.common.confirmDialog.cancel}
            onCancel={() => setEditing(null)} onTest={() => test('fallback')} onSave={() => save('fallback')} />}>
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
          <div className="mt-3 text-xs text-[var(--shell-crumb-text)]">ⓘ {a.complianceNote}</div>
        </Drawer>
      )}
    </>
  )
}
