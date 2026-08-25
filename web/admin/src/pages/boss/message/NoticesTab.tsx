// 消息中心 · 公告页签:列表含已下架(GET /notices) + 抽屉式发布(POST) + 上下架(PUT /notices/{id}/toggle)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { DataTable } from '../../../components/business/data-table'
import { Drawer } from '../../../components/Drawer'
import { useT } from '../../../i18n'
import type { Translations } from '../../../i18n/types'
import { filterNotices, fmtTime, type NoticeEntry } from './logic'
import { ctl } from './WorkerMessagesTab'

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
      .catch(() => setError(t.loadFail))
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
      .then(() => { setBusy(false); closeForm(); setHint(t.published); load() })
      .catch(() => { setBusy(false); setHint(t.publishFail) })
  }

  const toggle = (id: number) => {
    apiFetch(`/notices/${id}/toggle`, { method: 'PUT' })
      .then(load)
      .catch(() => setHint(t.toggleFail))
  }

  return (
    <div style={{ background: '#fff', border: '1px solid #f0f0f0', borderRadius: 8, padding: 16 }}>
      <div style={{ display: 'flex', gap: 8, marginBottom: 12, flexWrap: 'wrap', alignItems: 'center' }}>
        <input style={{ ...ctl, width: 200 }} placeholder={t.searchPlaceholder} value={keyword}
          onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
        <label style={{ fontSize: 13, color: '#666', display: 'flex', alignItems: 'center', gap: 4 }}>
          <input type="checkbox" checked={activeOnly}
            onChange={(e) => { setActiveOnly(e.target.checked); setPage(1) }} />
          {t.onShelf}
        </label>
        <span style={{ flex: 1 }} />
        <button style={{ ...ctl, cursor: 'pointer', background: '#1677ff', borderColor: '#1677ff', color: '#fff' }}
          onClick={() => setOpen(true)}>+ {t.publish}</button>
        {hint && <span style={{ fontSize: 12, color: hint === t.published ? '#52c41a' : '#e54545' }}>{hint}</span>}
      </div>
      {error ? <div style={{ color: '#e54545', fontSize: 13, padding: '12px 0' }}>{error}</div> : (
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
                <a style={{ color: '#1677ff', cursor: 'pointer' }} onClick={() => toggle(Number(r.id))}>
                  {r.active ? t.offShelf : t.onShelf}
                </a>
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
          <div style={{ display: 'grid', gap: 12 }}>
            <label style={{ fontSize: 13, color: '#666' }}>
              {t.noticeColumns[0]}
              <input style={{ ...ctl, width: '100%', marginTop: 4 }} placeholder={t.noticeTitlePlaceholder} value={title}
                onChange={(e) => setTitle(e.target.value)} />
            </label>
            <label style={{ fontSize: 13, color: '#666' }}>
              {t.noticeColumns[1]}
              <input style={{ ...ctl, width: '100%', marginTop: 4 }} placeholder={t.noticeCategoryPlaceholder} value={category}
                onChange={(e) => setCategory(e.target.value)} />
            </label>
            {hint && hint !== t.published && (
              <span style={{ fontSize: 12, color: '#e54545' }}>{hint}</span>
            )}
          </div>
        </Drawer>
      )}
    </div>
  )
}
