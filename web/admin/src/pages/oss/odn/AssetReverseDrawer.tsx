// 冲销抽屉(W8 转固凭证):危险写入右侧抽屉承载——凭证号只读预填,原因必填;
// 提交链保持原顺序:原因校验 → 二次确认 → POST reverse(接口与载荷不变)。
import { useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { Input } from '../../../components/ui/input'
import { FormField } from '../../../components/business/form-field'
import { ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { SubmitButton, type SubmitState } from '../../../components/business/submit-button'
import { useConfirm } from '../../../components/ConfirmDialog'

export function AssetReverseDrawer({ id, registrationNo, onClose, onReversed }: {
  id: number
  registrationNo: string
  onClose: () => void
  onReversed: () => void
}) {
  const confirmDialog = useConfirm()
  const [reason, setReason] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const save = async () => {
    if (!reason.trim()) { setError('冲销原因必填'); return }
    if (!(await confirmDialog('确认冲销该凭证?冲销后资产回 IN_STOCK,凭证保留历史。', { danger: true }))) return
    setBusy(true); setError('')
    try {
      await apiFetch('/odn/assets/registrations/' + id + '/reverse', { method: 'POST', body: { reason: reason.trim() } })
      toast.success('凭证已冲销')
      onReversed()
      onClose()
    } catch (e) {
      const msg = e instanceof Error ? e.message : '冲销失败'
      setError(msg)
      toast.error('凭证冲销失败', { description: msg })
    } finally { setBusy(false) }
  }
  const submitState: SubmitState = busy ? 'loading' : error ? 'failed' : 'idle'

  return (
    <Drawer title={'冲销 ' + registrationNo} onClose={onClose} width={480}
      footer={<>
        <ToolbarButton onClick={onClose} disabled={busy}>取消</ToolbarButton>
        <SubmitButton state={submitState} danger disabled={busy || !reason.trim()} onClick={() => void save()}
          labels={{ idle: '冲销', loading: '冲销中…', success: '已冲销', failed: '重试冲销' }} />
      </>}>
      <div className='flex flex-col gap-3.5'>
        {error && <ErrorBanner message={error} className='mx-0' />}
        <FormField label='凭证号'>
          <Input value={registrationNo} readOnly disabled />
        </FormField>
        <FormField label='冲销原因' required>
          <Input value={reason} onChange={(e) => setReason(e.target.value)} placeholder="冲销原因(必填)" />
        </FormField>
      </div>
    </Drawer>
  )
}
