// 审计日志页:列名与交互照抄 docs/admin/audit.html 原型。
// 契约: GET /audit-logs → {items:[audit.Entry]}(menu:audit 门禁)。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Dropdown } from '../../../components/Dropdown'
import { DatePicker } from '../../../components/DatePicker'
import { Pagination } from '../../../components/Pagination'
import { DataTable } from '../../../components/business/data-table'
import { filterAuditLogs, toAuditLog, type AuditEntry, type AuditLog } from './logic'

export default function AuditPage() {
  const t = useT()
  const [rows, setRows] = useState<AuditLog[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [type, setType] = useState('')
  const [date, setDate] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [detail, setDetail] = useState<string | null>(null)

  const load = () => {
    setError('')
    // 契约: GET /audit-logs → {items:[Entry]};Entry 见 pkg/audit,映射为页面行。
    apiFetch<{ items: AuditEntry[] }>('/audit-logs')
      .then((d) => setRows((d?.items ?? []).map(toAuditLog)))
      .catch((e) => setError(e instanceof Error ? e.message : t.pages.audit.loadFail))
  }

  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const filtered = useMemo(
    () => filterAuditLogs(rows, keyword, type, date),
    [rows, keyword, type, date],
  )
  const slice = filtered.slice((page - 1) * pageSize, page * pageSize)
  const detailRow = detail ? rows.find((r) => r.logId === detail) : null

  return (
    <div>
      <div style={{ marginBottom: 4 }}>
        <h2 style={{ margin: 0, fontSize: 20 }}>{t.pages.audit.title}</h2>
        <p style={{ margin: '4px 0 12px', color: '#888', fontSize: 12 }}>{t.pages.audit.desc}</p>
      </div>
      <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
        <input
          style={ctl}
          placeholder={t.pages.audit.searchPlaceholder}
          value={keyword}
          onChange={(e) => { setKeyword(e.target.value); setPage(1) }}
        />
        <Dropdown
          value={type}
          options={[{ value: '', label: t.pages.audit.allTypes }, ...t.pages.audit.types.map((ty) => ({ value: ty, label: ty }))]}
          onChange={(v) => { setType(v); setPage(1) }}
          ariaLabel={t.pages.audit.allTypes}
        />
        <DatePicker
          value={date}
          onChange={(v) => { setDate(v); setPage(1) }}
          ariaLabel={t.pages.audit.datePicker.label}
          placeholder={t.pages.audit.datePicker.placeholder}
          labels={t.pages.audit.datePicker}
        />
        <span style={{ flex: 1 }} />
        <button style={btn} onClick={load}>{t.pages.audit.refresh}</button>
      </div>
      <div style={{ background: '#fff', border: '1px solid #f0f0f0', borderRadius: 8, padding: 16 }}>
        <div style={{ fontWeight: 600, marginBottom: 12 }}>
          {t.pages.audit.cardTitle}
          <span style={{ marginLeft: 8, fontWeight: 400, fontSize: 12, color: '#999' }}>
            {filtered.length ? t.pages.audit.matched.replace('{count}', String(filtered.length)) : ''}
          </span>
        </div>
        {error ? <div style={{ color: '#e54545', fontSize: 13, padding: '12px 0' }}>{error}</div> : (
          <>
            <DataTable
              emptyText={t.pages.audit.empty}
              rows={slice as unknown as Record<string, unknown>[]}
              columns={[
                { key: 'time', label: t.pages.audit.columns[0], render: (r) => String(r.time ?? '') },
                { key: 'operator', label: t.pages.audit.columns[1], render: (r) => String(r.operator ?? '') },
                { key: 'type', label: t.pages.audit.columns[2], render: (r) => <span style={tag}>{String(r.type)}</span> },
                { key: 'action', label: t.pages.audit.columns[3], render: (r) => String(r.action ?? '') },
                { key: 'ip', label: t.pages.audit.columns[4], render: (r) => String(r.ip ?? '') },
                { key: 'op', label: t.pages.audit.columns[5], render: (r) => (
                  <a style={{ color: '#1677ff', cursor: 'pointer' }} onClick={() => setDetail(String(r.logId))}>{t.pages.audit.detail}</a>
                ) },
              ]}
            />
            <Pagination
              total={filtered.length}
              page={page}
              pageSize={pageSize}
              onPage={setPage}
              onSize={setPageSize}
              rangeText={t.pages.audit.rangeText}
              prevText={t.pages.audit.prev}
              nextText={t.pages.audit.next}
              perPageText={t.pages.audit.perPage}
              jumpText={t.pages.audit.jump}
              pageUnitText={t.pages.audit.pageUnit}
            />
          </>
        )}
      </div>
      {detailRow && (
        <div style={mask} onClick={() => setDetail(null)}>
          <div style={modal} onClick={(e) => e.stopPropagation()}>
            <h3 style={{ margin: '0 0 12px' }}>{t.pages.audit.detailTitle}</h3>
            <dl style={{ display: 'grid', gridTemplateColumns: '80px 1fr', gap: '8px 12px', fontSize: 13, margin: '0 0 16px' }}>
              <dt style={dtStyle}>{t.pages.audit.columns[0]}</dt><dd style={{ margin: 0 }}>{detailRow.time}</dd>
              <dt style={dtStyle}>{t.pages.audit.columns[1]}</dt><dd style={{ margin: 0 }}>{detailRow.operator}</dd>
              <dt style={dtStyle}>{t.pages.audit.columns[2]}</dt><dd style={{ margin: 0 }}>{detailRow.type}</dd>
              <dt style={dtStyle}>{t.pages.audit.columns[3]}</dt><dd style={{ margin: 0 }}>{detailRow.action}</dd>
              <dt style={dtStyle}>{t.pages.audit.columns[4]}</dt><dd style={{ margin: 0 }}>{detailRow.ip}</dd>
              <dt style={dtStyle}>{t.pages.audit.logId}</dt><dd style={{ margin: 0 }}>{detailRow.logId}</dd>
            </dl>
            <button style={{ ...btn, background: '#1677ff', borderColor: '#1677ff', color: '#fff' }} onClick={() => setDetail(null)}>{t.pages.audit.close}</button>
          </div>
        </div>
      )}
    </div>
  )
}

const ctl: React.CSSProperties = { padding: '6px 10px', border: '1px solid #d9d9d9', borderRadius: 6, fontSize: 13, background: '#fff' }
const btn: React.CSSProperties = { padding: '6px 14px', border: '1px solid #d9d9d9', borderRadius: 6, background: '#fff', cursor: 'pointer', fontSize: 13 }
const tag: React.CSSProperties = { padding: '1px 8px', borderRadius: 4, fontSize: 12, background: '#f0f5ff', color: '#2f54eb', border: '1px solid #adc6ff' }
const mask: React.CSSProperties = { position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.45)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 'var(--z-page-modal)' }
const modal: React.CSSProperties = { background: '#fff', borderRadius: 8, padding: 20, width: 380 }
const dtStyle: React.CSSProperties = { color: '#888' }
