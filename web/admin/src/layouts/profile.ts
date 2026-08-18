// 会话档案上下文:AuthGuard 加载后注入,菜单路由内消费。
import { createContext, useContext } from 'react'
import type { Profile } from '../api/auth'

export const ProfileContext = createContext<Profile | null>(null)

/** 取当前档案;守卫外调用抛错(仅路由树内使用)。 */
export function useProfile(): Profile {
  const p = useContext(ProfileContext)
  if (!p) throw new Error('useProfile 必须在 AuthGuard 内使用')
  return p
}
