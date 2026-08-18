// 悬浮按钮(FAB):内容区右下角固定,快捷新建入口(A1 批次接具体动作)。规格见 design-spec.md §2.4/§3.4。
import { useT } from '../i18n'
import { PlusIcon } from './icons'

export function Fab() {
  const t = useT()
  return (
    <button
      className="shell-fab"
      title={t.shell.quickCreate}
      aria-label={t.shell.quickCreate}
      onClick={() => {
        /* A1 批次接"新建"动作分发 */
      }}
    >
      <PlusIcon />
    </button>
  )
}
