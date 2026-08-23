// 数据备份两个抽屉:新建备份(候选表多选) + 导入恢复(.jsonl.gz 上传)。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../api/client'
import { useT } from '../../i18n'
import { Drawer } from '../../components/Drawer'
import { ErrorBanner } from '../../components/business/page-head'
import { filterTables, isBusyError } from './logic'
import { BTN, BTN_PRIMARY } from './styles'

interface DrawerProps {
  onClose: () => void
  onChanged: (msg: string) => void
  onBusy: () => void
}

/** 新建备份抽屉:候选表多选(可搜索),空选 = 全部表。 */
export function BackupCreateDrawer({ onClose, onChanged, onBusy }: DrawerProps) {
  const t = useT()
  const b = t.pages.backup
  const [tables, setTables] = useState<string[]>([])
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [kw, setKw] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [err, setErr] = useState('')

  useEffect(() => {
    apiFetch<{ items: string[] }>('/backup/tables')
      .then((d) => setTables(d?.items ?? []))
      .catch((e) => setErr(e instanceof Error ? e.message : b.loadFail))
  }, [b.loadFail])

  const visible = useMemo(() => filterTables(tables, kw), [tables, kw])
  const toggle = (name: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(name)) next.delete(name)
      else next.add(name)
      return next
    })
  }

  const submit = () => {
    setSubmitting(true)
    setErr('')
    apiFetch<{ id: number }>('/backup/jobs', { method: 'POST', body: { tables: [...selected] } })
      .then(() => { onClose(); onChanged(b.selectedCount.replace('{count}', String(selected.size))) })
      .catch((e) => {
        if (isBusyError(e)) { onBusy(); onClose(); return }
        setErr(e instanceof Error ? e.message : b.loadFail)
      })
      .finally(() => setSubmitting(false))
  }

  return (
    <Drawer
      title={b.createTitle}
      onClose={onClose}
      footer={
        <>
          <button className={BTN} onClick={onClose} disabled={submitting}>{t.common.confirmDialog.cancel}</button>
          <button className={BTN_PRIMARY} onClick={submit} disabled={submitting}>
            {submitting ? b.createSubmitting : b.createSubmit}
          </button>
        </>
      }
    >
      <p className="m-0 text-xs text-[var(--shell-crumb-text)]">{b.createHint}</p>
      {err && <div className="mt-3"><ErrorBanner message={err} /></div>}
      <p className="m-0 mt-3 text-[13px] font-medium text-[var(--shell-heading)]">
        {b.selectedCount.replace('{count}', String(selected.size))}
      </p>
      <input
        className="mt-3 w-full rounded-md border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-3 py-1.5 text-[13px] text-[var(--shell-content-text)] outline-none focus:border-[var(--shell-input-border-focus)]"
        placeholder={b.tableSearch}
        value={kw}
        onChange={(e) => setKw(e.target.value)}
      />
      <ul className="mt-3 max-h-[50vh] list-none overflow-y-auto p-0" role="listbox">
        {visible.map((name) => (
          <li key={name}>
            <label className="flex cursor-pointer items-center gap-2 rounded px-2 py-1.5 text-[13px] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
              <input type="checkbox" checked={selected.has(name)} onChange={() => toggle(name)} />
              <span className="font-mono">{name}</span>
            </label>
          </li>
        ))}
        {visible.length === 0 && <li className="px-2 py-4 text-center text-xs text-[var(--shell-group-title)]">{b.empty}</li>}
      </ul>
    </Drawer>
  )
}

/** 导入恢复抽屉:上传 .jsonl.gz 归档,追加语义导入。 */
export function RestoreDrawer({ onClose, onChanged, onBusy }: DrawerProps) {
  const t = useT()
  const b = t.pages.backup
  const [file, setFile] = useState<File | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [err, setErr] = useState('')

  const submit = () => {
    if (!file) { setErr(b.restoreNoFile); return }
    setSubmitting(true)
    setErr('')
    const form = new FormData()
    form.append('file', file)
    apiFetch<{ id: number }>('/backup/restore', { method: 'POST', body: form })
      .then(() => { onClose(); onChanged(file.name) })
      .catch((e) => {
        if (isBusyError(e)) { onBusy(); onClose(); return }
        setErr(e instanceof Error ? e.message : b.loadFail)
      })
      .finally(() => setSubmitting(false))
  }

  return (
    <Drawer
      title={b.restoreTitle}
      onClose={onClose}
      footer={
        <>
          <button className={BTN} onClick={onClose} disabled={submitting}>{t.common.confirmDialog.cancel}</button>
          <button className={BTN_PRIMARY} onClick={submit} disabled={submitting}>
            {submitting ? b.restoreUploading : b.restoreSubmit}
          </button>
        </>
      }
    >
      <p className="m-0 text-xs text-[var(--shell-crumb-text)]">{b.restoreHint}</p>
      <input
        className="mt-4 w-full text-[13px] text-[var(--shell-content-text)]"
        type="file"
        accept=".gz,.jsonl.gz,application/gzip"
        onChange={(e) => setFile(e.target.files?.[0] ?? null)}
      />
      {err && <div className="mt-3"><ErrorBanner message={err} /></div>}
    </Drawer>
  )
}
