// 岗位管理页:列名以 fields.md 1.4 为准(岗位代码/岗位/所属部门/绑定角色);原型数据范围列后端无字段,按契约裁剪。
// 契约: GET /posts(org.yaml;deptId 过滤,本次全量;roles 为岗位绑定角色码聚合)。
import { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useConfirm } from '../../../components/ConfirmDialog'
import { useT } from '../../../i18n'
import { DetailDrawer, PageHead, pagerTexts } from '../shared'
import { filterPosts, pageSlice, type PostRow } from './filter'
import { PostFormDrawer, emptyPostForm, rowToPostForm, type PostFormValues } from './PostForm'
import { BatchImportEntry } from '../../base/importer/BatchImportEntry'
import { Pagination } from '../../../components/Pagination'
import { TableStateRow } from '../../../components/business'
import { ErrorBanner } from '../../../components/business/page-head'
import { Card } from '../../../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { Input } from '../../../components/ui/input'

export default function PostPage() {
  const t = useT()
  const [rows, setRows] = useState<PostRow[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [detail, setDetail] = useState<PostRow | null>(null)
  const [form, setForm] = useState<PostFormValues | null>(null)
  const [formError, setFormError] = useState('')
  const [busy, setBusy] = useState(false)
  const [roleNames, setRoleNames] = useState<Record<string, string>>({})

  const load = () => {
    setError('')
    apiFetch<PostRow[]>('/posts')
      .then((d) => setRows(d ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : t.pages.post.loadFail))
    // 角色码→人可读名称映射(PostForm 同源 /roles);失败降级显示原码,不硬造。
    apiFetch<{ code: string; name: string }[]>('/roles')
      .then((d) => setRoleNames(Object.fromEntries((d ?? []).map((r) => [r.code, r.name]))))
      .catch(() => console.warn('[org-post] /roles 加载失败,绑定角色列降级显示角色码'))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const submit = async () => {
    if (!form || busy) return
    setBusy(true)
    setFormError('')
    try {
      const body = { deptId: form.deptId, code: form.code.trim(), name: form.name.trim(), roles: form.roles }
      if (form.id) await apiFetch(`/posts/${form.id}`, { method: 'PUT', body })
      else await apiFetch('/posts', { method: 'POST', body })
      toast.success(t.pages.staff.saved)
      setForm(null)
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : t.pages.post.saveFail)
    } finally {
      setBusy(false)
    }
  }

  const confirmDialog = useConfirm()

  // 补齐与后端 DELETE /posts/:postId 对齐的行内入口(仍有账号挂岗时后端 40900 拒)。
  const del = async (r: PostRow) => {
    if (!(await confirmDialog(t.pages.staff.delPostConfirm.replace('{name}', r.name), { danger: true }))) return
    try {
      await apiFetch(`/posts/${r.id}`, { method: 'DELETE' })
      toast.success(t.pages.staff.delOk)
      load()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t.pages.post.saveFail)
    }
  }

  const filtered = useMemo(() => filterPosts(rows, keyword), [rows, keyword])
  const slice = pageSlice(filtered, page, pageSize)
  const roleLabel = (codes: string[] | undefined) =>
    (codes ?? []).map((c) => roleNames[c] ?? c).join(', ')
  const act = 'cursor-pointer border-none bg-none p-0 text-[13px] text-[var(--shell-content-text)] underline-offset-2 hover:text-[var(--shell-heading)] hover:underline'
  const actDanger = 'cursor-pointer border-none bg-none p-0 text-[13px] text-[var(--color-danger)] underline-offset-2 hover:underline'

  return (
    <div>
      <PageHead title={t.pages.post.title} desc={t.pages.post.desc} />
      <Card>
        <div className="flex flex-wrap items-center gap-2 p-4">
          <Input className="w-56" placeholder={t.pages.post.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <BatchImportEntry kind="post" onImported={load} />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={load}>{t.pages.audit.refresh}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => setForm(emptyPostForm())}>
            {t.pages.post.create}
          </button>
        </div>
        {error && <ErrorBanner message={error} />}
        <Table>
          <TableHeader>
            <TableRow>
              {t.pages.post.columns.map((c) => (
                <TableHead key={c}>{c}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {slice.map((r) => (
              <TableRow key={r.id}>
                <TableCell>{r.code}</TableCell>
                <TableCell>{r.name}</TableCell>
                <TableCell>{r.deptName}</TableCell>
                <TableCell>{roleLabel(r.roles) || '—'}</TableCell>
                <TableCell>
                  <span className="inline-flex items-center gap-1.5">
                    <button className={act} onClick={() => setDetail(r)}>{t.pages.post.detail}</button>
                    <span className="text-[var(--shell-side-border)]">|</span>
                    <button className={act} onClick={() => setForm(rowToPostForm(r))}>{t.pages.account.edit}</button>
                    <span className="text-[var(--shell-side-border)]">|</span>
                    <button className={actDanger} onClick={() => del(r)}>{t.pages.staff.del}</button>
                  </span>
                </TableCell>
              </TableRow>
            ))}
            {!slice.length && <TableStateRow colSpan={5} loading={busy} text={t.pages.post.empty} />}
          </TableBody>
        </Table>
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(t.pages.post)} />
        </div>
      </Card>
      {detail && (
        <DetailDrawer
          title={t.pages.post.detail}
          closeText={t.pages.company.cancel}
          onClose={() => setDetail(null)}
          items={[
            { k: t.pages.post.columns[0], v: detail.code },
            { k: t.pages.post.columns[1], v: detail.name },
            { k: t.pages.post.columns[2], v: detail.deptName },
            { k: t.pages.post.columns[3], v: roleLabel(detail.roles) },
            { k: 'ID', v: String(detail.id) },
          ]}
        />
      )}
      <PostFormDrawer
        open={form !== null}
        values={form ?? emptyPostForm()}
        onChange={setForm}
        onClose={() => setForm(null)}
        onSubmit={submit}
        busy={busy}
        submitError={formError}
      />
    </div>
  )
}
