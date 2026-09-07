// 消息中心 · 公告页签:列表含已下架(GET /notices) + 抽屉式发布(POST) + 上下架(PUT /notices/{id}/toggle)。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { DataTable } from '../../../components/business/data-table'
import { ActionLink } from '../../../components/business/page-head'
import { Drawer } from '../../../components/Drawer'
import { useT } from '../../../i18n'
import type { Translations } from '../../../i18n/types'
import { filterNotices, fmtTime, type NoticeEntry } from './logic'
import { ctl, primaryBtn } from './WorkerMessagesTab'

type Ns = Translations['pages']['message']

export function NoticesTab({ t }: { t: Ns }) {
  const cancelText = useT().common.confirmDialog.cancel
  const [rows, setRows] = useState<NoticeEntry[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [activeOnly, setActiveOnly] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  // 发布表单(抽屉)
  const [open, setOpen] = useState(false)
  const [title, setTitle] = useState('')
  const [category, setCategory] = useState('')
  const [hint, setHint] = useState('')
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setHint('')
    apiFetch<{ items: NoticeEntry[] }>('/notices')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : t.loadFail))
  }

  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const filtered = filterNotices(rows, keyword, activeOnly)
  const slice = filtered.slice((page - 1) * pageSize, page * pageSize)

  const closeForm = () => {
    setOpen(false)
    setTitle('')
    setCategory('')
    setHint('')
  }

  const publish = () => {
    if (!title.trim()) { setHint(t.sendNeedTitle); return }
    setBusy(true)
    setHint('')
    apiFetch('/notices', { method: 'POST', body: { title: title.trim(), category: category.trim() } })
      .then(() => { setBusy(false); closeForm(); toast.success(t.published); load() })
      .catch((e) => { setBusy(false); setHint(e instanceof Error ? e.message : t.publishFail) })
  }

  const toggle = (id: number) => {
    apiFetch(`/notices/${id}/toggle`, { method: 'PUT' })
      .then(load)
      .catch((e) => toast.error(e instanceof Error ? e.message : t.toggleFail))
  }

  return (
    <div className="rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-4 shadow-[var(--shell-card-shadow)]">
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <input className={ctl + ' w-[200px]'} placeholder={t.searchPlaceholder} value={keyword}
          onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
        <label className="flex items-center gap-1 text-[13px] text-[var(--shell-group-title)]">
          <input type="checkbox" checked={activeOnly}
            onChange={(e) => { setActiveOnly(e.target.checked); setPage(1) }} />
          {t.onShelf}
        </label>
        <span className="spacer" />
        <button className={primaryBtn} onClick={() => setOpen(true)}>+ {t.publish}</button>
      </div>
      {error ? <div className="mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
        <>
          <DataTable
            emptyText={t.empty}
            rows={slice as unknown as Record<string, unknown>[]}
            columns={[
              { key: 'title', label: t.noticeColumns[0], render: (r) => String(r.title ?? '') },
              { key: 'category', label: t.noticeColumns[1], render: (r) => String(r.category || '-') },
              { key: 'active', label: t.noticeColumns[2], render: (r) => (
                <StatusTag domain="accountStatus" value={r.active ? '1' : '0'} />
              ) },
              { key: 'publishedAt', label: t.noticeColumns[3], render: (r) => fmtTime(String(r.publishedAt)) },
              { key: 'op', label: t.noticeColumns[4], render: (r) => (
                <ActionLink onClick={() => toggle(Number(r.id))} label={r.active ? t.offShelf : t.onShelf} />
              ) },
            ]}
          />
          <Pagination total={filtered.length} page={page} pageSize={pageSize} onPage={setPage} onSize={setPageSize}
            rangeText={t.rangeText} prevText={t.prev} nextText={t.next} perPageText={t.perPage}
            jumpText={t.jump} pageUnitText={t.pageUnit} />
        </>
      )}
      {open && (
        <Drawer title={t.publish} onClose={closeForm}
          footer={
            <>
              <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={closeForm}>
                {cancelText}
              </button>
              <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy} onClick={publish}>
                {t.publish}
              </button>
            </>
          }>
          <div className="grid gap-3">
            <label className="text-[13px] text-[var(--shell-group-title)]">
              {t.noticeColumns[0]}
              <input className={ctl + ' mt-1 w-full'} placeholder={t.noticeTitlePlaceholder} value={title}
                onChange={(e) => setTitle(e.target.value)} />
            </label>
            <label className="text-[13px] text-[var(--shell-group-title)]">
              {t.noticeColumns[1]}
              <input className={ctl + ' mt-1 w-full'} placeholder={t.noticeCategoryPlaceholder} value={category}
                onChange={(e) => setCategory(e.target.value)} />
            </label>
            {hint && <span className="text-xs text-[var(--color-danger)]">{hint}</span>}
          </div>
        </Drawer>
      )}
    </div>
  )
}
