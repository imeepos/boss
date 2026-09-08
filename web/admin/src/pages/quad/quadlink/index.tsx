// 关联查询页:GET /quad-links 全量列表 + POST /quad-links 建链 + by-asset/customer/port/address 反查。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Dropdown } from '../../../components/Dropdown'
import { pageSlice, type QuadLinkRow } from '../types'
import { IdRef, TableStateRow, ToolbarButton } from '../../../components/business'
import { ErrorBanner } from '../../../components/business/page-head'
import { Card, CardContent, CardFooter } from '../../../components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { CreateLinkDrawer } from './CreateLinkDrawer'

// 反查维度:by-asset/by-customer/by-port/by-address(后端四反查路由)。
const REVERSE_KEYS = ['by-asset', 'by-customer', 'by-port', 'by-address'] as const

export default function QuadLinkPage() {
  const t = useT()
  const q = t.pages.quadLinkPage
  const [rows, setRows] = useState<QuadLinkRow[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const [reverseBy, setReverseBy] = useState('by-asset')
  const [reverseId, setReverseId] = useState('')
  const [reverseNote, setReverseNote] = useState('')

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: QuadLinkRow[] }>('/quad-links')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : q.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  // 反查:命中把结果行置顶展示,未命中提示不臆造。
  const reverse = async () => {
    const id = Number(reverseId)
    if (!Number.isInteger(id) || id <= 0) return
    setError('')
    setBusy(true)
    setReverseNote('')
    try {
      const param = { 'by-asset': 'assetId', 'by-customer': 'customerId', 'by-port': 'portId', 'by-address': 'addressId' }[reverseBy as typeof REVERSE_KEYS[number]]
      const d = await apiFetch<QuadLinkRow | null>(`/quad-links/${reverseBy}`, { query: { [param]: id } })
      if (d && (d as QuadLinkRow).id) { setRows([(d as QuadLinkRow), ...rows.filter((x) => x.id !== (d as QuadLinkRow).id)]); setPage(1) }
      else setReverseNote(q.reverseEmpty)
    } catch (e) {
      setError(e instanceof Error ? e.message : q.loadFail)
    } finally {
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={q.title} desc={q.desc} />
      <Card>
        <CardContent>
          <div className="flex flex-wrap items-center gap-2">
            <ToolbarButton primary onClick={() => setCreateOpen(true)}>{q.createBtn}</ToolbarButton>
            <span className="flex-1" />
            <div className="w-36">
              <Dropdown value={reverseBy} ariaLabel={q.reverseLabel} onChange={setReverseBy} options={REVERSE_KEYS.map((k) => ({ value: k, label: q.columns[REVERSE_KEYS.indexOf(k)] ?? k }))} />
            </div>
            <input className="h-8 w-32 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" type="number" placeholder={q.reverseIdPh} value={reverseId} onChange={(e) => setReverseId(e.target.value)} onKeyDown={(e) => { if (e.key === 'Enter') reverse() }} />
            <ToolbarButton disabled={busy} onClick={reverse}>{q.reverseLabel}</ToolbarButton>
            <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
          </div>
        </CardContent>
        {error ? <ErrorBanner message={error} /> : (
          <div className="px-4 pb-4">
            <Table>
              <TableHeader>
                <TableRow>{q.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
              </TableHeader>
              <TableBody>
                {slice.map((x) => (
                  <TableRow key={x.id}>
                    <TableCell><IdRef value={x.id} /></TableCell>
                    <TableCell><IdRef value={x.assetId} /></TableCell>
                    <TableCell><IdRef value={x.customerId} /></TableCell>
                    <TableCell><IdRef value={x.portId} /></TableCell>
                    <TableCell><IdRef value={x.addressId} /></TableCell>
                    <TableCell>{x.legalEntityName || `#${x.legalEntityId}`}</TableCell>
                    <TableCell><StatusTag domain="quad" value={x.status} /></TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={7} loading={busy} text={q.empty} />}
              </TableBody>
            </Table>
          </div>
        )}
        {reverseNote && <p className="mx-4 mb-3 text-[13px] text-[var(--shell-group-title)]">{reverseNote}</p>}
        <CardFooter>
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(q)} />
        </CardFooter>
      </Card>
      {createOpen && <CreateLinkDrawer onDone={() => { setCreateOpen(false); load() }} onClose={() => setCreateOpen(false)} />}
    </div>
  )
}