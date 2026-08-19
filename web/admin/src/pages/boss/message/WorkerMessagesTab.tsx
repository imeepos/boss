// 消息中心 · 师傅消息页签:查询(GET /worker-messages?workerId=) + 下发(POST /worker-messages)。
import { useEffect, useState, type CSSProperties } from 'react'
import { apiFetch } from '../../../api/client'
import { Dropdown } from '../../../components/Dropdown'
import { Pagination } from '../../../components/Pagination'
import { StatusTag } from '../../../components/StatusTag'
import type { Translations } from '../../../i18n/types'
import { fmtTime, filterMessages, toId, MESSAGE_LEVELS, type WorkerMessageEntry } from './logic'

type Ns = Translations['pages']['message']

export const ctl: CSSProperties = {
  height: 30, padding: '0 8px', border: '1px solid #d9d9d9', borderRadius: 6, fontSize: 13, boxSizing: 'border-box',
}
export const th: CSSProperties = {
  textAlign: 'left', padding: '8px 10px', background: '#fafafa', borderBottom: '1px solid #f0f0f0', fontWeight: 600,
}
export const td: CSSProperties = { padding: '8px 10px', borderBottom: '1px solid #f0f0f0' }

export function WorkerMessagesTab({ t }: { t: Ns }) {
  const [rows, setRows] = useState<WorkerMessageEntry[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [level, setLevel] = useState('')
  const [read, setRead] = useState('')
  const [workerId, setWorkerId] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  // 下发表单
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

  const filtered = filterMessages(rows, keyword, level, read)
  const slice = filtered.slice((page - 1) * pageSize, page * pageSize)

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
        setSendTitle('')
        setSendContent('')
        setHint(t.sent)
        load()
      })
      .catch(() => { setSending(false); setHint(t.sendFail) })
  }

  return (
    <div style={{ background: '#fff', border: '1px solid #f0f0f0', borderRadius: 8, padding: 16 }}>
      <div style={{ display: 'flex', gap: 8, marginBottom: 12, flexWrap: 'wrap' }}>
        <input style={ctl} placeholder={t.searchPlaceholder} value={keyword}
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
        <input style={{ ...ctl, width: 120 }} placeholder={t.workerIdPlaceholder} value={workerId}
          onChange={(e) => { setWorkerId(e.target.value); setPage(1) }} onBlur={load} />
        <span style={{ flex: 1 }} />
        <button style={{ ...ctl, cursor: 'pointer' }} onClick={load}>{t.refresh}</button>
      </div>
      <div style={{ display: 'flex', gap: 8, marginBottom: 12, flexWrap: 'wrap', alignItems: 'center' }}>
        <span style={{ fontSize: 13, color: '#666' }}>{t.sendTitle}:</span>
        <input style={{ ...ctl, width: 100 }} placeholder={t.workerIdPlaceholder} value={sendWorker}
          onChange={(e) => setSendWorker(e.target.value)} />
        <Dropdown
          value={sendLevel}
          options={MESSAGE_LEVELS.map((lv) => ({ value: lv, label: lv }))}
          onChange={(v) => setSendLevel(v)}
          ariaLabel={t.sendLevel}
        />
        <input style={{ ...ctl, width: 180 }} placeholder={t.sendTitlePlaceholder} value={sendTitle}
          onChange={(e) => setSendTitle(e.target.value)} />
        <input style={{ ...ctl, width: 220 }} placeholder={t.sendContentPlaceholder} value={sendContent}
          onChange={(e) => setSendContent(e.target.value)} />
        <button style={{ ...ctl, cursor: 'pointer', background: '#1677ff', borderColor: '#1677ff', color: '#fff' }}
          disabled={sending} onClick={send}>{sending ? t.sending : t.send}</button>
        {hint && <span style={{ fontSize: 12, color: hint === t.sent ? '#52c41a' : '#e54545' }}>{hint}</span>}
      </div>
      <div style={{ fontWeight: 600, marginBottom: 12 }}>
        {t.cardTitle}
        <span style={{ marginLeft: 8, fontWeight: 400, fontSize: 12, color: '#999' }}>
          {filtered.length ? t.matched.replace('{count}', String(filtered.length)) : ''}
        </span>
      </div>
      {error ? <div style={{ color: '#e54545', fontSize: 13, padding: '12px 0' }}>{error}</div> : (
        <>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
            <thead>
              <tr>{t.msgColumns.map((c) => <th key={c} style={th}>{c}</th>)}</tr>
            </thead>
            <tbody>
              {slice.map((r) => (
                <tr key={r.id}>
                  <td style={td}><StatusTag domain="message" value={r.level} /></td>
                  <td style={td}>#{r.workerId}</td>
                  <td style={td}>{r.title}</td>
                  <td style={{ ...td, color: '#666' }}>{r.content}</td>
                  <td style={td}>{fmtTime(r.sentAt)}</td>
                  <td style={td}>{r.read ? t.read : t.unread}</td>
                </tr>
              ))}
              {!slice.length && <tr><td colSpan={6} style={{ ...td, textAlign: 'center', color: '#999' }}>{t.empty}</td></tr>}
            </tbody>
          </table>
          <Pagination total={filtered.length} page={page} pageSize={pageSize} onPage={setPage} onSize={setPageSize}
            rangeText={t.rangeText} prevText={t.prev} nextText={t.next} perPageText={t.perPage}
            jumpText={t.jump} pageUnitText={t.pageUnit} />
        </>
      )}
    </div>
  )
}
