// 端口台账页:列名以 fields.md §4.2 为准;契约 GET /resources + GET /ports?resourceId。
import { IdRef } from '../../../components/business'
import { useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { SimplePicker } from '../../../components/pickers/SimplePicker'
import { fmtTime } from '../../../lib/format'
import { Card, CardFooter } from '../../../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { pageSlice, type PortHistoryRow, type PortRow, type ResourceRow } from '../types'
import { PathDrawer } from './path-drawer'
import { TableStateRow, ErrorBanner, ToolbarButton } from '../../../components/business'
import { useProfile } from '../../../layouts/profile'

export default function ResourcePage() {
  const t = useT()
  const r = t.pages.resourcePage
  const profile = useProfile()
  // 置备端口入口:POST /provision/ports 属 menu:provision 域,无权限置灰(与批量导入同规)。
  const canProvision = (profile.permissionCodes ?? []).includes('menu:provision')
  const [devices, setDevices] = useState<ResourceRow[]>([])
  const [rows, setRows] = useState<PortRow[]>([])
  const [error, setError] = useState('')
  const [resourceId, setResourceId] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [history, setHistory] = useState<PortRow | null>(null)
  const [pathPort, setPathPort] = useState<PortRow | null>(null)
  const [historyRows, setHistoryRows] = useState<PortHistoryRow[] | null>(null)
  const [provOpen, setProvOpen] = useState(false)
  const [provDevice, setProvDevice] = useState(0)
  const [provPortId, setProvPortId] = useState('')
  const [provPortRow, setProvPortRow] = useState<PortRow | null>(null)
  // /ports 每次开关抽屉重新拉取;检索在已取全量上做关键字收窄(接口无 keyword 参数)。
  const provPortsRef = useRef<PortRow[] | null>(null)
  const [provError, setProvError] = useState('')

  const loadPorts = (rid: number) => {
    setError('')
    setBusy(true)
    apiFetch<{ items: PortRow[] }>('/ports', { query: { resourceId: rid || undefined } })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => {
        const msg = e instanceof Error ? e.message : r.loadFail
        setError(msg)
        toast.error(r.loadFail, { description: msg })
      })
      .finally(() => setBusy(false))
  }
  useEffect(() => {
    apiFetch<{ items: ResourceRow[] }>('/resources')
      .then((d) => setDevices(d?.items ?? []))
      .catch(() => setDevices([]))
    loadPorts(0)
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const deviceName = (rid: number) => devices.find((d) => d.id === rid)?.name ?? `#${rid}`
  const openHistory = (p: PortRow) => {
    setHistory(p)
    setHistoryRows(null)
    apiFetch<{ items: PortHistoryRow[] }>(`/ports/${p.portId}/change-history`)
      .then((d) => setHistoryRows(d?.items ?? []))
      .catch(() => setHistoryRows([]))
  }

  // 端口选择器数据源:SimplePicker 服务端模式,首拉 /ports 全量缓存,关键字按 portCode/quadCode 收窄。
  const searchProvPorts = async (keyword: string): Promise<{ value: string; label: string }[] | null> => {
    if (!provPortsRef.current) {
      const d = await apiFetch<{ items: PortRow[] }>('/ports')
      provPortsRef.current = d?.items ?? []
    }
    const kw = keyword.trim().toLowerCase()
    return provPortsRef.current
      .filter((p) => !kw || p.portCode.toLowerCase().includes(kw) || (p.quadCode ?? '').toLowerCase().includes(kw))
      .map((p) => ({ value: String(p.portId), label: p.portCode + ' (#' + p.portId + ')' }))
  }
  const onProvPortPick = (v: string) => {
    setProvPortId(v)
    setProvPortRow(provPortsRef.current?.find((p) => String(p.portId) === v) ?? null)
  }

  // provisionPort 置备端口:调 POST /provision/ports(权威置备端点,menu:provision 域),
  // 成功后刷新当前端口列表;portCode 唯一冲突/字段缺失由后端校验回显。
  const provisionPort = async () => {
    if (busy || !provDevice || !provPortRow) return
    setBusy(true)
    setProvError('')
    try {
      const dev = devices.find((d) => d.id === provDevice)
      await apiFetch('/provision/ports', {
        method: 'POST',
        body: {
          portCode: provPortRow.portCode,
          quadCode: provPortRow.quadCode || provPortRow.portCode,
          resourceId: provDevice,
          addressId: dev?.addressId ?? 0,
          legalEntityId: dev?.legalEntityId ?? 0,
        },
      })
      toast.success(r.provision)
      setProvOpen(false)
      setProvDevice(0)
      setProvPortId('')
      setProvPortRow(null)
      loadPorts(resourceId)
    } catch (e) {
      const msg = e instanceof Error ? e.message : r.provSaveFail
      setProvError(msg)
      toast.error(r.provSaveFail, { description: msg })
    } finally {
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, page, pageSize)
  const provDeviceRow = devices.find((d) => d.id === provDevice)

  return (
    <div>
      <PageHead title={r.title} desc={r.desc} />
      <Card className="mb-4">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <Dropdown
            value={resourceId ? String(resourceId) : ''}
            options={[{ value: '', label: r.allDevice }, ...devices.map((d) => ({ value: String(d.id), label: `${d.name} (${d.code})` }))]}
            onChange={(v) => { const n = Number(v) || 0; setResourceId(n); setPage(1); loadPorts(n) }}
            ariaLabel={r.allDevice}
            triggerStyle={{ minWidth: 200 }}
          />
          <span className="flex-1" />
          <span title={canProvision ? undefined : r.provNoPerm}>
            <ToolbarButton primary disabled={busy || !canProvision} onClick={() => { setProvOpen(true); setProvError(''); provPortsRef.current = null }}>
              {r.provision}
            </ToolbarButton>
          </span>
          <ToolbarButton onClick={() => loadPorts(resourceId)} disabled={busy}>
            {t.pages.audit.refresh}
          </ToolbarButton>
        </div>
        {error ? <div className="px-4 pb-3"><ErrorBanner message={error} /></div> : (
          <div className="px-4 pb-4">
            <Table>
              <TableHeader>
                <TableRow>
                  {r.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}
                </TableRow>
              </TableHeader>
              <TableBody>
                {slice.map((p) => (
                  <TableRow key={p.portId}>
                    <TableCell>{p.portCode}</TableCell>
                    <TableCell>{p.quadCode || '—'}</TableCell>
                    <TableCell>{deviceName(p.resourceId)}</TableCell>
                    <TableCell>{p.addressId ? <IdRef value={p.addressId} /> : '—'}</TableCell>
                    <TableCell><StatusTag domain="port" value={p.status} /></TableCell>
                    <TableCell>{p.orderId ? <IdRef value={p.orderId} /> : '—'}</TableCell>
                    <TableCell>
                      <span className="inline-flex items-center gap-2">
                        <button type="button" onClick={() => setPathPort(p)}>{r.linkView}</button>
                        <span className="text-[var(--shell-side-border)]">|</span>
                        <button type="button" onClick={() => openHistory(p)}>{r.history}</button>
                      </span>
                    </TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={7} loading={busy} text={r.empty} />}
              </TableBody>
            </Table>
          </div>
        )}
        <CardFooter>
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(r)} />
        </CardFooter>
      </Card>
      {pathPort && <PathDrawer port={pathPort} onClose={() => setPathPort(null)} />}
      {provOpen && (
        <Drawer title={r.provTitle} onClose={() => setProvOpen(false)}
          footer={
            <>
              <ToolbarButton onClick={() => setProvOpen(false)}>{t.pages.company.cancel}</ToolbarButton>
              <ToolbarButton primary
                disabled={busy || !provDevice || !provPortRow} onClick={provisionPort}>
                {busy ? t.pages.account.submitting : t.pages.company.save}
              </ToolbarButton>
            </>
          }>
          <div className="flex flex-col gap-3.5">
            <div className="flex flex-col gap-1.5">
              <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{r.provDevice}</label>
              <Dropdown
                value={provDevice ? String(provDevice) : ''}
                options={[{ value: '', label: r.allDevice }, ...devices.map((d) => ({ value: String(d.id), label: d.name + ' (' + d.code + ')' }))]}
                onChange={(v) => setProvDevice(Number(v) || 0)}
                ariaLabel={r.provDevice}
                triggerStyle={{ minWidth: 200 }}
              />
            </div>
            {provDeviceRow && (
              <div className="flex flex-col gap-1.5">
                <label>{r.provAddress}</label>
                <div className="text-[13px] text-[var(--shell-content-text)]"><IdRef value={provDeviceRow.addressId} /></div>
              </div>
            )}
            <div className="flex flex-col gap-1.5">
              <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{r.fPortCode}</label>
              <SimplePicker
                value={provPortId}
                search={searchProvPorts}
                onChange={onProvPortPick}
                ariaLabel={r.fPortCode}
                placeholder={r.pPortCode}
                searchPlaceholder={r.pPortCode}
                errorText={r.loadFail}
                clearable
                clearLabel={t.pages.pickers.common.clear}
                minWidth={220}
              />
            </div>
            {provError && <ErrorBanner message={provError} className="mx-0" />}
          </div>
        </Drawer>
      )}
      {history && (
        <Drawer title={`${r.historyTitle} · ${history.portCode}`} onClose={() => setHistory(null)}
          footer={<ToolbarButton primary onClick={() => setHistory(null)}>
            {t.pages.company.cancel}
          </ToolbarButton>}>
          <div className="px-4 pb-4">
            <Table>
              <TableHeader>
                <TableRow>
                  {r.historyColumns.map((x) => <TableHead key={x}>{x}</TableHead>)}
                </TableRow>
              </TableHeader>
              <TableBody>
                {(historyRows ?? []).map((h) => (
                  <TableRow key={h.id}>
                    <TableCell>{fmtTime(h.changedAt)}</TableCell>
                    <TableCell><StatusTag domain="port" value={h.status} /></TableCell>
                    <TableCell>{h.orderId ? `#${h.orderId}` : '—'}</TableCell>
                  </TableRow>
                ))}
                {historyRows !== null && !historyRows.length && (
                  <TableRow><TableCell colSpan={3} className="h-11 px-3 text-center text-[var(--shell-group-title)]">{r.empty}</TableCell></TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </Drawer>
      )}
    </div>
  )
}
