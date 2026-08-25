// 客户端崩溃日志:GET /crash-logs 列表 + 单条堆栈展开排查。
// 契约 admin/sys.yaml /crash-logs (menu:crash_logs);迁移 000140 授 sysadmin。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead } from '../../../components/business/page-head'

type CrashLog = {
  id: number
  subjectType: string
  subjectId: number
  app: string
  log: string
  createdAt: string
}

export default function CrashLogsPage() {
  const t = useT()
  const [logs, setLogs] = useState<CrashLog[]>([])
  const [error, setError] = useState('')
  const [openId, setOpenId] = useState<number | null>(null)

  const load = () => {
    setError('')
    apiFetch<CrashLog[]>('/crash-logs?limit=100')
      .then((d) => setLogs(Array.isArray(d) ? d : []))
      .catch((e) => setError(e instanceof Error ? e.message : t.pages.crashlogs.loadFail))
  }

  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div>
      <PageHead title={t.pages.crashlogs.title} desc={t.pages.crashlogs.desc} />
      <div style={{ marginBottom: 12 }}>
        <button
          style={{ padding: '6px 14px', border: '1px solid #d9d9d9', background: '#fff', borderRadius: 6, cursor: 'pointer' }}
          onClick={load}
        >
          {t.pages.crashlogs.refresh}
        </button>
      </div>
      {error ? (
        <div style={{ color: '#e54545', fontSize: 13, padding: '12px 0' }}>{error}</div>
      ) : logs.length === 0 ? (
        <div style={{ color: '#999', padding: '24px 0' }}>{t.pages.crashlogs.empty}</div>
      ) : (
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
          <thead>
            <tr style={{ borderBottom: '1px solid #f0f0f0', background: '#fafafa' }}>
              <th style={th}>{t.pages.crashlogs.colTime}</th>
              <th style={th}>{t.pages.crashlogs.colApp}</th>
              <th style={th}>{t.pages.crashlogs.colSubject}</th>
              <th style={th}>{t.pages.crashlogs.colOp}</th>
            </tr>
          </thead>
          <tbody>
            {logs.map((l) => (
              <>
                <tr key={l.id} style={{ borderBottom: '1px solid #f5f5f5' }}>
                  <td style={td}>{new Date(l.createdAt).toLocaleString()}</td>
                  <td style={td}>{l.app || '—'}</td>
                  <td style={td}>
                    {l.subjectType}/{l.subjectId || 0}
                  </td>
                  <td style={td}>
                    <button style={btn} onClick={() => setOpenId(openId === l.id ? null : l.id)}>
                      {openId === l.id ? t.pages.crashlogs.collapse : t.pages.crashlogs.expand}
                    </button>
                  </td>
                </tr>
                {openId === l.id ? (
                  <tr>
                    <td colSpan={4} style={{ background: '#1e1e1e', color: '#eaeaea', padding: 12, fontSize: 12, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                      {l.log}
                    </td>
                  </tr>
                ) : null}
              </>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}

const th: React.CSSProperties = { textAlign: 'left', padding: '10px 12px', fontWeight: 600, color: '#555' }
const td: React.CSSProperties = { padding: '10px 12px', verticalAlign: 'top' }
const btn: React.CSSProperties = { padding: '4px 10px', border: '1px solid #d9d9d9', background: '#fff', borderRadius: 4, cursor: 'pointer', fontSize: 12 }