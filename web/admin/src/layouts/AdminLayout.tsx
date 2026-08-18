// 顶栏布局:realName/roleName/legalEntityName + 数据域徽标(regionScope 空=全集团)。
import type { ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import { adminLogout, type Profile } from '../api/auth'

export function AdminLayout({ profile, children }: { profile: Profile; children: ReactNode }) {
  const nav = useNavigate()
  return (
    <div style={{ minHeight: '100vh', background: '#f5f6fa' }}>
      <header
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          padding: '0 24px',
          height: 56,
          background: '#1f2d3d',
          color: '#fff',
        }}
      >
        <span>BOSS 管理端</span>
        <span style={{ display: 'flex', gap: 16, alignItems: 'center', fontSize: 13 }}>
          <span>{profile.legalEntityName || '—'}</span>
          <span>{profile.regionScope ? `数据域:${profile.regionScope}` : '数据域:全集团'}</span>
          <span>
            {profile.realName}({profile.roleName})
          </span>
          <a
            onClick={async () => {
              await adminLogout()
              nav('/login', { replace: true })
            }}
            style={{ color: '#9cf', cursor: 'pointer' }}
          >
            退出
          </a>
        </span>
      </header>
      <main style={{ padding: 24 }}>{children}</main>
    </div>
  )
}
