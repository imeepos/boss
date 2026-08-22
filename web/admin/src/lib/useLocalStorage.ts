// useLocalStorage:JSON 序列化的 React state hook,持久化到 localStorage。
// mount 时读回(无 key 或解析失败 → initial);setValue 同步写回;跨标签 storage 事件同步(可选).
// SSR 安全(typeof window 兜底);不写时不做 JSON.stringify 性能浪费。

import { useCallback, useEffect, useState } from 'react'

export function useLocalStorage<T>(
  key: string,
  initial: T,
): [T, (v: T | ((prev: T) => T)) => void] {
  const [value, setValueState] = useState<T>(() => readStorage(key, initial))

  const setValue = useCallback((v: T | ((prev: T) => T)) => {
    setValueState((prev) => {
      const next = typeof v === 'function' ? (v as (p: T) => T)(prev) : v
      const ls = (globalThis as { localStorage?: Storage }).localStorage
      if (ls) {
        try {
          ls.setItem(key, JSON.stringify(next))
        } catch {
          // quota/隐私模式等;不阻塞 state 更新
        }
      }
      return next
    })
  }, [key])

  // 跨标签 storage 同步(同源不同 tab 改 key 时触发)
  useEffect(() => {
    const win = (globalThis as { window?: Window }).window
    if (!win) return
    const onStorage = (e: StorageEvent) => {
      if (e.key !== key || e.newValue == null) return
      try {
        setValueState(JSON.parse(e.newValue) as T)
      } catch {
        // ignore corrupted payload
      }
    }
    win.addEventListener('storage', onStorage)
    return () => win.removeEventListener('storage', onStorage)
  }, [key])

  return [value, setValue]
}

function readStorage<T>(key: string, initial: T): T {
  // 用 globalThis.localStorage(SSR 安全 + vi.stubGlobal 友好);window 在 node 环境不存在。
  const ls = (globalThis as { localStorage?: Storage }).localStorage
  if (!ls) return initial
  try {
    const raw = ls.getItem(key)
    if (raw == null) return initial
    return JSON.parse(raw) as T
  } catch {
    return initial
  }
}