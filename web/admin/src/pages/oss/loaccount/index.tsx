// 认证账号页(AAA 域,挂 oss 分组):契约 GET /lo-accounts。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Dropdown } from '../../../components/Dropdown'
import { Pagination } from '../../../components/Pagination'
import { Card, CardFooter } from '../../../components/ui/card'
import { Input } from '../../../components/ui/input'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { type LoAccountRow } from '../types'
import { TableStateRow, IdRef, ErrorBanner, ToolbarButton } from '../../../components/business'
import { useConfirm } from '../../../components/ConfirmDialog'
import { useDebouncedValue } from '../../../lib/useDebouncedValue'
import { ResetPasswordDialog } from './ResetPasswordDialog'

export default function LoAccountPage() {
  const t = useT()
  const l = t.pages.loAccountPage
  const [rows, setRows] = useState<LoAccountRow[]>([])
  const [total, setTotal] = useState(0)
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const debouncedKeyword = useDebouncedValue(keyword)
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
      query: { keyword: debouncedKeyword.trim() || undefined, status: status || undefined, page, pageSize },
    })
      .then((d) => { setRows(d?.items ?? []); setTotal(d?.total ?? 0) })
      .catch((e) => setError(e instanceof Error ? e.message : l.loadFail))
      .finally(() => setBusy(false))
  }
  // keyword 防抖:每击键即请求改 300ms 停顿触发。
  useEffect(load, [page, pageSize, debouncedKeyword, status])

  // 重置接入密码:二次确认 → POST reset-password → 弹层一次性展示随机密码(明文仅本次返回)。
  const handleReset = async (loid: string) => {
    if (!(await confirmDialog(l.resetConfirm, { title: l.resetPwd, danger: true }))) return
    setResetBusyLoid(loid)
    try {
      const d = await apiFetch<{ loid: string; password: string }>('/lo-accounts/' + encodeURIComponent(loid) + '/reset-password', { method: 'POST' })
      if (!d?.password) throw new Error(l.resetFail)
      setResetResult({ loid, password: d.password })
      toast.success(l.resetPwd)
    } catch (e) {
      const msg = e instanceof Error ? e.message : l.resetFail
      setError(msg)
      toast.error(l.resetPwd, { description: msg })
    } finally {
      setResetBusyLoid('')
    }
  }

  const slice = rows

  return (
    <div>
      <PageHead title={l.title} desc={l.desc} />
      <Card className="mb-4">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <Input className="w-56" placeholder={l.searchPlaceholder}
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
          <span className="flex-1" />
          <ToolbarButton onClick={load} disabled={busy}>{t.pages.audit.refresh}</ToolbarButton>
        </div>
        {error ? <div className="px-4 pb-3"><ErrorBanner message={error} /></div> : (
          <div className="px-4 pb-4">
            <Table>
              <TableHeader>
                <TableRow>
                  {[...l.columns, l.colActions].map((x) => <TableHead key={x}>{x}</TableHead>)}
                </TableRow>
              </TableHeader>
              <TableBody>
                {slice.map((r) => (
                  <TableRow key={r.id}>
                    <TableCell>{r.loid}</TableCell>
                    <TableCell><IdRef value={r.customerId} /></TableCell>
                    <TableCell>{r.legalEntityName || `#${r.legalEntityId}`}</TableCell>
                    <TableCell>{r.regionName || r.regionPath || '—'}</TableCell>
                    <TableCell>{r.qosTemplateId ? `#${r.qosTemplateId}` : '—'}</TableCell>
                    <TableCell>{r.billingMode === 'PREPAID' ? l.prepaid : l.postpaid}</TableCell>
                    <TableCell><StatusTag domain="loAccount" value={r.status} /></TableCell>
                    <TableCell>
                      <button type="button" data-testid={'reset-pwd-' + r.loid}
                        className="h-7 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[12px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)] disabled:cursor-not-allowed disabled:opacity-60"
                        disabled={resetBusyLoid === r.loid}
                        onClick={() => handleReset(r.loid)}>{resetBusyLoid === r.loid ? t.common.loading : l.resetPwd}</button>
                    </TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={8} loading={busy} text={l.empty} />}
              </TableBody>
            </Table>
          </div>
        )}
        <CardFooter>
          <Pagination total={total} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(l)} />
        </CardFooter>
      </Card>
      {resetResult && <ResetPasswordDialog loid={resetResult.loid} password={resetResult.password} onClose={() => setResetResult(null)} />}
    </div>
  )
}
