// URL search 参数驱动的页面状态:刷新/分享链接后搜索条件与分页不丢。
// 用 replace 更新地址栏,不产生浏览器历史记录。
import { useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'

export function useQueryState(key: string, fallback: string): [string, (v: string) => void] {
  const [params, setParams] = useSearchParams()
  const value = params.get(key) ?? fallback
  const setValue = useCallback(
    (v: string) => {
      setParams(
        (prev) => {
          const next = new URLSearchParams(prev)
          if (v === '' || v === fallback) next.delete(key)
          else next.set(key, v)
          return next
        },
        { replace: true },
      )
    },
    [key, fallback, setParams],
  )
  return [value, setValue]
}

export function useQueryInt(key: string, fallback: number): [number, (v: number) => void] {
  const [raw, setRaw] = useQueryState(key, String(fallback))
  const parsed = Number.parseInt(raw, 10)
  const value = Number.isFinite(parsed) && parsed > 0 ? parsed : fallback
  const setValue = useCallback((v: number) => setRaw(String(Math.max(1, Math.floor(v)))), [setRaw])
  return [value, setValue]
}
