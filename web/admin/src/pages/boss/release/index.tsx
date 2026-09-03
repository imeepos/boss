// 客户端版本管理:client_releases 列表 + 抽屉式上传/灰度白名单状态编辑。
// 菜单 key=release,权限 menu:release;契约见 docs/contract/fields.md 8F。
import { useEffect, useState } from 'react'
import { useT } from '../../../i18n'
import { PageHead } from '../../org/shared'
import { TableStateRow, ToolbarButton } from '../../../components/business'
import { Dropdown } from '../../../components/Dropdown'
import { Drawer } from '../../../components/Drawer'
import { apiBaseUrl } from '../../../api/client'
import { fmtTime } from '../../../lib/format'
import {
  listReleases, patchRelease, uploadRelease,
  type ClientReleaseDTO, type ReleasePatchInput,
} from './logic'

type Status = ClientReleaseDTO['status']

const EMPTY_FORM = { app: 'user', version: '', versionCode: '', minSupportedCode: '', notes: '', file: undefined as File | undefined }

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

  const load = () => {
    setBusy(true); setError('')
    listReleases(appFilter)
      .then((items) => setRows(items ?? []))
      .catch(() => setError(s.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, [appFilter])

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
      setEditing(null); setPatch({}); load()
    } catch (e) { setError(e instanceof Error ? e.message : s.saveFail); setBusy(false) }
  }

  const appLabel = (v: string) => (v === 'worker' ? s.appWorker : s.appUser)
  const stLabel = (v: Status) => v === 'DRAFT' ? s.stDraft : v === 'GRAY' ? s.stGray : v === 'PUBLISHED' ? s.stPublished : s.stRolledBack
  const inputCls = 'h-8 w-full rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'
  const td = 'border-b border-[var(--shell-side-border)] px-3 py-2'
  const btnPrimary = 'h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]'
  const btnPlain = 'h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]'

  return <div>
    <PageHead title={s.title} desc={s.desc} />
    {error && <div className="mb-3 text-sm text-[var(--color-danger)]">{error}</div>}

    <div className="mb-3 flex items-center gap-3">
      <Dropdown value={appFilter === 'worker' ? s.appWorker : appFilter === 'user' ? s.appUser : s.columns[0]}
        ariaLabel={s.fApp} onChange={(v) => setAppFilter(v)}
        options={[{ value: '', label: s.columns[0] }, { value: 'user', label: s.appUser }, { value: 'worker', label: s.appWorker }]} />
      <span className="flex-1" />
      <ToolbarButton primary onClick={() => setUploadOpen(true)}>+ {s.upload}</ToolbarButton>
    </div>

    <div className="rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-4">
      <div className="overflow-x-auto">
        <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
          <thead><tr>{s.columns.map((x) => <th key={x} className="border-b border-[var(--shell-side-border)] px-3 py-2 text-left text-xs">{x}</th>)}</tr></thead>
          <tbody>
            {rows.map((r) => <tr key={r.id}>
              <td className={td}>{appLabel(r.app)}</td>
              <td className={td}>v{r.version} ({r.versionCode})</td>
              <td className={td}>{r.minSupportedCode}</td>
              <td className={td}>{stLabel(r.status)}</td>
              <td className={td}>{r.status === 'GRAY' ? `${r.rolloutPercent}% / ${r.whitelistIds.length}` : '—'}</td>
              <td className={td} title={r.sha256}>{(r.apkSize / 1048576).toFixed(1)}MB</td>
              <td className={td}>{fmtTime(r.updatedAt)}</td>
              <td className={td}>
                <button className="cursor-pointer border-none bg-none text-[13px] text-[var(--shell-content-text)] underline-offset-2 hover:text-[var(--shell-heading)] hover:underline" onClick={() => { setEditing(r); setPatch({}) }}>{s.edit}</button>
                <a className="ml-3 text-[13px] text-[var(--shell-fab-bg)] underline-offset-2 hover:underline" href={`${apiBaseUrl()}/client-releases/${r.id}/apk`} download>{s.download}</a>
              </td>
            </tr>)}
            {!rows.length && <TableStateRow colSpan={s.columns.length} loading={busy} text={s.empty} />}
          </tbody>
        </table>
      </div>
    </div>

    {uploadOpen && <Drawer title={s.upload} onClose={closeUpload}
      footer={<>
        <button className={btnPlain} onClick={closeUpload}>{s.cancel}</button>
        <button className={btnPrimary} disabled={uploading} onClick={submitUpload}>{s.upload}</button>
      </>}>
      <div className="grid gap-3">
        <label className="text-xs">{s.fApp}
          <div className="mt-1"><Dropdown value={appLabel(form.app)} ariaLabel={s.fApp}
            options={[{ value: 'user', label: s.appUser }, { value: 'worker', label: s.appWorker }]}
            onChange={(v) => setForm({ ...form, app: v })} /></div>
        </label>
        <label className="text-xs">{s.fVersion}<input className={inputCls + ' mt-1'} placeholder="1.2.0" value={form.version} onChange={(e) => setForm({ ...form, version: e.target.value })} /></label>
        <label className="text-xs">{s.fVersionCode}<input className={inputCls + ' mt-1'} placeholder="12" inputMode="numeric" value={form.versionCode} onChange={(e) => setForm({ ...form, versionCode: e.target.value })} /></label>
        <label className="text-xs">{s.fMinSupported}<input className={inputCls + ' mt-1'} placeholder="10" inputMode="numeric" value={form.minSupportedCode} onChange={(e) => setForm({ ...form, minSupportedCode: e.target.value })} /></label>
        <label className="text-xs">{s.fFile}
          <input className="mt-1 block w-full text-xs text-[var(--shell-content-text)] file:mr-3 file:cursor-pointer file:rounded-sm file:border file:border-[var(--shell-input-border)] file:bg-[var(--shell-input-bg)] file:px-3 file:py-1.5 file:text-xs" type="file" accept=".apk"
            onChange={(e) => setForm({ ...form, file: e.target.files?.[0] })} />
        </label>
        <label className="block text-xs">{s.fNotes}<input className={inputCls + ' mt-1'} value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} /></label>
      </div>
      {formError && <p className="mt-3 text-sm text-[var(--color-danger)]">{formError}</p>}
    </Drawer>}

    {editing && <Drawer title={s.edit} onClose={() => { setEditing(null); setPatch({}) }}
      footer={<>
        <button className={btnPlain} onClick={() => { setEditing(null); setPatch({}) }}>{s.cancel}</button>
        <button className={btnPrimary} disabled={busy} onClick={savePatch}>{s.save}</button>
      </>}>
      <div className="grid gap-3">
        <label className="text-xs">{s.fStatus}
          <div className="mt-1"><Dropdown value={stLabel(patch.status ?? editing.status)} ariaLabel={s.fStatus} onChange={(v) => setPatch({ ...patch, status: v as Status })}
            options={[
              { value: 'DRAFT', label: s.stDraft }, { value: 'GRAY', label: s.stGray },
              { value: 'PUBLISHED', label: s.stPublished }, { value: 'ROLLED_BACK', label: s.stRolledBack },
            ]} /></div>
        </label>
        <label className="text-xs">{s.fRollout}<input className={inputCls + ' mt-1'} inputMode="numeric" placeholder={String(editing.rolloutPercent)} onChange={(e) => setPatch({ ...patch, rolloutPercent: Number(e.target.value) })} /></label>
        <label className="text-xs">{s.fWhitelist}<input className={inputCls + ' mt-1'} placeholder={editing.whitelistIds.join(',')} onChange={(e) => setPatch({ ...patch, whitelistIds: e.target.value.split(',').map((x) => Number(x.trim())).filter((n) => n > 0) })} /></label>
        <label className="text-xs">{s.fMinSupported}<input className={inputCls + ' mt-1'} inputMode="numeric" placeholder={String(editing.minSupportedCode)} onChange={(e) => setPatch({ ...patch, minSupportedCode: Number(e.target.value) })} /></label>
        <label className="text-xs">{s.fNotes}<input className={inputCls + ' mt-1'} placeholder={editing.notes} onChange={(e) => setPatch({ ...patch, notes: e.target.value })} /></label>
        <label className="mt-1 flex items-center gap-2 text-xs">
          <input type="checkbox" checked={patch.force ?? editing.force} onChange={(e) => setPatch({ ...patch, force: e.target.checked })} />
          {s.fForce}
        </label>
      </div>
      <p className="mt-2 text-xs text-[var(--shell-group-title)]">{s.grayHint}</p>
    </Drawer>}
  </div>
}
