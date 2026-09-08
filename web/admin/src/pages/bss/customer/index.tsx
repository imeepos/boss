// 客户档案页:列名以 fields.md §2.1 为准;契约 GET /customers(keyword/phone/status 过滤)。
// 代客开户一站式入口:直建(POST /customers)与自助注册审核队列同页挂载。
import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { useQueryState } from '../../../lib/useQueryState'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Dropdown } from '../../../components/Dropdown'
import { Table, TableBody, TableHead, TableHeader, TableRow, TableCell } from '../../../components/ui/table'
import { Card } from '../../../components/ui/card'
import { Input } from '../../../components/ui/input'
import { ActionLink, ActionLinks, ActionSep, ErrorBanner, ToolbarButton } from '../../../components/business'
import { SERVICE_STATUSES, filterCustomers, pageSlice } from './filter'
import type { CustomerRow } from './types'
import { VerifyLogsDrawer } from './VerifyLogsDrawer'
import { RealNameDrawer } from './RealNameDrawer'
import { CustomerCreateDrawer } from './CustomerCreateDrawer'
import { AddressChainDrawer } from '../../boss/order/AddressChainDrawer'
import { RegistrationQueueDrawer } from './RegistrationQueueDrawer'
import { BatchImportEntry } from '../../base/importer/BatchImportEntry'
import { fmtTime } from '../../../lib/format'
import { TableStateRow } from '../../../components/business'
import { Drawer } from '../../../components/Drawer'

export default function CustomerPage() {
  const t = useT()
  const c = t.pages.customer
  const navigate = useNavigate()
  const [rows, setRows] = useState<CustomerRow[]>([])
  const [error, setError] = useState('')
  const [urlKeyword, setUrlKeyword] = useQueryState('kw', '')
  const [keyword, setKeyword] = useState(urlKeyword)
  const [phone, setPhone] = useState('')
  const [status, setStatus] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [detail, setDetail] = useState<CustomerRow | null>(null)
  const [verifyId, setVerifyId] = useState<CustomerRow | null>(null)
  const [rnId, setRnId] = useState<CustomerRow | null>(null)
  const [addrRow, setAddrRow] = useState<CustomerRow | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const [regOpen, setRegOpen] = useState(false)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: CustomerRow[] }>('/customers', {
      query: { keyword: keyword || undefined, phone: phone || undefined, status: status || undefined },
    })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : c.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const filtered = useMemo(() => filterCustomers(rows, keyword, phone), [rows, keyword, phone])
  const slice = pageSlice(filtered, page, pageSize)

  return (
    <div>
      <PageHead title={c.title} desc={c.desc} />
      <Card>
        <div className="flex flex-wrap items-center gap-2 p-4">
          <Input className="w-44" placeholder={c.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setUrlKeyword(e.target.value); setPage(1) }} />
          <Input className="w-40" placeholder={c.phonePlaceholder}
            value={phone} onChange={(e) => { setPhone(e.target.value); setPage(1) }} />
          <Dropdown
            value={status}
            options={[{ value: '', label: c.allStatus }, ...SERVICE_STATUSES.map((s, i) => ({ value: s, label: c.statusOptions[i] }))]}
            onChange={(v) => { setStatus(v); setPage(1) }}
            ariaLabel={c.allStatus}
          />
          <span className="spacer" />
          <ToolbarButton primary onClick={() => setCreateOpen(true)}>{c.createBtn}</ToolbarButton>
          <ToolbarButton onClick={() => setRegOpen(true)}>{c.regBtn}</ToolbarButton>
          <BatchImportEntry kind="customer" onImported={load} />
          <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
        </div>
        {error ? <ErrorBanner message={error} /> : (
          <div className="overflow-x-auto px-4 pb-4">
            <Table>
              <TableHeader>
                <TableRow>{c.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
              </TableHeader>
              <TableBody>
                {slice.map((r) => (
                  <TableRow key={r.id}>
                    <TableCell>{r.name}</TableCell>
                    <TableCell>{r.phone}</TableCell>
                    <TableCell>{r.idType || '—'}</TableCell>
                    <TableCell>{r.idNo || '—'}</TableCell>
                    <TableCell><StatusTag domain="realName" value={r.realNameStatus} /></TableCell>
                    <TableCell><StatusTag domain="service" value={r.serviceStatus} /></TableCell>
                    <TableCell>
                      <ActionLinks>
                        <ActionLink onClick={() => setDetail(r)} label={c.detail} testId={'cust-detail-' + r.id} />
                        <ActionSep />
                        <ActionLink onClick={() => setRnId(r)} label={c.rnBtn} />
                        <ActionSep />
                        <ActionLink onClick={() => setAddrRow(r)} label={c.addrBtn} />
                        <ActionSep />
                        <ActionLink onClick={() => navigate(`/bss/onboarding?customerId=${r.id}`)} label={c.orderBtn} />
                        <ActionSep />
                        <ActionLink onClick={() => setVerifyId(r)} label={c.verify} />
                      </ActionLinks>
                    </TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={7} loading={busy} text={c.empty} />}
              </TableBody>
            </Table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(c)} />
        </div>
      </Card>
      {detail && (
        <CustomerDetailDrawer detail={detail} onClose={() => setDetail(null)} />
      )}
      {verifyId && (
        <VerifyLogsDrawer customerId={verifyId.id} customerName={verifyId.name} onClose={() => setVerifyId(null)} />
      )}
      {rnId && (
        <RealNameDrawer customerId={rnId.id} customerName={rnId.name} onClose={() => setRnId(null)} onSubmitted={load} />
      )}
      {addrRow && (
        <AddressChainDrawer customerId={String(addrRow.id)} customerAddressId={addrRow.addressId} backfill
          endpoint="/customers/address" onDone={load} onClose={() => setAddrRow(null)} />
      )}
      {createOpen && (
        <CustomerCreateDrawer open onClose={() => setCreateOpen(false)} onCreated={load} />
      )}
      {regOpen && (
        <RegistrationQueueDrawer open onClose={() => setRegOpen(false)} onChanged={load} />
      )}
    </div>
  )
}

/** 客户详情抽屉:归属链(运营主体→区域→客户)+ 档案字段 + 关联记录区块。
 * 归属公司名直用行内 legalEntityName(服务端 JOIN 现值,data-relations §6.1 已销账);
 * 关联计数:订单/账单走 customerId 服务端过滤;缴费接口无 customerId 过滤(102 实测),
 * 取不到显示 —(data-relations §0 铁律 4,禁止臆造)。 */
export function CustomerDetailDrawer({ detail, onClose }: { detail: CustomerRow; onClose: () => void }) {
  const t = useT()
  const c = t.pages.customer
  const navigate = useNavigate()
  const [rel, setRel] = useState<{ orders: number | null; bills: number | null; fail: boolean }>({
    orders: null, bills: null, fail: false,
  })

  useEffect(() => {
    let alive = true
    Promise.all([
      apiFetch<{ items: unknown[] }>('/orders', { query: { customerId: detail.id } }),
      apiFetch<{ items: unknown[] }>('/bills', { query: { customerId: detail.id } }),
    ])
      .then(([o, b]) => {
        if (alive) setRel({ orders: o?.items?.length ?? 0, bills: b?.items?.length ?? 0, fail: false })
      })
      .catch(() => {
        if (alive) { setRel({ orders: null, bills: null, fail: true }); console.warn('[customer] 关联计数拉取失败 customerId=' + detail.id) }
      })
    return () => { alive = false }
  }, [detail.id]) // eslint-disable-line react-hooks/exhaustive-deps

  const dash = <span title={c.relLoadFail}>—</span>
  const relRows: { label: string; value: React.ReactNode; drill?: () => void }[] = [
    { label: c.relOrders, value: rel.orders === null ? dash : String(rel.orders),
      drill: rel.orders === null ? undefined : () => { onClose(); navigate('/bss/onboarding?customerId=' + detail.id) } },
    { label: c.relBills, value: rel.bills === null ? dash : String(rel.bills) },
    { label: c.relPayments, value: '—' },
  ]

  return (
    <Drawer title={c.detail} onClose={onClose}
      footer={
        <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={onClose}>
          {t.pages.company.cancel}
        </button>
      }>
      <div className="mb-4">
        <h4 className="mb-2 mt-0 text-xs font-semibold text-[var(--shell-group-title)]">{c.chainTitle}</h4>
        <div className="flex flex-wrap items-center gap-2 text-[13px] text-[var(--shell-content-text)]">
          <span className="rounded-sm border border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] px-2 py-0.5">
            {detail.legalEntityName || '—'}
          </span>
          <span className="text-[var(--shell-side-border)]">→</span>
          <span className="rounded-sm border border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] px-2 py-0.5">
            {detail.regionName || '—'}
          </span>
          <span className="text-[var(--shell-side-border)]">→</span>
          <span className="rounded-sm border border-[var(--color-border-focus)] bg-[var(--shell-menu-hover-bg)] px-2 py-0.5 font-medium">
            {detail.name} #{detail.id}
          </span>
        </div>
      </div>
      <div className="flex flex-col gap-2.5">
        {([
          { k: c.columns[0], v: detail.name },
          { k: c.columns[1], v: detail.phone },
          { k: c.columns[2], v: detail.idType },
          { k: c.columns[3], v: detail.idNo },
          { k: c.columns[4], v: <StatusTag domain="realName" value={detail.realNameStatus} /> },
          { k: c.columns[5], v: <StatusTag domain="service" value={detail.serviceStatus} /> },
          { k: 'ID', v: String(detail.id) },
          { k: c.regionLabel, v: detail.regionName },
          { k: 'createdAt', v: fmtTime(detail.createdAt) },
        ] as { k: string; v?: React.ReactNode }[]).map((it) => (
          <div key={it.k} className="flex gap-3 text-[13px]">
            <span className="w-24 flex-none text-[var(--shell-group-title)]">{it.k}</span>
            <span className="break-all text-[var(--shell-content-text)]">{it.v || '—'}</span>
          </div>
        ))}
      </div>
      <div className="mt-4 border-t border-[var(--shell-side-border)] pt-3">
        <h4 className="mb-2 mt-0 text-xs font-semibold text-[var(--shell-group-title)]">{c.relTitle}</h4>
        <div className="flex flex-col gap-2">
          {relRows.map((r) => (
            <div key={r.label} className="flex items-center gap-3 text-[13px]">
              <span className="w-24 flex-none text-[var(--shell-group-title)]">{r.label}</span>
              <span className="text-[var(--shell-content-text)]">{r.value}</span>
              {r.drill && (
                <button data-testid={'customer-rel-drill-' + r.label} className="border-none bg-none cursor-pointer text-xs text-[var(--color-text-link)] hover:underline" onClick={r.drill}>
                  {c.relDrill}
                </button>
              )}
            </div>
          ))}
        </div>
      </div>
    </Drawer>
  )
}
