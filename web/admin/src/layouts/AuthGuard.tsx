// 路由守卫:无 token 跳 /login;有 token 预取 /auth/me,失败(401/网络)登出。
import { useEffect, useState, type ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import { adminLogout, fetchMe, type Profile } from '../api/auth'
import { getAuthToken } from '../api/client'
import { Loading } from '../components/Loading'

/** 会话上下文:守卫加载完成后经 props 下发,避免引额外依赖。 */
export function AuthGuard({ children }: { children: (profile: Profile) => ReactNode }) {
  const [profile, setProfile] = useState<Profile | null>(null)
  const [state, setState] = useState<'loading' | 'ok' | 'denied'>('loading')

  useEffect(() => {
    if (!getAuthToken()) {
      setState('denied')
      return
    }
    fetchMe()
      .then((p) => {
        if (p) {
          setProfile(p)
          setState('ok')
        } else {
          setState('denied')
        }
      })
      .catch(() => {
        void adminLogout()
        setState('denied')
      })
  }, [])

  if (state === 'denied') return <Navigate to="/login" replace />
  if (state === 'loading') return <Loading />
  return <>{profile ? children(profile) : <Loading />}</>
}
