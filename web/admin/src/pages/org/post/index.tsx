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

  const load = () => {
    setError('')
    apiFetch<PostRow[]>('/posts')
      .then((d) => setRows(d ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : t.pages.post.loadFail))
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

  return (
    <div>
      <PageHead title={t.pages.post.title} desc={t.pages.post.desc} />
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" placeholder={t.pages.post.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <BatchImportEntry kind="post" onImported={load} />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={load}>{t.pages.audit.refresh}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => setForm(emptyPostForm())}>
            {t.pages.post.create}
          </button>
        </div>
        {error && <ErrorBanner message={error} />}
        <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{t.pages.post.columns.map((c) => <th key={c} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{c}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.code}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.name}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.deptName}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{(r.roles ?? []).join(', ') || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      <span className="inline-flex items-center">
                        <button onClick={() => setDetail(r)}>{t.pages.post.detail}</button>
                        <span className="text-[var(--shell-side-border)]">|</span>
                        <button onClick={() => setForm(rowToPostForm(r))}>{t.pages.account.edit}</button>
                        <span className="text-[var(--shell-side-border)]">|</span>
                        <button className="text-[var(--color-danger)]" onClick={() => del(r)}>{t.pages.staff.del}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={5} loading={busy} text={t.pages.post.empty} />}
              </tbody>
            </table>
          </div>
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(t.pages.post)} />
        </div>
      </div>
      {detail && (
        <DetailDrawer
          title={t.pages.post.detail}
          closeText={t.pages.company.cancel}
          onClose={() => setDetail(null)}
          items={[
            { k: t.pages.post.columns[0], v: detail.code },
            { k: t.pages.post.columns[1], v: detail.name },
            { k: t.pages.post.columns[2], v: detail.deptName },
            { k: t.pages.post.columns[3], v: (detail.roles ?? []).join(', ') },
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
