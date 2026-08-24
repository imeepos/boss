// 官网内容管理:cms_posts CRUD(动态/新闻/文章),菜单 key=site,权限 menu:site。
// 正文 Markdown 编辑:textarea + 简易预览切换,不引重型所见即所得(adopted note)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { TableStateRow } from '../../../components/business'
import { useConfirm } from '../../../components/ConfirmDialog'
import { Dropdown } from '../../../components/Dropdown'

type Post = {
  id: number; slug: string; title: string; category: string; summary: string
  coverAttachmentId: number; content: string; status: string; publishedAt: string
  version: number; authorName: string; updatedAt: string
}

type Form = Pick<Post, 'slug' | 'title' | 'category' | 'summary' | 'content' | 'status' | 'authorName'>

const emptyForm: Form = { slug: '', title: '', category: 'NEWS', summary: '', content: '', status: 'DRAFT', authorName: '' }
const inputCls = 'h-8 w-full rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'

export default function SitePostsPage() {
  const t = useT(); const s = t.pages.sitePage; const confirm = useConfirm()
  const [rows, setRows] = useState<Post[]>([]); const [busy, setBusy] = useState(false)
  const [error, setError] = useState(''); const [page, setPage] = useState(1); const pageSize = 10
  const [editing, setEditing] = useState<Post | null | undefined>(undefined)
  const [form, setForm] = useState<Form>(emptyForm)
  const [preview, setPreview] = useState(false)

  const load = () => {
    setBusy(true); setError('')
    apiFetch<{ items: Post[] }>('/site-posts')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : s.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, [])

  const open = (p?: Post) => {
    setEditing(p ?? null); setPreview(false)
    setForm(p ? { slug: p.slug, title: p.title, category: p.category, summary: p.summary, content: p.content, status: p.status, authorName: p.authorName } : emptyForm)
  }
  const save = async () => {
    setBusy(true)
    try {
      await apiFetch(editing ? `/site-posts/${editing.id}` : '/site-posts', { method: editing ? 'PUT' : 'POST', body: form })
      setEditing(undefined); load()
    } catch (e) { setError(e instanceof Error ? e.message : s.saveFail); setBusy(false) }
  }
  const remove = async (p: Post) => {
    if (!(await confirm(s.deleteConfirm, { danger: true }))) return
    setBusy(true)
    try { await apiFetch(`/site-posts/${p.id}`, { method: 'DELETE' }); load() }
    catch (e) { setError(e instanceof Error ? e.message : s.actionFail); setBusy(false) }
  }

  const catLabel = (v: string) => (v === 'ARTICLE' ? s.catArticle : s.catNews)
  const stLabel = (v: string) => (v === 'PUBLISHED' ? s.stPublished : v === 'OFFLINE' ? s.stOffline : s.stDraft)
  const slice = rows.slice((page - 1) * pageSize, page * pageSize)
  const td = 'border-b border-[var(--shell-side-border)] px-3 py-2'

  return <div>
    <PageHead title={s.title} desc={s.desc} />
    <div className="mb-4 flex justify-end">
      <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)]" onClick={() => open()}>{s.create}</button>
    </div>
    <div className="rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-4">
      {error && <div className="mb-3 text-sm text-[var(--color-danger)]">{error}</div>}
      <div className="overflow-x-auto">
        <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
          <thead><tr>{s.columns.map((x) => <th key={x} className="border-b border-[var(--shell-side-border)] px-3 py-2 text-left text-xs">{x}</th>)}</tr></thead>
          <tbody>
            {slice.map((p) => <tr key={p.id}>
              <td className={td}>{p.title}</td>
              <td className={td}>{p.slug}</td>
              <td className={td}>{catLabel(p.category)}</td>
              <td className={td}>{stLabel(p.status)}</td>
              <td className={td}>{p.publishedAt || '—'}</td>
              <td className={td}>v{p.version}</td>
              <td className={td}>
                <button className="cursor-pointer border-none bg-none text-[13px] text-[var(--shell-content-text)] underline-offset-2 hover:text-[var(--shell-heading)] hover:underline" onClick={() => open(p)}>{s.edit}</button>
                <button className="ml-3 cursor-pointer border-none bg-none text-[13px] text-[var(--color-danger)] underline-offset-2 hover:underline" onClick={() => remove(p)}>{s.delete}</button>
              </td>
            </tr>)}
            {!slice.length && <TableStateRow colSpan={7} loading={busy} text={s.empty} />}
          </tbody>
        </table>
      </div>
      <div className="flex justify-end pt-3">
        <Pagination total={rows.length} page={page} pageSize={pageSize} onPage={setPage} onSize={() => {}} {...pagerTexts(t.pages.company)} />
      </div>
    </div>

    {editing !== undefined && <div className="mt-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-4">
      <div className="grid gap-3 md:grid-cols-2">
        <label className="text-xs">{s.fTitle}<input className={inputCls + ' mt-1'} value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} /></label>
        <label className="text-xs">{s.fSlug}<input className={inputCls + ' mt-1'} value={form.slug} placeholder="hello-world" onChange={(e) => setForm({ ...form, slug: e.target.value })} /></label>
        <label className="text-xs">{s.fCategory}
          <div className="mt-1"><Dropdown options={s.categoryOptions.map((l, i) => ({ value: i === 1 ? 'ARTICLE' : 'NEWS', label: l }))} value={catLabel(form.category)} onChange={(v) => setForm({ ...form, category: v })} ariaLabel={s.fCategory} /></div>
        </label>
        <label className="text-xs">{s.fStatus}
          <div className="mt-1"><Dropdown options={[{ value: 'DRAFT', label: s.stDraft }, { value: 'PUBLISHED', label: s.stPublished }, { value: 'OFFLINE', label: s.stOffline }]} value={stLabel(form.status)} onChange={(v) => setForm({ ...form, status: v })} ariaLabel={s.fStatus} /></div>
        </label>
        <label className="text-xs">{s.fAuthor}<input className={inputCls + ' mt-1'} value={form.authorName} onChange={(e) => setForm({ ...form, authorName: e.target.value })} /></label>
        <label className="text-xs">{s.fSummary}<input className={inputCls + ' mt-1'} value={form.summary} onChange={(e) => setForm({ ...form, summary: e.target.value })} /></label>
      </div>
      <div className="mt-3">
        <div className="mb-1 flex items-center justify-between">
          <span className="text-xs">{s.fContent}</span>
          <button className="cursor-pointer border-none bg-none px-1 py-0.5 text-xs text-[var(--shell-group-title)] hover:text-[var(--shell-content-text)]" onClick={() => setPreview(!preview)}>{preview ? 'Markdown' : 'Preview'}</button>
        </div>
        {preview
          ? <pre className="min-h-48 overflow-auto rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] p-3 text-[13px] whitespace-pre-wrap text-[var(--shell-content-text)]">{form.content}</pre>
          : <textarea className="min-h-48 w-full rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] p-3 text-[13px] text-[var(--shell-content-text)] outline-none focus:border-[var(--color-border-focus)]" value={form.content} onChange={(e) => setForm({ ...form, content: e.target.value })} />}
      </div>
      <div className="mt-4 flex gap-3">
        <button className="primary h-8 cursor-pointer rounded-sm px-4 text-[13px]" disabled={busy} onClick={save}>{s.save}</button>
        <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)]" onClick={() => setEditing(undefined)}>{t.pages.company.cancel}</button>
      </div>
    </div>}
  </div>
}
