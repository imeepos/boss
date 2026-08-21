// 顶栏下拉浮层统一样式,对齐 antd v5 Menu/Popover 规范:
// 卡片 8px 圆角 + boxShadowSecondary;菜单项 4px 圆角 / 32px 行高 / 12px 内边距。
// 令牌见 tokens.css --shell-popover-*,双主题随 [data-theme] 切换。
export const POPOVER = 'absolute right-0 top-[calc(100%+8px)] z-50 min-w-[180px] rounded-lg border border-[var(--shell-popover-border)] bg-[var(--shell-popover-bg)] p-1 shadow-[var(--shell-popover-shadow)]'
export const POPOVER_ITEM = 'flex h-8 w-full cursor-pointer items-center justify-between gap-2 rounded border-0 bg-none px-3 text-left text-[13px] text-[var(--popover-foreground)] hover:bg-[var(--shell-menu-hover-bg)]'
export const POPOVER_DIVIDER = 'my-1 border-t border-[var(--shell-popover-border)]'
