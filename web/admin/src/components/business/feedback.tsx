// 统一反馈态组件:加载中(spinner+文案)与空数据(图标+文案),全站唯一实现。
// 颜色一律走 shell-* 令牌;文案默认取 i18n common.loading,空数据文案由调用方传入。
import { useEffect, useRef, useState } from 'react'
import { useT } from '../../i18n'

/** CSS 圆环 spinner,颜色随主题令牌。 */
export function Spinner({ size = 16 }: { size?: number }) {
  return (
    <span
      aria-hidden
      style={{ width: size, height: size, borderWidth: Math.max(2, Math.round(size / 8)) }}
      className="inline-block animate-spin rounded-full border-solid border-[var(--shell-side-border)] border-t-[var(--shell-heading)] align-[-2px]"
    />
  )
}

/** 块级加载态:spinner + 文案居中。 */
export function LoadingState({ text }: { text?: string }) {
  const t = useT()
  return (
    <div className="flex items-center justify-center gap-2 py-8 text-[13px] text-[var(--shell-group-title)]">
      <Spinner />
      <span>{text ?? t.common.loading}</span>
    </div>
  )
}

/** 空数据图标:描边 SVG(24 viewBox / stroke 1.8 / round / currentColor)。 */
function EmptyIcon() {
  return (
    <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden className="shrink-0">
      <path d="M22 12h-6l-2 3h-4l-2-3H2" />
      <path d="M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z" />
    </svg>
  )
}

/** 块级空数据态:图标 + 文案居中。 */
export function EmptyState({ text }: { text: string }) {
  return (
    <div className="flex flex-col items-center justify-center gap-2 py-8 text-[13px] text-[var(--shell-group-title)]">
      <EmptyIcon />
      <span>{text}</span>
    </div>
  )
}

/** 表格内统一状态行:loading 时渲染加载态,否则渲染空数据态。 */
export function TableStateRow({ colSpan, loading, text }: { colSpan: number; loading?: boolean; text: string }) {
  const t = useT()
  return (
    <tr>
      <td colSpan={colSpan} className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
        {loading ? <LoadingState text={t.common.loading} /> : <EmptyState text={text} />}
      </td>
    </tr>
  )
}

/** 复制到剪贴板;非安全上下文(http)降级 execCommand。导出仅供单测。 */
export function copyText(text: string): boolean {
  if (navigator.clipboard?.writeText) {
    void navigator.clipboard.writeText(text)
    return true
  }
  const ta = document.createElement('textarea')
  ta.value = text
  ta.style.position = 'fixed'
  ta.style.opacity = '0'
  document.body.appendChild(ta)
  ta.select()
  const ok = document.execCommand('copy')
  ta.remove()
  return ok
}

/** 一键复制按钮:复制传入文本,成功后短暂切换为"已复制"。 */
export function CopyButton({ text, className = '' }: { text: string; className?: string }) {
  const t = useT()
  const [copied, setCopied] = useState(false)
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)
  useEffect(() => () => { if (timer.current) clearTimeout(timer.current) }, [])
  const onClick = () => {
    if (copyText(text)) {
      setCopied(true)
      if (timer.current) clearTimeout(timer.current)
      timer.current = setTimeout(() => setCopied(false), 1500)
    }
  }
  return (
    <button
      type="button"
      onClick={onClick}
      className={`h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)] ${className}`}
    >{copied ? t.common.copied : t.common.copy}</button>
  )
}
