import { useT } from '../i18n'
import { Spinner } from './business/feedback'

export function Loading() {
  const t = useT()
  return (
    <div className="flex items-center justify-center gap-2 p-12 text-[13px] text-[var(--shell-group-title)]">
      <Spinner />
      <span>{t.common.loading}</span>
    </div>
  )
}
