// 认证账号页(AAA 域,挂 oss 分组):契约 GET /lo-accounts。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Dropdown } from '../../../components/Dropdown'
import { Pagination } from '../../../components/Pagination'
import { type LoAccountRow } from '../types'
import { TableStateRow } from '../../../components/business'
import { useConfirm } from '../../../components/ConfirmDialog'
import { ResetPasswordDialog } from './ResetPasswordDialog'

export default function LoAccountPage() {
  const t = useT()
  const l = t.pages.loAccountPage
  const [rows, setRows] = useState<LoAccountRow[]>([])
  const [total, setTotal] = useState(0)
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [status, setStatus] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [resetBusyLoid, setResetBusyLoid] = useState('')
  const [resetResult, setResetResult] = useState<{ loid: string; password: string } | null>(null)
  const confirmDialog = useConfirm()

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: LoAccountRow[]; total: number }>('/lo-accounts', {
      query: { keyword: keyword.trim() || undefined, status: status || undefined, page, pageSize },
    })
      .then((d) => { setRows(d?.items ?? []); setTotal(d?.total ?? 0) })
      .catch((e) => setError(e instanceof Error ? e.message : l.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, [page, pageSize, keyword, status])

  // 重置接入密码:二次确认 → POST reset-password → 弹层一次性展示随机密码(明文仅本次返回)。
  const handleReset = async (loid: string) => {
    if (!(await confirmDialog(l.resetConfirm, { title: l.resetPwd, danger: true }))) return
    setResetBusyLoid(loid)
    try {
      const d = await apiFetch<{ loid: string; password: string }>('/lo-accounts/' + encodeURIComponent(loid) + '/reset-password', { method: 'POST' })
      if (!d?.password) throw new Error(l.resetFail)
      setResetResult({ loid, password: d.password })
    } catch (e) {
      setError(e instanceof Error ? e.message : l.resetFail)
    } finally {
      setResetBusyLoid('')
    }
  }

  const slice = rows

  return (
    <div>
      <PageHead title={l.title} desc={l.desc} />
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" placeholder={l.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <Dropdown
            value={status}
            options={[
              { value: '', label: l.allStatus },
              { value: 'ACTIVE', label: l.active },
              { value: 'SUSPENDED', label: l.suspended },
              { value: 'CLOSED', label: l.closed },
            ]}
            onChange={(value) => { setStatus(value); setPage(1) }}
            ariaLabel={l.allStatus}
          />
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{[...l.columns, l.colActions].map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.loid}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">#{r.customerId}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.legalEntityName || `#${r.legalEntityId}`}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.regionName || r.regionPath || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.qosTemplateId ? `#${r.qosTemplateId}` : '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.billingMode === 'PREPAID' ? l.prepaid : l.postpaid}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="loAccount" value={r.status} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      <button type="button" data-testid={'reset-pwd-' + r.loid}
                        className="h-7 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[12px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)] disabled:cursor-not-allowed disabled:opacity-60"
                        disabled={resetBusyLoid === r.loid}
                        onClick={() => handleReset(r.loid)}>{l.resetPwd}</button>
                    </td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={8} loading={busy} text={l.empty} />}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={total} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(l)} />
        </div>
      </div>
      {resetResult && <ResetPasswordDialog loid={resetResult.loid} password={resetResult.password} onClose={() => setResetResult(null)} />}
    </div>
  )
}
