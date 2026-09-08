// 覆盖关联登记抽屉(P1 覆盖关联,迁移 000197):与全站表单口径一致——右侧 Drawer + FormField,
// 地址=服务端检索、设施/设备=本城市主数据静态源(经 props 注入,原面板 loadRefs 同源),
// 状态枚举走 Dropdown(禁原生 select);文案全走 pages.odn 三语词条。
import { useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { SimplePicker } from '../../../components/pickers/SimplePicker'
import { Input } from '../../../components/ui/input'
import { FormField } from '../../../components/business/form-field'
import { ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { SubmitButton, type SubmitState } from '../../../components/business/submit-button'
import { useT } from '../../../i18n'

type Opt = { value: string; label: string }

export function CoverageRegisterDrawer({ facOpts, devOpts, searchAddresses, onClose, onCreated }: {
  facOpts: Opt[]
  devOpts: Opt[]
  searchAddresses: (kw: string) => Promise<Opt[]>
  onClose: () => void
  onCreated: () => void
}) {
  const t = useT()
  const g = t.pages.odn
  const clear = t.pages.pickers.common.clear
  const [addressId, setAddressId] = useState('')
  const [facilityCode, setFacilityCode] = useState('')
  const [deviceId, setDeviceId] = useState('')
  const [status, setStatus] = useState('SERVED')
  const [note, setNote] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const save = async () => {
    setBusy(true); setError('')
    try {
      await apiFetch('/odn/coverage', { method: 'POST', body: {
        addressId: Number(addressId),
        facilityCode: facilityCode || undefined,
        deviceId: deviceId ? Number(deviceId) : undefined,
        status: status, note: note } })
      toast.success(g.saveOk)
      onCreated()
      onClose()
    } catch (e) {
      const msg = e instanceof Error ? e.message : g.saveFail
      setError(msg)
      toast.error(g.saveFail, { description: msg })
    } finally { setBusy(false) }
  }
  const submitState: SubmitState = busy ? 'loading' : error ? 'failed' : 'idle'
  const statusOptions = [
    { value: 'SERVED', label: g.covServed },
    { value: 'PENDING', label: g.covPending },
    { value: 'UNSERVED', label: g.covUnserved },
  ]

  return (
    <Drawer title='登记覆盖关联' onClose={onClose} width={480}
      footer={<>
        <ToolbarButton onClick={onClose} disabled={busy}>{g.cancel}</ToolbarButton>
        <SubmitButton state={submitState} disabled={busy || !addressId} onClick={() => void save()}
          labels={{ idle: g.save, loading: g.saving, success: g.saveOk, failed: g.saveFail }} />
      </>}>
      <div className='flex flex-col gap-3.5'>
        {error && <ErrorBanner message={error} className='mx-0' />}
        <FormField label={g.addressId} required>
          <SimplePicker value={addressId} onChange={setAddressId} search={searchAddresses} ariaLabel={g.addressId} placeholder={g.addressId} searchPlaceholder={g.addressId} minWidth={180} />
        </FormField>
        <FormField label={g.covFacility}>
          <SimplePicker value={facilityCode} onChange={setFacilityCode} options={facOpts} ariaLabel={g.covFacility} placeholder={g.covFacility} clearable clearLabel={clear} minWidth={180} />
        </FormField>
        <FormField label={g.covDevice}>
          <SimplePicker value={deviceId} onChange={setDeviceId} options={devOpts} ariaLabel={g.covDevice} placeholder={g.covDevice} clearable clearLabel={clear} minWidth={180} />
        </FormField>
        <FormField label={g.covStatus}>
          <Dropdown value={status} options={statusOptions} ariaLabel={g.covStatus} onChange={setStatus} />
        </FormField>
        <FormField label={g.covNote}>
          <Input value={note} onChange={(e) => setNote(e.target.value)} />
        </FormField>
      </div>
    </Drawer>
  )
}
