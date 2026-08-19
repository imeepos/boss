// 悬浮按钮(FAB):内容区右下角固定,快捷新建入口(A1 批次接具体动作)。规格见 design-spec.md §2.4/§3.4。
// 样式:tailwind 原子类(原 shell.css 已删除)。
import { useT } from '../i18n'
import { PlusIcon } from './icons'

export function Fab() {
  const t = useT()
  return (
    <button
      className="fixed right-8 bottom-16 z-20 grid h-12 w-12 cursor-pointer animate-in zoom-in-75 duration-200 place-items-center rounded-[14px] border-0 bg-[var(--shell-fab-bg)] text-[var(--shell-fab-icon)] shadow-[var(--shell-fab-shadow)] hover:bg-[var(--shell-fab-bg-hover)]"
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
