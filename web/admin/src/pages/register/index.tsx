// 注册页:规格对齐 visual-design-prompts.md §3.4 注册页纵向参考 + §7 差异表。
import { useState, type CSSProperties, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { adminLogin, adminRegister } from '../../api/auth'
import logoLine from '../../assets/brand/logo-mark-lineart.png'
import ornamentShield from '../../assets/brand/ornament-shield.png'
import { AuthShell, BrandAside, inputStyle, buttonStyle, BRAND_NAVY, BRAND_GOLD } from '../auth-shell'
import { AdCarousel } from '../auth-ads'

const USERNAME_RE = /^[A-Za-z0-9_\-.]{3,64}$/

export default function RegisterPage() {
  const nav = useNavigate()
  const [form, setForm] = useState({ username: '', password: '', confirm: '', realName: '' })
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const set = (k: keyof typeof form) => (e: { target: { value: string } }) =>
    setForm((f) => ({ ...f, [k]: e.target.value }))

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (!form.realName.trim()) return setError('请输入姓名')
    if (!USERNAME_RE.test(form.username)) return setError('账号需 3-64 位,仅限字母/数字/_-.')
    if (form.password.length < 6) return setError('密码不少于 6 位')
    if (form.password !== form.confirm) return setError('两次输入的密码不一致')
    setBusy(true)
    setError('')
    try {
      await adminRegister(form.username, form.password, form.realName.trim())
      await adminLogin(form.username, form.password)
      nav('/', { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : '注册失败')
    } finally {
      setBusy(false)
    }
  }

  return (
    <AuthShell
      aside={
        <BrandAside tip="注册即开通业务运营账号">
          <AdCarousel />
        </BrandAside>
      }
    >
      {/* 注册页表单:104px 起(比登录高 38px) */}
      <div style={regFormStyle}>
        <img src={logoLine} alt="Sphere Boss" style={logoStyle} />
        <h2 style={titleStyle}>注册账号</h2>
        <p style={subStyle}>创建 Sphere Boss 管理端账号</p>
        <form onSubmit={onSubmit} style={{ marginTop: 8 }}>
          <input placeholder="姓名" value={form.realName} onChange={set('realName')} style={inputStyle} />
          <input
            placeholder="账号(3-64 位,字母/数字/_/.)"
            value={form.username}
            onChange={set('username')}
            style={inputStyle}
          />
          <input
            placeholder="密码(不少于 6 位)"
            type="password"
            value={form.password}
            onChange={set('password')}
            style={inputStyle}
          />
          <input
            placeholder="确认密码"
            type="password"
            value={form.confirm}
            onChange={set('confirm')}
            style={{ ...inputStyle, marginBottom: 0 }}
          />
          {error && <div style={errStyle}>{error}</div>}
          <button type="submit" disabled={busy} style={{ ...buttonStyle, marginTop: 12 }}>
            {busy ? '注册中…' : '注 册'}
          </button>
        </form>
        <div style={linkRowStyle}>
          <img src={ornamentShield} alt="" style={shieldStyle} />
          <span style={{ color: '#7C8799' }}>已有账号?</span>
          <Link to="/login" style={linkStyle}>
            返回登录
          </Link>
        </div>
      </div>
    </AuthShell>
  )
}

const regFormStyle: CSSProperties = {
  width: 274,
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'center',
}

const logoStyle: CSSProperties = {
  width: 32, height: 32, display: 'block',
}

const titleStyle: CSSProperties = {
  margin: '18px 0 0', fontSize: 20, lineHeight: '28px', fontWeight: 700,
  color: BRAND_NAVY, textAlign: 'center',
}

const subStyle: CSSProperties = {
  margin: '6px 0 0', fontSize: 12, lineHeight: '18px', fontWeight: 400,
  color: '#7C8799', textAlign: 'center',
}

const errStyle: CSSProperties = {
  color: '#D94B4B', fontSize: 11, lineHeight: '16px', marginTop: 8,
}

const linkRowStyle: CSSProperties = {
  marginTop: 20, display: 'flex', alignItems: 'center', justifyContent: 'center',
  gap: 6, fontSize: 12,
}

const shieldStyle: CSSProperties = { width: 14, height: 14 }

const linkStyle: CSSProperties = {
  color: BRAND_GOLD, fontWeight: 600, fontSize: 12, textDecoration: 'none',
}