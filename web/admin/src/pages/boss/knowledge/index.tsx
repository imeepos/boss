// 客服知识库列表页:cs_knowledge_articles CRUD(后端复用 menu:complaint 权限,前端菜单 key=knowledge)。
// 状态枚举 DRAFT/PUBLISHED/OFFLINE 与 sitePage 同源,stLabel 三语映射;颜色全走主题 token。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { TableStateRow, ToolbarButton } from '../../../components/business'
import { useConfirm } from '../../../components/ConfirmDialog'
import { Dropdown } from '../../../components/Dropdown'

type Article = {
  id: number; code: string; title: string; content: string
  status: string; version: number; ownerId: number; updatedAt: string
}

const emptyForm = { code: '', title: '', content: '', status: 'DRAFT' }
const inputCls = 'block w-full rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-3 py-2 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--shell-input-border-focus)]'
const td = 'border-b border-[var(--shell-side-border)] px-3 py-2'

export default function KnowledgePage() {
  const t = useT(); const k = t.pages.knowledgePage; const confirm = useConfirm()
  const [rows, setRows] = useState<Article[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [editing, setEditing] = useState<Article | null | undefined>(undefined)
  const [form, setForm] = useState(emptyForm)

  const load = () => {
    setBusy(true); setError('')
    apiFetch<{ items: Article[] }>('/knowledge-articles')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : k.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, [])

  const open = (a?: Article) => {
    setEditing(a ?? null)
    setForm(a ? { code: a.code, title: a.title, content: a.content, status: a.status } : emptyForm)
  }
  const save = async () => {
    setBusy(true)
    try {
      await apiFetch(editing ? `/knowledge-articles/${editing.id}` : '/knowledge-articles', { method: editing ? 'PUT' : 'POST', body: form })
      toast.success(k.toastSaved)
      setEditing(null); load()
    } catch (e) { setError(e instanceof Error ? e.message : k.saveFail); setBusy(false) }
  }
  const remove = async (a: Article) => {
    if (!(await confirm(k.deleteConfirm, { danger: true }))) return
    setBusy(true)
    try { await apiFetch(`/knowledge-articles/${a.id}`, { method: 'DELETE' }); toast.success(k.toastDeleted); load() }
    catch (e) { setError(e instanceof Error ? e.message : k.actionFail); setBusy(false) }
  }

  const stLabel = (v: string) => (v === 'PUBLISHED' ? k.stPublished : v === 'OFFLINE' ? k.stOffline : k.stDraft)
  const statusOptions = [
    { value: 'DRAFT', label: k.stDraft },
    { value: 'PUBLISHED', label: k.stPublished },
    { value: 'OFFLINE', label: k.stOffline },
  ]
  const slice = rows.slice((page - 1) * pageSize, page * pageSize)

  return <div>
    <PageHead title={k.title} desc={k.desc} />
    <div className="mb-4 flex justify-end">
      <ToolbarButton primary onClick={() => open()}>{k.create}</ToolbarButton>
    </div>
    <div className="rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-4">
      {error && <div className="mb-3 text-sm text-[var(--color-danger)]">{error}</div>}
      <div className="overflow-x-auto">
        <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
          <thead><tr>{k.columns.map((x) => <th key={x} className="border-b border-[var(--shell-side-border)] px-3 py-2 text-left text-xs">{x}</th>)}</tr></thead>
          <tbody>
            {slice.map((a) => <tr key={a.id}>
              <td className={td}>{a.code}</td>
              <td className={td}>{a.title}</td>
              <td className={`${td} max-w-md truncate`}>{a.content}</td>
              <td className={td}>{stLabel(a.status)}</td>
              <td className={td}>v{a.version}</td>
              <td className={td}>
                <button className="cursor-pointer border-none bg-none text-[13px] text-[var(--shell-content-text)] underline-offset-2 hover:text-[var(--shell-heading)] hover:underline" onClick={() => open(a)}>{k.edit}</button>
                <button className="ml-3 cursor-pointer border-none bg-none text-[13px] text-[var(--color-danger)] underline-offset-2 hover:underline" onClick={() => remove(a)}>{k.delete}</button>
              </td>
            </tr>)}
            {!slice.length && <TableStateRow colSpan={6} loading={busy} text={k.empty} />}
          </tbody>
        </table>
      </div>
      <div className="flex justify-end pt-3">
        <Pagination total={rows.length} page={page} pageSize={pageSize} onPage={setPage} onSize={setPageSize} {...pagerTexts(k)} />
      </div>
    </div>
    {editing !== undefined ? <div className="mt-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-4">
      <div className="grid gap-3 md:grid-cols-2">
        <label className="text-xs">{k.fCode}<input className={`mt-1 ${inputCls}`} value={form.code} onChange={(e) => setForm({ ...form, code: e.target.value })} /></label>
        <label className="text-xs">{k.fTitle}<input className={`mt-1 ${inputCls}`} value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} /></label>
        <label className="text-xs md:col-span-2">{k.fContent}<textarea className={`mt-1 min-h-32 ${inputCls}`} value={form.content} onChange={(e) => setForm({ ...form, content: e.target.value })} /></label>
        <label className="text-xs">{k.fStatus}
          <div className="mt-1"><Dropdown value={form.status} options={statusOptions} onChange={(value) => setForm({ ...form, status: value })} ariaLabel={k.fStatus} /></div>
        </label>
      </div>
      <div className="mt-4"><ToolbarButton primary disabled={busy} onClick={save}>{k.save}</ToolbarButton></div>
    </div> : null}
  </div>
}
