// 底部信息栏:左版权,右版本号。规格见 design-spec.md §2.5/§3.5。
// 样式:tailwind 原子类(原 shell.css 已删除)。
import { useT } from '../i18n'

export const APP_VERSION = 'v0.1.0'

export function BottomBar() {
  const t = useT()
  return (
    <footer className="flex h-8 flex-none items-center justify-between border-t border-[var(--shell-bottom-border)] bg-[var(--shell-bottom-bg)] px-4 text-xs text-[var(--shell-bottom-text)]">
      <span>{t.common.footer}</span>
      <span>
        {t.shell.version} {APP_VERSION}
      </span>
    </footer>
  )
}
