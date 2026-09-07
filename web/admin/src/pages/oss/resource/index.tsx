// 端口台账页:列名以 fields.md §4.2 为准;契约 GET /resources + GET /ports?resourceId。
import { IdRef } from '../../../components/business'
import { useEffect, useRef, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { SimplePicker } from '../../../components/pickers/SimplePicker'
import { fmtTime } from '../../../lib/format'
import { pageSlice, type PortHistoryRow, type PortRow, type ResourceRow } from '../types'
import { PathDrawer } from './path-drawer'
import { TableStateRow, EmptyState, ErrorBanner } from '../../../components/business'
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
      .catch((e) => setError(e instanceof Error ? e.message : r.loadFail))
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
      setProvOpen(false)
      setProvDevice(0)
      setProvPortId('')
      setProvPortRow(null)
      loadPorts(resourceId)
    } catch (e) {
      setProvError(e instanceof Error ? e.message : r.provSaveFail)
    } finally {
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, page, pageSize)
  const provDeviceRow = devices.find((d) => d.id === provDevice)

  return (
    <div>
      <PageHead title={r.title} desc={r.desc} />
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <Dropdown
            value={resourceId ? String(resourceId) : ''}
            options={[{ value: '', label: r.allDevice }, ...devices.map((d) => ({ value: String(d.id), label: `${d.name} (${d.code})` }))]}
            onChange={(v) => { const n = Number(v) || 0; setResourceId(n); setPage(1); loadPorts(n) }}
            ariaLabel={r.allDevice}
            triggerStyle={{ minWidth: 200 }}
          />
          <span className="spacer" />
          <span title={canProvision ? undefined : r.provNoPerm}>
            <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)] disabled:cursor-not-allowed" disabled={busy || !canProvision} onClick={() => { setProvOpen(true); setProvError(''); provPortsRef.current = null }}>
              {r.provision}
            </button>
          </span>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={() => loadPorts(resourceId)}>
            {t.pages.audit.refresh}
          </button>
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{r.columns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((p) => (
                  <tr key={p.portId}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{p.portCode}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{p.quadCode || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{deviceName(p.resourceId)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{p.addressId ? <IdRef value={p.addressId} /> : '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="port" value={p.status} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{p.orderId ? <IdRef value={p.orderId} /> : '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      <span className="inline-flex items-center gap-2">
                        <button onClick={() => setPathPort(p)}>{r.linkView}</button>
                        <button onClick={() => openHistory(p)}>{r.history}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={7} loading={busy} text={r.empty} />}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(r)} />
        </div>
      </div>
      {pathPort && <PathDrawer port={pathPort} onClose={() => setPathPort(null)} />}
      {provOpen && (
        <Drawer title={r.provTitle} onClose={() => setProvOpen(false)}
          footer={
            <>
              <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={() => setProvOpen(false)}>{t.pages.company.cancel}</button>
              <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]"
                disabled={busy || !provDevice || !provPortRow} onClick={provisionPort}>
                {busy ? t.pages.account.submitting : t.pages.company.save}
              </button>
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
          footer={<button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => setHistory(null)}>
            {t.pages.company.cancel}
          </button>}>
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{r.historyColumns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {(historyRows ?? []).map((h) => (
                  <tr key={h.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtTime(h.changedAt)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="port" value={h.status} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{h.orderId ? `#${h.orderId}` : '—'}</td>
                  </tr>
                ))}
                {historyRows !== null && !historyRows.length && (
                  <tr><td colSpan={3} className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><EmptyState text={r.empty} /></td></tr>
                )}
              </tbody>
            </table>
          </div>
        </Drawer>
      )}
    </div>
  )
}
