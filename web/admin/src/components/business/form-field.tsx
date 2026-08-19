// FormField: form field wrapper with label + hint + error, replaces .flex flex-col gap-1.5
import type { ReactNode } from 'react'
import { Label } from '../ui/label'

export function FormField({
  label, required, hint, error, children,
}: {
  label: string
  required?: boolean
  hint?: string
  error?: string
  children: ReactNode
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label>
        {label}
        {required && <span className="ml-0.5 text-[var(--color-danger)]">*</span>}
      </Label>
      {children}
      {hint && <span className="text-[11px] text-[var(--shell-group-title)]">{hint}</span>}
      {error && <span className="text-[11px] text-[var(--color-danger)]">{error}</span>}
    </div>
  )
}