// 官网内容添加/编辑独立页:路由 /boss/site/new 与 /boss/site/:postId。
// 表单字段与列表页原内联表单一致;正文换 MarkdownEditor 双栏编辑;
// 分类下拉读 /site-categories 字典(启用项,按界面语言展示本地化名);
// 语言(000155)选变体:同 slug 多语言各存一行,公开端按 ?lang= 命中。
import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { apiFetch } from '../../../api/client'
import { AttachmentPickerDialog } from '../../../components/AttachmentManager/PickerDialog'
import { useT, useLocale, localeOptions, type Locale } from '../../../i18n'
import { PageHead } from '../../org/shared'
import { Dropdown } from '../../../components/Dropdown'
import { MarkdownEditor } from './MarkdownEditor'

type Post = {
  id: number; slug: string; lang: string; title: string; category: string; summary: string
  coverAttachmentId: number; content: string; status: string; publishedAt: string
  version: number; authorName: string; updatedAt: string
}
type Cat = { id: number; code: string; name: string; names?: Record<string, string>; sortNo: number; enabled: boolean }

type Form = Pick<Post, 'slug' | 'lang' | 'title' | 'category' | 'summary' | 'content' | 'status' | 'authorName' | 'coverAttachmentId'>

const emptyForm: Form = { slug: '', lang: 'zh-CN', title: '', category: '', summary: '', content: '', status: 'DRAFT', authorName: '', coverAttachmentId: 0 }
const inputCls = 'h-8 w-full rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'

export default function SitePostEditorPage() {
  const t = useT(); const s = t.pages.sitePage
  const locale = useLocale()
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
          setForm({ slug: p.slug, lang: p.lang || 'zh-CN', title: p.title, category: p.category, summary: p.summary, content: p.content, status: p.status, authorName: p.authorName, coverAttachmentId: p.coverAttachmentId })
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
  const [pickCover, setPickCover] = useState(false)
  const stLabel = (v: string) => (v === 'PUBLISHED' ? s.stPublished : v === 'OFFLINE' ? s.stOffline : s.stDraft)
  // 分类展示名:语言覆盖 → 默认名 → code。
  const catName = (code: string) => {
    const c = cats.find((x) => x.code === code)
    return c ? c.names?.[locale] || c.name : code
  }

  return <div>
    <PageHead title={editing ? s.editTitle : s.newTitle} desc={s.desc} />
    <div className="rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-4">
      {error && <div className="mb-3 text-sm text-[var(--color-danger)]">{error}</div>}
      <div className="grid gap-3 md:grid-cols-2">
        <label className="text-xs">{s.fTitle}<input className={inputCls + ' mt-1'} value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} /></label>
        <label className="text-xs">{s.fSlug}<input className={inputCls + ' mt-1'} value={form.slug} placeholder="hello-world" onChange={(e) => setForm({ ...form, slug: e.target.value })} /></label>
        <label className="text-xs">{s.fLang}
          <div className="mt-1"><Dropdown options={localeOptions()} value={localeOptions().find((o) => o.value === form.lang)?.label ?? form.lang} onChange={(v) => setForm({ ...form, lang: v as Locale })} ariaLabel={s.fLang} /></div>
        </label>
        <label className="text-xs">{s.fCategory}
          <div className="mt-1"><Dropdown options={cats.map((c) => ({ value: c.code, label: c.names?.[locale] || c.name }))} value={catName(form.category || cats[0]?.code || '')} onChange={(v) => setForm({ ...form, category: v })} ariaLabel={s.fCategory} /></div>
        </label>
        <label className="text-xs">{s.fStatus}
          <div className="mt-1"><Dropdown options={[{ value: 'DRAFT', label: s.stDraft }, { value: 'PUBLISHED', label: s.stPublished }, { value: 'OFFLINE', label: s.stOffline }]} value={stLabel(form.status)} onChange={(v) => setForm({ ...form, status: v })} ariaLabel={s.fStatus} /></div>
        </label>
        <label className="text-xs">{s.fAuthor}<input className={inputCls + ' mt-1'} value={form.authorName} onChange={(e) => setForm({ ...form, authorName: e.target.value })} /></label>
        <div className="text-xs">{s.fCover}
          <div className="mt-1 flex items-center gap-3">
            <button type="button" className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)]" onClick={() => setPickCover(true)}>{s.pickCover}</button>
            {form.coverAttachmentId > 0 && <>
              <span className="text-xs text-[var(--shell-content-text)]">#{form.coverAttachmentId}</span>
              <button type="button" className="h-7 cursor-pointer rounded-sm border border-[var(--shell-input-border)] px-3 text-xs" onClick={() => setForm({ ...form, coverAttachmentId: 0 })}>{s.fCoverRemove}</button>
            </>}
          </div>
        </div>
        <label className="text-xs md:col-span-2">{s.fSummary}<input className={inputCls + ' mt-1'} value={form.summary} onChange={(e) => setForm({ ...form, summary: e.target.value })} /></label>
      </div>
      <div className="mt-4">
        <div className="mb-1 text-xs">{s.fContent}</div>
        <MarkdownEditor value={form.content} onChange={(v) => setForm({ ...form, content: v })} />
      </div>
      <AttachmentPickerDialog open={pickCover} onClose={() => setPickCover(false)} imageOnly
        onPick={(items) => { if (items[0]) setForm((v) => ({ ...v, coverAttachmentId: items[0].id })) }} />
      <div className="mt-4 flex gap-3">
        <button type="button" className="primary h-8 cursor-pointer rounded-sm px-4 text-[13px]" disabled={busy} onClick={save}>{s.save}</button>
        <button type="button" className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)]" onClick={() => nav('/boss/site')}>{s.back}</button>
      </div>
    </div>
  </div>
}
