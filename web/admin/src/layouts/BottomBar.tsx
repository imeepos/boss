// 底部信息栏:左版权,右版本号。规格见 design-spec.md §2.5/§3.5。
import { useT } from '../i18n'

export const APP_VERSION = 'v0.1.0'

export function BottomBar() {
  const t = useT()
  return (
    <footer className="shell-bottom">
      <span>{t.common.footer}</span>
      <span>
        {t.shell.version} {APP_VERSION}
      </span>
    </footer>
  )
}
