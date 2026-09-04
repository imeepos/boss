// SubmitButton:提交按钮微反馈展示层,状态机 idle→loading→success/failed。
// 状态与复位定时由调用方持有;文案走 i18n,颜色走主题令牌,图标为描边 SVG。
import { cn } from '../../lib/cn'

export type SubmitState = 'idle' | 'loading' | 'success' | 'failed'

const BASE = 'inline-flex h-8 cursor-pointer items-center justify-center gap-1.5 rounded-sm border-none px-4 text-[13px] transition-colors disabled:cursor-not-allowed disabled:opacity-70'

const STATE_CLS: Record<SubmitState, string> = {
  idle: 'bg-[var(--shell-fab-bg)] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]',
  loading: 'bg-[var(--shell-fab-bg)] text-[var(--shell-fab-icon)]',
  success: 'bg-[var(--color-success)] text-white',
  failed: 'bg-[var(--color-danger)] text-white',
}

/** 状态图标:描边 SVG,24 viewBox,尺寸适配 13px 文案。 */
function StateIcon({ kind }: { kind: 'check' | 'cross' }) {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
      <path d={kind === 'check' ? 'M4 12.5l5 5L20 6.5' : 'M6 6l12 12M18 6L6 18'} />
    </svg>
  )
}

export function SubmitButton({ state, labels, disabled, onClick }: {
  state: SubmitState
  labels: Record<SubmitState, string>
  disabled?: boolean
  onClick?: () => void
}) {
  return (
    <button type="button" data-submit-state={state} disabled={disabled || state === 'loading'}
      className={cn(BASE, STATE_CLS[state])} onClick={onClick}>
      {state === 'loading' && <span aria-hidden className="inline-block h-3 w-3 animate-spin rounded-full border-[1.5px] border-current border-t-transparent" />}
      {state === 'success' && <StateIcon kind="check" />}
      {state === 'failed' && <StateIcon kind="cross" />}
      <span>{labels[state]}</span>
    </button>
  )
}