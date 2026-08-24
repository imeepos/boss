// 官网内容添加/编辑独立页:路由 /boss/site/new 与 /boss/site/:postId。
// 表单字段与列表页原内联表单一致;正文换 MarkdownEditor 双栏编辑;
// 分类下拉读 /site-categories 字典(启用项)。
import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { apiFetch } from '../../../api/client'
import { uploadAttachment } from '../../../api/attachments'
import { useT } from '../../../i18n'
import { PageHead } from '../../org/shared'
import { Dropdown } from '../../../components/Dropdown'
import { MarkdownEditor } from './MarkdownEditor'

type Post = {
  id: number; slug: string; title: string; category: string; summary: string
  coverAttachmentId: number; content: string; status: string; publishedAt: string
  version: number; authorName: string; updatedAt: string
}
type Cat = { id: number; code: string; name: string; sortNo: number; enabled: boolean }

type Form = Pick<Post, 'slug' | 'title' | 'category' | 'summary' | 'content' | 'status' | 'authorName' | 'coverAttachmentId'>

const emptyForm: Form = { slug: '', title: '', category: '', summary: '', content: '', status: 'DRAFT', authorName: '', coverAttachmentId: 0 }
const inputCls = 'h-8 w-full rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'

export default function SitePostEditorPage() {
  const t = useT(); const s = t.pages.sitePage
  const nav = useNavigate()
  const { postId } = useParams()
  const editing = postId !== undefined
  const [form, setForm] = useState<Form>(emptyForm)
  const [cats, setCats] = useState<Cat[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    apiFetch<{ items: Cat[] }>('/site-categories')
      .then((d) => setCats((d?.items ?? []).filter((c) => c.enabled)))
      .catch(() => setCats([]))
    if (!editing) return
    apiFetch<{ items: Post[] }>('/site-posts')
      .then((d) => {
        const p = (d?.items ?? []).find((x) => String(x.id) === postId)
        if (p) {
          setForm({ slug: p.slug, title: p.title, category: p.category, summary: p.summary, content: p.content, status: p.status, authorName: p.authorName, coverAttachmentId: p.coverAttachmentId })
        } else { setError(s.loadFail) }
      })
      .catch((e) => setError(e instanceof Error ? e.message : s.loadFail))
  }, [editing, postId])

  const save = async () => {
    setBusy(true); setError('')
    try {
      const body = { ...form, category: form.category || cats[0]?.code || 'NEWS' }
      await apiFetch(editing ? `/site-posts/${postId}` : '/site-posts', { method: editing ? 'PUT' : 'POST', body })
      nav('/boss/site')
    } catch (e) { setError(e instanceof Error ? e.message : s.saveFail); setBusy(false) }
  }
  const uploadCover = async (f: File | undefined) => {
    if (!f) return
    setBusy(true)
    try {
      const at = await uploadAttachment(f)
      if (at) setForm((v) => ({ ...v, coverAttachmentId: at.id }))
    } catch (e) { setError(e instanceof Error ? e.message : s.actionFail) }
    setBusy(false)
  }
  const stLabel = (v: string) => (v === 'PUBLISHED' ? s.stPublished : v === 'OFFLINE' ? s.stOffline : s.stDraft)
  const catName = (code: string) => cats.find((c) => c.code === code)?.name ?? code

  return <div>
    <PageHead title={editing ? s.editTitle : s.newTitle} desc={s.desc} />
    <div className="rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-4">
      {error && <div className="mb-3 text-sm text-[var(--color-danger)]">{error}</div>}
      <div className="grid gap-3 md:grid-cols-2">
        <label className="text-xs">{s.fTitle}<input className={inputCls + ' mt-1'} value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} /></label>
        <label className="text-xs">{s.fSlug}<input className={inputCls + ' mt-1'} value={form.slug} placeholder="hello-world" onChange={(e) => setForm({ ...form, slug: e.target.value })} /></label>
        <label className="text-xs">{s.fCategory}
          <div className="mt-1"><Dropdown options={cats.map((c) => ({ value: c.code, label: c.name }))} value={catName(form.category || cats[0]?.code || '')} onChange={(v) => setForm({ ...form, category: v })} ariaLabel={s.fCategory} /></div>
        </label>
        <label className="text-xs">{s.fStatus}
          <div className="mt-1"><Dropdown options={[{ value: 'DRAFT', label: s.stDraft }, { value: 'PUBLISHED', label: s.stPublished }, { value: 'OFFLINE', label: s.stOffline }]} value={stLabel(form.status)} onChange={(v) => setForm({ ...form, status: v })} ariaLabel={s.fStatus} /></div>
        </label>
        <label className="text-xs">{s.fAuthor}<input className={inputCls + ' mt-1'} value={form.authorName} onChange={(e) => setForm({ ...form, authorName: e.target.value })} /></label>
        <label className="text-xs">{s.fCover}
          <input className="mt-1 block w-full text-xs text-[var(--shell-content-text)] file:mr-3 file:cursor-pointer file:rounded-sm file:border file:border-[var(--shell-input-border)] file:bg-[var(--shell-input-bg)] file:px-3 file:py-1.5 file:text-xs" type="file" accept="image/*" onChange={(e) => uploadCover(e.target.files?.[0])} />
        </label>
        {form.coverAttachmentId > 0 && <div className="flex items-center gap-3 text-xs text-[var(--shell-content-text)]">
          <span>#{form.coverAttachmentId}</span>
          <button type="button" className="h-7 cursor-pointer rounded-sm border border-[var(--shell-input-border)] px-3 text-xs" onClick={() => setForm({ ...form, coverAttachmentId: 0 })}>{s.fCoverRemove}</button>
        </div>}
        <label className="text-xs md:col-span-2">{s.fSummary}<input className={inputCls + ' mt-1'} value={form.summary} onChange={(e) => setForm({ ...form, summary: e.target.value })} /></label>
      </div>
      <div className="mt-4">
        <div className="mb-1 text-xs">{s.fContent}</div>
        <MarkdownEditor value={form.content} onChange={(v) => setForm({ ...form, content: v })} texts={{ uploadImg: s.uploadImg, uploadFail: s.uploadFail }} />
      </div>
      <div className="mt-4 flex gap-3">
        <button type="button" className="primary h-8 cursor-pointer rounded-sm px-4 text-[13px]" disabled={busy} onClick={save}>{s.save}</button>
        <button type="button" className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)]" onClick={() => nav('/boss/site')}>{s.back}</button>
      </div>
    </div>
  </div>
}
