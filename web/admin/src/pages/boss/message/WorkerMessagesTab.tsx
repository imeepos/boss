// 消息中心 · 师傅消息页签:查询(GET /worker-messages?workerId=) + 抽屉式下发(POST /worker-messages)。
import { useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Dropdown } from '../../../components/Dropdown'
import { Pagination } from '../../../components/Pagination'
import { ResourcePicker } from '../../../components/ResourcePicker'
import { StatusTag } from '../../../components/StatusTag'
import { DataTable } from '../../../components/business/data-table'
import { Drawer } from '../../../components/Drawer'
import { searchWorkers } from '../../../api/pickers'
import { useT } from '../../../i18n'
import type { Translations } from '../../../i18n/types'
import { fmtTime, filterMessages, toId, MESSAGE_LEVELS, type WorkerMessageEntry } from './logic'

type Ns = Translations['pages']['message']

// 筛选控件/按钮统一令牌化 className(收敛原内联 style,禁裸色值);NoticesTab 复用 ctl/primaryBtn。
export const ctl = 'h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'
export const ctlBtn = 'h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]'
export const primaryBtn = 'h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]'
const formLabel = 'text-[13px] text-[var(--shell-group-title)]'
const fieldInput = ctl + ' mt-1 w-full'
const errBanner = 'mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]'

export function WorkerMessagesTab({ t }: { t: Ns }) {
  const p = useT().pages.pickers
  const cancelText = useT().common.confirmDialog.cancel
  const [rows, setRows] = useState<WorkerMessageEntry[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [level, setLevel] = useState('')
  const [read, setRead] = useState('')
  const [workerId, setWorkerId] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  // 下发表单(抽屉)
  const [sendOpen, setSendOpen] = useState(false)
  const [sendWorker, setSendWorker] = useState('')
  const [sendLevel, setSendLevel] = useState<string>('INFO')
  const [sendTitle, setSendTitle] = useState('')
  const [sendContent, setSendContent] = useState('')
  const [sending, setSending] = useState(false)
  const [hint, setHint] = useState('')

  const load = () => {
    setError('')
    setHint('')
    apiFetch<{ items: WorkerMessageEntry[] }>('/worker-messages', { query: { workerId: toId(workerId) || undefined } })
      .then((d) => setRows(d?.items ?? []))
      .catch(() => setError(t.loadFail))
  }

  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps
  // 筛选师傅变化后自动重查(替代原输入框 onBlur 触发)。
  const reloadRef = useRef(false)
  useEffect(() => {
    if (!reloadRef.current) return
    reloadRef.current = false
    load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [workerId])

  const filtered = filterMessages(rows, keyword, level, read)
  const slice = filtered.slice((page - 1) * pageSize, page * pageSize)

  const closeSend = () => {
    setSendOpen(false)
    setSendWorker('')
    setSendTitle('')
    setSendContent('')
    setHint('')
  }

  const send = () => {
    const wid = toId(sendWorker)
    if (!wid) { setHint(t.sendNeedWorker); return }
    if (!sendTitle.trim()) { setHint(t.sendNeedTitle); return }
    setSending(true)
    setHint('')
    apiFetch('/worker-messages', {
      method: 'POST',
      body: { workerId: wid, level: sendLevel, title: sendTitle.trim(), content: sendContent.trim() },
    })
      .then(() => {
        setSending(false)
        closeSend()
        toast.success(t.sent)
        load()
      })
      .catch(() => { setSending(false); setHint(t.sendFail) })
  }

  return (
    <div className="rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-4 shadow-[var(--shell-card-shadow)]">
      <div className="mb-3 flex flex-wrap gap-2">
        <input className={ctl} placeholder={t.searchPlaceholder} value={keyword}
          onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
        <Dropdown
          value={level}
          options={[{ value: '', label: t.allLevels }, ...MESSAGE_LEVELS.map((lv) => ({ value: lv, label: lv }))]}
          onChange={(v) => { setLevel(v); setPage(1) }}
          ariaLabel={t.allLevels}
        />
        <Dropdown
          value={read}
          options={[
            { value: '', label: t.allRead },
            { value: 'read', label: t.read },
            { value: 'unread', label: t.unread },
          ]}
          onChange={(v) => { setRead(v); setPage(1) }}
          ariaLabel={t.allRead}
        />
        <ResourcePicker
          value={workerId}
          onChange={(v) => { setWorkerId(v); setPage(1); reloadRef.current = true }}
          search={searchWorkers}
          toOption={(w) => ({ value: String(w.id), label: `${w.name} · ${w.staffNo}` })}
          ariaLabel={t.workerIdPlaceholder}
          emptyLabel={p.common.all}
          searchPlaceholder={p.common.placeholder}
          errorText={t.loadFail}
        />
        <span className="spacer" />
        <button className={ctlBtn} onClick={load}>{t.refresh}</button>
      </div>
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <button className={primaryBtn} onClick={() => setSendOpen(true)}>+ {t.send}</button>
      </div>
      <div className="mb-3 text-sm font-semibold text-[var(--shell-heading)]">
        {t.cardTitle}
        <span className="ml-2 text-xs font-normal text-[var(--shell-group-title)]">
          {filtered.length ? t.matched.replace('{count}', String(filtered.length)) : ''}
        </span>
      </div>
      {error ? <div className={errBanner}>{error}</div> : (
        <>
          <DataTable
            emptyText={t.empty}
            rows={slice as unknown as Record<string, unknown>[]}
            columns={[
              { key: 'level', label: t.msgColumns[0], render: (r) => <StatusTag domain="message" value={String(r.level)} /> },
              { key: 'workerId', label: t.msgColumns[1], render: (r) => `#${r.workerId}` },
              { key: 'title', label: t.msgColumns[2], render: (r) => String(r.title ?? '') },
              { key: 'content', label: t.msgColumns[3], render: (r) => <span className="text-[var(--shell-group-title)]">{String(r.content ?? '')}</span> },
              { key: 'sentAt', label: t.msgColumns[4], render: (r) => fmtTime(String(r.sentAt)) },
              { key: 'read', label: t.msgColumns[5], render: (r) => (r.read ? t.read : t.unread) },
            ]}
          />
          <Pagination total={filtered.length} page={page} pageSize={pageSize} onPage={setPage} onSize={setPageSize}
            rangeText={t.rangeText} prevText={t.prev} nextText={t.next} perPageText={t.perPage}
            jumpText={t.jump} pageUnitText={t.pageUnit} />
        </>
      )}
      {sendOpen && (
        <Drawer title={t.send} onClose={closeSend}
          footer={
            <>
              <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={closeSend}>
                {cancelText}
              </button>
              <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={sending} onClick={send}>
                {sending ? t.sending : t.send}
              </button>
            </>
          }>
          <div className="grid gap-3">
            <label className={formLabel}>
              {t.msgColumns[1]}
              <div className="mt-1">
                <ResourcePicker
                  value={sendWorker}
                  onChange={setSendWorker}
                  search={searchWorkers}
                  toOption={(w) => ({ value: String(w.id), label: `${w.name} · ${w.staffNo}` })}
                  ariaLabel={t.workerIdPlaceholder}
                  emptyLabel={p.common.all}
                  searchPlaceholder={p.common.placeholder}
                  errorText={t.loadFail}
                />
              </div>
            </label>
            <label className={formLabel}>
              {t.sendLevel}
              <div className="mt-1">
                <Dropdown
                  value={sendLevel}
                  options={MESSAGE_LEVELS.map((lv) => ({ value: lv, label: lv }))}
                  onChange={(v) => setSendLevel(v)}
                  ariaLabel={t.sendLevel}
                />
              </div>
            </label>
            <label className={formLabel}>
              {t.msgColumns[2]}
              <input className={fieldInput} placeholder={t.sendTitlePlaceholder} value={sendTitle}
                onChange={(e) => setSendTitle(e.target.value)} />
            </label>
            <label className={formLabel}>
              {t.msgColumns[3]}
              <input className={fieldInput} placeholder={t.sendContentPlaceholder} value={sendContent}
                onChange={(e) => setSendContent(e.target.value)} />
            </label>
            {hint && <span className="text-xs text-[var(--color-danger)]">{hint}</span>}
          </div>
        </Drawer>
      )}
    </div>
  )
}
