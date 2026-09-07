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
import { TableStateRow } from '../../../components/business'
import { TabBar } from '../../../components/business/tab-bar'

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
        .catch((e) => setError(e instanceof Error ? e.message : w.loadFail))
    } else if (key === 'hall') {
      apiFetch<{ items: DispatchTicketRow[] }>('/dispatch/pool')
        .then((d) => setPool(d?.items ?? []))
        .catch((e) => setError(e instanceof Error ? e.message : w.loadFail))
    }
  }
  useEffect(() => { loadTab(tab) }, [tab]) // eslint-disable-line react-hooks/exhaustive-deps

  const maintSlice = pageSlice(maints, page, pageSize)
  const hallSlice = pageSlice(pool, page, pageSize)

  return (
    <div>
      <PageHead title={w.title} desc={w.desc} />
      <TabBar
        tabs={[{ key: 'notice', label: w.tabNotice }, { key: 'maint', label: w.tabMaint }, { key: 'hall', label: w.tabHall }]}
        value={tab}
        onChange={(k) => { setTab(k); setPage(1) }}
      />
      {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : tab === 'notice' ? (
        <NoticesTab t={t.pages.message} />
      ) : tab === 'maint' ? (
        <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>
                <th className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{t.pages.devicePage.maintColumns[0]}</th><th className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{t.pages.devicePage.maintColumns[1]}</th>
                <th className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{t.pages.devicePage.maintColumns[2]}</th><th className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{t.pages.devicePage.maintColumns[3]}</th>
                <th className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{t.pages.devicePage.maintColumns[4]}</th><th className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{t.pages.devicePage.maintColumns[5]}</th>
                <th className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{t.pages.devicePage.maintColumns[6]}</th>
              </tr></thead>
              <tbody>
                {maintSlice.map((m) => (
                  <tr key={m.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{m.deviceNo}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{m.deviceType || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{m.healthScore}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{m.faultCount}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{m.ageYears ?? '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="maintPriority" value={m.priority} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{m.reason || '—'}</td>
                  </tr>
                ))}
                {!maintSlice.length && <TableStateRow colSpan={7} text={w.empty} />}
              </tbody>
            </table>
          </div>
          <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
            <Pagination total={maints.length} page={page} pageSize={pageSize}
              onPage={setPage} onSize={setPageSize} {...pagerTexts(w)} />
          </div>
        </div>
      ) : (
        <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{w.hallColumns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {hallSlice.map((x) => (
                  <tr key={x.ticketId}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.ticketNo}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]" title={'orderId=' + x.orderId}>#{x.orderId}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.regionName || `#${x.regionId}`}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.legalEntityName || `#${x.legalEntityId}`}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="ticket" value={x.status} /></td>
                  </tr>
                ))}
                {!hallSlice.length && <TableStateRow colSpan={5} text={w.empty} />}
              </tbody>
            </table>
          </div>
          <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
            <Pagination total={pool.length} page={page} pageSize={pageSize}
              onPage={setPage} onSize={setPageSize} {...pagerTexts(w)} />
          </div>
        </div>
      )}
    </div>
  )
}
