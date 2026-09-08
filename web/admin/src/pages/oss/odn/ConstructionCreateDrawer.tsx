// 新建施工单抽屉:W1 页内内联卡片迁入,与全站表单口径统一(右侧 Drawer + FormField)。
// 字段/校验/载荷与迁出前一致;失败错误在抽屉内展示,成功后关抽屉并刷新列表。
import { useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { Input } from '../../../components/ui/input'
import { FormField } from '../../../components/business/form-field'
import { ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { SubmitButton, type SubmitState } from '../../../components/business/submit-button'

export function ConstructionCreateDrawer({ onClose, onCreated }: {
  onClose: () => void
  onCreated: () => void
}) {
  const [projNo, setProjNo] = useState('')
  const [name, setName] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  const save = async () => {
    if (!projNo.trim()) { setError('施工单号必填'); return }
    setBusy(true); setError('')
    try {
      await apiFetch('/odn/constructions', { method: 'POST', body: { projNo: projNo.trim(), name: name.trim() } })
      toast.success('施工单已创建')
      onCreated()
      onClose()
    } catch (e) {
      const msg = e instanceof Error ? e.message : '保存失败'
      setError(msg)
      toast.error('施工单创建失败', { description: msg })
    } finally { setBusy(false) }
  }
  const submitState: SubmitState = busy ? 'loading' : error ? 'failed' : 'idle'

  return (
    <Drawer title='新建施工单' onClose={onClose} width={480}
      footer={<>
        <ToolbarButton onClick={onClose} disabled={busy}>取消</ToolbarButton>
        <SubmitButton state={submitState} disabled={busy || !projNo.trim()} onClick={() => void save()}
          labels={{ idle: '保存', loading: '保存中…', success: '已创建', failed: '重试保存' }} />
      </>}>
      <div className='flex flex-col gap-3.5'>
        {error && <ErrorBanner message={error} className='mx-0' />}
        <FormField label='施工单号' required>
          <Input value={projNo} onChange={(e) => setProjNo(e.target.value)} placeholder="C-20260907-001" />
        </FormField>
        <FormField label='名称'>
          <Input value={name} onChange={(e) => setName(e.target.value)} />
        </FormField>
      </div>
    </Drawer>
  )
}
