// 入驻申请审核页(sysadmin,menu:partner):状态页签 + 审核(通过开通/驳回记意见)。
// 契约:GET/POST /partner/applications*(fields.md 8B);状态口径 PENDING/APPROVED/REJECTED。
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'
import {
  listPartnerApplications, approvePartnerApplication, rejectPartnerApplication,
  type PartnerApplication, type PartnerApproveResult,
} from '../../../api/partner'
import { useT } from '../../../i18n'
import { useQueryState } from '../../../lib/useQueryState'
import { Badge } from '../../../components/ui/badge'
import { Card } from '../../../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { PageHead, ErrorBanner, EmptyState, ToolbarButton } from '../../../components/business/page-head'
import { Drawer } from '../../../components/Drawer'
import { Pagination } from '../../../components/Pagination'
import { PartnerReviewActions } from './ReviewActions'

const TABS = ['', 'PENDING', 'APPROVED', 'REJECTED'] as const

/** 状态徽标颜色映射。 */
function StatusBadge({ status }: { status: string }) {
  const t = useT()
  const map = {
    PENDING: { label: t.pages.partnerReview.statusPending, variant: 'warning' as const },
    APPROVED: { label: t.pages.partnerReview.statusApproved, variant: 'success' as const },
    REJECTED: { label: t.pages.partnerReview.statusRejected, variant: 'danger' as const },
  } as const
  const it = map[status as keyof typeof map]
  if (!it) return <span>{status}</span>
  return <Badge variant={it.variant}>{it.label}</Badge>
}

export default function PartnerReviewPage() {
  const t = useT()
  const [urlStatus, setUrlStatus] = useQueryState('status', '')
  const status = (TABS as readonly string[]).includes(urlStatus) ? urlStatus : ''
  const [items, setItems] = useState<PartnerApplication[]>([])
  const [error, setError] = useState('')
  const [detail, setDetail] = useState<PartnerApplication | null>(null)
  const [approveResult, setApproveResult] = useState<PartnerApproveResult | null>(null)
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)

  const load = useCallback(() => {
    setError('')
    listPartnerApplications(status)
      .then((d) => setItems(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : t.pages.partnerReview.loadFail))
  }, [status, t]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(load, [load])

  // 检索+客户端分页(全量拉取页,数据量小;服务端分页登记为后续契约项)。
  const kw = keyword.trim().toLowerCase()
  const shown = items.filter((r) => !kw
    || r.companyName.toLowerCase().includes(kw) || r.creditCode.toLowerCase().includes(kw)
    || r.contactName.toLowerCase().includes(kw))
  const slice = shown.slice((page - 1) * pageSize, page * pageSize)

  const tabLabel = (v: string) =>
    v === '' ? t.pages.partnerReview.tabAll
      : v === 'PENDING' ? t.pages.partnerReview.statusPending
        : v === 'APPROVED' ? t.pages.partnerReview.statusApproved
          : t.pages.partnerReview.statusRejected

  const onApprove = async (id: number) => {
    try {
      const res = await approvePartnerApplication(id)
      if (!res) throw new Error(t.pages.partnerReview.loadFail)
      setApproveResult(res)
    } catch (e) {
      setError(e instanceof Error ? e.message : t.pages.partnerReview.loadFail)
    }
    load()
    setDetail(null)
  }

  const onReject = async (id: number, note: string) => {
    try {
      await rejectPartnerApplication(id, note)
      toast.success(t.pages.partnerReview.rejectOk)
    } catch (e) {
      setError(e instanceof Error ? e.message : t.pages.partnerReview.loadFail)
    }
    load()
    setDetail(null)
  }

  return (
    <div>
      <PageHead title={t.pages.partnerReview.title} desc={t.pages.partnerReview.desc} />
      <nav className="mb-4 flex gap-6 border-b border-[var(--shell-side-border)]" role="tablist">
        {TABS.map((v) => (
          <button
            key={v || 'all'}
            role="tab"
            aria-selected={status === v}
            className={`mb-[-1px] cursor-pointer border-b-2 bg-none px-0.5 pt-2.5 pb-3 text-sm ${
              status === v
                ? 'font-semibold text-[var(--shell-heading)] border-b-[var(--color-brand-gold-500)]'
                : 'border-transparent text-[var(--shell-content-text)] hover:text-[var(--shell-heading)]'
            }`}
            onClick={() => setUrlStatus(v)}
          >
            {tabLabel(v)}
          </button>
        ))}
        <div className="flex-1" />
        <input
          className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]"
          placeholder={t.pages.partnerReview.colCompany}
          value={keyword}
          onChange={(e) => { setKeyword(e.target.value); setPage(1) }}
        />
        <ToolbarButton onClick={load}>{t.pages.partnerReview.refresh}</ToolbarButton>
      </nav>
      <Card className="p-4">
        {error ? <ErrorBanner message={error} /> : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t.pages.partnerReview.colCompany}</TableHead>
                <TableHead>{t.pages.partnerReview.colCreditCode}</TableHead>
                <TableHead>{t.pages.partnerReview.colContact}</TableHead>
                <TableHead>{t.pages.partnerReview.colStatus}</TableHead>
                <TableHead>{t.pages.partnerReview.colSubmitted}</TableHead>
                <TableHead>{t.pages.partnerReview.colOp}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {slice.map((r) => (
                <TableRow key={r.id} className="cursor-pointer" onClick={() => setDetail(r)}>
                  <TableCell className="font-medium">{r.companyName}</TableCell>
                  <TableCell>{r.creditCode}</TableCell>
                  <TableCell>{r.contactName} / {r.contactPhone}</TableCell>
                  <TableCell><StatusBadge status={r.status} /></TableCell>
                  <TableCell className="whitespace-nowrap">{r.submittedAt.slice(0, 10)}</TableCell>
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <PartnerReviewActions row={r} onApprove={onApprove} onReject={onReject} />
                  </TableCell>
                </TableRow>
              ))}
              {!shown.length && (
                <TableRow><TableCell colSpan={6}><EmptyState text={t.pages.partnerReview.empty} /></TableCell></TableRow>
              )}
            </TableBody>
          </Table>
        )}
        <div className="mt-2 flex items-center justify-between text-xs text-[var(--shell-group-title)]">
          <span>{shown.length}</span>
          <Pagination
            total={shown.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={(s) => { setPageSize(s); setPage(1) }}
            rangeText={t.pages.company.rangeText} prevText={t.pages.company.prev}
            nextText={t.pages.company.next} perPageText={t.pages.company.perPage}
            jumpText={t.pages.company.jumpText} pageUnitText={t.pages.company.pageUnit}
          />
        </div>
      </Card>

      {detail && (
        <Drawer title={detail.companyName} onClose={() => setDetail(null)}>
          <dl className="grid grid-cols-[110px_1fr] gap-x-3 gap-y-2 text-[13px]">
            <dt className="text-[var(--shell-crumb-text)]">{t.pages.partnerReview.colCompany}</dt>
            <dd className="m-0">{detail.companyName}</dd>
            <dt className="text-[var(--shell-crumb-text)]">{t.pages.partnerReview.colCreditCode}</dt>
            <dd className="m-0">{detail.creditCode}</dd>
            <dt className="text-[var(--shell-crumb-text)]">{t.pages.partnerReview.colContact}</dt>
            <dd className="m-0">{detail.contactName} / {detail.contactPhone}</dd>
            <dt className="text-[var(--shell-crumb-text)]">{t.pages.partnerReview.email}</dt>
            <dd className="m-0">{detail.email || '-'}</dd>
            <dt className="text-[var(--shell-crumb-text)]">{t.pages.partnerReview.businessDesc}</dt>
            <dd className="m-0">{detail.businessDesc}</dd>
            <dt className="text-[var(--shell-crumb-text)]">{t.pages.partnerReview.colStatus}</dt>
            <dd className="m-0"><StatusBadge status={detail.status} /></dd>
            {detail.reviewNote && (
              <>
                <dt className="text-[var(--shell-crumb-text)]">{t.pages.partnerReview.rejectReason}</dt>
                <dd className="m-0">{detail.reviewNote}</dd>
              </>
            )}
            <dt className="text-[var(--shell-crumb-text)]">{t.pages.partnerReview.colSubmitted}</dt>
            <dd className="m-0">{detail.submittedAt.slice(0, 19).replace('T', ' ')}</dd>
          </dl>
        </Drawer>
      )}

      {approveResult && (
        <div className="fixed inset-0 z-page-modal flex items-center justify-center bg-black/45"
          onClick={() => setApproveResult(null)}>
          <div className="w-105 rounded-md bg-[var(--shell-card-bg)] p-5 shadow-[var(--shadow-panel)]"
            onClick={(e) => e.stopPropagation()}>
            <h3 className="mb-3 text-base font-semibold text-[var(--shell-heading)]">
              {t.pages.partnerReview.approveResultTitle}
            </h3>
            <dl className="mb-4 grid grid-cols-[110px_1fr] gap-x-3 gap-y-2 text-[13px]">
              <dt className="text-[var(--shell-crumb-text)]">{t.pages.partnerReview.approveResultUser}</dt>
              <dd className="m-0 font-mono">{approveResult.username}</dd>
              <dt className="text-[var(--shell-crumb-text)]">{t.pages.partnerReview.approveResultPwd}</dt>
              <dd className="m-0 font-mono">{approveResult.initialPassword}</dd>
            </dl>
            <p className="m-0 mb-4 text-xs text-[var(--color-danger)]">{t.pages.partnerReview.pwdTip}</p>
            <ToolbarButton primary onClick={() => setApproveResult(null)}>
              {t.pages.partnerReview.close}
            </ToolbarButton>
          </div>
        </div>
      )}
    </div>
  )
}
