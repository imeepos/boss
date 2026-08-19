// Input: shadcn-style input, replaces .h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]
import { type InputHTMLAttributes, forwardRef } from 'react'
import { cn } from '../../lib/cn'

export const Input = forwardRef<HTMLInputElement, InputHTMLAttributes<HTMLInputElement>>(
  ({ className, type, ...props }, ref) => (
    <input
      type={type}
      ref={ref}
      className={cn(
        'flex h-8 w-full rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 py-1 text-xs text-[var(--shell-input-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--shell-input-border-focus)] focus:shadow-[var(--shell-input-focus-shadow)] disabled:cursor-not-allowed disabled:bg-[var(--shell-input-disabled-bg)]',
        className,
      )}
      {...props}
    />
  ),
)
Input.displayName = 'Input'