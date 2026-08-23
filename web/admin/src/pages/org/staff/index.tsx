// 组织架构与人员页:左树(企业→部门→岗位,成员计数) + 右成员列表;
// 部门/岗位增删改(DELETE 占用拒 40900),成员添加/编辑(复用账号表单级联赋岗)/启停。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { useConfirm } from '../../../components/ConfirmDialog'
import { PageHead } from '../../../components/business/page-head'
import { EmptyState } from '../../../components/business/feedback'
import { buildOrgTree, filterTree, membersOf, type DeptNode, type PostNode, type Selection } from './tree'
import { OrgTree } from './OrgTree'
import { MemberPanel } from './MemberPanel'
import { DeptFormDrawer, emptyDeptForm, type DeptFormValues } from '../department/DeptForm'
import { PostFormDrawer, emptyPostForm, type PostFormValues } from '../post/PostForm'
import { AccountFormDrawer } from '../../base/account/AccountForm'
import { buildAccountPayload, validateAccount, type AccountFormValues } from '../../base/account/form'
import type { AccountRow } from '../../base/account/list'

interface EntityRow { id: number; code: string; name: string }
interface DeptRow { id: number; legalEntityId: number; name: string }

export default function StaffOrgPage() {
  const t = useT()
  const confirmDialog = useConfirm()
  const [entities, setEntities] = useState<EntityRow[]>([])
  const [depts, setDepts] = useState<DeptRow[]>([])
  const [posts, setPosts] = useState<PostNode[]>([])
  const [accounts, setAccounts] = useState<AccountRow[]>([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [treeKw, setTreeKw] = useState('')
  const [selection, setSelection] = useState<Selection | null>(null)
  const [deptForm, setDeptForm] = useState<DeptFormValues | null>(null)
  const [postForm, setPostForm] = useState<PostFormValues | null>(null)
  const [memberForm, setMemberForm] = useState<AccountFormValues | null>(null)
  const [formError, setFormError] = useState('')

  const load = () => {
    setError('')
    Promise.all([
      apiFetch<EntityRow[]>('/legal-entities'),
      apiFetch<DeptRow[]>('/departments'),
      apiFetch<PostNode[]>('/posts'),
      apiFetch<AccountRow[]>('/accounts'),
    ])
      .then(([e, d, p, a]) => {
        setEntities(e ?? [])
        setDepts(d ?? [])
        setPosts(p ?? [])
        setAccounts(a ?? [])
      })
      .catch(() => setError(t.pages.staff.loadFail))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const tree = useMemo(
    () => filterTree(buildOrgTree(entities, depts, posts, accounts), treeKw),
    [entities, depts, posts, accounts, treeKw],
  )
  const members = useMemo(() => membersOf(accounts, selection), [accounts, selection])

  const title = useMemo(() => {
    if (!selection) return ''
    if (selection.kind === 'entity') {
      const e = entities.find((x) => x.id === selection.id)
      return e ? `${e.code} ${e.name}` : ''
    }
    if (selection.kind === 'dept') return depts.find((x) => x.id === selection.id)?.name ?? ''
    return posts.find((x) => x.id === selection.id)?.name ?? ''
  }, [selection, entities, depts, posts])

  const guard = async (fn: () => Promise<void>) => {
    if (busy) return
    setBusy(true)
    setFormError('')
    try { await fn(); load() } catch (e) {
      setFormError(e instanceof Error ? e.message : t.pages.staff.saveFail)
    } finally { setBusy(false) }
  }

  const submitDept = () => guard(async () => {
    if (!deptForm) return
    const body = { legalEntityId: deptForm.legalEntityId, name: deptForm.name.trim() }
    if (deptForm.id) await apiFetch(`/departments/${deptForm.id}`, { method: 'PUT', body })
    else await apiFetch('/departments', { method: 'POST', body })
    setDeptForm(null)
  })

  const submitPost = () => guard(async () => {
    if (!postForm) return
    const body = { deptId: postForm.deptId, code: postForm.code.trim(), name: postForm.name.trim(), roles: postForm.roles }
    if (postForm.id) await apiFetch(`/posts/${postForm.id}`, { method: 'PUT', body })
    else await apiFetch('/posts', { method: 'POST', body })
    setPostForm(null)
  })

  const submitMember = () => guard(async () => {
    if (!memberForm) return
    if (validateAccount(memberForm, Boolean(memberForm.id)).length) return
    const body = buildAccountPayload(memberForm, Boolean(memberForm.id))
    if (memberForm.id) await apiFetch(`/accounts/${memberForm.id}`, { method: 'PUT', body })
    else await apiFetch('/accounts', { method: 'POST', body })
    setMemberForm(null)
  })

  const toggleMember = (r: AccountRow) => guard(async () => {
    const f = rowToForm(r)
    await apiFetch(`/accounts/${r.id}`, {
      method: 'PUT',
      body: { ...buildAccountPayload(f, true), status: r.status === 1 ? 0 : 1 },
    })
  })

  const delDept = async (d: DeptNode) => {
    if (!(await confirmDialog(t.pages.staff.delDeptConfirm.replace('{name}', d.name), { danger: true }))) return
    await guard(async () => { await apiFetch(`/departments/${d.id}`, { method: 'DELETE' }) })
  }

  const delPost = async (p: PostNode) => {
    if (!(await confirmDialog(t.pages.staff.delPostConfirm.replace('{name}', p.name), { danger: true }))) return
    await guard(async () => { await apiFetch(`/posts/${p.id}`, { method: 'DELETE' }) })
  }

  const addMember = () => {
    const preset = selection?.kind === 'entity'
      ? { legalEntityId: selection.id }
      : selection?.kind === 'dept'
        ? { legalEntityId: depts.find((d) => d.id === selection.id)?.legalEntityId ?? 0, deptId: selection.id }
        : {}
    setMemberForm({ ...emptyMemberForm(), ...preset })
  }

  const treeBox = 'w-72 flex-none overflow-y-auto rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)] p-2'
  return (
    <div>
      <PageHead title={t.pages.staff.title} desc={t.pages.staff.desc} />
      {error && (
        <div className="mb-4 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">
          {error} <button className="ml-2 cursor-pointer border-none bg-none text-[var(--color-danger)] underline" onClick={load}>{t.pages.audit.refresh}</button>
        </div>
      )}
      {formError && !deptForm && !postForm && !memberForm && (
        <div className="mb-4 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{formError}</div>
      )}
      <div className="flex items-stretch gap-4">
        <div className={treeBox}>
          <input
            className="mb-2 h-8 w-full rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]"
            placeholder={t.pages.department.searchPlaceholder}
            value={treeKw}
            onChange={(e) => setTreeKw(e.target.value)}
          />
          <OrgTree
            tree={tree}
            selection={selection}
            onSelect={setSelection}
            onAddDept={(entityId) => setDeptForm({ ...emptyDeptForm(), legalEntityId: entityId })}
            onEditDept={(d) => setDeptForm({ id: d.id, legalEntityId: d.legalEntityId, name: d.name })}
            onDelDept={delDept}
            onAddPost={(deptId) => setPostForm({ ...emptyPostForm(), deptId })}
            onEditPost={(p) => setPostForm({ id: p.id, deptId: p.deptId, code: p.code, name: p.name, roles: p.roles ?? [] })}
            onDelPost={delPost}
          />
        </div>
        {selection
          ? <MemberPanel title={title} members={members} busy={busy} onAdd={addMember} onEdit={(r) => setMemberForm(rowToForm(r))} onToggle={toggleMember} />
          : (
            <div className="flex min-w-0 flex-1 items-center justify-center rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)] py-20">
              <EmptyState text={t.pages.staff.selectTip} />
            </div>
          )}
      </div>

      <DeptFormDrawer
        open={deptForm !== null}
        values={deptForm ?? emptyDeptForm()}
        onChange={setDeptForm}
        onClose={() => setDeptForm(null)}
        onSubmit={submitDept}
        busy={busy}
        submitError={deptForm ? formError : ''}
      />
      <PostFormDrawer
        open={postForm !== null}
        values={postForm ?? emptyPostForm()}
        onChange={setPostForm}
        onClose={() => setPostForm(null)}
        onSubmit={submitPost}
        busy={busy}
        submitError={postForm ? formError : ''}
      />
      <AccountFormDrawer
        open={memberForm !== null}
        values={memberForm ?? emptyMemberForm()}
        onChange={setMemberForm}
        onClose={() => setMemberForm(null)}
        onSubmit={submitMember}
        busy={busy}
        submitError={memberForm ? formError : ''}
      />
    </div>
  )
}

function emptyMemberForm(): AccountFormValues {
  return { username: '', password: '', realName: '', phone: '', roleCode: '', legalEntityId: 0, deptId: 0, postId: 0, regionScope: '' }
}

function rowToForm(r: AccountRow): AccountFormValues {
  return {
    id: r.id,
    username: r.username,
    password: '',
    realName: r.realName,
    phone: r.phone ?? '',
    roleCode: r.roleCode ?? '',
    legalEntityId: r.legalEntityId ?? 0,
    deptId: r.deptId ?? 0,
    postId: r.postId ?? 0,
    regionScope: r.regionScope ?? '',
  }
}
