// 登录页:规格对齐 visual-design-prompts.md §3.4 登录页纵向参考,文本走 i18n。
// 服务端选择器在账号密码上方:不自动弹框;未配置时选择器提示并拦截提交,由用户点"管理服务端"手动配置。
// 样式:令牌走 --color-* 静态色板(品牌面不随主题翻转);表单控件复用 auth-shell 体系。
import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { adminLogin } from '../../api/auth'
import logoFull from '../../assets/brand/logo-mark-gradient.png'
import ornamentShield from '../../assets/brand/ornament-shield.png'
import { AuthShell, BrandAside, inputStyle, buttonStyle } from '../auth-shell'
import { AdCarousel } from '../auth-ads'
import { useT } from '../../i18n'
import { ServerManagerDialog } from '../../components/ServerManagerDialog'
import { SimplePicker } from '../../components/pickers/SimplePicker'
import { initialPickerState, pickServer, type ServerPickerState } from './serverPicker'

export default function LoginPage() {
  const nav = useNavigate()
  const t = useT()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [picker, setPicker] = useState<ServerPickerState>(initialPickerState)
  const [manageOpen, setManageOpen] = useState(false)

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (!picker.activeId) {
      setError(t.auth.login.serverRequired)
      return
    }
    if (!username || !password) {
      setError(t.auth.login.emptyFields)
      return
    }
    setBusy(true)
    setError('')
    try {
      await adminLogin(username, password)
      nav('/', { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : t.auth.login.fail)
    } finally {
      setBusy(false)
    }
  }

  const onPick = (id: string) => setPicker({ ...picker, activeId: pickServer(id) })

  return (
    <AuthShell
      aside={
        <BrandAside>
          <AdCarousel />
        </BrandAside>
      }
    >
      <div className="flex w-[274px] flex-col items-center pt-[38px]">
        <img src={logoFull} alt="Sphere Boss" className="block h-10 w-10" />
        <h2 className="m-0 mt-4 text-center text-[20px] font-bold leading-7 text-[var(--color-brand-navy-950)]">{t.auth.login.title}</h2>
        <p className="m-0 mt-1.5 text-center text-xs leading-[18px] text-[var(--color-text-secondary)]">{t.auth.login.subtitle}</p>
        <form onSubmit={onSubmit} className="mt-3">
          <SimplePicker
            value={picker.activeId}
            options={[
              ...(!picker.servers.length ? [{ value: '', label: t.auth.login.serverNone }] : []),
              ...picker.servers.map((s) => ({ value: s.id, label: `${s.name} (${s.baseUrl})` })),
            ]}
            onChange={onPick}
            ariaLabel={t.auth.login.serverNone}
            minWidth={274}
          />
          <span className="mb-3 flex items-center justify-between">
            <a onClick={() => setManageOpen(true)} className="mb-3 block cursor-pointer text-[11px] text-[var(--color-text-link)]">{t.auth.login.serverManage}</a>
            <a onClick={() => nav('/partner/apply')} className="mb-3 block cursor-pointer text-[11px] text-[var(--color-text-link)]">{t.auth.login.partnerApply}</a>
          </span>
          <input
            placeholder={t.auth.login.usernamePlaceholder}
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            style={inputStyle}
          />
          <input
            placeholder={t.auth.login.passwordPlaceholder}
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            style={inputStyle}
            className="!mb-0"
          />
          {error && <div className="mt-2 text-[11px] leading-4 text-[var(--color-danger)]">{error}</div>}
          <button type="submit" disabled={busy} style={buttonStyle} className="!mt-3">
            {busy ? t.auth.login.submitting : t.auth.login.submit}
          </button>
        </form>
        <div className="mt-5 flex items-center justify-center gap-1.5 text-xs">
          <img src={ornamentShield} alt="" className="h-3.5 w-3.5" />
          <span className="text-[var(--color-text-secondary)]">{t.auth.login.assignedByAdmin}</span>
        </div>
      </div>
      {manageOpen && <ServerManagerDialog onClose={() => { setManageOpen(false); setPicker(initialPickerState()) }} />}
    </AuthShell>
  )
}
