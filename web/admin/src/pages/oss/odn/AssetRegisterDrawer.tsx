// 资产化登记抽屉(W8 转固凭证):与全站表单口径一致——右侧 Drawer + FormField,
// 对象/来源枚举走 Dropdown(禁原生 select),设备/资产/项目走 SimplePicker;
// 设备与施工项目静态源开抽屉时一次拉取(全网 ≤500/100 条,本地过滤;原面板内联表单同口径)。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { Dropdown, type DropdownOption } from '../../../components/Dropdown'
import { SimplePicker } from '../../../components/pickers/SimplePicker'
import { Input } from '../../../components/ui/input'
import { FormField } from '../../../components/business/form-field'
import { ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { SubmitButton, type SubmitState } from '../../../components/business/submit-button'
import { useT } from '../../../i18n'

// 选择器数据源行类型(本域列表接口;字段以 internal/domain/odn 与 asset 为准)。
interface DeviceLite { id: number; code: string; name: string; status: string }
interface AssetLite { assetId: number; assetCode: string; status: string; type: string }
interface ProjectLite { id: number; projNo: string; name: string; status: string }

export function AssetRegisterDrawer({ onClose, onCreated }: {
  onClose: () => void
  onCreated: () => void
}) {
  const o = useT().pages.odn
  const [entityKind, setEntityKind] = useState('FACILITY')
  const [facilityCode, setFacilityCode] = useState('')
  const [deviceId, setDeviceId] = useState('')
  const [assetId, setAssetId] = useState('')
  const [sourceKind, setSourceKind] = useState('DIRECT')
  const [projectId, setProjectId] = useState('')
  const [valueAmount, setValueAmount] = useState('')
  const [remark, setRemark] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [deviceOpts, setDeviceOpts] = useState<DropdownOption[]>([])
  const [projectOpts, setProjectOpts] = useState<DropdownOption[]>([])

  // 设备/施工项目静态源一次拉取(口径同原面板内联表单)。
  useEffect(() => {
    void (async () => {
      try {
        const devs = (await apiFetch<DeviceLite[]>('/odn/devices')) ?? []
        setDeviceOpts(devs.map((d) => ({ value: String(d.id), label: (d.name || d.code) + ' (' + d.code + ')' })))
      } catch { setDeviceOpts([]) }
      try {
        const projs = (await apiFetch<ProjectLite[]>('/odn/constructions', { query: { limit: 100 } })) ?? []
        setProjectOpts(projs.map((p) => ({ value: String(p.id), label: p.projNo + ' ' + (p.name || '') + ' [' + (o.projStatus[p.status as keyof typeof o.projStatus] ?? p.status) + ']' })))
      } catch { setProjectOpts([]) }
    })()
  }, [])

  // 资产服务端检索(/assets 分页接口 q 关键字;label 标注状态便于识别可登记状态)。
  const searchAssets = async (kw: string): Promise<DropdownOption[] | null> => {
    const d = await apiFetch<{ items: AssetLite[] }>('/assets', { query: { q: kw || undefined, limit: 50 } })
    return (d?.items ?? []).map((a) => ({ value: String(a.assetId), label: a.assetCode + ' #' + a.assetId + ' [' + a.status + ']' }))
  }

  const save = async () => {
    const body: Record<string, unknown> = { entityKind, sourceKind, assetId: Number(assetId) || 0 }
    if (entityKind === 'FACILITY') body.facilityCode = facilityCode.trim(); else body.deviceId = Number(deviceId) || 0
    if (sourceKind === 'CONSTRUCTION') body.projectId = Number(projectId) || 0
    if (valueAmount !== '') body.valueAmount = Number(valueAmount) || 0
    if (remark.trim()) body.remark = remark.trim()
    setBusy(true); setError('')
    try {
      await apiFetch('/odn/assets/registrations', { method: 'POST', body })
      toast.success('已登记资产化凭证,资产转为 DEPLOYED')
      onCreated()
      onClose()
    } catch (e) {
      const msg = e instanceof Error ? e.message : '登记失败'
      setError(msg)
      toast.error('凭证登记失败', { description: msg })
    } finally { setBusy(false) }
  }
  const formOk = assetId !== '' && (entityKind === 'FACILITY' ? facilityCode.trim() !== '' : deviceId !== '')
  const submitState: SubmitState = busy ? 'loading' : error ? 'failed' : 'idle'

  return (
    <Drawer title='新建资产化凭证' onClose={onClose} width={480}
      footer={<>
        <ToolbarButton onClick={onClose} disabled={busy}>取消</ToolbarButton>
        <SubmitButton state={submitState} disabled={busy || !formOk} onClick={() => void save()}
          labels={{ idle: '登记', loading: '登记中…', success: '已登记', failed: '重试登记' }} />
      </>}>
      <div className='flex flex-col gap-3.5'>
        {error && <ErrorBanner message={error} className='mx-0' />}
        <FormField label='对象类型'>
          <Dropdown value={entityKind} ariaLabel="对象类型" options={[{ value: 'FACILITY', label: '设施' }, { value: 'DEVICE', label: '设备' }]} onChange={(v) => { setEntityKind(v); setFacilityCode(''); setDeviceId('') }} />
        </FormField>
        {entityKind === 'FACILITY'
          ? <FormField label='设施编码'><Input value={facilityCode} onChange={(e) => setFacilityCode(e.target.value)} placeholder="P01001 / CLS00001" /></FormField>
          : <FormField label='设备 ID'><SimplePicker value={deviceId} onChange={setDeviceId} options={deviceOpts} ariaLabel={o.pickDevice} searchPlaceholder={o.pickDeviceSearch} minWidth={200} /></FormField>}
        <FormField label='资产 ID(须 IN_STOCK/IN_TRANSIT)'>
          <SimplePicker value={assetId} onChange={setAssetId} search={searchAssets} ariaLabel={o.pickAsset} searchPlaceholder={o.pickAssetSearch} minWidth={220} />
        </FormField>
        <FormField label='来源'>
          <Dropdown value={sourceKind} ariaLabel="来源" options={[{ value: 'DIRECT', label: '直购直转' }, { value: 'PROCUREMENT', label: '采购入库' }, { value: 'CONSTRUCTION', label: '施工建成' }]} onChange={setSourceKind} />
        </FormField>
        {sourceKind === 'CONSTRUCTION' && <FormField label='施工项目 ID(须已竣工)'>
          <SimplePicker value={projectId} onChange={setProjectId} options={projectOpts} ariaLabel={o.pickProject} searchPlaceholder={o.pickProjectSearch} minWidth={240} />
        </FormField>}
        <FormField label='转固价值'>
          <Input value={valueAmount} onChange={(e) => setValueAmount(e.target.value)} placeholder="0.00" inputMode="decimal" />
        </FormField>
        <FormField label='备注'>
          <Input value={remark} onChange={(e) => setRemark(e.target.value)} placeholder="可空" />
        </FormField>
      </div>
    </Drawer>
  )
}
