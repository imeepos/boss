// 师傅端内容页:公告(复用消息中心 NoticesTab)+ 健康观察(GET /device/maintenances)+ 接单大厅(GET /dispatch/pool)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { NoticesTab } from '../message/NoticesTab'
import { pageSlice } from '../types'
import type { DispatchTicketRow } from '../types'
import type { MaintenanceRow } from '../../oss/types'
import '../../org/org.css'

export default function WorkerOpsPage() {
  const t = useT()
  const w = t.pages.workerOps
  const [tab, setTab] = useState<'notice' | 'maint' | 'hall'>('notice')
  const [maints, setMaints] = useState<MaintenanceRow[]>([])
  const [pool, setPool] = useState<DispatchTicketRow[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)

  const loadTab = (key: string) => {
    setError('')
    if (key === 'maint') {
      apiFetch<{ items: MaintenanceRow[] }>('/device/maintenances')
        .then((d) => setMaints(d?.items ?? []))
        .catch(() => setError(w.loadFail))
    } else if (key === 'hall') {
      apiFetch<{ items: DispatchTicketRow[] }>('/dispatch/pool')
        .then((d) => setPool(d?.items ?? []))
        .catch(() => setError(w.loadFail))
    }
  }
  useEffect(() => { loadTab(tab) }, [tab]) // eslint-disable-line react-hooks/exhaustive-deps

  const maintSlice = pageSlice(maints, page, pageSize)
  const hallSlice = pageSlice(pool, page, pageSize)

  return (
    <div>
      <PageHead title={w.title} desc={w.desc} />
      <div style={{ display: 'flex', gap: 4, marginBottom: 12, borderBottom: '1px solid #f0f0f0' }}>
        {(['notice', 'maint', 'hall'] as const).map((key) => (
          <button key={key} onClick={() => { setTab(key); setPage(1) }}
            style={{
              padding: '8px 16px', fontSize: 14, cursor: 'pointer', background: 'none', border: 'none',
              borderBottom: tab === key ? '2px solid #1677ff' : '2px solid transparent',
              color: tab === key ? '#1677ff' : '#666', fontWeight: tab === key ? 600 : 400,
            }}>
            {key === 'notice' ? w.tabNotice : key === 'maint' ? w.tabMaint : w.tabHall}
          </button>
        ))}
      </div>
      {error ? <div className="org-error">{error}</div> : tab === 'notice' ? (
        <NoticesTab t={t.pages.message} />
      ) : tab === 'maint' ? (
        <div className="org-card">
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>
                <th>{t.pages.devicePage.maintColumns[0]}</th><th>{t.pages.devicePage.maintColumns[1]}</th>
                <th>{t.pages.devicePage.maintColumns[2]}</th><th>{t.pages.devicePage.maintColumns[3]}</th>
                <th>{t.pages.devicePage.maintColumns[4]}</th><th>{t.pages.devicePage.maintColumns[5]}</th>
                <th>{t.pages.devicePage.maintColumns[6]}</th>
              </tr></thead>
              <tbody>
                {maintSlice.map((m) => (
                  <tr key={m.id}>
                    <td>{m.deviceNo}</td>
                    <td>{m.deviceType || '—'}</td>
                    <td>{m.healthScore}</td>
                    <td>{m.faultCount}</td>
                    <td>{m.ageYears ?? '—'}</td>
                    <td><StatusTag domain="maintPriority" value={m.priority} /></td>
                    <td>{m.reason || '—'}</td>
                  </tr>
                ))}
                {!maintSlice.length && <tr><td colSpan={7}><div className="org-empty">{w.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
          <div className="org-footer">
            <Pagination total={maints.length} page={page} pageSize={pageSize}
              onPage={setPage} onSize={setPageSize} {...pagerTexts(w)} />
          </div>
        </div>
      ) : (
        <div className="org-card">
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{w.hallColumns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {hallSlice.map((x) => (
                  <tr key={x.ticketId}>
                    <td>{x.ticketNo}</td>
                    <td>#{x.orderId}</td>
                    <td>{x.regionName || `#${x.regionId}`}</td>
                    <td>{x.legalEntityName || `#${x.legalEntityId}`}</td>
                    <td><StatusTag domain="ticket" value={x.status} /></td>
                  </tr>
                ))}
                {!hallSlice.length && <tr><td colSpan={5}><div className="org-empty">{w.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
          <div className="org-footer">
            <Pagination total={pool.length} page={page} pageSize={pageSize}
              onPage={setPage} onSize={setPageSize} {...pagerTexts(w)} />
          </div>
        </div>
      )}
    </div>
  )
}
