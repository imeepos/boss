// AttachmentManager 内部共享的 tailwind 类名常量:卡片壳/输入框/危险文字按钮。
// 双主题令牌均来自 tokens.css,与页面其他组件保持一致。
export const CARD = 'border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]'
export const INPUT = 'h-8 w-44 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] placeholder:text-[var(--shell-crumb-text)] focus:border-[var(--shell-input-border-focus)] focus:outline-none'
export const DANGER_BTN = 'h-7 cursor-pointer rounded-sm border-0 bg-transparent px-2 text-xs text-[var(--color-danger)] hover:underline disabled:cursor-not-allowed disabled:opacity-50'
