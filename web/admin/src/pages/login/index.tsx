// 登录页:规格对齐 visual-design-prompts.md §3.4 登录页纵向参考,文本走 i18n。
import { useState, type CSSProperties, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { adminLogin } from '../../api/auth'
import logoFull from '../../assets/brand/logo-mark-gradient.png'
import ornamentShield from '../../assets/brand/ornament-shield.png'
import { AuthShell, BrandAside, inputStyle, buttonStyle, BRAND_NAVY, BRAND_GOLD } from '../auth-shell'
import { AdCarousel } from '../auth-ads'
import { useT } from '../../i18n'

export default function LoginPage() {
  const nav = useNavigate()
  const t = useT()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault()
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

  return (
    <AuthShell
      aside={
        <BrandAside>
          <AdCarousel />
        </BrandAside>
      }
    >
      <div style={loginFormStyle}>
        <img src={logoFull} alt="Sphere Boss" style={logoStyle} />
        <h2 style={titleStyle}>{t.auth.login.title}</h2>
        <p style={subStyle}>{t.auth.login.subtitle}</p>
        <form onSubmit={onSubmit} style={{ marginTop: 12 }}>
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
            style={{ ...inputStyle, marginBottom: 0 }}
          />
          {error && <div style={errStyle}>{error}</div>}
          <button type="submit" disabled={busy} style={{ ...buttonStyle, marginTop: 12 }}>
            {busy ? t.auth.login.submitting : t.auth.login.submit}
          </button>
        </form>
        <div style={linkRowStyle}>
          <img src={ornamentShield} alt="" style={shieldStyle} />
          <span style={{ color: '#7C8799' }}>{t.auth.login.noAccount}</span>
          <Link to="/register" style={linkStyle}>
            {t.auth.login.toRegister}
          </Link>
        </div>
      </div>
    </AuthShell>
  )
}

const loginFormStyle: CSSProperties = {
  width: 274,
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'center',
  paddingTop: 38,
}

const logoStyle: CSSProperties = {
  width: 40, height: 40, display: 'block',
}

const titleStyle: CSSProperties = {
  margin: '16px 0 0', fontSize: 20, lineHeight: '28px', fontWeight: 700,
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
  transition: 'color 160ms',
}