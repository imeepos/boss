// 官网分类管理页:cms_categories 字典 CRUD,菜单 key=site-cats,权限 menu:site-cats。
// code 被文章引用时后端拒删/拒改 code(40900),前端转成可读提示。
// 名称多语言(000155):name 为默认/回退,zh/en/ms 覆盖名可留空(官网按语言展示)。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT, useLocale } from '../../../i18n'
import { PageHead } from '../../org/shared'
import { ErrorBanner, ToolbarButton, TableStateRow } from '../../../components/business'
import { useConfirm } from '../../../components/ConfirmDialog'
import { Dropdown } from '../../../components/Dropdown'
import { Card } from '../../../components/ui/card'
import { Input } from '../../../components/ui/input'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'

type Cat = {
  id: number; code: string; name: string; names?: Record<string, string>
  sortNo: number; enabled: boolean; updatedAt: string
}

type Form = Pick<Cat, 'code' | 'name' | 'sortNo' | 'enabled'> & { names: Record<string, string> }

const emptyForm: Form = { code: '', name: '', names: {}, sortNo: 0, enabled: true }

const isConflict = (e: unknown) => e instanceof Error && e.message.includes('40900')

export default function SiteCategoriesPage() {
  const t = useT(); const s = t.pages.siteCatsPage; const confirm = useConfirm()
  const locale = useLocale()
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
    setForm(c ? { code: c.code, name: c.name, names: c.names ?? {}, sortNo: c.sortNo, enabled: c.enabled } : emptyForm)
  }
  // 空串覆盖名视为未填,不入 names(后端 NameFor 空值同样回退默认名)。
  const namesBody = () => {
    const out: Record<string, string> = {}
    for (const [k, v] of Object.entries(form.names)) if (v.trim()) out[k] = v.trim()
    return out
  }
  const save = async () => {
    setBusy(true)
    try {
      await apiFetch(editing ? `/site-categories/${editing.id}` : '/site-categories', { method: editing ? 'PUT' : 'POST', body: { ...form, names: namesBody() } })
      toast.success(s.toastSaved)
      setEditing(undefined); load()
    } catch (e) {
      setError(isConflict(e) ? s.inUse : e instanceof Error ? e.message : s.saveFail)
      setBusy(false)
    }
  }
  const remove = async (c: Cat) => {
    if (!(await confirm(s.deleteConfirm, { danger: true }))) return
    setBusy(true)
    try { await apiFetch(`/site-categories/${c.id}`, { method: 'DELETE' }); toast.success(s.toastDeleted); load() }
    catch (e) {
      setError(isConflict(e) ? s.inUse : e instanceof Error ? e.message : s.actionFail)
      setBusy(false)
    }
  }

  // 名称列按界面语言展示本地化名(缺覆盖回退默认名)。
  const localName = (c: Cat) => c.names?.[locale] || c.name
  return <div>
    <PageHead title={s.title} desc={s.desc} />
    <div className="mb-4 flex justify-end">
      <ToolbarButton primary onClick={() => open()}>{s.create}</ToolbarButton>
    </div>
    <Card className="p-4">
      {error && <ErrorBanner message={error} />}
      <Table>
        <TableHeader>
          <TableRow>{s.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((c) => <TableRow key={c.id}>
            <TableCell>{c.code}</TableCell>
            <TableCell>{localName(c)}</TableCell>
            <TableCell>{c.sortNo}</TableCell>
            <TableCell>{c.enabled ? s.enabledOn : s.enabledOff}</TableCell>
            <TableCell>
              <button className="cursor-pointer border-none bg-none text-[13px] text-[var(--shell-content-text)] underline-offset-2 hover:text-[var(--shell-heading)] hover:underline" onClick={() => open(c)}>{s.edit}</button>
              <button className="ml-3 cursor-pointer border-none bg-none text-[13px] text-[var(--color-danger)] underline-offset-2 hover:underline" onClick={() => remove(c)}>{s.delete}</button>
            </TableCell>
          </TableRow>)}
          {!rows.length && <TableStateRow colSpan={5} loading={busy} text={s.empty} />}
        </TableBody>
      </Table>
    </Card>

    {editing !== undefined && <Card className="mt-0 p-4">
      <div className="grid max-w-2xl gap-3 md:grid-cols-2">
        <label className="text-xs">{s.fCode}<Input className="mt-1" value={form.code} placeholder={s.fCodePh} onChange={(e) => setForm({ ...form, code: e.target.value.toUpperCase() })} /></label>
        <label className="text-xs">{s.fName}<Input className="mt-1" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /></label>
        <label className="text-xs">{s.fNameZh}<Input className="mt-1" value={form.names['zh-CN'] ?? ''} onChange={(e) => setForm({ ...form, names: { ...form.names, 'zh-CN': e.target.value } })} /></label>
        <label className="text-xs">{s.fNameEn}<Input className="mt-1" value={form.names['en-US'] ?? ''} onChange={(e) => setForm({ ...form, names: { ...form.names, 'en-US': e.target.value } })} /></label>
        <label className="text-xs">{s.fNameMy}<Input className="mt-1" value={form.names['ms-MY'] ?? ''} onChange={(e) => setForm({ ...form, names: { ...form.names, 'ms-MY': e.target.value } })} /></label>
        <label className="text-xs">{s.fSort}<Input className="mt-1" type="number" value={form.sortNo} onChange={(e) => setForm({ ...form, sortNo: Number(e.target.value) })} /></label>
        <label className="text-xs">{s.fEnabled}
          <div className="mt-1"><Dropdown options={[{ value: 'on', label: s.enabledOn }, { value: 'off', label: s.enabledOff }]} value={form.enabled ? 'on' : 'off'} onChange={(v) => setForm({ ...form, enabled: v === 'on' })} ariaLabel={s.fEnabled} /></div>
        </label>
      </div>
      <div className="mt-4 flex gap-3">
        <ToolbarButton primary disabled={busy} onClick={save}>{busy ? t.pages.account.submitting : s.save}</ToolbarButton>
        <ToolbarButton onClick={() => setEditing(undefined)}>{t.pages.company.cancel}</ToolbarButton>
      </div>
    </Card>}
  </div>
}
