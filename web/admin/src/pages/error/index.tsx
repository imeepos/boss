// 403/404 错误页,文本走 i18n。
import { useT } from '../../i18n'

export function ForbiddenPage() {
  const t = useT()
  return (
    <div className="p-16 text-center">
      <h1>403</h1>
      <p>{t.pages.error.forbidden}</p>
    </div>
  )
}

export function NotFoundPage() {
  const t = useT()
  return (
    <div className="p-16 text-center">
      <h1>404</h1>
      <p>{t.pages.error.notFound}</p>
    </div>
  )
}