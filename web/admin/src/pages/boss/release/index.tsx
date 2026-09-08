// 客户端版本管理:client_releases 列表 + 抽屉式上传/灰度白名单状态编辑。
// 菜单 key=release,权限 menu:release;契约见 docs/contract/fields.md 8F。
import { useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'
import { useT } from '../../../i18n'
import { PageHead } from '../../org/shared'
import { TableStateRow, ToolbarButton, ErrorBanner } from '../../../components/business'
import { Dropdown } from '../../../components/Dropdown'
import { Drawer } from '../../../components/Drawer'
import { Card } from '../../../components/ui/card'
import { Input } from '../../../components/ui/input'
import { Textarea } from '../../../components/ui/textarea'
import { Checkbox } from '../../../components/ui/checkbox'
import { Button } from '../../../components/ui/button'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { apiBaseUrl, apiFetch } from '../../../api/client'
import { DialogPicker } from '../../../components/pickers/DialogPicker'
import { fmtTime } from '../../../lib/format'
import {
  listReleases, patchRelease, uploadRelease,
  type ClientReleaseDTO, type ReleasePatchInput,
} from './logic'

type Status = ClientReleaseDTO['status']

// 灰度白名单账号(sys.yaml AccountRow 子集;GET /accounts 裸数组信封)。
type AccountRow = { id: number; username: string; realName: string; phone: string }

const EMPTY_FORM = { app: 'user', version: '', versionCode: '', minSupportedCode: '', notes: '', file: undefined as File | undefined }

// 状态机(terms.md §4):ROLLED_BACK 终态不可再投放;灰度可全量/回滚;草稿可灰度/全量。
const ALLOWED_NEXT: Record<Status, Status[]> = {
  DRAFT: ['DRAFT', 'GRAY', 'PUBLISHED'],
  GRAY: ['GRAY', 'PUBLISHED', 'ROLLED_BACK'],
  PUBLISHED: ['PUBLISHED', 'ROLLED_BACK'],
  ROLLED_BACK: ['ROLLED_BACK'],
}

export default function ClientReleasePage() {
  const t = useT(); const s = t.pages.releasePage
  const [rows, setRows] = useState<ClientReleaseDTO[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [appFilter, setAppFilter] = useState('')
  const [uploadOpen, setUploadOpen] = useState(false)
  const [editing, setEditing] = useState<ClientReleaseDTO | null>(null)
  const [patch, setPatch] = useState<ReleasePatchInput>({})
  const [uploading, setUploading] = useState(false)
  const [formError, setFormError] = useState('')
  const [form, setForm] = useState(EMPTY_FORM)
  const [wlOpen, setWlOpen] = useState(false)
  /** 上次确认的白名单实体:重开抽屉时经 initialItems 预勾选回显(W0-R4)。 */
  const [wlPicked, setWlPicked] = useState<AccountRow[]>([])
  const accRef = useRef<AccountRow[]>([])
  const accLoaded = useRef(false)

  const load = () => {
    setBusy(true); setError('')
    listReleases(appFilter)
      .then((items) => setRows(items ?? []))
      .catch(() => setError(s.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, [appFilter])

  const loadAccounts = async () => {
    try {
      const list = await apiFetch<AccountRow[]>('/accounts', {})
      accRef.current = list ?? []
      accLoaded.current = true
    } catch { accRef.current = []; accLoaded.current = false }
  }
  // 已选回显 label 的映射缓存:抽屉打开时预拉(无 menu:account 权限失败则回退 #id 显示)。
  useEffect(() => { if (editing) loadAccounts() }, [editing]) // eslint-disable-line react-hooks/exhaustive-deps
  const openWlPicker = () => { accLoaded.current = false; accRef.current = []; setWlOpen(true) }
  // DialogPicker query 包装:账号接口无 keyword/分页参数,前端过滤+切片(items/total 信封)。
  const accountQuery = async (q: { keyword: string; filters: Record<string, string>; page: number; pageSize: number }) => {
    if (!accLoaded.current) await loadAccounts()
    const kw = q.keyword.trim().toLowerCase()
    const hit = kw
      ? accRef.current.filter((a) => [a.username, a.realName, a.phone].some((x) => (x || '').toLowerCase().includes(kw)))
      : accRef.current
    return { items: hit.slice((q.page - 1) * q.pageSize, q.page * q.pageSize), total: hit.length }
  }
  const wlCurrent = patch.whitelistIds ?? editing?.whitelistIds ?? []
  const wlLabels = (ids: number[]) => {
    const byId = new Map(accRef.current.map((a) => [String(a.id), a.realName + '(' + a.username + ')']))
    return ids.map((id) => byId.get(String(id)) ?? '#' + id)
  }
  const wlColumns = [
    { key: 'username', title: s.colAccount },
    { key: 'realName', title: s.colName },
    { key: 'phone', title: s.colPhone },
  ]

  const closeUpload = () => {
    setUploadOpen(false)
    setForm(EMPTY_FORM)
    setFormError('')
  }

  const submitUpload = async () => {
    if (!form.file || !form.version || !form.versionCode) { setFormError(s.uploadFail); return }
    setUploading(true); setFormError('')
    try {
      await uploadRelease({
        app: form.app as 'user' | 'worker', version: form.version,
        versionCode: Number(form.versionCode), minSupportedCode: Number(form.minSupportedCode) || 1,
        notes: form.notes, file: form.file,
      })
      toast.success(s.toastUploadOk)
      closeUpload()
      load()
    } catch (e) { setFormError(e instanceof Error ? e.message : s.uploadFail) }
    setUploading(false)
  }

  const savePatch = async () => {
    if (!editing) return
    setBusy(true)
    try {
      await patchRelease(editing.id, patch)
      toast.success(s.toastPatchOk)
      setEditing(null); setPatch({}); load()
    } catch (e) { setError(e instanceof Error ? e.message : s.saveFail); setBusy(false) }
  }

  const appLabel = (v: string) => (v === 'worker' ? s.appWorker : s.appUser)
  const stLabel = (v: Status) => v === 'DRAFT' ? s.stDraft : v === 'GRAY' ? s.stGray : v === 'PUBLISHED' ? s.stPublished : s.stRolledBack

  return <div>
    <PageHead title={s.title} desc={s.desc} />
    {error && <ErrorBanner message={error} />}

    <div className="mb-3 flex items-center gap-3">
      <Dropdown value={appFilter}
        ariaLabel={s.fApp} onChange={(v) => setAppFilter(v)}
        options={[{ value: '', label: s.columns[0] }, { value: 'user', label: s.appUser }, { value: 'worker', label: s.appWorker }]} />
      <span className="flex-1" />
      <ToolbarButton primary onClick={() => setUploadOpen(true)}>+ {s.upload}</ToolbarButton>
    </div>

    <Card className="p-4">
      <div className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>{s.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((r) => <TableRow key={r.id}>
              <TableCell>{appLabel(r.app)}</TableCell>
              <TableCell>v{r.version} ({r.versionCode})</TableCell>
              <TableCell>{r.minSupportedCode}</TableCell>
              <TableCell>{stLabel(r.status)}</TableCell>
              <TableCell>{r.status === 'GRAY' ? `${r.rolloutPercent}% / ${r.whitelistIds.length}` : '—'}</TableCell>
              <TableCell title={r.sha256}>{(r.apkSize / 1048576).toFixed(1)}MB</TableCell>
              <TableCell>{fmtTime(r.updatedAt)}</TableCell>
              <TableCell>
                <button className="cursor-pointer border-none bg-none text-[13px] text-[var(--shell-content-text)] underline-offset-2 hover:text-[var(--shell-heading)] hover:underline" onClick={() => { setEditing(r); setPatch({}) }}>{s.edit}</button>
                <a className="ml-3 text-[13px] text-[var(--shell-fab-bg)] underline-offset-2 hover:underline" href={`${apiBaseUrl()}/client-releases/${r.id}/apk`} download>{s.download}</a>
              </TableCell>
            </TableRow>)}
            {!rows.length && <TableStateRow colSpan={s.columns.length} loading={busy} text={s.empty} />}
          </TableBody>
        </Table>
      </div>
    </Card>

    {uploadOpen && <Drawer title={s.upload} onClose={closeUpload}
      footer={<>
        <Button variant="outline" size="sm" onClick={closeUpload}>{s.cancel}</Button>
        <Button size="sm" disabled={uploading} onClick={submitUpload}>{uploading ? t.pages.account.submitting : s.upload}</Button>
      </>}>
      <div className="grid gap-3">
        <label className="text-xs">{s.fApp}
          <div className="mt-1"><Dropdown value={form.app} ariaLabel={s.fApp}
            options={[{ value: 'user', label: s.appUser }, { value: 'worker', label: s.appWorker }]}
            onChange={(v) => setForm({ ...form, app: v })} /></div>
        </label>
        <label className="text-xs">{s.fVersion}<Input className="mt-1" placeholder="1.2.0" value={form.version} onChange={(e) => setForm({ ...form, version: e.target.value })} /></label>
        <label className="text-xs">{s.fVersionCode}<Input className="mt-1" placeholder="12" inputMode="numeric" value={form.versionCode} onChange={(e) => setForm({ ...form, versionCode: e.target.value })} /></label>
        <label className="text-xs">{s.fMinSupported}<Input className="mt-1" placeholder="10" inputMode="numeric" value={form.minSupportedCode} onChange={(e) => setForm({ ...form, minSupportedCode: e.target.value })} /></label>
        <label className="text-xs">{s.fFile}
          <input className="mt-1 block w-full text-xs text-[var(--shell-content-text)] file:mr-3 file:cursor-pointer file:rounded-sm file:border file:border-[var(--shell-input-border)] file:bg-[var(--shell-input-bg)] file:px-3 file:py-1.5 file:text-xs" type="file" accept=".apk"
            onChange={(e) => setForm({ ...form, file: e.target.files?.[0] })} />
        </label>
        <label className="block text-xs">{s.fNotes}<Textarea className="mt-1" rows={3} value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} /></label>
      </div>
      {formError && <p className="mt-3 text-sm text-[var(--color-danger)]">{formError}</p>}
    </Drawer>}

    {editing && <Drawer title={s.edit} onClose={() => { setEditing(null); setPatch({}) }}
      footer={<>
        <Button variant="outline" size="sm" onClick={() => { setEditing(null); setPatch({}) }}>{s.cancel}</Button>
        <Button size="sm" disabled={busy} onClick={savePatch}>{busy ? t.pages.account.submitting : s.save}</Button>
      </>}>
      <div className="grid gap-3">
        <label className="text-xs">{s.fStatus}
          <div className="mt-1"><Dropdown value={patch.status ?? editing.status} ariaLabel={s.fStatus} onChange={(v) => setPatch({ ...patch, status: v as Status })}
            options={[
              { value: 'DRAFT', label: s.stDraft }, { value: 'GRAY', label: s.stGray },
              { value: 'PUBLISHED', label: s.stPublished }, { value: 'ROLLED_BACK', label: s.stRolledBack },
            ].map((o) => ({ ...o, disabled: !ALLOWED_NEXT[editing.status].includes(o.value as Status) }))} /></div>
        </label>
        <label className="text-xs">{s.fRollout}<Input className="mt-1" inputMode="numeric" placeholder={String(editing.rolloutPercent)} onChange={(e) => setPatch({ ...patch, rolloutPercent: Number(e.target.value) })} /></label>
        <label className="text-xs">{s.fWhitelist}
          <div className="mt-1 flex flex-wrap items-center gap-1.5">
            <ToolbarButton onClick={openWlPicker}>{s.wlPickBtn}</ToolbarButton>
            <span className="text-[11px] text-[var(--shell-group-title)]">{s.wlSelectedCount.replace('{n}', String(wlCurrent.length))}</span>
            {wlCurrent.length > 0 && <span className="text-[11px] text-[var(--shell-content-text)]">{wlLabels(wlCurrent).join('; ')}</span>}
          </div>
        </label>
        <label className="text-xs">{s.fMinSupported}<Input className="mt-1" inputMode="numeric" placeholder={String(editing.minSupportedCode)} onChange={(e) => setPatch({ ...patch, minSupportedCode: Number(e.target.value) })} /></label>
        <label className="text-xs">{s.fNotes}<Textarea className="mt-1" rows={3} placeholder={editing.notes} onChange={(e) => setPatch({ ...patch, notes: e.target.value })} /></label>
        <label className="mt-1 flex items-center gap-2 text-xs">
          <Checkbox checked={patch.force ?? editing.force} onCheckedChange={(v) => setPatch({ ...patch, force: v === true })} />
          {s.fForce}
        </label>
      </div>
      <p className="mt-2 text-xs text-[var(--shell-group-title)]">{s.grayHint}</p>
    </Drawer>}

    {wlOpen && editing && <DialogPicker<AccountRow>
      open={wlOpen} mode="multiple" title={s.wlPickerTitle}
      onClose={() => setWlOpen(false)}
      onPick={(picked) => { setPatch({ ...patch, whitelistIds: picked.map((a) => a.id) }); setWlPicked(picked); setWlOpen(false) }}
      initialItems={wlPicked}
      columns={wlColumns}
      query={accountQuery}
      rowKey={(a) => String(a.id)}
      rowLabel={(a) => a.realName + '(' + a.username + ')'}
      texts={t.pages.pickers.dialog}
    />}
  </div>
}
