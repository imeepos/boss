// 客户档案页:列名以 fields.md §2.1 为准;契约 GET /customers(keyword/phone/status 过滤)。
// 代客开户一站式入口:直建(POST /customers)与自助注册审核队列同页挂载。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { useQueryState } from '../../../lib/useQueryState'
import { DetailDrawer, PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Dropdown } from '../../../components/Dropdown'
import { SERVICE_STATUSES, filterCustomers, pageSlice } from './filter'
import type { CustomerRow } from './types'
import { VerifyLogsDrawer } from './VerifyLogsDrawer'
import { RealNameDrawer } from './RealNameDrawer'
import { CustomerCreateDrawer } from './CustomerCreateDrawer'
import { RegistrationQueueDrawer } from './RegistrationQueueDrawer'
import { BatchImportEntry } from '../../base/importer/BatchImportEntry'
import { fmtTime } from '../../../lib/format'
import { TableStateRow } from '../../../components/business'

export default function CustomerPage() {
  const t = useT()
  const c = t.pages.customer
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
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" placeholder={c.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setUrlKeyword(e.target.value); setPage(1) }} />
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" placeholder={c.phonePlaceholder}
            value={phone} onChange={(e) => { setPhone(e.target.value); setPage(1) }} />
          <Dropdown
            value={status}
            options={[{ value: '', label: c.allStatus }, ...SERVICE_STATUSES.map((s, i) => ({ value: s, label: c.statusOptions[i] }))]}
            onChange={(v) => { setStatus(v); setPage(1) }}
            ariaLabel={c.allStatus}
          />
          <span className="spacer" />
          <button type="button" className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => setCreateOpen(true)}>{c.createBtn}</button>
          <button type="button" className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={() => setRegOpen(true)}>{c.regBtn}</button>
          <BatchImportEntry kind="customer" onImported={load} />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{c.columns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.name}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.phone}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.idType || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.idNo || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="realName" value={r.realNameStatus} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="service" value={r.serviceStatus} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      <span className="inline-flex items-center">
                        <button onClick={() => setDetail(r)}>{c.detail}</button>
                        <span className="text-[var(--shell-side-border)]">|</span>
                        <button onClick={() => setRnId(r)}>{c.rnBtn}</button>
                        <span className="text-[var(--shell-side-border)]">|</span>
                        <button onClick={() => setVerifyId(r)}>{c.verify}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={7} loading={busy} text={c.empty} />}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(c)} />
        </div>
      </div>
      {detail && (
        <DetailDrawer
          title={c.detail}
          closeText={t.pages.company.cancel}
          onClose={() => setDetail(null)}
          items={[
            { k: c.columns[0], v: detail.name },
            { k: c.columns[1], v: detail.phone },
            { k: c.columns[2], v: detail.idType },
            { k: c.columns[3], v: detail.idNo },
            { k: c.columns[4], v: detail.realNameStatus },
            { k: c.columns[5], v: detail.serviceStatus },
            { k: 'ID', v: String(detail.id) },
            { k: 'regionName', v: detail.regionName },
            { k: 'createdAt', v: fmtTime(detail.createdAt) },
          ]}
        />
      )}
      {verifyId && (
        <VerifyLogsDrawer customerId={verifyId.id} customerName={verifyId.name} onClose={() => setVerifyId(null)} />
      )}
      {rnId && (
        <RealNameDrawer customerId={rnId.id} customerName={rnId.name} onClose={() => setRnId(null)} onSubmitted={load} />
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
