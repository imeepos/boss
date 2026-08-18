// 客户档案页:列名以 fields.md §2.1 为准;契约 GET /customers(keyword/phone/status 过滤)。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { DetailDrawer, PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { SERVICE_STATUSES, filterCustomers, pageSlice } from './filter'
import type { CustomerRow } from './types'
import { VerifyLogsDrawer } from './VerifyLogsDrawer'
import { fmtTime } from '../../../lib/format'
import '../../org/org.css'

export default function CustomerPage() {
  const t = useT()
  const c = t.pages.customer
  const [rows, setRows] = useState<CustomerRow[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [phone, setPhone] = useState('')
  const [status, setStatus] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [detail, setDetail] = useState<CustomerRow | null>(null)
  const [verifyId, setVerifyId] = useState<CustomerRow | null>(null)
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
      <div className="org-card">
        <div className="org-toolbar">
          <input className="org-input" placeholder={c.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <input className="org-input" placeholder={c.phonePlaceholder}
            value={phone} onChange={(e) => { setPhone(e.target.value); setPage(1) }} />
          <select className="org-select" value={status}
            onChange={(e) => { setStatus(e.target.value); setPage(1) }}>
            <option value="">{c.allStatus}</option>
            {SERVICE_STATUSES.map((s, i) => <option key={s} value={s}>{c.statusOptions[i]}</option>)}
          </select>
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{c.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.id}>
                    <td>{r.name}</td>
                    <td>{r.phone}</td>
                    <td>{r.idType || '—'}</td>
                    <td>{r.idNo || '—'}</td>
                    <td><StatusTag domain="realName" value={r.realNameStatus} /></td>
                    <td><StatusTag domain="service" value={r.serviceStatus} /></td>
                    <td>
                      <span className="org-act">
                        <button onClick={() => setDetail(r)}>{c.detail}</button>
                        <span className="sep">|</span>
                        <button onClick={() => setVerifyId(r)}>{c.verify}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={7}><div className="org-empty">{c.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
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
    </div>
  )
}
