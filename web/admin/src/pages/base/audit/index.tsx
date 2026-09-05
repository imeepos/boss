// 审计日志页:列名与交互照抄 docs/admin/audit.html 原型。
// 契约: GET /audit-logs → {items:[audit.Entry]}(menu:audit 门禁)。
// 样式统一走 shell-* 令牌与 business/ui 组件,详情用 Drawer(禁内联裸色值)。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Dropdown } from '../../../components/Dropdown'
import { DatePicker } from '../../../components/DatePicker'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { Badge } from '../../../components/ui/badge'
import { DataTable } from '../../../components/business/data-table'
import { PageHead, ErrorBanner, ToolbarButton, ActionLink } from '../../../components/business/page-head'
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
      <PageHead title={t.pages.audit.title} desc={t.pages.audit.desc} />
      <div className="mb-3 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <input
            className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]"
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
          <span className="flex-1" />
          <ToolbarButton onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
        </div>
        {error ? <ErrorBanner message={error} className="!mx-0" /> : (
          <div className="px-4 pb-4">
            <div className="mb-3 flex items-baseline gap-2">
              <span className="text-[15px] font-semibold text-[var(--shell-heading)]">{t.pages.audit.cardTitle}</span>
              <span className="text-xs text-[var(--shell-group-title)]">
                {filtered.length ? t.pages.audit.matched.replace('{count}', String(filtered.length)) : ''}
              </span>
            </div>
            <DataTable
              emptyText={t.pages.audit.empty}
              rows={slice as unknown as Record<string, unknown>[]}
              columns={[
                { key: 'time', label: t.pages.audit.columns[0], render: (r) => String(r.time ?? '') },
                { key: 'operator', label: t.pages.audit.columns[1], render: (r) => String(r.operator ?? '') },
                { key: 'type', label: t.pages.audit.columns[2], render: (r) => <Badge>{String(r.type)}</Badge> },
                { key: 'action', label: t.pages.audit.columns[3], render: (r) => String(r.action ?? '') },
                { key: 'ip', label: t.pages.audit.columns[4], render: (r) => String(r.ip ?? '') },
                { key: 'op', label: t.pages.audit.columns[5], render: (r) => (
                  <ActionLink label={t.pages.audit.detail} onClick={() => setDetail(String(r.logId))} />
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
          </div>
        )}
      </div>
      {detailRow && (
        <Drawer title={t.pages.audit.detailTitle} onClose={() => setDetail(null)}
          footer={<ToolbarButton onClick={() => setDetail(null)}>{t.pages.audit.close}</ToolbarButton>}>
          <dl className="m-0 mb-4 grid grid-cols-[80px_1fr] gap-x-3 gap-y-2 text-[13px]">
            <dt className="text-[var(--shell-group-title)]">{t.pages.audit.columns[0]}</dt><dd className="m-0 text-[var(--shell-content-text)]">{detailRow.time}</dd>
            <dt className="text-[var(--shell-group-title)]">{t.pages.audit.columns[1]}</dt><dd className="m-0 text-[var(--shell-content-text)]">{detailRow.operator}</dd>
            <dt className="text-[var(--shell-group-title)]">{t.pages.audit.columns[2]}</dt><dd className="m-0 text-[var(--shell-content-text)]">{detailRow.type}</dd>
            <dt className="text-[var(--shell-group-title)]">{t.pages.audit.columns[3]}</dt><dd className="m-0 text-[var(--shell-content-text)]">{detailRow.action}</dd>
            <dt className="text-[var(--shell-group-title)]">{t.pages.audit.columns[4]}</dt><dd className="m-0 text-[var(--shell-content-text)]">{detailRow.ip}</dd>
            <dt className="text-[var(--shell-group-title)]">{t.pages.audit.logId}</dt><dd className="m-0 text-[var(--shell-content-text)]">{detailRow.logId}</dd>
          </dl>
        </Drawer>
      )}
    </div>
  )
}
