// 官网内容列表页:cms_posts 列表 + 删除;新增/编辑跳独立编辑页 /boss/site(/new|/:postId)。
// 菜单 key=site,权限 menu:site。分类列展示字典名(懒加载 /site-categories)。
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { TableStateRow } from '../../../components/business'
import { useConfirm } from '../../../components/ConfirmDialog'

type Post = {
  id: number; slug: string; title: string; category: string; summary: string
  coverAttachmentId: number; content: string; status: string; publishedAt: string
  version: number; authorName: string; updatedAt: string
}
type Cat = { code: string; name: string }

export default function SitePostsPage() {
  const t = useT(); const s = t.pages.sitePage; const confirm = useConfirm()
  const nav = useNavigate()
  const [rows, setRows] = useState<Post[]>([])
  const [cats, setCats] = useState<Record<string, string>>({})
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const pageSize = 10

  const load = () => {
    setBusy(true); setError('')
    apiFetch<{ items: Post[] }>('/site-posts')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : s.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, [])
  useEffect(() => {
    apiFetch<{ items: Cat[] }>('/site-categories')
      .then((d) => {
        const m: Record<string, string> = {}
        for (const c of d?.items ?? []) m[c.code] = c.name
        setCats(m)
      })
      .catch(() => setCats({}))
  }, [])

  const remove = async (p: Post) => {
    if (!(await confirm(s.deleteConfirm, { danger: true }))) return
    setBusy(true)
    try { await apiFetch(`/site-posts/${p.id}`, { method: 'DELETE' }); load() }
    catch (e) { setError(e instanceof Error ? e.message : s.actionFail); setBusy(false) }
  }

  const stLabel = (v: string) => (v === 'PUBLISHED' ? s.stPublished : v === 'OFFLINE' ? s.stOffline : s.stDraft)
  const slice = rows.slice((page - 1) * pageSize, page * pageSize)
  const td = 'border-b border-[var(--shell-side-border)] px-3 py-2'

  return <div>
    <PageHead title={s.title} desc={s.desc} />
    <div className="mb-4 flex justify-end gap-3">
      <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)]" onClick={() => nav('/boss/site/cats')}>{s.manageCats}</button>
      <button className="primary h-8 cursor-pointer rounded-sm px-4 text-[13px]" onClick={() => nav('/boss/site/new')}>{s.create}</button>
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
              <td className={td}>{cats[p.category] ?? p.category}</td>
              <td className={td}>{stLabel(p.status)}</td>
              <td className={td}>{p.publishedAt || '—'}</td>
              <td className={td}>v{p.version}</td>
              <td className={td}>
                <button className="cursor-pointer border-none bg-none text-[13px] text-[var(--shell-content-text)] underline-offset-2 hover:text-[var(--shell-heading)] hover:underline" onClick={() => nav(`/boss/site/${p.id}`)}>{s.edit}</button>
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
  </div>
}
