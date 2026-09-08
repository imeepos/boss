// 师傅端内容页:公告(复用消息中心 NoticesTab)+ 健康观察(GET /device/maintenances)+ 接单大厅(GET /dispatch/pool)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { ErrorBanner, IdRef } from '../../../components/business'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { NoticesTab } from '../message/NoticesTab'
import { pageSlice } from '../types'
import type { DispatchTicketRow } from '../types'
import type { MaintenanceRow } from '../../oss/types'
import { TableStateRow } from '../../../components/business'
import { TabBar } from '../../../components/business/tab-bar'
import { Card } from '../../../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'

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
  const resetPage = (s: number) => { setPageSize(s); setPage(1) }

  return (
    <div>
      <PageHead title={w.title} desc={w.desc} />
      <TabBar
        tabs={[{ key: 'notice', label: w.tabNotice }, { key: 'maint', label: w.tabMaint }, { key: 'hall', label: w.tabHall }]}
        value={tab}
        onChange={(k) => { setTab(k); setPage(1) }}
      />
      {error ? <ErrorBanner message={error} /> : tab === 'notice' ? (
        <NoticesTab t={t.pages.message} />
      ) : tab === 'maint' ? (
        <Card>
          <div className="px-4 pb-4">
            <Table>
              <TableHeader>
                <TableRow>{t.pages.devicePage.maintColumns.map((x: string) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
              </TableHeader>
              <TableBody>
                {maintSlice.map((m) => (
                  <TableRow key={m.id}>
                    <TableCell>{m.deviceNo}</TableCell>
                    <TableCell>{m.deviceType || '—'}</TableCell>
                    <TableCell>{m.healthScore}</TableCell>
                    <TableCell>{m.faultCount}</TableCell>
                    <TableCell>{m.ageYears ?? '—'}</TableCell>
                    <TableCell><StatusTag domain="maintPriority" value={m.priority} /></TableCell>
                    <TableCell className="whitespace-normal">{m.reason || '—'}</TableCell>
                  </TableRow>
                ))}
                {!maintSlice.length && <TableStateRow colSpan={7} text={w.empty} />}
              </TableBody>
            </Table>
          </div>
          <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
            <Pagination total={maints.length} page={page} pageSize={pageSize}
              onPage={setPage} onSize={resetPage} {...pagerTexts(w)} />
          </div>
        </Card>
      ) : (
        <Card>
          <div className="px-4 pb-4">
            <Table>
              <TableHeader>
                <TableRow>{w.hallColumns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
              </TableHeader>
              <TableBody>
                {hallSlice.map((x) => (
                  <TableRow key={x.ticketId}>
                    <TableCell>{x.ticketNo}</TableCell>
                    <TableCell><IdRef value={x.orderId} /></TableCell>
                    <TableCell>{x.regionName || `#${x.regionId}`}</TableCell>
                    <TableCell>{x.legalEntityName || `#${x.legalEntityId}`}</TableCell>
                    <TableCell><StatusTag domain="ticket" value={x.status} /></TableCell>
                  </TableRow>
                ))}
                {!hallSlice.length && <TableStateRow colSpan={5} text={w.empty} />}
              </TableBody>
            </Table>
          </div>
          <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
            <Pagination total={pool.length} page={page} pageSize={pageSize}
              onPage={setPage} onSize={resetPage} {...pagerTexts(w)} />
          </div>
        </Card>
      )}
    </div>
  )
}
