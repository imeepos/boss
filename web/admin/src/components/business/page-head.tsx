// PageHead: page title + description, replaces .mb-4
export function PageHead({ title, desc }: { title: string; desc: string }) {
  return (
    <div className="mb-4">
      <h2 className="m-0 text-xl font-bold text-[var(--shell-heading)]">{title}</h2>
      {desc && <p className="mt-1 text-xs text-[var(--shell-crumb-text)]">{desc}</p>}
    </div>
  )
}

/** pagerTexts 装配 Pagination 所需文案(各 namespace 同名 key),可直接展开传给 Pagination。 */
export function pagerTexts(ns: {
  rangeText: string; prev: string; next: string; perPage: string; jumpText: string; pageUnit: string
}) {
  return {
    rangeText: ns.rangeText, prevText: ns.prev, nextText: ns.next,
    perPageText: ns.perPage, jumpText: ns.jumpText, pageUnitText: ns.pageUnit,
  }
}

// EmptyState 统一由 feedback.tsx 提供(图标+文案),此处 re-export 兼容既有 import 路径。
export { EmptyState } from './feedback'

/** ErrorBanner: error message block, replaces .mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)] */
export function ErrorBanner({ message, className = '' }: { message: string; className?: string }) {
  return (
    <div className={`mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-xs text-[var(--color-danger)] ${className}`}>
      {message}
    </div>
  )
}

/** ActionLinks: action buttons row with separator, replaces .inline-flex items-center */
export function ActionLinks({ children }: { children: React.ReactNode }) {
  return <span className="inline-flex items-center gap-0">{children}</span>
}

export function ActionLink({
  onClick, label, testId,
}: { onClick: () => void; label: string; testId?: string }) {
  return (
    <button
      data-testid={testId}
      className="px-1 text-xs text-[var(--color-text-link)] bg-none border-none cursor-pointer hover:text-[var(--color-brand-gold-600)] hover:underline"
      onClick={onClick}
    >
      {label}
    </button>
  )
}

export function ActionSep() {
  return <span className="text-[var(--shell-side-border)]">|</span>
}

/** SearchBar: toolbar with search input, spacer, and action buttons */
export function SearchBar({
  keyword, onKeywordChange, onResetKeyword, children,
}: {
  keyword: string
  onKeywordChange: (v: string) => void
  onResetKeyword?: () => void
  children?: React.ReactNode
}) {
  return (
    <div className="flex items-center gap-2 p-4 flex-wrap">
      <input
        className="flex h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 py-1 text-xs text-[var(--shell-input-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--shell-input-border-focus)]"
        placeholder="搜索..."
        value={keyword}
        onChange={(e) => {
          onKeywordChange(e.target.value)
          onResetKeyword?.()
        }}
      />
      <div className="flex-1" />
      {children}
    </div>
  )
}

/** Toolbar button: replaces .h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)] */
export function ToolbarButton({
  onClick, disabled, children, primary,
}: {
  onClick?: () => void
  disabled?: boolean
  children: React.ReactNode
  primary?: boolean
}) {
  const base = 'h-8 px-4 text-xs rounded-sm cursor-pointer border'
  if (primary) {
    return (
      <button
        className={`${base} text-[var(--shell-fab-icon)] bg-[var(--shell-fab-bg)] border-none hover:bg-[var(--shell-fab-bg-hover)] disabled:opacity-50 disabled:cursor-not-allowed`}
        onClick={onClick}
        disabled={disabled}
      >
        {children}
      </button>
    )
  }
  return (
    <button
      className={`${base} text-[var(--shell-content-text)] bg-[var(--shell-input-bg)] border-[var(--shell-input-border)] hover:border-[var(--shell-input-border-hover)] hover:text-[var(--shell-heading)] disabled:opacity-50 disabled:cursor-not-allowed`}
      onClick={onClick}
      disabled={disabled}
    >
      {children}
    </button>
  )
}