// 拆机管理页:契约 GET /dismantles(裸列表)+ POST /dismantles(无 json tag,键为 Go 字段名)。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { ErrorBanner, ToolbarButton, IdRef } from '../../../components/business'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { ResourcePicker } from '../../../components/ResourcePicker'
import { Card } from '../../../components/ui/card'
import { Button } from '../../../components/ui/button'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { pageSlice, type DismantleRow, type OrderListRow } from '../types'
import type { AssetRow } from '../../ams/types'
import type { PortRow } from '../../oss/types'
import { TableStateRow } from '../../../components/business'

export default function DismantlePage() {
  const t = useT()
  const d = t.pages.dismantlePage
  const [rows, setRows] = useState<DismantleRow[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [open, setOpen] = useState(false)
  const [orderId, setOrderId] = useState('')
  const [assetId, setAssetId] = useState('')
  const [portId, setPortId] = useState('')
  const [formError, setFormError] = useState('')

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<DismantleRow[]>('/dismantles')
      .then((x) => setRows(Array.isArray(x) ? x : []))
      .catch((e) => setError(e instanceof Error ? e.message : d.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const submit = async () => {
    if (busy) return
    setBusy(true)
    setFormError('')
    try {
      await apiFetch('/dismantles', {
        method: 'POST',
        body: { OrderID: Number(orderId), AssetID: Number(assetId) || 0, PortID: Number(portId) || 0 },
      })
      toast.success(d.toastCreateOk)
      setOpen(false)
      setOrderId('')
      setAssetId('')
      setPortId('')
      load()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : d.saveFail)
    } finally {
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, page, pageSize)
  const orderOk = /^\d+$/.test(orderId) && Number(orderId) > 0

  return (
    <div>
      <PageHead title={d.title} desc={d.desc} />
      <Card>
        <div className="flex flex-wrap items-center gap-2 p-4">
          <span className="spacer" />
          <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
          <ToolbarButton primary onClick={() => setOpen(true)}>{d.create}</ToolbarButton>
        </div>
        {error && <ErrorBanner message={error} />}
        <div className="px-4 pb-4">
          <Table>
            <TableHeader>
              <TableRow>{d.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
            </TableHeader>
            <TableBody>
              {slice.map((x) => (
                <TableRow key={x.id}>
                  <TableCell>{x.dismantleNo || `#${x.id}`}</TableCell>
                  <TableCell><IdRef value={x.orderId} /></TableCell>
                  <TableCell>{x.legalEntityName || `#${x.legalEntityId}`}</TableCell>
                  <TableCell>{x.assetId ? <IdRef value={x.assetId} /> : '—'}</TableCell>
                  <TableCell>{x.portId ? <IdRef value={x.portId} /> : '—'}</TableCell>
                  <TableCell><StatusTag domain="task" value={x.status} /></TableCell>
                </TableRow>
              ))}
              {!slice.length && <TableStateRow colSpan={6} loading={busy} text={d.empty} />}
            </TableBody>
          </Table>
        </div>
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={(s) => { setPageSize(s); setPage(1) }} {...pagerTexts(d)} />
        </div>
      </Card>
      {open && (
        <Drawer title={d.createTitle} onClose={() => setOpen(false)}
          footer={
            <>
              <Button variant="outline" size="sm" onClick={() => setOpen(false)}>{t.pages.company.cancel}</Button>
              <Button size="sm" disabled={busy || !orderOk} onClick={submit}>
                {busy ? t.pages.account.submitting : t.pages.company.save}
              </Button>
            </>
          }>
          <div className="flex flex-col gap-3.5">
            <div className="flex flex-col gap-1.5">
              <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{d.fOrder}</label>
              <ResourcePicker
                value={orderId}
                onChange={setOrderId}
                load={() => apiFetch<{ items: OrderListRow[] }>('/orders').then((x) => x?.items ?? [])}
                toOption={(o) => ({ value: String(o.id), label: `${o.orderNo}${o.customer ? ` · ${o.customer}` : ''}` })}
                ariaLabel={d.fOrder}
                searchPlaceholder={d.pickSearch}
                errorText={d.loadFail}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <label>{d.fAsset}</label>
              <ResourcePicker
                value={assetId}
                onChange={setAssetId}
                load={() => apiFetch<{ items: AssetRow[] }>('/assets').then((x) => x?.items ?? [])}
                toOption={(a) => ({ value: String(a.assetId), label: `${a.assetCode} (#${a.assetId})` })}
                ariaLabel={d.fAsset}
                emptyLabel={d.pickEmpty}
                searchPlaceholder={d.pickSearch}
                errorText={d.loadFail}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <label>{d.fPort}</label>
              <ResourcePicker
                value={portId}
                onChange={setPortId}
                load={() => apiFetch<{ items: PortRow[] }>('/ports').then((x) => x?.items ?? [])}
                toOption={(p) => ({ value: String(p.portId), label: `${p.portCode} (#${p.portId})` })}
                ariaLabel={d.fPort}
                emptyLabel={d.pickEmpty}
                searchPlaceholder={d.pickSearch}
                errorText={d.loadFail}
              />
            </div>
            {formError && <div className="rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] break-all text-[var(--color-danger)]">{formError}</div>}
          </div>
        </Drawer>
      )}
    </div>
  )
}
