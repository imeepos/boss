// 登录页:交互照抄 docs/admin/login.html(账号密码 + 错误提示)。
import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { adminLogin } from '../../api/auth'

export default function LoginPage() {
  const nav = useNavigate()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (!username || !password) {
      setError('请输入账号与密码')
      return
    }
    setBusy(true)
    setError('')
    try {
      await adminLogin(username, password)
      nav('/', { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : '登录失败')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div style={{ display: 'grid', placeItems: 'center', minHeight: '100vh', background: '#f5f6fa' }}>
      <form
        onSubmit={onSubmit}
        style={{ width: 320, padding: 32, background: '#fff', borderRadius: 8, boxShadow: '0 2px 8px rgba(0,0,0,.08)' }}
      >
        <h2 style={{ marginTop: 0 }}>BOSS 综合业务支撑平台</h2>
        <input
          placeholder="账号"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          style={inputStyle}
        />
        <input
          placeholder="密码"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          style={inputStyle}
        />
        {error && <div style={{ color: '#e54545', fontSize: 13, marginBottom: 12 }}>{error}</div>}
        <button type="submit" disabled={busy} style={{ width: '100%', padding: '8px 0' }}>
          {busy ? '登录中…' : '登录'}
        </button>
      </form>
    </div>
  )
}

const inputStyle: React.CSSProperties = {
  width: '100%',
  boxSizing: 'border-box',
  padding: '8px 10px',
  marginBottom: 12,
}
