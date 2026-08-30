// 版本角标(版本自证消费端):轮询服务端 /healthz 的 commit,与构建注入的
// VITE_BUILD_COMMIT 比对,漂移即右下角提示"系统已更新,点击刷新"。
// 治"发版后用户端停留在旧 bundle,改了看不到还以为是 bug"(累犯坑)。
// 仅生产构建轮询(dev 下 import.meta.env.DEV 短路,避免本地必漂移的常驻误报)。
// seen 记忆:server-only 部署时 web 内容不变,刷新后 commit 仍不匹配,
// 无记忆角标会永久驻留——每次部署只提示一次,点过即记(b2a1d950 实测推演)。
import { useEffect, useRef, useState } from 'react'
import { VERSION_SEEN_KEY, healthzUrl, shouldShowVersionBadge } from '../lib/version'

const POLL_MS = 60_000
const FETCH_TIMEOUT_MS = 5_000

export function VersionBadge() {
  const [show, setShow] = useState(false)
  const lastServerCommit = useRef('')

  useEffect(() => {
    if (import.meta.env.DEV) return
    const build = import.meta.env.VITE_BUILD_COMMIT ?? ''
    let alive = true
    const check = async () => {
      const url = healthzUrl()
      if (!url) return
      try {
        const ctl = new AbortController()
        const timer = setTimeout(() => ctl.abort(), FETCH_TIMEOUT_MS)
        const res = await fetch(url, { signal: ctl.signal })
        clearTimeout(timer)
        if (!res.ok) return
        const body = (await res.json()) as { commit?: string }
        const server = body.commit ?? ''
        if (!alive) return
        lastServerCommit.current = server
        const seen = localStorage.getItem(VERSION_SEEN_KEY) ?? ''
        if (shouldShowVersionBadge(build, server, seen)) setShow(true)
      } catch {
        // 网络抖动/服务未起:静默,不打扰操作;下次轮询再试。
      }
    }
    void check()
    const id = setInterval(check, POLL_MS)
    window.addEventListener('focus', check)
    return () => {
      alive = false
      clearInterval(id)
      window.removeEventListener('focus', check)
    }
  }, [])

  if (!show) return null
  return (
    <button
      type="button"
      onClick={() => {
        // 刷新前记录"已提示到当前服务端版本";未知 commit 记空串,下次会再提示(安全侧)。
        localStorage.setItem(VERSION_SEEN_KEY, lastServerCommit.current)
        window.location.reload()
      }}
      className="fixed bottom-6 right-6 z-fab rounded-full border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] px-4 py-2 text-sm text-[var(--shell-content-text)] shadow-[var(--shell-card-shadow)] hover:opacity-90"
    >
      系统已更新，点击刷新
    </button>
  )
}
