// 新建许可单抽屉(ROW/PECE):与全站表单口径一致——右侧 Drawer + FormField,
// 枚举走 Dropdown(禁原生 select),日期走 DatePicker(禁原生 date),设施走 SimplePicker。
// 日历结构性文案复用 pages.audit.datePicker 三语词条(纯通用月/周/今天/清除,无业务语义)。
import { useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { DatePicker } from '../../../components/DatePicker'
import { Dropdown, type DropdownOption } from '../../../components/Dropdown'
import { SimplePicker } from '../../../components/pickers/SimplePicker'
import { Input } from '../../../components/ui/input'
import { FormField } from '../../../components/business/form-field'
import { ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { SubmitButton, type SubmitState } from '../../../components/business/submit-button'
import { useT } from '../../../i18n'

export interface PermitFormValues {
  kind: string
  title: string
  approvalNo: string
  authority: string
  validFrom: string
  validUntil: string
  facilityCode: string
  note: string
}

export const emptyPermitForm: PermitFormValues = {
  kind: 'ROW', title: '', approvalNo: '', authority: '', validFrom: '', validUntil: '', facilityCode: '', note: '',
}

const KIND_OPTIONS: DropdownOption[] = [
  { value: 'ROW', label: 'ROW 路权' },
  { value: 'PECE', label: 'PECE 许可' },
]

export function PermitCreateDrawer({ facilityOptions, onClose, onCreated }: {
  facilityOptions: DropdownOption[]
  onClose: () => void
  onCreated: () => void
}) {
  const t = useT()
  const o = t.pages.odn
  const cal = t.pages.audit.datePicker
  const [form, setForm] = useState<PermitFormValues>(emptyPermitForm)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const set = <K extends keyof PermitFormValues>(k: K, v: string) => setForm((m) => ({ ...m, [k]: v }))

  const save = async () => {
    if (!form.title.trim()) { setError('名称必填'); return }
    setBusy(true); setError('')
    try {
      await apiFetch('/odn/permits', { method: 'POST', body: { ...form, title: form.title.trim(), projectId: 0 } })
      toast.success('许可单已创建')
      onCreated()
      onClose()
    } catch (e) {
      const msg = e instanceof Error ? e.message : '保存失败'
      setError(msg)
      toast.error('许可单创建失败', { description: msg })
    } finally { setBusy(false) }
  }

  const text = (k: keyof PermitFormValues, label: string, placeholder = '', required = false) => (
    <FormField label={label} required={required}>
      <Input value={form[k]} placeholder={placeholder} onChange={(e) => set(k, e.target.value)} />
    </FormField>
  )
  const date = (k: 'validFrom' | 'validUntil', label: string) => (
    <FormField label={label}>
      <DatePicker value={form[k]} onChange={(v) => set(k, v)} ariaLabel={label} placeholder={label} labels={cal} />
    </FormField>
  )
  const submitState: SubmitState = busy ? 'loading' : error ? 'failed' : 'idle'

  return (
    <Drawer title='新建许可单' onClose={onClose} width={480}
      footer={<>
        <ToolbarButton onClick={onClose} disabled={busy}>取消</ToolbarButton>
        <SubmitButton state={submitState} disabled={busy || !form.title.trim()} onClick={() => void save()}
          labels={{ idle: '保存', loading: '保存中…', success: '已创建', failed: '重试保存' }} />
      </>}>
      <div className='flex flex-col gap-3.5'>
        {error && <ErrorBanner message={error} className='mx-0' />}
        <FormField label='类型' required>
          <Dropdown value={form.kind} ariaLabel='许可类型' options={KIND_OPTIONS} onChange={(v) => set('kind', v)} />
        </FormField>
        {text('title', '名称', '如 人民路架空段路权', true)}
        {text('approvalNo', '批复号', '批准前可留空')}
        {text('authority', '管辖机构', '如 市政公用局')}
        <div className='grid grid-cols-2 gap-3'>
          {date('validFrom', '有效期起')}
          {date('validUntil', '有效期止')}
        </div>
        <FormField label='关联设施'>
          <SimplePicker value={form.facilityCode} onChange={(v) => set('facilityCode', v)} options={facilityOptions}
            ariaLabel={o.pickFacility} searchPlaceholder={o.pickFacilitySearch} clearable
            clearLabel={t.pages.pickers.common.clear} minWidth={240} />
        </FormField>
        {text('note', '备注')}
      </div>
    </Drawer>
  )
}
