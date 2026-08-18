import { useT } from '../i18n'

export function Loading() {
  const t = useT()
  return <div style={{ padding: 48, textAlign: 'center', color: '#888' }}>{t.common.loading}</div>
}