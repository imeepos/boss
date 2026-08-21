// 后台提醒轮询 hook:30s 拉未读数;TopBar 铃铛与消息中心页共用。
// API:GET /notifications/unread-count、POST /notifications/read(契约 docs/plan/admin-notify-center.md)。
import { useCallback, useEffect, useRef, useState } from 'react'
import { apiFetch } from '../api/client'

const POLL_MS = 30_000

export function fetchUnreadCount(): Promise<number> {
  return apiFetch<{ count: number }>('/notifications/unread-count')
    .then((d) => d?.count ?? 0)
    .catch(() => 0)
}

export function markNotificationsRead(ids: number[]): Promise<unknown> {
  return apiFetch('/notifications/read', { method: 'POST', body: { ids } })
}

/** 轮询未读数;返回 count 与手动刷新。 */
export function useUnreadCount(enabled: boolean): { count: number; refresh: () => void } {
  const [count, setCount] = useState(0)
  const timer = useRef<ReturnType<typeof setInterval>>()

  const refresh = useCallback(() => {
    void fetchUnreadCount().then(setCount)
  }, [])

  useEffect(() => {
    if (!enabled) return
    refresh()
    timer.current = setInterval(refresh, POLL_MS)
    return () => clearInterval(timer.current)
  }, [enabled, refresh])

  return { count, refresh }
}
