// 消息中心 · 师傅消息页签:查询(GET /worker-messages?workerId=) + 抽屉式下发(POST /worker-messages)。
import { useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Dropdown } from '../../../components/Dropdown'
import { Pagination } from '../../../components/Pagination'
import { ResourcePicker } from '../../../components/ResourcePicker'
import { StatusTag } from '../../../components/StatusTag'
import { DataTable, CopyButton } from '../../../components/business'
import { Drawer } from '../../../components/Drawer'
import { Card } from '../../../components/ui/card'
import { Input } from '../../../components/ui/input'
import { Textarea } from '../../../components/ui/textarea'
import { Button } from '../../../components/ui/button'
import { ToolbarButton } from '../../../components/business'
import { searchWorkers } from '../../../api/pickers'
import { useT } from '../../../i18n'
import type { Translations } from '../../../i18n/types'
import { fmtTime, filterMessages, toId, MESSAGE_LEVELS, type WorkerMessageEntry } from './logic'

type Ns = Translations['pages']['message']
const errBanner = 'mb-3 flex items-start justify-between gap-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]'

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
  const [workerNames, setWorkerNames] = useState<Map<number, string>>(new Map())
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
      .catch((e) => setError(e instanceof Error ? e.message : t.loadFail))
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

  // 师傅 id → 人读名映射:列表列与筛选钉选回显共用;失败降级 #id(不空转)。
  useEffect(() => {
    apiFetch<{ items: { id: number; name: string; staffNo: string }[] }>('/workers')
      .then((d) => setWorkerNames(new Map((d?.items ?? []).map((w) => [w.id, `${w.name} · ${w.staffNo}`]))))
      .catch(() => setWorkerNames(new Map()))
  }, [])

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
      .catch((e) => { setSending(false); setHint(e instanceof Error ? e.message : t.sendFail) })
  }

  const workerCell = (id: number) => {
    const name = workerNames.get(id)
    return <span title={'workerId=' + String(id)}>{name ?? `#${id}`}</span>
  }
  /** 已选师傅钉选回显:页签重挂/检索失败时触发器不再跌回裸编号(W0 契约 6)。 */
  const workerPinOptions = (id: string) => {
    const label = id ? workerNames.get(Number(id)) : undefined
    return label ? [{ value: id, label }] : undefined
  }

  return (
    <Card className="p-4">
      <div className="mb-3 flex flex-wrap gap-2">
        <Input className="w-55" placeholder={t.searchPlaceholder} value={keyword}
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
          pinnedOptions={workerPinOptions(workerId)}
        />
        <span className="spacer" />
        <ToolbarButton onClick={load}>{t.refresh}</ToolbarButton>
      </div>
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <ToolbarButton primary onClick={() => setSendOpen(true)}>+ {t.send}</ToolbarButton>
      </div>
      <div className="mb-3 text-sm font-semibold text-[var(--shell-heading)]">
        {t.cardTitle}
        <span className="ml-2 text-xs font-normal text-[var(--shell-group-title)]">
          {filtered.length ? t.matched.replace('{count}', String(filtered.length)) : ''}
        </span>
      </div>
      {error ? <div className={errBanner}><span className="break-all">{error}</span><CopyButton text={error} className="h-6 shrink-0 border-none bg-none px-1 text-[11px]" /></div> : (
        <>
          <DataTable
            emptyText={t.empty}
            rows={slice as unknown as Record<string, unknown>[]}
            columns={[
              { key: 'level', label: t.msgColumns[0], render: (r) => <StatusTag domain="message" value={String(r.level)} /> },
              { key: 'workerId', label: t.msgColumns[1], render: (r) => workerCell(Number(r.workerId)) },
              { key: 'title', label: t.msgColumns[2], render: (r) => String(r.title ?? '') },
              { key: 'content', label: t.msgColumns[3], render: (r) => <span className="text-[var(--shell-group-title)]">{String(r.content ?? '')}</span> },
              { key: 'sentAt', label: t.msgColumns[4], render: (r) => fmtTime(String(r.sentAt)) },
              { key: 'read', label: t.msgColumns[5], render: (r) => (r.read ? t.read : t.unread) },
            ]}
          />
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={(s) => { setPageSize(s); setPage(1) }}
            rangeText={t.rangeText} prevText={t.prev} nextText={t.next} perPageText={t.perPage}
            jumpText={t.jump} pageUnitText={t.pageUnit} />
        </>
      )}
      {sendOpen && (
        <Drawer title={t.send} onClose={closeSend}
          footer={
            <>
              <Button variant="outline" size="sm" onClick={closeSend}>
                {cancelText}
              </Button>
              <Button size="sm" disabled={sending} onClick={send}>
                {sending ? t.sending : t.send}
              </Button>
            </>
          }>
          <div className="grid gap-3">
            <label>
              <span className="text-[13px] text-[var(--shell-group-title)]">{t.msgColumns[1]}</span>
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
                  pinnedOptions={workerPinOptions(sendWorker)}
                />
              </div>
            </label>
            <label>
              <span className="text-[13px] text-[var(--shell-group-title)]">{t.sendLevel}</span>
              <div className="mt-1">
                <Dropdown
                  value={sendLevel}
                  options={MESSAGE_LEVELS.map((lv) => ({ value: lv, label: lv }))}
                  onChange={(v) => setSendLevel(v)}
                  ariaLabel={t.sendLevel}
                />
              </div>
            </label>
            <label>
              <span className="text-[13px] text-[var(--shell-group-title)]">{t.msgColumns[2]}</span>
              <Input className="mt-1" placeholder={t.sendTitlePlaceholder} value={sendTitle}
                onChange={(e) => setSendTitle(e.target.value)} />
            </label>
            <label>
              <span className="text-[13px] text-[var(--shell-group-title)]">{t.msgColumns[3]}</span>
              <Textarea className="mt-1" rows={3} placeholder={t.sendContentPlaceholder} value={sendContent}
                onChange={(e) => setSendContent(e.target.value)} />
            </label>
            {hint && <span className="text-xs text-[var(--color-danger)]">{hint}</span>}
          </div>
        </Drawer>
      )}
    </Card>
  )
}
