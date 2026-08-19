// Badge: shadcn-style badge, replaces .org-tag-on / .org-tag-off
import { type HTMLAttributes, forwardRef } from 'react'
import { cn } from '../../lib/cn'

type Variant = 'default' | 'success' | 'danger' | 'warning' | 'info'

const variantClass: Record<Variant, string> = {
  default: 'border-[var(--shell-input-border)] text-[var(--shell-group-title)] bg-transparent',
  success: 'border-[color-mix(in_srgb,var(--color-success)_35%,transparent)] text-[var(--color-success)] bg-[color-mix(in_srgb,var(--color-success)_10%,transparent)]',
  danger: 'border-[color-mix(in_srgb,var(--color-danger)_35%,transparent)] text-[var(--color-danger)] bg-[color-mix(in_srgb,var(--color-danger)_10%,transparent)]',
  warning: 'border-[color-mix(in_srgb,var(--color-brand-gold-500)_35%,transparent)] text-[var(--color-brand-gold-500)] bg-[color-mix(in_srgb,var(--color-brand-gold-500)_10%,transparent)]',
  info: 'border-[color-mix(in_srgb,var(--color-border-focus)_35%,transparent)] text-[var(--color-border-focus)] bg-[color-mix(in_srgb,var(--color-border-focus)_10%,transparent)]',
}

interface BadgeProps extends HTMLAttributes<HTMLSpanElement> {
  variant?: Variant
}

export const Badge = forwardRef<HTMLSpanElement, BadgeProps>(
  ({ className, variant = 'default', ...props }, ref) => (
    <span
      ref={ref}
      className={cn(
        'inline-block rounded-[4px] border px-2 text-xs leading-[22px]',
        variantClass[variant],
        className,
      )}
      {...props}
    />
  ),
)
Badge.displayName = 'Badge'