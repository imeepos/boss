// 消息中心页:师傅站内消息 + 师傅端公告,两个页签(契约 worker.yaml Worker tag,perm menu:dispatch)。
import { useState } from 'react'
import { useT } from '../../../i18n'
import { WorkerMessagesTab } from './WorkerMessagesTab'
import { NoticesTab } from './NoticesTab'

export default function MessageCenterPage() {
  const t = useT()
  const m = t.pages.message
  const [tab, setTab] = useState<'worker' | 'notice'>('worker')
  return (
    <div>
      <h2 style={{ margin: 0, fontSize: 20 }}>{m.title}</h2>
      <p style={{ margin: '4px 0 12px', color: '#888', fontSize: 12 }}>{m.desc}</p>
      <div style={{ display: 'flex', gap: 4, marginBottom: 12, borderBottom: '1px solid #f0f0f0' }}>
        {([['worker', m.tabWorker], ['notice', m.tabNotice]] as const).map(([key, label]) => (
          <button
            key={key}
            onClick={() => setTab(key)}
            style={{
              padding: '8px 16px', fontSize: 14, cursor: 'pointer', background: 'none',
              border: 'none', borderBottom: tab === key ? '2px solid #1677ff' : '2px solid transparent',
              color: tab === key ? '#1677ff' : '#666', fontWeight: tab === key ? 600 : 400,
            }}
          >
            {label}
          </button>
        ))}
      </div>
      {tab === 'worker' ? <WorkerMessagesTab t={m} /> : <NoticesTab t={m} />}
    </div>
  )
}
