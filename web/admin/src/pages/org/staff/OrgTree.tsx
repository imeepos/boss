// 左侧组织树:企业→部门→岗位,节点带成员数与操作按钮(建部门/建岗位/编辑/删除)。
import { useT } from '../../../i18n'
import { EmptyState } from '../../../components/business/feedback'
import type { DeptNode, EntityNode, PostNode, Selection } from './tree'

interface OrgTreeProps {
  tree: EntityNode[]
  selection: Selection | null
  onSelect: (sel: Selection) => void
  onAddDept: (entityId: number) => void
  onEditDept: (d: DeptNode) => void
  onDelDept: (d: DeptNode) => void
  onAddPost: (deptId: number) => void
  onEditPost: (p: PostNode) => void
  onDelPost: (p: PostNode) => void
}

const ACT_BTN = 'ml-1 cursor-pointer border-none bg-none px-1 text-[11px] text-[var(--shell-menu-icon)] hover:text-[var(--shell-heading)]'

export function OrgTree(p: OrgTreeProps) {
  const t = useT()
  return (
    <div className="flex flex-col gap-1 text-[13px] text-[var(--shell-content-text)]">
      {p.tree.map((e) => (
        <EntityRow key={e.id} e={e} {...p} />
      ))}
      {!p.tree.length && <div className="px-2 py-4"><EmptyState text={t.pages.staff.empty} /></div>}
    </div>
  )
}

function EntityRow({ e, ...p }: OrgTreeProps & { e: EntityNode }) {
  const t = useT()
  const active = p.selection?.kind === 'entity' && p.selection.id === e.id
  return (
    <div>
      <div className={`group flex items-center gap-1 rounded-sm px-2 py-1.5 ${active ? 'bg-[var(--shell-menu-active-bg)] font-semibold text-[var(--shell-menu-active-text)]' : 'hover:bg-[var(--shell-menu-hover-bg)]'}`}>
        <button
          className="flex-1 cursor-pointer truncate border-none bg-none p-0 text-left text-[13px] font-semibold text-[var(--shell-heading)]"
          onClick={() => p.onSelect({ kind: 'entity', id: e.id })}
        >
          {e.code} {e.name}
          <span className="ml-2 text-[11px] font-normal text-[var(--shell-group-title)]">
            {e.memberCount} {t.pages.staff.memberUnit} · {e.depts.length} {t.pages.staff.deptUnit}
          </span>
        </button>
        <button className={ACT_BTN} onClick={() => p.onAddDept(e.id)}>{t.pages.staff.addDept}</button>
      </div>
      <div className="ml-3 flex flex-col gap-0.5 border-l border-[var(--shell-side-border)] pl-2">
        {e.depts.map((d) => (
          <DeptRow key={d.id} d={d} {...p} />
        ))}
      </div>
    </div>
  )
}

function DeptRow({ d, ...p }: OrgTreeProps & { d: DeptNode }) {
  const t = useT()
  const active = p.selection?.kind === 'dept' && p.selection.id === d.id
  return (
    <div>
      <div className={`group flex items-center gap-1 rounded-sm px-2 py-1 ${active ? 'bg-[var(--shell-menu-active-bg)] font-semibold text-[var(--shell-menu-active-text)]' : 'hover:bg-[var(--shell-menu-hover-bg)]'}`}>
        <button
          className="flex-1 cursor-pointer truncate border-none bg-none p-0 text-left text-[13px] text-[var(--shell-content-text)]"
          onClick={() => p.onSelect({ kind: 'dept', id: d.id })}
        >
          {d.name}
          <span className="ml-2 text-[11px] font-normal text-[var(--shell-group-title)]">
            {d.memberCount} {t.pages.staff.memberUnit} · {d.posts.length} {t.pages.staff.postUnit}
          </span>
        </button>
        <span className="hidden items-center group-hover:flex">
          <button className={ACT_BTN} onClick={() => p.onAddPost(d.id)}>{t.pages.staff.addPost}</button>
          <button className={ACT_BTN} onClick={() => p.onEditDept(d)}>{t.pages.staff.edit}</button>
          <button className={ACT_BTN} onClick={() => p.onDelDept(d)}>{t.pages.staff.del}</button>
        </span>
      </div>
      <div className="ml-3 flex flex-col gap-0.5 border-l border-[var(--shell-side-border)] pl-2">
        {d.posts.map((post) => (
          <PostRow key={post.id} post={post} {...p} />
        ))}
      </div>
    </div>
  )
}

function PostRow({ post, ...p }: OrgTreeProps & { post: PostNode }) {
  const t = useT()
  const active = p.selection?.kind === 'post' && p.selection.id === post.id
  return (
    <div className={`group flex items-center gap-1 rounded-sm px-2 py-1 ${active ? 'bg-[var(--shell-menu-active-bg)] text-[var(--shell-menu-active-text)]' : 'hover:bg-[var(--shell-menu-hover-bg)]'}`}>
      <button
        className="flex-1 cursor-pointer truncate border-none bg-none p-0 text-left text-[12px] text-[var(--shell-menu-text)]"
        onClick={() => p.onSelect({ kind: 'post', id: post.id })}
      >
        {post.name}
        <span className="ml-2 text-[11px] text-[var(--shell-group-title)]">
          {post.memberCount} {t.pages.staff.memberUnit}
        </span>
      </button>
      <span className="hidden items-center group-hover:flex">
        <button className={ACT_BTN} onClick={() => p.onEditPost(post)}>{t.pages.staff.edit}</button>
        <button className={ACT_BTN} onClick={() => p.onDelPost(post)}>{t.pages.staff.del}</button>
      </span>
    </div>
  )
}
