// 岗位管理页:列名以 fields.md 1.4 为准(岗位代码/岗位/所属部门/绑定角色);原型数据范围列后端无字段,按契约裁剪。
// 契约: GET /posts(org.yaml;deptId 过滤,本次全量;roles 为岗位绑定角色码聚合)。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { DetailDrawer, PageHead, pagerTexts } from '../shared'
import { filterPosts, pageSlice, type PostRow } from './filter'
import { PostFormDrawer, emptyPostForm, rowToPostForm, type PostFormValues } from './PostForm'
import { Pagination } from '../../../components/Pagination'
import '../org.css'

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
      setForm(null)
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : t.pages.post.saveFail)
    } finally {
      setBusy(false)
    }
  }

  const filtered = useMemo(() => filterPosts(rows, keyword), [rows, keyword])
  const slice = pageSlice(filtered, page, pageSize)

  return (
    <div>
      <PageHead title={t.pages.post.title} desc={t.pages.post.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <input className="org-input" placeholder={t.pages.post.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <button className="org-btn" onClick={load}>{t.pages.audit.refresh}</button>
          <button className="org-btn org-btn-primary" onClick={() => setForm(emptyPostForm())}>
            {t.pages.post.create}
          </button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{t.pages.post.columns.map((c) => <th key={c}>{c}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.id}>
                    <td>{r.code}</td>
                    <td>{r.name}</td>
                    <td>{r.deptName}</td>
                    <td>{(r.roles ?? []).join(', ') || '—'}</td>
                    <td>
                      <span className="org-act">
                        <button onClick={() => setDetail(r)}>{t.pages.post.detail}</button>
                        <span className="sep">|</span>
                        <button onClick={() => setForm(rowToPostForm(r))}>{t.pages.account.edit}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={5}><div className="org-empty">{t.pages.post.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
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
