// 官网分类管理页:cms_categories 字典 CRUD,菜单 key=site-cats,权限 menu:site-cats。
// code 被文章引用时后端拒删/拒改 code(40900),前端转成可读提示。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead } from '../../org/shared'
import { TableStateRow } from '../../../components/business'
import { useConfirm } from '../../../components/ConfirmDialog'
import { Dropdown } from '../../../components/Dropdown'

type Cat = { id: number; code: string; name: string; sortNo: number; enabled: boolean; updatedAt: string }

type Form = Pick<Cat, 'code' | 'name' | 'sortNo' | 'enabled'>

const emptyForm: Form = { code: '', name: '', sortNo: 0, enabled: true }
const inputCls = 'h-8 w-full rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'

const isConflict = (e: unknown) => e instanceof Error && e.message.includes('40900')

export default function SiteCategoriesPage() {
  const t = useT(); const s = t.pages.siteCatsPage; const confirm = useConfirm()
  const [rows, setRows] = useState<Cat[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [editing, setEditing] = useState<Cat | null | undefined>(undefined)
  const [form, setForm] = useState<Form>(emptyForm)

  const load = () => {
    setBusy(true); setError('')
    apiFetch<{ items: Cat[] }>('/site-categories')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : s.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, [])

  const open = (c?: Cat) => {
    setEditing(c ?? null)
    setForm(c ? { code: c.code, name: c.name, sortNo: c.sortNo, enabled: c.enabled } : emptyForm)
  }
  const save = async () => {
    setBusy(true)
    try {
      await apiFetch(editing ? `/site-categories/${editing.id}` : '/site-categories', { method: editing ? 'PUT' : 'POST', body: form })
      setEditing(undefined); load()
    } catch (e) {
      setError(isConflict(e) ? s.inUse : e instanceof Error ? e.message : s.saveFail)
      setBusy(false)
    }
  }
  const remove = async (c: Cat) => {
    if (!(await confirm(s.deleteConfirm, { danger: true }))) return
    setBusy(true)
    try { await apiFetch(`/site-categories/${c.id}`, { method: 'DELETE' }); load() }
    catch (e) {
      setError(isConflict(e) ? s.inUse : e instanceof Error ? e.message : s.actionFail)
      setBusy(false)
    }
  }

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
            {rows.map((c) => <tr key={c.id}>
              <td className={td}>{c.code}</td>
              <td className={td}>{c.name}</td>
              <td className={td}>{c.sortNo}</td>
              <td className={td}>{c.enabled ? s.enabledOn : s.enabledOff}</td>
              <td className={td}>
                <button className="cursor-pointer border-none bg-none text-[13px] text-[var(--shell-content-text)] underline-offset-2 hover:text-[var(--shell-heading)] hover:underline" onClick={() => open(c)}>{s.edit}</button>
                <button className="ml-3 cursor-pointer border-none bg-none text-[13px] text-[var(--color-danger)] underline-offset-2 hover:underline" onClick={() => remove(c)}>{s.delete}</button>
              </td>
            </tr>)}
            {!rows.length && <TableStateRow colSpan={5} loading={busy} text={s.empty} />}
          </tbody>
        </table>
      </div>
    </div>

    {editing !== undefined && <div className="mt-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-4">
      <div className="grid max-w-2xl gap-3 md:grid-cols-2">
        <label className="text-xs">{s.fCode}<input className={inputCls + ' mt-1'} value={form.code} placeholder="FAQ" onChange={(e) => setForm({ ...form, code: e.target.value.toUpperCase() })} /></label>
        <label className="text-xs">{s.fName}<input className={inputCls + ' mt-1'} value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /></label>
        <label className="text-xs">{s.fSort}<input className={inputCls + ' mt-1'} type="number" value={form.sortNo} onChange={(e) => setForm({ ...form, sortNo: Number(e.target.value) })} /></label>
        <label className="text-xs">{s.fEnabled}
          <div className="mt-1"><Dropdown options={[{ value: 'on', label: s.enabledOn }, { value: 'off', label: s.enabledOff }]} value={form.enabled ? s.enabledOn : s.enabledOff} onChange={(v) => setForm({ ...form, enabled: v === 'on' })} ariaLabel={s.fEnabled} /></div>
        </label>
      </div>
      <div className="mt-4 flex gap-3">
        <button className="primary h-8 cursor-pointer rounded-sm px-4 text-[13px]" disabled={busy} onClick={save}>{s.save}</button>
        <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)]" onClick={() => setEditing(undefined)}>{t.pages.company.cancel}</button>
      </div>
    </div>}
  </div>
}
