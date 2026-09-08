// 官网内容列表页:cms_posts 列表 + 删除;新增/编辑跳独立编辑页 /boss/site(/new|/:postId)。
// 菜单 key=site,权限 menu:site。分类列展示字典本地化名(懒加载 /site-categories);
// 语言列(000155):同 slug 多语言变体各一行,可按语言筛选。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { useNavigate } from 'react-router-dom'
import { apiFetch } from '../../../api/client'
import { useT, useLocale, localeOptions } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { ErrorBanner, ToolbarButton } from '../../../components/business'
import { Pagination } from '../../../components/Pagination'
import { TableStateRow } from '../../../components/business'
import { useConfirm } from '../../../components/ConfirmDialog'
import { Dropdown } from '../../../components/Dropdown'
import { Card } from '../../../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { fmtTime } from '../../../lib/format'

type Post = {
  id: number; slug: string; lang: string; title: string; category: string; summary: string
  coverAttachmentId: number; content: string; status: string; publishedAt: string
  version: number; authorName: string; updatedAt: string
}
type Cat = { code: string; name: string; names?: Record<string, string> }

export default function SitePostsPage() {
  const t = useT(); const s = t.pages.sitePage; const confirm = useConfirm()
  const locale = useLocale()
  const nav = useNavigate()
  const [rows, setRows] = useState<Post[]>([])
  const [cats, setCats] = useState<Record<string, Cat>>({})
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [fCat, setFCat] = useState('')
  const [fSt, setFSt] = useState('')
  const [fLang, setFLang] = useState('')

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
        const m: Record<string, Cat> = {}
        for (const c of d?.items ?? []) m[c.code] = c
        setCats(m)
      })
      .catch(() => setCats({}))
  }, [])

  const remove = async (p: Post) => {
    if (!(await confirm(s.deleteConfirm, { danger: true }))) return
    setBusy(true)
    try { await apiFetch(`/site-posts/${p.id}`, { method: 'DELETE' }); toast.success(s.toastDeleted); load() }
    catch (e) { setError(e instanceof Error ? e.message : s.actionFail); setBusy(false) }
  }

  const stLabel = (v: string) => (v === 'PUBLISHED' ? s.stPublished : v === 'OFFLINE' ? s.stOffline : s.stDraft)
  const catLabel = (code: string) => {
    const c = cats[code]
    return c ? c.names?.[locale] || c.name : code
  }
  const langLabel = (v: string) => localeOptions().find((o) => o.value === v)?.label ?? v
  const filtered = rows.filter((p) => (!fCat || p.category === fCat) && (!fSt || p.status === fSt) && (!fLang || p.lang === fLang))
  const slice = filtered.slice((page - 1) * pageSize, page * pageSize)

  return <div>
    <PageHead title={s.title} desc={s.desc} />
    <div className="mb-4 flex justify-end gap-3">
      <ToolbarButton onClick={() => nav('/boss/site/cats')}>{s.manageCats}</ToolbarButton>
      <ToolbarButton primary onClick={() => nav('/boss/site/new')}>{s.create}</ToolbarButton>
    </div>
    <Card className="p-4">
      {error && <ErrorBanner message={error} />}
      <div className="mb-3 flex items-center gap-3">
        <Dropdown ariaLabel={s.fCategory} value={fCat}
          onChange={(v) => { setFCat(v); setPage(1) }}
          options={[{ value: '', label: s.filterAll }, ...Object.entries(cats).map(([code, c]) => ({ value: code, label: c.names?.[locale] || c.name }))]} />
        <Dropdown ariaLabel={s.fStatus} value={fSt}
          onChange={(v) => { setFSt(v); setPage(1) }}
          options={[{ value: '', label: s.filterAll }, { value: 'DRAFT', label: s.stDraft }, { value: 'PUBLISHED', label: s.stPublished }, { value: 'OFFLINE', label: s.stOffline }]} />
        <Dropdown ariaLabel={s.fLang} value={fLang}
          onChange={(v) => { setFLang(v); setPage(1) }}
          options={[{ value: '', label: s.filterAll }, ...localeOptions()]} />
      </div>
      <Table>
        <TableHeader>
          <TableRow>{s.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
        </TableHeader>
        <TableBody>
          {slice.map((p) => <TableRow key={p.id}>
            <TableCell>{p.title}</TableCell>
            <TableCell>{p.slug}</TableCell>
            <TableCell>{langLabel(p.lang)}</TableCell>
            <TableCell>{catLabel(p.category)}</TableCell>
            <TableCell>{stLabel(p.status)}</TableCell>
            <TableCell>{fmtTime(p.publishedAt)}</TableCell>
            <TableCell>v{p.version}</TableCell>
            <TableCell>
              <button className="cursor-pointer border-none bg-none text-[13px] text-[var(--shell-content-text)] underline-offset-2 hover:text-[var(--shell-heading)] hover:underline" onClick={() => nav(`/boss/site/${p.id}`)}>{s.edit}</button>
              <button className="ml-3 cursor-pointer border-none bg-none text-[13px] text-[var(--color-danger)] underline-offset-2 hover:underline" onClick={() => remove(p)}>{s.delete}</button>
            </TableCell>
          </TableRow>)}
          {!slice.length && <TableStateRow colSpan={8} loading={busy} text={s.empty} />}
        </TableBody>
      </Table>
      <div className="flex justify-end pt-3">
        <Pagination total={filtered.length} page={page} pageSize={pageSize}
          onPage={setPage} onSize={(sz) => { setPageSize(sz); setPage(1) }} {...pagerTexts(t.pages.company)} />
      </div>
    </Card>
  </div>
}
