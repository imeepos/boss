import { useT } from '../i18n'

export function Loading() {
  const t = useT()
  return <div className="p-12 text-center text-[var(--shell-group-title)]">{t.common.loading}</div>
}
