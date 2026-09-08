// 出账管理页:列名以 fields.md §3.3 + billing.html 为准;契约 GET /bills(customerId 过滤)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { DetailDrawer, PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { ResourcePicker } from '../../../components/ResourcePicker'
import { searchCustomers } from '../../../api/pickers'
import { pageSlice, type BillRow } from '../types'
import { useCustomerPin } from '../useCustomerPin'
import { InvoicePanel } from './invoices'
import { BillingRunModal, INVOICES_REFRESH } from './run-modal'
import { fmtFee } from '../../../lib/format'
import { TableStateRow, ErrorBanner, ActionLink } from '../../../components/business'
import { Card, CardFooter } from '../../../components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { ToolbarButton } from '../../../components/business/page-head'

export default function BillPage() {
  const t = useT()
  const b = t.pages.billPage
  const [rows, setRows] = useState<BillRow[]>([])
  const [error, setError] = useState('')
  const [customerId, setCustomerId] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [detail, setDetail] = useState<BillRow | null>(null)
  const [runOpen, setRunOpen] = useState(false)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: BillRow[] }>('/bills', {
      query: { customerId: customerId || undefined },
    })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : b.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const slice = pageSlice(rows, page, pageSize)
  const statusText = (s: string) => t.common.statusTags['bill.' + s] ?? s
  // 钉选回显(W0 基线交接项):已选客户名经详情接口取,保证触发器不回显裸编号。
  const pinnedCustomer = useCustomerPin(customerId)

  return (
    <div>
      <PageHead title={b.title} desc={b.desc} />
      <Card>
        <div className="flex flex-wrap items-center gap-2 p-4">
          <ResourcePicker
            value={customerId}
            onChange={(v) => { setCustomerId(v); setPage(1) }}
            search={searchCustomers}
            toOption={(c) => ({ value: String(c.id), label: `${c.name} · ${c.phone || c.customerCode}` })}
            ariaLabel={b.filterCustomer}
            emptyLabel={t.pages.pickers.common.all}
            searchPlaceholder={t.pages.pickers.common.placeholder}
            errorText={b.loadFail}
            pinnedOptions={pinnedCustomer}
          />
          <span className="spacer" />
          <ToolbarButton primary onClick={() => setRunOpen(true)}>{b.run.btn}</ToolbarButton>
          <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
        </div>
        {error ? <ErrorBanner message={error} /> : (
          <div className="px-4 pb-4">
            <Table>
              <TableHeader><TableRow>{b.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow></TableHeader>
              <TableBody>
                {slice.map((r) => (
                  <TableRow key={r.billId}>
                    <TableCell className="font-mono">{r.billNo}</TableCell>
                    <TableCell>{r.customerName || `#${r.customerId}`}</TableCell>
                    <TableCell>{r.period}</TableCell>
                    <TableCell>{fmtFee(r.amount)}</TableCell>
                    <TableCell><StatusTag domain="bill" value={r.status} /></TableCell>
                    <TableCell>
                      <ActionLink onClick={() => setDetail(r)} label={b.detail} />
                    </TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={6} loading={busy} text={b.empty} />}
              </TableBody>
            </Table>
          </div>
        )}
        <CardFooter>
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(b)} />
        </CardFooter>
      </Card>
      <InvoicePanel />
      <BillingRunModal open={runOpen} onClose={() => setRunOpen(false)} onDone={() => { load(); window.dispatchEvent(new CustomEvent(INVOICES_REFRESH)) }} />
      {detail && (
        <DetailDrawer
          title={b.detail}
          closeText={t.pages.company.cancel}
          onClose={() => setDetail(null)}
          items={[
            { k: b.columns[0], v: detail.billNo },
            { k: b.columns[1], v: detail.customerName || `#${detail.customerId}` },
            { k: b.columns[2], v: detail.period },
            { k: b.columns[3], v: fmtFee(detail.amount) },
            { k: b.columns[4], v: statusText(detail.status) },
            { k: 'legalEntity', v: detail.legalEntityName || `#${detail.legalEntityId}` },
            { k: 'region', v: detail.regionName || `#${detail.regionId}` },
          ]}
        />
      )}
    </div>
  )
}
