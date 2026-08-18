// 认证接口(契约:internal/app/http.go —— /auth/login、/auth/me、/auth/logout)。
import { apiFetch, setAuthToken } from './client'

export interface LoginResult {
  token: string
  accountId: number
  realName: string
  roleName: string
}

export interface Profile {
  accountId: number
  username: string
  realName: string
  roleCode: string
  roleName: string
  legalEntityName: string
  regionScope: string
}

/** 登录:成功后立即持久化 token。 */
export async function adminLogin(username: string, password: string): Promise<LoginResult> {
  const data = await apiFetch<LoginResult>('/auth/login', {
    method: 'POST',
    body: { username, password },
  })
  if (!data?.token) throw new Error('登录响应缺少 token')
  setAuthToken(data.token)
  return data
}

/** 当前账号档案(菜单可见性按 roleCode 驱动)。 */
export function fetchMe(): Promise<Profile | null> {
  return apiFetch<Profile>('/auth/me')
}

/** 登出:清本地态;后端登出失败不阻断。 */
export async function adminLogout(): Promise<void> {
  try {
    await apiFetch('/auth/logout', { method: 'POST' })
  } finally {
    setAuthToken(null)
  }
}
