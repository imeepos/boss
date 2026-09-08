// 扩容申请页:契约 GET /expansions、POST /expansions、POST /expansions/:expansionNo/execute|reject。
// 执行=PENDING 单选目标设备批量建端口(写侧补齐);驳回=终态;两者均 menu:transfer 权限组。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { Card, CardFooter } from '../../../components/ui/card'
import { Input } from '../../../components/ui/input'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { pageSlice, type ExpansionRow, type LegalEntityRow, type RegionRefRow, type ResourceRow } from '../types'
import { TableStateRow, ErrorBanner, ToolbarButton } from '../../../components/business'
import { useConfirm } from '../../../components/ConfirmDialog'

export default function ExpandPage() {
  const t = useT()
  const confirmDialog = useConfirm()
  const e = t.pages.expandPage
  const [rows, setRows] = useState<ExpansionRow[]>([])
  const [companies, setCompanies] = useState<LegalEntityRow[]>([])
  const [regions, setRegions] = useState<RegionRefRow[]>([])
  const [devices, setDevices] = useState<ResourceRow[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [open, setOpen] = useState(false)
  const [legalEntityId, setLegalEntityId] = useState(0)
  const [regionId, setRegionId] = useState(0)
  const [expectedPorts, setExpectedPorts] = useState('')
  const [formError, setFormError] = useState('')
  const [execTarget, setExecTarget] = useState<ExpansionRow | null>(null)
  const [execDevice, setExecDevice] = useState(0)
  const [execError, setExecError] = useState('')

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: ExpansionRow[] }>('/expansions')
      .then((d) => setRows(d?.items ?? []))
      .catch((x) => {
        const msg = x instanceof Error ? x.message : e.loadFail
        setError(msg)
        toast.error(e.loadFail, { description: msg })
      })
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => {
    // 公司选项取自 QoS 模板(模板按公司归属,与 /expansions 同 menu:transfer 权限组)。
    apiFetch<{ items: { legalEntityId: number; legalEntityName?: string }[] }>('/expansions/qos-templates')
      .then((d) => {
        const seen = new Map<number, string>()
        for (const q of d?.items ?? []) seen.set(q.legalEntityId, q.legalEntityName ?? '')
        setCompanies([...seen].map(([id, name]) => ({ id, name: name || '#' + id })))
      })
      .catch(() => setCompanies([]))
    apiFetch<RegionRefRow[]>('/regions')
      .then((d) => setRegions(d ?? []))
      .catch(() => setRegions([]))
    apiFetch<{ items: ResourceRow[] }>('/resources')
      .then((d) => setDevices(d?.items ?? []))
      .catch(() => setDevices([]))
  }, [])

  const submit = async () => {
    if (busy) return
    setBusy(true)
    setFormError('')
    try {
      await apiFetch('/expansions', {
        method: 'POST',
        body: { legalEntityId, regionId, expectedPorts: Number(expectedPorts) },
      })
      setOpen(false)
      setLegalEntityId(0)
      setRegionId(0)
      setExpectedPorts('')
      toast.success(e.create)
      load()
    } catch (x) {
      const msg = x instanceof Error ? x.message : e.saveFail
      setFormError(msg)
      toast.error(e.saveFail, { description: msg })
    } finally {
      setBusy(false)
    }
  }

  // reject 驳回扩容单(PENDING→DONE 终态)。
  const reject = async (row: ExpansionRow) => {
    if (busy) return
    const no = row.expansionNo || '#' + row.id
    if (!(await confirmDialog(e.rejectConfirm.replace('{no}', no), { danger: true }))) return
    setBusy(true)
    try {
      await apiFetch('/expansions/' + encodeURIComponent(row.expansionNo) + '/reject', { method: 'POST' })
      toast.success(e.reject)
      load()
    } catch (x) {
      const msg = x instanceof Error ? x.message : e.actionFail
      setError(msg)
      toast.error(e.actionFail, { description: msg })
    } finally {
      setBusy(false)
    }
  }

  // execute 执行扩容:选同法人设备,后端按 expectedPorts 批量建端口(PENDING→DONE)。
  const execute = async () => {
    if (busy || !execTarget || !execDevice) return
    setBusy(true)
    setExecError('')
    try {
      const res = await apiFetch<{ created: number; totalPorts: number }>('/expansions/' + encodeURIComponent(execTarget.expansionNo) + '/execute', {
        method: 'POST',
        body: { resourceId: execDevice },
      })
      const msg = e.execOk.replace('{n}', String(res?.created ?? 0))
      toast.success(msg)
      setExecTarget(null)
      setExecDevice(0)
      load()
    } catch (x) {
      const msg = x instanceof Error ? x.message : e.actionFail
      setExecError(msg)
      toast.error(e.actionFail, { description: msg })
    } finally {
      setBusy(false)
    }
  }

  const companyName = (id: number) => companies.find((c) => c.id === id)?.name ?? '#' + id
  const regionName = (id: number) => regions.find((x) => x.id === id)?.name ?? '#' + id
  // execDevices 执行抽屉设备选项:限扩容单同法人(后端归属校验的前端前置)。
  const execDevices = execTarget ? devices.filter((d) => d.legalEntityId === execTarget.legalEntityId) : []
  const slice = pageSlice(rows, page, pageSize)
  const portsOk = /^\d+$/.test(expectedPorts) && Number(expectedPorts) > 0

  return (
    <div>
      <PageHead title={e.title} desc={e.desc} />
      <Card className="mb-4">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <span className="flex-1" />
          <ToolbarButton onClick={load} disabled={busy}>{t.pages.audit.refresh}</ToolbarButton>
          <ToolbarButton primary onClick={() => setOpen(true)}>{e.create}</ToolbarButton>
        </div>
        {error ? <div className="px-4 pb-3"><ErrorBanner message={error} /></div> : (
          <div className="px-4 pb-4">
            <Table>
              <TableHeader>
                <TableRow>
                  {e.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}
                </TableRow>
              </TableHeader>
              <TableBody>
                {slice.map((x) => (
                  <TableRow key={x.id}>
                    <TableCell>{x.expansionNo || '#' + x.id}</TableCell>
                    <TableCell>{companyName(x.legalEntityId)}</TableCell>
                    <TableCell>{regionName(x.regionId)}</TableCell>
                    <TableCell>{x.expectedPorts}</TableCell>
                    <TableCell><StatusTag domain="task" value={x.status} /></TableCell>
                    <TableCell>
                      {x.status === 'PENDING' ? (
                        <span className="inline-flex items-center gap-2">
                          <button type="button" disabled={busy} onClick={() => { setExecTarget(x); setExecDevice(0); setExecError('') }}>{e.exec}</button>
                          <span className="text-[var(--shell-side-border)]">|</span>
                          <button type="button" disabled={busy} onClick={() => reject(x)}>{e.reject}</button>
                        </span>
                      ) : '—'}
                    </TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={6} loading={busy} text={e.empty} />}
              </TableBody>
            </Table>
          </div>
        )}
        <CardFooter>
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(e)} />
        </CardFooter>
      </Card>
      {open && (
        <Drawer title={e.createTitle} onClose={() => setOpen(false)}
          footer={
            <>
              <ToolbarButton onClick={() => setOpen(false)}>{t.pages.company.cancel}</ToolbarButton>
              <ToolbarButton primary
                disabled={busy || !legalEntityId || !regionId || !portsOk} onClick={submit}>
                {busy ? t.pages.account.submitting : t.pages.company.save}
              </ToolbarButton>
            </>
          }>
          <div className="flex flex-col gap-3.5">
            <div className="flex flex-col gap-1.5">
              <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{e.fCompany}</label>
              <Dropdown
                value={legalEntityId ? String(legalEntityId) : ''}
                options={[{ value: '', label: e.pCompany }, ...companies.map((c) => ({ value: String(c.id), label: c.name }))]}
                onChange={(v) => setLegalEntityId(Number(v) || 0)}
                ariaLabel={e.pCompany}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{e.fRegion}</label>
              <Dropdown
                value={regionId ? String(regionId) : ''}
                options={[{ value: '', label: e.pRegion }, ...regions.map((x) => ({ value: String(x.id), label: x.name }))]}
                onChange={(v) => setRegionId(Number(v) || 0)}
                ariaLabel={e.pRegion}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{e.fPorts}</label>
              <Input className="h-8" type="number" value={expectedPorts} placeholder="0"
                onChange={(ev) => setExpectedPorts(ev.target.value)} />
              {!portsOk && expectedPorts !== '' && <span className="text-[11px] text-[var(--color-danger)]">{e.ePorts}</span>}
            </div>
            {formError && <ErrorBanner message={formError} className="mx-0" />}
          </div>
        </Drawer>
      )}
      {execTarget && (
        <Drawer title={e.execTitle + ' · ' + (execTarget.expansionNo || '#' + execTarget.id)} onClose={() => setExecTarget(null)}
          footer={
            <>
              <ToolbarButton onClick={() => setExecTarget(null)}>{t.pages.company.cancel}</ToolbarButton>
              <ToolbarButton primary
                disabled={busy || !execDevice} onClick={execute}>
                {busy ? t.pages.account.submitting : e.exec}
              </ToolbarButton>
            </>
          }>
          <div className="flex flex-col gap-3.5">
            <div className="flex flex-col gap-1.5">
              <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{e.fDevice}</label>
              <Dropdown
                value={execDevice ? String(execDevice) : ''}
                options={[{ value: '', label: e.pDevice }, ...execDevices.map((d) => ({ value: String(d.id), label: d.name + ' (' + d.code + ')' }))]}
                onChange={(v) => setExecDevice(Number(v) || 0)}
                ariaLabel={e.pDevice}
                triggerStyle={{ minWidth: 200 }}
              />
              {!execDevices.length && <span className="text-[11px] text-[var(--color-danger)]">{e.noDevice}</span>}
            </div>
            {execError && <ErrorBanner message={execError} className="mx-0" />}
          </div>
        </Drawer>
      )}
    </div>
  )
}
