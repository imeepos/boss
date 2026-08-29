// URL search 参数驱动的页面状态:刷新/分享链接后搜索条件与分页不丢。
// 用 replace 更新地址栏,不产生浏览器历史记录。
import { useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'

export function useQueryState(key: string, fallback: string): [string, (v: string) => void] {
  const [params, setParams] = useSearchParams()
  const value = params.get(key) ?? fallback
  const setValue = useCallback(
    (v: string) => {
      // prev 取 window.location.search 而非函数式入参:同批次连续两次 setParams(如
      // 过滤器互斥双写)时,第二次的函数式 prev 仍是未提交导航前的旧值,会用旧参数集
      // 覆盖第一次写入,表现为「URL query 被整体清空」;history 同步写,直读即权威值。
      const cur = new URLSearchParams(window.location.search)
      if (v === '' || v === fallback) cur.delete(key)
      else cur.set(key, v)
      setParams(cur, { replace: true })
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
