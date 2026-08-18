// 框架布局:顶栏(档案/数据域)+ 侧栏(roleCode 过滤)+ 内容区 Outlet。
import { Outlet, useNavigate } from 'react-router-dom'
import type { ReactNode } from 'react'
import { adminLogout, type Profile } from '../api/auth'
import logoMark from '../assets/brand/logo-mark-navy.png'
import { ProfileContext } from './profile'
import { Sidebar } from './Sidebar'

export function AdminLayout({ profile }: { profile: Profile; children?: ReactNode }) {
  const nav = useNavigate()
  return (
    <ProfileContext.Provider value={profile}>
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
          <span style={{ display: 'flex', gap: 10, alignItems: 'center', fontSize: 15, fontWeight: 600 }}>
            <img
              src={logoMark}
              alt="Sphere Boss"
              style={{ width: 26, height: 26, background: '#fff', borderRadius: 6, padding: 1 }}
            />
            Sphere Boss · BOSS 管理端
          </span>
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
        <div style={{ display: 'flex', minHeight: 'calc(100vh - 56px)' }}>
          <Sidebar roleCode={profile.roleCode} />
          <main style={{ flex: 1, padding: 24 }}>
            <Outlet />
          </main>
        </div>
      </div>
    </ProfileContext.Provider>
  )
}
