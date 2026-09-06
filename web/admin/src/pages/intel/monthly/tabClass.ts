// 页签状态类。active/idle 两组 bg-*/text-* 工具类必须互斥:同一元素重复定义时,
// 生效方由 CSS 产物顺序决定而非书写顺序,曾致激活页签亮色白字白底、暗色深底深字。
const TAB_BASE = 'h-8 cursor-pointer rounded-t-sm border border-b-0 px-4 text-[13px] transition-colors'
const TAB_IDLE = 'border-[var(--shell-side-border)] bg-[var(--shell-input-bg)] text-[var(--shell-content-text)] hover:text-[var(--shell-heading)]'
const TAB_ACTIVE = 'border-[var(--shell-fab-bg)] bg-[var(--shell-fab-bg)] font-medium text-[var(--shell-fab-icon)] hover:text-[var(--shell-fab-icon)]'

export function monthlyTabClass(active: boolean): string {
  return TAB_BASE + ' ' + (active ? TAB_ACTIVE : TAB_IDLE)
}
