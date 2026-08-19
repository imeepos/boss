// 路由守卫:无 token 跳 /login;有 token 预取 /auth/me,失败(401/网络)登出。
// 会话保活:每 6h 及窗口重新聚焦时静默续期(滑动 TTL),活跃用户不再被踢回登录页。
import { useEffect, useState, type ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import { adminLogout, fetchMe, refreshAuthToken, type Profile } from '../api/auth'
import { getAuthToken } from '../api/client'
import { Loading } from '../components/Loading'

const REFRESH_INTERVAL_MS = 6 * 60 * 60 * 1000

/** 静默续期:失败不惊扰用户,交由下一次请求的 401 路径登出。 */
function silentRefresh(): void {
  refreshAuthToken().catch(() => undefined)
}

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

  useEffect(() => {
    if (state !== 'ok') return
    silentRefresh()
    const timer = window.setInterval(silentRefresh, REFRESH_INTERVAL_MS)
    const onVisible = () => {
      if (document.visibilityState === 'visible') silentRefresh()
    }
    document.addEventListener('visibilitychange', onVisible)
    return () => {
      window.clearInterval(timer)
      document.removeEventListener('visibilitychange', onVisible)
    }
  }, [state])

  if (state === 'denied') return <Navigate to="/login" replace />
  if (state === 'loading') return <Loading />
  return <>{profile ? children(profile) : <Loading />}</>
}
